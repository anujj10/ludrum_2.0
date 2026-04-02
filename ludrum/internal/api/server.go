package api

import (
	"encoding/json"
	"ludrum/internal/cache"
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
