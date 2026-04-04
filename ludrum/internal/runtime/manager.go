package runtime

import (
	"context"
	"fmt"
	"sync"

	"ludrum/internal/broker/fyers"
)

type Manager struct {
	mu       sync.RWMutex
	runtimes map[string]*UserRuntime
}

func NewManager() *Manager {
	return &Manager{
		runtimes: make(map[string]*UserRuntime),
	}
}

func (m *Manager) EnsureUserRuntime(ctx context.Context, config fyers.RuntimeConfig) (*UserRuntime, error) {
	if config.UserID == 0 || config.AccountID == 0 {
		return nil, fmt.Errorf("user runtime requires user id and account id")
	}

	key := runtimeKey(config.UserID, config.AccountID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.runtimes[key]; ok {
		existing.ApplyConfig(config)
		return existing, nil
	}

	runtime := NewUserRuntime(config)
	if err := runtime.Start(ctx); err != nil {
		return nil, err
	}
	m.runtimes[key] = runtime
	return runtime, nil
}

func (m *Manager) GetUserRuntime(userID, accountID int64) (*UserRuntime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	runtime, ok := m.runtimes[runtimeKey(userID, accountID)]
	return runtime, ok
}

func (m *Manager) StopUserRuntime(userID, accountID int64) {
	key := runtimeKey(userID, accountID)

	m.mu.Lock()
	runtime, ok := m.runtimes[key]
	if ok {
		delete(m.runtimes, key)
	}
	m.mu.Unlock()

	if ok {
		runtime.Stop()
	}
}

func runtimeKey(userID, accountID int64) string {
	return fmt.Sprintf("%d:%d", userID, accountID)
}
