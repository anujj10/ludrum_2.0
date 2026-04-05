package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"ludrum/internal/cache"
	"ludrum/internal/broker/fyers"
	"ludrum/internal/runtime"
	"ludrum/internal/storage/postgres"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	State *cache.EngineState
	RuntimeManager *runtime.Manager
}

func NewServer(state *cache.EngineState, runtimeManager *runtime.Manager) *Server {
	return &Server{State: state, RuntimeManager: runtimeManager}
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

	user, _ := authorizeRequest(r)

	if user != nil && s.RuntimeManager != nil {
		runtimeInstance, err := ensureFyersRuntimeForUser(r.Context(), s.RuntimeManager, user.ID)
		if err == nil {
			if data, ok := runtimeInstance.GetPairs(); ok && len(data) > 0 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(data)
				return
			}
		}
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

func ensureFyersRuntimeForUser(ctx context.Context, manager *runtime.Manager, userID int64) (*runtime.UserRuntime, error) {
	config, err := loadFyersRuntimeConfigForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return manager.EnsureUserRuntime(ctx, config)
}

func loadFyersRuntimeConfigForUser(ctx context.Context, userID int64) (fyers.RuntimeConfig, error) {
	account, err := postgres.GetFyersAccountByUserID(ctx, userID)
	if err != nil {
		return fyers.RuntimeConfig{}, err
	}

	token, err := postgres.GetFyersTokenByAccountID(ctx, account.ID)
	if err != nil {
		return fyers.RuntimeConfig{}, err
	}

	accessToken, err := decodeStoredToken(token.AccessTokenEncrypted)
	if err != nil {
		return fyers.RuntimeConfig{}, err
	}

	return fyers.RuntimeConfig{
		UserID:          userID,
		AccountID:       account.ID,
		AppID:           account.AppID,
		BrokerUserID:    account.BrokerUserID,
		AccessToken:     accessToken,
		OptionChainRoot: "NSE:NIFTY50-INDEX",
		StrikeCount:     3,
		PollInterval:    time.Second,
	}, nil
}

func decodeStoredToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", context.DeadlineExceeded
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil && strings.TrimSpace(string(decoded)) != "" {
		return string(decoded), nil
	}

	return value, nil
}
