package simulator

import (
	"fmt"
	"sync"
)

type Simulator struct {
	mu sync.RWMutex

	positions map[string]*Position
	closed    []*Position

	portfolio *Portfolio

	lastPrices map[string]float64
}

func NewSimulator() *Simulator {
	return &Simulator{
		positions:  make(map[string]*Position),
		closed:     []*Position{},
		portfolio:  NewPortfolio(300000),
		lastPrices: make(map[string]float64),
	}
}

func positionKey(strike float64, opt string) string {
	return fmt.Sprintf("%.0f_%s", strike, opt)
}

// ================= PRICE UPDATE =================

func (s *Simulator) OnPriceUpdate(strike float64, opt string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := positionKey(strike, opt)

	// ✅ store latest price
	s.lastPrices[key] = price

	pos, ok := s.positions[key]
	if !ok {
		return
	}

	pos.UpdatePnL(price)

	// SL / TARGET CHECK
	if s.checkExits(pos, price) {
		s.closePosition(key, price)
	}
}

// ================= GET LAST PRICE =================

func (s *Simulator) GetLastPrice(strike float64, opt string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := positionKey(strike, opt)

	price, ok := s.lastPrices[key]
	if !ok {
		return 0
	}

	return price
}

// ================= CLOSE POSITION =================

func (s *Simulator) closePosition(key string, price float64) {
	pos := s.positions[key]

	if pos.Side == LONG {
		// return premium
		s.portfolio.AvailableCapital += price * float64(pos.Qty)
	} else {
		// release margin
		margin := pos.AvgPrice * float64(pos.Qty) * MarginFactor
		s.portfolio.ReleaseMargin(margin)
	}

	s.portfolio.RealizedPnL += pos.UnrealizedPnL

	delete(s.positions, key)
	s.closed = append(s.closed, pos)
}

// ================= GET STATE =================

func (s *Simulator) GetState() ([]Position, []*Position, Portfolio) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	open := []Position{}
	for _, p := range s.positions {
		open = append(open, *p)
	}

	return open, s.closed, *s.portfolio
}

// ================= RESET =================

func (s *Simulator) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.positions = make(map[string]*Position)
	s.closed = []*Position{}
	s.portfolio = NewPortfolio(300000)
	s.lastPrices = make(map[string]float64)
}

func (s *Simulator) ManualExit(strike float64, opt string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := positionKey(strike, opt)

	_, ok := s.positions[key]
	if !ok {
		return
	}

	price, ok := s.lastPrices[key]
	if !ok {
		return
	}

	s.closePosition(key, price)
}