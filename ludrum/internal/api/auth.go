package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ludrum/internal/runtime"
	"ludrum/internal/storage/postgres"
)

const authTokenHeader = "Authorization"

type AuthAPI struct{
	RuntimeManager *runtime.Manager
}

type betaRequestPayload struct {
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	TradingStyle string `json:"trading_style"`
}

type loginPayload struct {
	ClientID string `json:"client_id"`
	Password string `json:"password"`
}

type verifyOTPPayload struct {
	ClientID string `json:"client_id"`
	OTP      string `json:"otp"`
}

type resendOTPPayload struct {
	ClientID string `json:"client_id"`
}

type marketOverridePayload struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type adminTokenClaims struct {
	ClientID  string `json:"client_id"`
	ExpiresAt int64  `json:"exp"`
}

func RegisterAuthRoutes(mux *http.ServeMux) {
	api := &AuthAPI{}
	mux.HandleFunc("/auth/beta-request", api.handleBetaRequest)
	mux.HandleFunc("/auth/login", api.handleLogin)
	mux.HandleFunc("/auth/verify-otp", api.handleVerifyOTP)
	mux.HandleFunc("/auth/resend-otp", api.handleResendOTP)
	mux.HandleFunc("/auth/me", api.handleMe)
	mux.HandleFunc("/auth/logout", api.handleLogout)
	mux.HandleFunc("/auth/admin/login", api.handleAdminLogin)
	mux.HandleFunc("/auth/admin/me", api.handleAdminMe)
	mux.HandleFunc("/auth/admin/logout", api.handleAdminLogout)
	mux.HandleFunc("/auth/admin/market-override", api.handleAdminMarketOverride)
	mux.HandleFunc("/auth/admin/fyers/overview", api.handleAdminFyersOverview)
	mux.HandleFunc("/auth/admin/system-overview", api.handleAdminSystemOverview)
}

func (a *AuthAPI) handleBetaRequest(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload betaRequestPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, password, err := postgres.CreateOrRotateBetaUser(r.Context(), payload.FullName, payload.Email, payload.Phone, payload.TradingStyle)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	subject := "Your Index Options beta credentials"
	body := fmt.Sprintf(
		"Hello %s,\n\nYour beta access is ready.\n\nClient ID: %s\nPassword: %s\n\nUse these credentials on the login page, then verify the OTP sent to this email.\n",
		user.FullName,
		user.ClientID,
		password,
	)

	delivery := map[string]interface{}{
		"message": "Beta request received. Credentials have been prepared.",
	}

	if err := sendEmail(user.Email, subject, body); err != nil {
		delivery["delivery"] = "preview"
		delivery["client_id"] = user.ClientID
		delivery["password"] = password
		delivery["warning"] = "Email provider not configured, returning credentials in preview mode."
	} else {
		delivery["delivery"] = "email"
	}

	writeJSON(w, http.StatusOK, delivery)
}

func (a *AuthAPI) handleLogin(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload loginPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := postgres.AuthenticateClient(r.Context(), payload.ClientID, payload.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid client id or password"})
		return
	}

	otpLength := configuredOTPLength()
	otpExpiry := configuredOTPExpiry()

	otp, err := generateOTP(otpLength)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate otp"})
		return
	}

	if err := postgres.SaveEmailOTP(r.Context(), user.ID, otp, time.Now().Add(otpExpiry)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store otp"})
		return
	}

	delivery := map[string]interface{}{
		"message": fmt.Sprintf("OTP sent to %s", user.Email),
	}

	subject := "Your Ludrum verification code"
	body := fmt.Sprintf("Hello %s,\n\nYour Ludrum verification code is %s.\nIt expires in %d minutes.\n\nDo not share this code with anyone.\n", user.FullName, otp, int(otpExpiry/time.Minute))
	if err := sendEmail(user.Email, subject, body); err != nil {
		delivery["delivery"] = "preview"
		delivery["otp_preview"] = otp
		delivery["warning"] = "Email provider not configured, returning OTP in preview mode."
	} else {
		delivery["delivery"] = "email"
	}

	writeJSON(w, http.StatusOK, delivery)
}

func (a *AuthAPI) handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload verifyOTPPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := postgres.VerifyEmailOTP(r.Context(), payload.ClientID, payload.OTP)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired otp"})
		return
	}

	token, err := postgres.CreateSession(r.Context(), user.ID, 7*24*time.Hour)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]string{
			"client_id": user.ClientID,
			"email":     user.Email,
			"full_name": user.FullName,
		},
	})
}

func (a *AuthAPI) handleResendOTP(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var payload resendOTPPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	user, err := postgres.GetBetaUserByClientID(r.Context(), payload.ClientID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "client account not found"})
		return
	}
	if strings.TrimSpace(user.Status) != "active" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "client account is not active"})
		return
	}

	otpLength := configuredOTPLength()
	otpExpiry := configuredOTPExpiry()

	otp, err := generateOTP(otpLength)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate otp"})
		return
	}

	if err := postgres.SaveEmailOTP(r.Context(), user.ID, otp, time.Now().Add(otpExpiry)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store otp"})
		return
	}

	delivery := map[string]interface{}{
		"message": fmt.Sprintf("A new OTP has been sent to %s", user.Email),
	}

	subject := "Your Ludrum verification code"
	body := fmt.Sprintf("Hello %s,\n\nYour Ludrum verification code is %s.\nIt expires in %d minutes.\n\nDo not share this code with anyone.\n", user.FullName, otp, int(otpExpiry/time.Minute))
	if err := sendEmail(user.Email, subject, body); err != nil {
		delivery["delivery"] = "preview"
		delivery["otp_preview"] = otp
		delivery["warning"] = "Email provider not configured, returning OTP in preview mode."
	} else {
		delivery["delivery"] = "email"
	}

	writeJSON(w, http.StatusOK, delivery)
}

