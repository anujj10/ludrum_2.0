package optionData

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"

	"ludrum/internal/cache"
	"ludrum/internal/logger"
	"ludrum/internal/models"
	"ludrum/internal/parser"
	"ludrum/internal/processor"
	"ludrum/internal/simulator"
	"ludrum/internal/storage/postgres"
	"ludrum/internal/storage/redis"
)

// ==========================
// INGESTOR
// ==========================
type WSIngestor struct {
	store       *processor.MarketStore // ✅ POINTER
	pipeline    *processor.Pipeline
	state       *cache.EngineState
	redisClient *redis.RedisClient
	dbWorker    *postgres.DBWorker
	sim         *simulator.Simulator

	marketChan chan models.MarketTick
	cancel     context.CancelFunc

	onTick func(float64)
}

// ==========================
// CONSTRUCTOR
// ==========================
func NewWSIngestor(
	store *processor.MarketStore,
	pipeline *processor.Pipeline,
	state *cache.EngineState,
	redisClient *redis.RedisClient,
	dbWorker *postgres.DBWorker,
	sim *simulator.Simulator,
) *WSIngestor {

	w := &WSIngestor{
		store:       store,
		pipeline:    pipeline,
		state:       state,
		redisClient: redisClient,
		dbWorker:    dbWorker,
		sim:         sim,
		marketChan:  make(chan models.MarketTick, 100),
	}

	go w.consumeTicks()

	return w
}

// ==========================
// SET HOOK
// ==========================
func (w *WSIngestor) SetOnTick(fn func(float64)) {
	w.onTick = fn
}

// ==========================
// START WS
// ==========================
func (w *WSIngestor) Start(ctx context.Context, appID, token string, symbols []string) {

	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel

	accessToken := fmt.Sprintf("%s:%s", appID, token)

	var dataSocket *fyersws.FyersDataSocket

	onConnect := func() {
		log.Println("✅ WS Connected")
		dataSocket.Subscribe(symbols, "SymbolUpdate")
	}

	onMessage := func(message fyersws.DataResponse) {

		msgBytes, err := json.Marshal(message)
		if err != nil {
			logger.Error("ws", "marshal failed", err, nil)
			return
		}

		parser.HandleMessage(msgBytes, w.store, w.marketChan)
	}

	onError := func(message fyersws.DataError) {
		logger.Error("ws", "fyers error", nil, map[string]interface{}{
			"data": message,
		})
	}

	onClose := func(message fyersws.DataClose) {
		log.Println("⚠️ WS Closed:", message)
	}

	dataSocket = fyersws.NewFyersDataSocket(
		accessToken,
		"",
		false,
		true, // ✅ reconnect
		true,
		50,
		onConnect,
		onClose,
		onError,
		onMessage,
	)

	if err := dataSocket.Connect(); err != nil {
		log.Println("❌ WS Connect error:", err)
		return
	}

	go func() {
		<-ctx.Done()
		dataSocket.CloseConnection()
		close(w.marketChan)
	}()
}

// ==========================
// CONSUMER
// ==========================
func (w *WSIngestor) consumeTicks() {

	for tick := range w.marketChan {

		// 🔥 LTP engine hook
		if tick.Symbol == "NSE:NIFTY50-INDEX" && w.onTick != nil {
			w.onTick(tick.LTP)
		}

		// existing pipeline
		state, ok := w.store.Get(tick.Symbol)
		if !ok {
			continue
		}

		alpha := processor.ComputeAlpha(state)

		payload := processor.BuildPayloadFromTickWithAlpha(state.LastTick, alpha)

		w.redisClient.PublishPayload(payload)
		w.redisClient.PublishPayloadStream(payload)

		w.state.UpdatePairs(nil, payload.Meta.Timestamp)
	}
}

// ==========================
// STOP
// ==========================
func (w *WSIngestor) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}