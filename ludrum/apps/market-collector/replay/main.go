package main

import (
	"context"
	"log"
	"net/http"
	"time"

	config "ludrum/configs"

	"ludrum/internal/api"
	"ludrum/internal/cache"
	payloadbuilder "ludrum/internal/engine/payload-builder"
	"ludrum/internal/models"

	"ludrum/internal/processor"

	"ludrum/internal/replay"
	"ludrum/internal/simulator"
	"ludrum/internal/storage/postgres"
	"ludrum/internal/storage/redis"
	"os"
)

func main() {
	log.Println("🎬 Starting REPLAY mode...")
		mode := os.Getenv("MODE")
	if mode == "" {
		mode = "test"
	}

	// ================= INIT =================
	postgres.InitDB()
	_ = config.LoadConfig()

	state := cache.NewEngineState()
	redisClient := redis.NewRedisClient("localhost:6379", mode)
	dbWorker := postgres.NewDBWorker()
	dbWorker.Start()

	pipeline := processor.NewPipeline()
	sim := simulator.NewSimulator()

	// 🔥 NEW: payload builder
	builder := payloadbuilder.NewBuilder(300000, 0.01)

	// ================= TRADE API =================
	tradeAPI := &api.TradeAPI{
		Sim: sim,
	}

	go func() {
		log.Println("🚀 Trading API running on :8082")

		mux := http.NewServeMux()
		tradeAPI.RegisterRoutes(mux)

		err := http.ListenAndServe(":8082", mux)
		if err != nil {
			log.Fatal("Trading API error:", err)
		}
	}()

	// ================= API + WS =================
	apiServer := api.NewServer(state)
	apiServer.Start()
	api.StartWS(redisClient, "8083")

	// ================= REPLAY CONFIG =================
	ctx := context.Background()

	startTime := time.Date(2026, 3, 25, 9, 21, 0, 0, time.Local)
	endTime := time.Date(2026, 3, 25, 13, 39, 0, 0, time.Local)

	speed := 2 * time.Second

	engine := replay.NewReplayEngine(
		postgres.DB,
		"NIFTY",
		startTime,
		endTime,
		speed,
	)

	// ================= START REPLAY =================
	go engine.Start(ctx, func(snap *models.MarketSnapshot) {
		runPipelineWithSnapshot(
			snap,
			pipeline,
			builder, // 🔥 pass builder
			state,
			redisClient,
			dbWorker,
			sim,
		)
	})

	select {}
}
// ================= PIPELINE =================

func runPipelineWithSnapshot(
	snap *models.MarketSnapshot,
	pipeline *processor.Pipeline,
	builder *payloadbuilder.Builder,
	state *cache.EngineState,
	redisClient *redis.RedisClient,
	dbWorker *postgres.DBWorker,
	sim *simulator.Simulator,
) {
start := time.Now()

	// ✅ SINGLE ENTRY POINT
	payload := pipeline.Process(snap, sim)

	// ✅ Optional: keep for REST API
	state.UpdatePairs(payload.Pairs, snap.Timestamp)

	// ✅ Redis stream (non-blocking)
	go redisClient.PublishPayloadWithType(payload, payload.Type)

	// ✅ DB write (async)
	dbWorker.PairChan <- payload.Pairs

	log.Printf(
		"Replay Tick | Pairs: %d | OpenPos: %d | Time: %v\n",
		len(payload.Pairs),
		len(payload.OpenPositions),
		time.Since(start),
	)
	log.Println("SNAP STRIKES:", len(snap.Strikes))
	log.Println("SNAP TIMESTAMP:", snap.Timestamp)
}