func (a *AuthAPI) handleMe(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

func (a *AuthAPI) handleLogout(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	token := extractBearerToken(r)
	if token != "" {
		_ = postgres.DeleteSession(r.Context(), token)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (a *AuthAPI) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminClientID := strings.TrimSpace(os.Getenv("ADMIN_CLIENT_ID"))
	adminPassword := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	adminSecret := strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET"))
	if adminClientID == "" || adminPassword == "" || adminSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin auth is not configured"})
		return
	}

	var payload loginPayload
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if payload.ClientID != adminClientID || payload.Password != adminPassword {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid admin client id or password"})
		return
	}

	token, err := issueAdminToken(adminClientID, adminSecret, 12*time.Hour)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create admin session"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"admin": map[string]string{
			"client_id": adminClientID,
		},
	})
}

func (a *AuthAPI) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminClientID := strings.TrimSpace(os.Getenv("ADMIN_CLIENT_ID"))
	adminSecret := strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET"))
	if adminClientID == "" || adminSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin auth is not configured"})
		return
	}

	token := extractBearerToken(r)
	claims, err := verifyAdminToken(token, adminSecret)
	if err != nil || claims.ClientID != adminClientID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"admin": map[string]string{
			"client_id": claims.ClientID,
		},
	})
}

func (a *AuthAPI) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (a *AuthAPI) handleAdminMarketOverride(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	adminClientID := strings.TrimSpace(os.Getenv("ADMIN_CLIENT_ID"))
	adminSecret := strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET"))
	if adminClientID == "" || adminSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin auth is not configured"})
		return
	}

	claims, err := authorizeAdminRequest(r, adminSecret)
	if err != nil || claims.ClientID != adminClientID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		reason, err := postgres.GetMarketOverride(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch market override"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": reason != "",
			"reason":  reason,
		})
	case http.MethodPost:
		var payload marketOverridePayload
		if err := decodeJSON(r, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if payload.Enabled {
			reason := strings.TrimSpace(payload.Reason)
			if reason == "" {
				reason = "Markets are down right now. Please check back shortly."
			}
			if err := postgres.SetMarketOverride(r.Context(), reason); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store market override"})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"enabled": true,
				"reason":  reason,
			})
			return
		}

		if err := postgres.ClearMarketOverride(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to clear market override"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": false,
			"reason":  "",
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (a *AuthAPI) handleAdminFyersOverview(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminClientID := strings.TrimSpace(os.Getenv("ADMIN_CLIENT_ID"))
	adminSecret := strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET"))
	if adminClientID == "" || adminSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin auth is not configured"})
		return
	}

	claims, err := authorizeAdminRequest(r, adminSecret)
	if err != nil || claims.ClientID != adminClientID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	overview, err := postgres.GetFyersOverview(r.Context(), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load fyers overview"})
		return
	}

	writeJSON(w, http.StatusOK, overview)
}

func (a *AuthAPI) handleAdminSystemOverview(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	adminClientID := strings.TrimSpace(os.Getenv("ADMIN_CLIENT_ID"))
	adminSecret := strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET"))
	if adminClientID == "" || adminSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "admin auth is not configured"})
		return
	}

	claims, err := authorizeAdminRequest(r, adminSecret)
	if err != nil || claims.ClientID != adminClientID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	overview, err := postgres.GetAuthOverview(r.Context(), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load auth overview"})
		return
	}

	writeJSON(w, http.StatusOK, overview)
}

func issueAdminToken(clientID, secret string, ttl time.Duration) (string, error) {
	claims := adminTokenClaims{
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := signAdminPayload(encodedPayload, secret)
	return encodedPayload + "." + signature, nil
}

func verifyAdminToken(token, secret string) (*adminTokenClaims, error) {
	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}

	expected := signAdminPayload(parts[0], secret)
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, fmt.Errorf("invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	var claims adminTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	if claims.ClientID == "" || time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

func authorizeAdminRequest(r *http.Request, secret string) (*adminTokenClaims, error) {
	token := extractBearerToken(r)
	if token == "" {
		return nil, fmt.Errorf("missing token")
	}
	return verifyAdminToken(token, secret)
}

func signAdminPayload(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func generateOTP(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	return randomDigits(length)
}

func randomDigits(length int) (string, error) {
	var builder strings.Builder
	for range length {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}
	return builder.String(), nil
}

func extractBearerToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get(authTokenHeader))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}
	return ""
}

func authorizeRequest(r *http.Request) (*postgres.BetaUser, error) {
	token := extractBearerToken(r)
	if token == "" {
		return nil, postgres.ErrSessionNotFound
	}
	return postgres.GetUserBySessionToken(r.Context(), token)
}

func decodeJSON(r *http.Request, target interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

func sendEmail(to, subject, body string) error {
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	from := strings.TrimSpace(os.Getenv("RESEND_FROM_EMAIL"))
	if apiKey == "" || from == "" {
		return fmt.Errorf("resend not configured")
	}

	payload := map[string]interface{}{
		"from":    from,
		"to":      []string{to},
		"subject": subject,
		"text":    body,
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(rawBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("resend send failed: %s", strings.TrimSpace(string(responseBody)))
}

func configuredOTPExpiry() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("OTP_EXPIRY_MINUTES")); raw != "" {
		if minutes, err := strconv.Atoi(raw); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return 5 * time.Minute
}

func configuredOTPLength() int {
	if raw := strings.TrimSpace(os.Getenv("OTP_LENGTH")); raw != "" {
		if length, err := strconv.Atoi(raw); err == nil && length >= 4 && length <= 9 {
			return length
		}
	}
	return 6
}
