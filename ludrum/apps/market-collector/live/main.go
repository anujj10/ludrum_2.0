package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	config "ludrum/configs"

	"ludrum/internal/api"
	"ludrum/internal/cache"
	optionData "ludrum/internal/ingestion/data"
	ltpSeries "ludrum/internal/ltp-series"
	"ludrum/internal/parser"
	"ludrum/internal/processor"
	"ludrum/internal/runtime"
	"ludrum/internal/simulator"
	"ludrum/internal/storage/postgres"
	"ludrum/internal/storage/redis"
	storageTypes "ludrum/internal/storage/types"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
)

var indiaLocation = mustLoadLocation("Asia/Kolkata")

type snapshotCache struct {
	mu     sync.RWMutex
	latest []storageTypes.DBOption
	ok     bool
}

func (c *snapshotCache) Set(options []storageTypes.DBOption) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latest = append([]storageTypes.DBOption(nil), options...)
	c.ok = true
}

func (c *snapshotCache) Get() ([]storageTypes.DBOption, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]storageTypes.DBOption(nil), c.latest...), c.ok
}

func main() {

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "live"
	}
	log.Println("🚀 Starting LIVE mode...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ==========================
	// INIT
	// ==========================
	postgres.InitDB()

	cfg := config.LoadConfig()
	fyModel := optionData.CreateFyersModel(cfg.AppID, cfg.AccessToken)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	state := cache.NewEngineState()
	redisClient := redis.NewRedisClient(redisAddr, mode)
	dbWorker := postgres.NewDBWorker()
	dbWorker.Start()
	runtimeManager := runtime.NewManager()

	pipeline := processor.NewPipeline()
	sim := simulator.NewSimulator()
	liveSnapshotCache := &snapshotCache{}

	// ==========================
	// 🔥 LTP SERIES ENGINE
	// ==========================
	atmTracker := ltpSeries.NewATMTracker(50)

	selector := ltpSeries.NewStrikeSelector(
		2,       // ±2 strikes
		"24APR", // expiry
		50,
	)

	ltpStore := ltpSeries.NewMarketStore(5)
	fetcher := ltpSeries.NewFyersFetcher(fyModel)
	poller := ltpSeries.NewLTPPoller(fetcher, ltpStore)

	ltpEngine := ltpSeries.NewLTPEngine(atmTracker, selector, poller)
	isMarketHours := isIndianMarketOpen(time.Now())

	// start poller once
	if isMarketHours {
		ltpEngine.Start(ctx)
	}

	// ==========================
	// 🔥 LTP → ALPHA → REDIS LOOP
	// ==========================
	if isMarketHours {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case <-ticker.C:

					for symbol := range poller.GetTrackedSymbols() {

						state := ltpStore.GetState(symbol)
						if state == nil {
							continue
						}

						alpha := processor.ComputeAlphaFromLTP(state.History)

						payload := map[string]interface{}{
							"symbol": symbol,
							"alpha":  alpha,
						}
						log.Println("LTP ALPHA:", symbol, alpha.Signal)

						redisClient.PublishLTP(payload)
					}
				}
			}
		}()
	}

	// ==========================
	// EXISTING MARKET STORE
	// ==========================
	store := processor.NewMarketStore()

	// ==========================
	// API + WS
	// ==========================
	apiServer := api.NewServer(state, runtimeManager)
	apiServer.Start()
	tradeAPI := &api.TradeAPI{Sim: sim}
	tradeAPI.RegisterRoutes(http.DefaultServeMux)
	api.RegisterOIEventRoutes(http.DefaultServeMux)
	api.RegisterAuthRoutes(http.DefaultServeMux)
	api.RegisterBrokerRoutesWithRuntime(http.DefaultServeMux, runtimeManager)
	api.StartWS(redisClient, "8081", runtimeManager)

	// ==========================
	// WS INGESTOR
	// ==========================
	wsIngestor := optionData.NewWSIngestor(
		store,
		pipeline,
		state,
		redisClient,
		dbWorker,
		sim,
	)

	// connect WS → LTP engine
	wsIngestor.SetOnTick(func(ltp float64) {
		ltpEngine.OnIndexTick(ltp)
	})

	if isMarketHours {
		go wsIngestor.Start(ctx, cfg.AppID, cfg.AccessToken, []string{"NSE:NIFTY50-INDEX"})

		// ==========================
		// REST FETCH + UI BROADCAST (1s)
		// ==========================
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					runLivePipeline(fyModel, pipeline, state, redisClient, dbWorker, sim, liveSnapshotCache)
				}
			}
		}()

		// ==========================
		// MARKET SNAPSHOT PERSIST (1m)
		// ==========================
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					snapshotRows, ok := liveSnapshotCache.Get()
					if !ok {
						continue
					}
					go postgres.SaveMinuteSnapshots(snapshotRows)
					if len(snapshotRows) > 0 {
						log.Printf("Persisted %d market snapshot rows @ %s", len(snapshotRows), snapshotRows[0].Time.Format("15:04:05"))
					}
				}
			}
		}()
	} else {
		log.Println("Market closed: auth/API stays online, live collectors paused.")
	}

	// ==========================
	// SHUTDOWN
	// ==========================
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received shutdown signal: %v", sig)
	}

	log.Println("Shutting down system...")
	cancel()
	wsIngestor.Stop()
	log.Println("Shutdown complete")
}

