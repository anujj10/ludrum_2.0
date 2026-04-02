package cache

import (
	"ludrum/internal/models"
	"sync"
)

type EngineState struct {
	mu 				sync.RWMutex

	LatestPairs		map[float64]models.PairSignal
	LastUpdated		int64
}

func NewEngineState() *EngineState {
	return &EngineState{
		LatestPairs: make(map[float64]models.PairSignal),
	}
}

func (s *EngineState) UpdatePairs(pairs []models.PairSignal, ts int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range pairs {
		s.LatestPairs[p.Strike] = p
	}
	s.LastUpdated = ts
}

func (s *EngineState) GetAllPairs() []models.PairSignal {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.PairSignal, 0, len(s.LatestPairs))
	for _, v := range s.LatestPairs {
		result = append(result, v)
	}
	return result
}