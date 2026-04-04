package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ludrum/internal/broker/fyers"
	"ludrum/internal/storage/postgres"
)

type fyersOAuthState struct {
	UserID    int64  `json:"user_id"`
	ClientID  string `json:"client_id"`
	ExpiresAt int64  `json:"exp"`
}

type fyersStatusResponse struct {
	Connected       bool   `json:"connected"`
	Status          string `json:"status"`
	BrokerUserID    string `json:"broker_user_id,omitempty"`
	TokenExpiresAt  string `json:"token_expires_at,omitempty"`
	LastConnectedAt string `json:"last_connected_at,omitempty"`
	LoginURL        string `json:"login_url,omitempty"`
}

func RegisterBrokerRoutes(mux *http.ServeMux) {
	api := &AuthAPI{}
	mux.HandleFunc("/broker/fyers/status", api.handleFyersStatus)
	mux.HandleFunc("/broker/fyers/connect/start", api.handleFyersConnectStart)
	mux.HandleFunc("/broker/fyers/callback", api.handleFyersCallback)
}

func (a *AuthAPI) handleFyersStatus(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	user, err := authorizeRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	account, err := postgres.GetFyersAccountByUserID(r.Context(), user.ID)
	if err != nil {
		if err == postgres.ErrFyersAccountNotFound {
			writeJSON(w, http.StatusOK, fyersStatusResponse{
				Connected: false,
				Status:    "unlinked",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load broker status"})
		return
	}

	response := fyersStatusResponse{
		Connected: account.Status == "linked" || account.Status == "active",
		Status:    account.Status,
	}
	if account.BrokerUserID != "" {
		response.BrokerUserID = account.BrokerUserID
	}
	if account.LastConnectedAt != nil {
		response.LastConnectedAt = account.LastConnectedAt.UTC().Format(time.RFC3339)
	}

	if token, tokenErr := postgres.GetFyersTokenByAccountID(r.Context(), account.ID); tokenErr == nil && token.ExpiresAt != nil {
		response.TokenExpiresAt = token.ExpiresAt.UTC().Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *AuthAPI) handleFyersConnectStart(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	user, err := authorizeRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	authConfig, err := loadFyersAuthConfig()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	account, err := postgres.UpsertFyersAccount(r.Context(), user.ID, authConfig.AppID, authConfig.RedirectURL, "", "pending")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to initialize broker account"})
		return
	}

	state, err := issueFyersOAuthState(fyersOAuthState{
		UserID:    user.ID,
		ClientID:  user.ClientID,
		ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to prepare broker auth"})
		return
	}

	client := fyers.NewClient(authConfig)
	loginURL := client.LoginURL()
	if loginURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "failed to generate broker login url"})
		return
	}

	loginURL = appendQueryParam(loginURL, "state", state)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connected":  false,
		"status":     account.Status,
		"account_id": account.ID,
		"login_url":  loginURL,
	})
}

func (a *AuthAPI) handleFyersCallback(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	authCode := strings.TrimSpace(r.URL.Query().Get("auth_code"))
	if authCode == "" {
		authCode = strings.TrimSpace(r.URL.Query().Get("authCode"))
	}

	redirectURL := frontendSuccessURL()
	if state == "" || authCode == "" {
		http.Redirect(w, r, frontendFailureURL("missing_broker_callback_params"), http.StatusFound)
		return
	}

	claims, err := verifyFyersOAuthState(state)
	if err != nil {
		http.Redirect(w, r, frontendFailureURL("invalid_broker_state"), http.StatusFound)
		return
	}

	authConfig, err := loadFyersAuthConfig()
	if err != nil {
		http.Redirect(w, r, frontendFailureURL("broker_app_not_configured"), http.StatusFound)
		return
	}

	account, err := postgres.UpsertFyersAccount(r.Context(), claims.UserID, authConfig.AppID, authConfig.RedirectURL, "", "linking")
	if err != nil {
		http.Redirect(w, r, frontendFailureURL("account_setup_failed"), http.StatusFound)
		return
	}

	client := fyers.NewClient(authConfig)
	tokenSet, err := client.ExchangeAuthCode(context.Background(), authCode)
	if err != nil || tokenSet == nil || tokenSet.AccessToken == "" {
		http.Redirect(w, r, frontendFailureURL("token_exchange_failed"), http.StatusFound)
		return
	}

	brokerUserID := deriveBrokerUserID(tokenSet.Raw)
	now := time.Now().UTC()

	if _, err := postgres.UpsertFyersAccount(r.Context(), claims.UserID, authConfig.AppID, authConfig.RedirectURL, brokerUserID, "linked"); err != nil {
		http.Redirect(w, r, frontendFailureURL("account_store_failed"), http.StatusFound)
		return
	}

	if _, err := postgres.UpsertFyersToken(
		r.Context(),
		account.ID,
		encryptToken(tokenSet.AccessToken),
		encryptToken(tokenSet.RefreshToken),
		tokenSet.TokenType,
		tokenSet.ExpiresAt,
	); err != nil {
		http.Redirect(w, r, frontendFailureURL("token_store_failed"), http.StatusFound)
		return
	}

	if _, err := postgres.UpsertUserRuntimeStatus(r.Context(), claims.UserID, account.ID, "linked", &now, nil, ""); err != nil {
		http.Redirect(w, r, frontendFailureURL("runtime_status_failed"), http.StatusFound)
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func loadFyersAuthConfig() (fyers.AuthConfig, error) {
	appID := strings.TrimSpace(getEnvAny("FYERS_APP_ID", "APP_ID"))
	secretID := strings.TrimSpace(getEnvAny("FYERS_SECRET_ID", "SECRET_ID"))
	redirectURL := strings.TrimSpace(getEnvAny("FYERS_REDIRECT_URL", "REDIRECT_URL"))
	if appID == "" || secretID == "" || redirectURL == "" {
		return fyers.AuthConfig{}, fmt.Errorf("fyers oauth is not configured")
	}

	return fyers.AuthConfig{
		AppID:       appID,
		SecretID:    secretID,
		RedirectURL: redirectURL,
	}, nil
}

func getEnvAny(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func issueFyersOAuthState(claims fyersOAuthState) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signFyersPayload(encodedPayload, fyersStateSecret())
	return encodedPayload + "." + signature, nil
}

func verifyFyersOAuthState(token string) (*fyersOAuthState, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state format")
	}

	expected := signFyersPayload(parts[0], fyersStateSecret())
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, fmt.Errorf("invalid state signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	var claims fyersOAuthState
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	if claims.UserID == 0 || claims.ExpiresAt == 0 || time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("expired state")
	}

	return &claims, nil
}

func fyersStateSecret() string {
	if secret := strings.TrimSpace(os.Getenv("FYERS_STATE_SECRET")); secret != "" {
		return secret
	}
	if secret := strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET")); secret != "" {
		return secret
	}
	return "ludrum-fyers-state"
}

func signFyersPayload(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encryptToken(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func deriveBrokerUserID(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	for _, key := range []string{"fy_id", "user_name", "user_id"} {
		if value := strings.TrimSpace(stringMapValue(raw[key])); value != "" {
			return value
		}
	}
	return ""
}

func stringMapValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func frontendBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "https://ludrum.online"
}

func frontendSuccessURL() string {
	return frontendBaseURL() + "/?broker=fyers&connected=1"
}

func frontendFailureURL(reason string) string {
	return frontendBaseURL() + "/?broker=fyers&error=" + reason
}

func appendQueryParam(url, key, value string) string {
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	return url + separator + key + "=" + urlpkgEscape(value)
}

func urlpkgEscape(value string) string {
	return url.QueryEscape(value)
}
