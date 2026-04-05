package runtime

import (
	"context"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ludrum/internal/broker/fyers"
	"ludrum/internal/models"
	"ludrum/internal/processor"
	"ludrum/internal/simulator"
	"ludrum/internal/storage/postgres"
	"ludrum/internal/storage/types"
)

type UserRuntime struct {
	mu         sync.RWMutex
	config     fyers.RuntimeConfig
	state      string
	lastError  string
	lastTickAt *time.Time
	lastWSConnectAt *time.Time
	latestPayload models.StreamPayload
	hasPayload bool
	revision uint64
	cancel     context.CancelFunc
	pipeline   *processor.Pipeline
	sim        *simulator.Simulator
	oiEvents   map[string][]OIEvent
}

func NewUserRuntime(config fyers.RuntimeConfig) *UserRuntime {
	return &UserRuntime{
		config: config,
		state:  "pending",
		pipeline: processor.NewPipeline(),
		sim: simulator.NewSimulator(),
		oiEvents: make(map[string][]OIEvent),
	}
}

func (r *UserRuntime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		return nil
	}

	runtimeCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	now := time.Now().UTC()
	r.state = "starting"
	r.lastWSConnectAt = &now

	go func() {
		r.run(runtimeCtx)
	}()

	return nil
}

func (r *UserRuntime) Stop() {
	r.mu.RLock()
	cancel := r.cancel
	r.mu.RUnlock()

	if cancel != nil {
		cancel()
	}
}

func (r *UserRuntime) ApplyConfig(config fyers.RuntimeConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
}

func (r *UserRuntime) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return Snapshot{
		UserID:          r.config.UserID,
		AccountID:       r.config.AccountID,
		State:           r.state,
		LastError:       r.lastError,
		LastTickAt:      r.lastTickAt,
		LastWSConnectAt: r.lastWSConnectAt,
	}
}

func (r *UserRuntime) SetLastError(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastError = message
}

func (r *UserRuntime) MarkTick(ts time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastTickAt = &ts
}

func (r *UserRuntime) GetPairs() ([]models.PairSignal, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.hasPayload {
		return nil, false
	}

	return append([]models.PairSignal(nil), r.latestPayload.Pairs...), true
}

func (r *UserRuntime) LatestPayload() (models.StreamPayload, uint64, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.hasPayload {
		return models.StreamPayload{}, 0, false
	}

	payload := r.latestPayload
	payload.Pairs = append([]models.PairSignal(nil), r.latestPayload.Pairs...)
	payload.OpenPositions = append([]simulator.Position(nil), r.latestPayload.OpenPositions...)
	payload.ClosedPositions = append([]*simulator.Position(nil), r.latestPayload.ClosedPositions...)
	return payload, r.revision, true
}

type Snapshot struct {
	UserID          int64
	AccountID       int64
	State           string
	LastError       string
	LastTickAt      *time.Time
	LastWSConnectAt *time.Time
}

type OIEvent struct {
	Time       time.Time `json:"time"`
	Symbol     string    `json:"symbol"`
	Strike     float64   `json:"strike"`
	OptionType string    `json:"option_type"`
	OIChange   int64     `json:"oi_change"`
	LTPChange  float64   `json:"ltp_change"`
}

