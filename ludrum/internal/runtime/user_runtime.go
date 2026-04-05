package runtime

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"ludrum/internal/broker/fyers"
	"ludrum/internal/models"
	"ludrum/internal/processor"
	"ludrum/internal/simulator"
	"ludrum/internal/storage/postgres"
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
}

func NewUserRuntime(config fyers.RuntimeConfig) *UserRuntime {
	return &UserRuntime{
		config: config,
		state:  "pending",
		pipeline: processor.NewPipeline(),
		sim: simulator.NewSimulator(),
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
	r.mu.Lock()
	r.state = "running"
	r.lastError = ""
	r.lastTickAt = &now
	r.latestPayload = streamPayload
	r.hasPayload = true
	r.revision++
	r.mu.Unlock()

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
