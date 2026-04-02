package api

import (
	"encoding/json"
	"log"
	"net/http"

	"ludrum/internal/simulator"
)

type TradeAPI struct {
	Sim *simulator.Simulator
}

func (t *TradeAPI) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/trade", t.handleTrade)
	mux.HandleFunc("/exit", t.handleExit)
	mux.HandleFunc("/positions", t.handlePositions)
}

func allowCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (t *TradeAPI) handleTrade(w http.ResponseWriter, r *http.Request) {
	log.Println("/trade hit")

	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req simulator.TradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Lots <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lots must be greater than zero"})
		return
	}

	price := t.Sim.GetLastPrice(req.Strike, req.OptionType)
	if price == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price not available yet"})
		return
	}

	if err := t.Sim.Execute(req, price); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	open, _, portfolio := t.Sim.GetState()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "trade executed",
		"price":     price,
		"positions": open,
		"portfolio": portfolio,
	})
}

func (t *TradeAPI) handleExit(w http.ResponseWriter, r *http.Request) {
	log.Println("/exit hit")

	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		Strike     float64 `json:"strike"`
		OptionType string  `json:"optionType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	price := t.Sim.GetLastPrice(req.Strike, req.OptionType)
	if price == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price not available"})
		return
	}

	t.Sim.ManualExit(req.Strike, req.OptionType)

	open, _, portfolio := t.Sim.GetState()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "position closed",
		"price":     price,
		"positions": open,
		"portfolio": portfolio,
	})
}

func (t *TradeAPI) handlePositions(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	open, _, portfolio := t.Sim.GetState()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"positions": open,
		"portfolio": portfolio,
	})
}
