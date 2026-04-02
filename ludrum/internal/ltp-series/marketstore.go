package ltpSeries

import "sync"

type MarketState struct {
	History []float64
}

type MarketStore struct {
	mu    sync.RWMutex
	state map[string]*MarketState
	limit int
}

func NewMarketStore(limit int) *MarketStore {
	return &MarketStore{
		state: make(map[string]*MarketState),
		limit: limit,
	}
}

func (m *MarketStore) UpdateLTP(symbol string, ltp float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, exists := m.state[symbol]
	if !exists {
		s = &MarketState{}
		m.state[symbol] = s
	}

	s.History = append(s.History, ltp)

	if len(s.History) > m.limit {
		s.History = s.History[1:]
	}
}

func (m *MarketStore) GetState(symbol string) *MarketState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.state[symbol]
}