func (r *UserRuntime) run(ctx context.Context) {
	defer func() {
		r.mu.Lock()
		r.state = "stopped"
		r.cancel = nil
		r.mu.Unlock()
		_, _ = postgres.UpsertUserRuntimeStatus(ctx, r.config.UserID, r.config.AccountID, "stopped", r.lastWSConnectAt, r.lastTickAt, r.lastError)
	}()

	if err := r.writeRuntimeStatus(ctx, "running", ""); err != nil {
		log.Printf("runtime status update failed for user %d: %v", r.config.UserID, err)
	}

	interval := r.config.PollInterval
	if interval <= 0 {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := r.runCycle(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			r.SetLastError(err.Error())
			_ = r.writeRuntimeStatus(ctx, "degraded", err.Error())
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *UserRuntime) runCycle(ctx context.Context) error {
	root := strings.TrimSpace(r.config.OptionChainRoot)
	if root == "" {
		root = "NSE:NIFTY50-INDEX"
	}

	strikeCount := r.config.StrikeCount
	if strikeCount <= 0 {
		strikeCount = 3
	}

	client := fyers.NewAPIClient(strings.TrimSpace(r.config.AppID), strings.TrimSpace(r.config.AccessToken))
	payload, err := client.FetchOptionChain(ctx, root, "", strikeCount)
	if err != nil {
		return err
	}
	if payload == nil || len(payload.Data.OptionsChain) == 0 {
		return context.DeadlineExceeded
	}

	snap := processor.BuildMarketSnapshot(payload)
	if snap == nil {
		return context.DeadlineExceeded
	}
	snap.Timestamp = time.Now().Unix()

	streamPayload := r.pipeline.Process(snap, r.sim)

	now := time.Now().UTC()
	newEvents := r.recordOIEvents(snap.Symbol, now, streamPayload.Pairs)

	r.mu.Lock()
	r.state = "running"
	r.lastError = ""
	r.lastTickAt = &now
	r.latestPayload = streamPayload
	r.hasPayload = true
	r.revision++
	r.mu.Unlock()

	if err := postgres.SaveUserRuntimeSnapshot(ctx, r.config.UserID, r.config.AccountID, streamPayload); err != nil && ctx.Err() == nil {
		log.Printf("failed to persist runtime snapshot for user %d: %v", r.config.UserID, err)
	}

	if len(newEvents) > 0 {
		scopedEvents := make([]types.UserScopedOIChangeEvent, 0, len(newEvents))
		for _, event := range newEvents {
			scopedEvents = append(scopedEvents, types.UserScopedOIChangeEvent{
				UserID:    r.config.UserID,
				AccountID: r.config.AccountID,
				DBOIChangeEvent: types.DBOIChangeEvent{
					Time:       event.Time,
					Symbol:     event.Symbol,
					Strike:     event.Strike,
					OptionType: event.OptionType,
					OIChange:   event.OIChange,
					LTPChange:  event.LTPChange,
				},
			})
		}
		if err := postgres.SaveUserRuntimeOIEvents(ctx, scopedEvents); err != nil && ctx.Err() == nil {
			log.Printf("failed to persist runtime OI events for user %d: %v", r.config.UserID, err)
		}
	}

	return r.writeRuntimeStatus(ctx, "running", "")
}

func (r *UserRuntime) writeRuntimeStatus(ctx context.Context, state, lastError string) error {
	r.mu.RLock()
	lastWSConnectAt := r.lastWSConnectAt
	lastTickAt := r.lastTickAt
	r.mu.RUnlock()

	_, err := postgres.UpsertUserRuntimeStatus(ctx, r.config.UserID, r.config.AccountID, state, lastWSConnectAt, lastTickAt, lastError)
	return err
}

func (r *UserRuntime) GetOIEvents(symbol string, strikes []float64, limit int) []OIEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 12
	}

	result := make([]OIEvent, 0, len(strikes)*2*limit)
	for _, strike := range strikes {
		for _, optionType := range []string{"CE", "PE"} {
			key := oiEventKey(symbol, strike, optionType)
			entries := r.oiEvents[key]
			if len(entries) == 0 {
				continue
			}
			result = append(result, recentOIEvents(entries, limit)...)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Strike == result[j].Strike {
			if result[i].OptionType == result[j].OptionType {
				return result[i].Time.After(result[j].Time)
			}
			return result[i].OptionType < result[j].OptionType
		}
		return result[i].Strike < result[j].Strike
	})

	return result
}

func (r *UserRuntime) recordOIEvents(symbol string, ts time.Time, pairs []models.PairSignal) []OIEvent {
	newEvents := make([]OIEvent, 0, len(pairs)*2)

	for _, pair := range pairs {
		if pair.CE.OIChange != 0 {
			event := OIEvent{
				Time:       ts,
				Symbol:     symbol,
				Strike:     pair.Strike,
				OptionType: "CE",
				OIChange:   pair.CE.OIChange,
				LTPChange:  pair.CE.LTPChange,
			}
			key := oiEventKey(symbol, pair.Strike, "CE")
			updated, appended := appendOIEvent(r.oiEvents[key], event)
			r.oiEvents[key] = updated
			if appended {
				newEvents = append(newEvents, event)
			}
		}
		if pair.PE.OIChange != 0 {
			event := OIEvent{
				Time:       ts,
				Symbol:     symbol,
				Strike:     pair.Strike,
				OptionType: "PE",
				OIChange:   pair.PE.OIChange,
				LTPChange:  pair.PE.LTPChange,
			}
			key := oiEventKey(symbol, pair.Strike, "PE")
			updated, appended := appendOIEvent(r.oiEvents[key], event)
			r.oiEvents[key] = updated
			if appended {
				newEvents = append(newEvents, event)
			}
		}
	}

	return newEvents
}

func appendOIEvent(existing []OIEvent, next OIEvent) ([]OIEvent, bool) {
	if len(existing) > 0 {
		last := existing[len(existing)-1]
		if last.OIChange == next.OIChange && last.LTPChange == next.LTPChange {
			return existing, false
		}
	}

	existing = append(existing, next)
	const maxOIEvents = 24
	if len(existing) > maxOIEvents {
		existing = existing[len(existing)-maxOIEvents:]
	}
	return existing, true
}

func recentOIEvents(entries []OIEvent, limit int) []OIEvent {
	start := 0
	if len(entries) > limit {
		start = len(entries) - limit
	}
	out := append([]OIEvent(nil), entries[start:]...)
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}

func oiEventKey(symbol string, strike float64, optionType string) string {
	return symbol + "|" + optionType + "|" + strconv.FormatFloat(strike, 'f', 2, 64)
}
