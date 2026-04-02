package api

import (
	"encoding/json"
	"net/http"
	"ludrum/internal/cache"
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
	data := s.State.GetAllPairs()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}