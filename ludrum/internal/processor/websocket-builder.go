package processor

import (
	"sync"

	"ludrum/internal/models"
)

const MaxHistory = 5

// ==========================
// 🔥 STATE PER SYMBOL
// ==========================
type MarketState struct {
	LastTick models.MarketTick
	History  []float64 // LTP history (FIFO)
}

// ==========================
// 🔥 STORE
// ==========================
type MarketStore struct {
	mu    sync.RWMutex
	state map[string]*MarketState
}

// ==========================
// INIT
// ==========================
func NewMarketStore() *MarketStore {
	return &MarketStore{
		state: make(map[string]*MarketState),
	}
}

// ==========================
// UPDATE (CORE LOGIC)
// ==========================
func (ms *MarketStore) Update(t models.MarketTick) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	s, exists := ms.state[t.Symbol]

	if !exists {
		s = &MarketState{
			History: make([]float64, 0, MaxHistory),
		}
		ms.state[t.Symbol] = s
	}

	// update last tick
	s.LastTick = t

	// 🔥 maintain FIFO LTP history
	s.History = append(s.History, t.LTP)

	if len(s.History) > MaxHistory {
		s.History = s.History[1:] // remove oldest
	}
}

// ==========================
// GET STATE
// ==========================
func (ms *MarketStore) Get(symbol string) (*MarketState, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	s, ok := ms.state[symbol]
	return s, ok
}

// ==========================
// GET HISTORY ONLY
// ==========================
func (ms *MarketStore) GetHistory(symbol string) ([]float64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	s, ok := ms.state[symbol]
	if !ok {
		return nil, false
	}

	return s.History, true
}