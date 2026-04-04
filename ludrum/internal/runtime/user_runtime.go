package runtime

import (
	"context"
	"sync"
	"time"

	"ludrum/internal/broker/fyers"
)

type UserRuntime struct {
	mu         sync.RWMutex
	config     fyers.RuntimeConfig
	state      string
	lastError  string
	lastTickAt *time.Time
	cancel     context.CancelFunc
}

func NewUserRuntime(config fyers.RuntimeConfig) *UserRuntime {
	return &UserRuntime{
		config: config,
		state:  "pending",
	}
}

func (r *UserRuntime) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cancel != nil {
		return nil
	}

	runtimeCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.state = "running"

	go func() {
		<-runtimeCtx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		r.state = "stopped"
		r.cancel = nil
	}()

	return nil
}

func (r *UserRuntime) Stop() {
	r.mu.RLock()
	cancel := r.cancel
	r.mu.RUnlock()

	if cancel != nil {
		cancel()
	}
}

func (r *UserRuntime) ApplyConfig(config fyers.RuntimeConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = config
}

func (r *UserRuntime) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return Snapshot{
		UserID:     r.config.UserID,
		AccountID:  r.config.AccountID,
		State:      r.state,
		LastError:  r.lastError,
		LastTickAt: r.lastTickAt,
	}
}

func (r *UserRuntime) SetLastError(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastError = message
}

func (r *UserRuntime) MarkTick(ts time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastTickAt = &ts
}

type Snapshot struct {
	UserID     int64
	AccountID  int64
	State      string
	LastError  string
	LastTickAt *time.Time
}
