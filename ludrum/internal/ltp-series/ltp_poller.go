package ltpSeries

import (
	"context"
	"log"
	"time"

)

type LTPPoller struct {
	fetcher *FyersFetcher
	store   *MarketStore

	symbols map[string]bool
}

func NewLTPPoller(fetcher *FyersFetcher, store *MarketStore) *LTPPoller {
	return &LTPPoller{
		fetcher: fetcher,
		store:   store,
		symbols: make(map[string]bool),
	}
}

// 🔥 REQUIRED METHOD
func (p *LTPPoller) GetTrackedSymbols() map[string]bool {
	return p.symbols
}

func (p *LTPPoller) UpdateTrackedSymbols(symbols []string) {
	newMap := make(map[string]bool)

	for _, s := range symbols {
		newMap[s] = true
	}

	p.symbols = newMap
}

func (p *LTPPoller) Start(ctx context.Context) {

	ticker := time.NewTicker(2 * time.Second) // 🔥 FIXED (was 1s)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			p.safePoll()
		}
	}
}

// 🔥 PANIC SAFE WRAPPER
func (p *LTPPoller) safePoll() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("🔥 Recovered from poller panic:", r)
		}
	}()

	p.poll()
}

func (p *LTPPoller) poll() {

	options, err := p.fetcher.FetchOptionChain()
	if err != nil {
		log.Println("⚠️ Poll skipped:", err)
		return
	}

	if len(options) == 0 {
		log.Println("⚠️ Empty options chain")
		return
	}

	for _, opt := range options {

		if opt.OptionType == "" {
			continue
		}

		// only track selected symbols
		if !p.symbols[opt.Symbol] {
			continue
		}

		p.store.UpdateLTP(opt.Symbol, opt.LTP)
	}
}