// ==========================
// SNAPSHOT PIPELINE
// ==========================
func runLivePipeline(
	fyModel *fyersgosdk.FyersModel,
	pipeline *processor.Pipeline,
	state *cache.EngineState,
	redisClient *redis.RedisClient,
	dbWorker *postgres.DBWorker,
	sim *simulator.Simulator,
	liveSnapshotCache *snapshotCache,
) {
	if fyModel == nil || pipeline == nil || state == nil || redisClient == nil || dbWorker == nil || sim == nil || liveSnapshotCache == nil {
		log.Println("Pipeline skipped: required dependency is nil")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println(" Recovered from pipeline panic:", r)
		}
	}()
	start := time.Now()

	raw, err := optionData.OptionsData("NSE:NIFTY50-INDEX", "", 3, fyModel)
	if err != nil {
		log.Println("API error:", err)
		return
	}
	if raw == "" {
		log.Println("Empty/failed API response")
		return
	}

	parsed, err := parser.ParseOptionChain([]byte(raw))
	if err != nil {
		log.Println("Parse error:", err)
		return
	}
	if parsed == nil || len(parsed.Data.OptionsChain) == 0 {
		log.Println(" Invalid parsed data")
		return
	}
	// ==========================
	// HOT PATH DB EVENT TRACKING
	// ==========================
	dbOptions := postgres.ConvertToDBOptions(parsed.Data.OptionsChain)
	liveSnapshotCache.Set(dbOptions)
	go postgres.SaveOptionsBatch(dbOptions)
	go postgres.SaveOptionsAndOIEvents(dbOptions)

	// ==========================
	// EXISTING PIPELINE (UNCHANGED)
	// ==========================
	snap := processor.BuildMarketSnapshot(parsed)
	if snap == nil {
		log.Println("Snapshot build failed")
		return
	}
	snap.Timestamp = time.Now().Unix()

	// ==========================
	// 🔥 BUILD OPTIONS CHAIN
	// ==========================
	optionsChain := processor.BuildOptionsChainPayload(snap, 200)

	// publish to redis
	data, _ := json.Marshal(optionsChain)

	// store latest
	mode := "live"
	redisClient.Client.Set(context.Background(), mode+"latest:options_chain", data, 0)

	// publish
	redisClient.Publish("options_chain", optionsChain)

	payloadRaw := pipeline.Process(snap, sim)
	go redisClient.PublishPayloadWithType(payloadRaw, payloadRaw.Type)
	go redisClient.PublishPayloadStream(payloadRaw)

	// redisClient.Publish("options_chain", map[string]interface{}{
	// "debug": "hello_from_backend",
	// "time":  time.Now().Unix(),
	// })

	// (optional: keep for future analytics DB)
	dbWorker.PairChan <- payloadRaw.Pairs

	state.UpdatePairs(payloadRaw.Pairs, snap.Timestamp)
	log.Println(" BUILDING OPTIONS CHAIN")
	optionsChain2 := processor.BuildOptionsChainPayload(snap, 200)
	log.Println(" PUBLISHING options_chain", len(optionsChain2.Strikes))

	log.Println("Options count:", len(parsed.Data.OptionsChain))
	log.Println("Strikes built:", len(snap.Strikes))
	log.Println("Pairs to DB:", len(payloadRaw.Pairs))
	log.Printf("Snapshot Time: %v\n", time.Since(start))
}

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		log.Fatalf("failed to load timezone %s: %v", name, err)
	}
	return location
}

func isIndianMarketOpen(now time.Time) bool {
	local := now.In(indiaLocation)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return false
	}

	minutes := local.Hour()*60 + local.Minute()
	openMinutes := 9*60 + 15
	closeMinutes := 15*60 + 30

	return minutes >= openMinutes && minutes < closeMinutes
}
