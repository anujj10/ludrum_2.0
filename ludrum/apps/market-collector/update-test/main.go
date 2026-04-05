package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "ludrum/configs"

	"ludrum/internal/api"
	"ludrum/internal/cache"
	optionData "ludrum/internal/ingestion/data"
	ltpSeries "ludrum/internal/ltp-series"
	"ludrum/internal/parser"
	"ludrum/internal/processor"
	"ludrum/internal/simulator"
	"ludrum/internal/storage/postgres"
	"ludrum/internal/storage/redis"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
)

func main() {

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "test"
	}

	log.Println("🚀 Starting Demo mode...")

	ctx, cancel := context.WithCancel(context.Background())

	// ==========================
	// INIT
	// ==========================
	postgres.InitDB()

	cfg := config.LoadConfig()
	fyModel := optionData.CreateFyersModel(cfg.AppID, cfg.AccessToken)

	state := cache.NewEngineState()
	redisClient := redis.NewRedisClient("localhost:6379", mode)

	dbWorker := postgres.NewDBWorker()
	dbWorker.Start()

	pipeline := processor.NewPipeline()
	sim := simulator.NewSimulator()

	// ==========================
	// 🔥 LTP SERIES ENGINE
	// ==========================
	atmTracker := ltpSeries.NewATMTracker(50)

	selector := ltpSeries.NewStrikeSelector(
		2,
		"24APR",
		50,
	)

	ltpStore := ltpSeries.NewMarketStore(5)
	fetcher := ltpSeries.NewFyersFetcher(fyModel)
	poller := ltpSeries.NewLTPPoller(fetcher, ltpStore)

	ltpEngine := ltpSeries.NewLTPEngine(atmTracker, selector, poller)

	// start poller
	ltpEngine.Start(ctx)

	// ==========================
	// 🔥 LTP → ALPHA → REDIS LOOP
	// ==========================
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

	// ==========================
	// EXISTING MARKET STORE
	// ==========================
	store := processor.NewMarketStore()

	// ==========================
	// API + WS
	// ==========================
	apiServer := api.NewServer(state, nil)
	apiServer.Start()

	api.StartWS(redisClient, "8082", nil)

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

	wsIngestor.SetOnTick(func(ltp float64) {
		ltpEngine.OnIndexTick(ltp)
	})

	go wsIngestor.Start(ctx, cfg.AppID, cfg.AccessToken, []string{"NSE:NIFTY50-INDEX"})

	// ==========================
	// REST SNAPSHOT PIPELINE LOOP
	// ==========================
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			runLivePipeline(fyModel, pipeline, state, redisClient, dbWorker, sim)
		}
	}()

	// ==========================
	// SHUTDOWN
	// ==========================
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

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
) {

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered from pipeline panic:", r)
		}
	}()

	start := time.Now()

	raw, err := optionData.OptionsData("NSE:NIFTY50-INDEX", "", 3, fyModel)
	if err != nil || raw == "" {
		log.Println("Empty/failed API response")
		return
	}

	parsed, err := parser.ParseOptionChain([]byte(raw))
	if parsed == nil || len(parsed.Data.OptionsChain) == 0 {
		log.Println("Invalid parsed data")
		return
	}

	// ==========================
	// BUILD SNAPSHOT
	// ==========================
	snap := processor.BuildMarketSnapshot(parsed)
	snap.Timestamp = time.Now().Unix()

	// ==========================
	// OPTIONS CHAIN (UI RAW DATA)
	// ==========================
	optionsChain := processor.BuildOptionsChainPayload(snap, 200)

	data, _ := json.Marshal(optionsChain)
	mode := redisClient.GetPrefix()

	// store latest options chain
	redisClient.Client.Set(context.Background(), mode+":latest:options_chain", data, 0)

	// publish options chain
	redisClient.Publish("options_chain", optionsChain)

	// ==========================
	// 🔥 MAIN PIPELINE
	// ==========================
	payload := pipeline.Process(snap, sim)

	// 🔥 SEND SNAPSHOT / DELTA (CORE FIX)
	redisClient.PublishPayloadWithType(payload, payload.Type)

	// ==========================
	// DB + STATE
	// ==========================
	dbWorker.PairChan <- payload.Pairs
	state.UpdatePairs(payload.Pairs, snap.Timestamp)

	// ==========================
	// LOGS
	// ==========================
	log.Println("BUILDING OPTIONS CHAIN")
	log.Println("Options count:", len(parsed.Data.OptionsChain))
	log.Println("Strikes built:", len(snap.Strikes))
	log.Println("Pairs to DB:", len(payload.Pairs))
	log.Printf("Snapshot Time: %v\n", time.Since(start))
}
