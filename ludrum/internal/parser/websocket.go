package parser

import (
	"encoding/json"

	"ludrum/internal/logger"
	"ludrum/internal/models"
	processor "ludrum/internal/processor"
)

func HandleMessage(
	msg []byte,
	store *processor.MarketStore,
	out chan models.MarketTick,
) {
	var raw map[string]interface{}

	if err := json.Unmarshal(msg, &raw); err != nil {
		logger.Error("parser", "json unmarshal failed", err, nil)
		return
	}

msgType, _ := raw["type"].(string)

switch msgType {

case "if", "sf": // ✅ handle both

	var tick models.MarketTick
	if err := json.Unmarshal(msg, &tick); err != nil {
		logger.Error("parser", "tick parse failed", err, map[string]interface{}{
			"raw": string(msg),
		})
		return
	}

	store.Update(tick)
	out <- tick

default:
	logger.Debug("parser", "unknown message type", map[string]interface{}{
		"type": msgType,
	})
}
}