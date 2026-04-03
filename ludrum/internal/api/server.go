package api

import (
	"encoding/json"
	"ludrum/internal/cache"
	"ludrum/internal/storage/postgres"
	"net/http"
)

type Server struct {
	State *cache.EngineState
}

func NewServer(state *cache.EngineState) *Server {
	return &Server{State: state}
}

func (s *Server) Start() {
	http.HandleFunc("/pairs", s.handlePairs)
	http.HandleFunc("/market-status", s.handleMarketStatus)

	go http.ListenAndServe(":8080", nil)
}

func (s *Server) handlePairs(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if _, err := authorizeRequest(r); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	data := s.State.GetAllPairs()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleMarketStatus(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	reason, err := postgres.GetMarketOverride(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch market status"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"forced_closed": reason != "",
		"reason":        reason,
	})
}
