package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"ludrum/internal/storage/postgres"
)

const authTokenHeader = "Authorization"

type AuthAPI struct{}

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

func RegisterAuthRoutes(mux *http.ServeMux) {
	api := &AuthAPI{}
	mux.HandleFunc("/auth/beta-request", api.handleBetaRequest)
	mux.HandleFunc("/auth/login", api.handleLogin)
	mux.HandleFunc("/auth/verify-otp", api.handleVerifyOTP)
	mux.HandleFunc("/auth/me", api.handleMe)
	mux.HandleFunc("/auth/logout", api.handleLogout)
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
		delivery["warning"] = "SMTP not configured, returning credentials in preview mode."
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

	otp, err := generateOTP()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate otp"})
		return
	}

	if err := postgres.SaveEmailOTP(r.Context(), user.ID, otp, time.Now().Add(10*time.Minute)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store otp"})
		return
	}

	delivery := map[string]interface{}{
		"message": fmt.Sprintf("OTP sent to %s", user.Email),
	}

	subject := "Your Index Options login OTP"
	body := fmt.Sprintf("Hello %s,\n\nYour login OTP is %s.\nIt expires in 10 minutes.\n", user.FullName, otp)
	if err := sendEmail(user.Email, subject, body); err != nil {
		delivery["delivery"] = "preview"
		delivery["otp_preview"] = otp
		delivery["warning"] = "SMTP not configured, returning OTP in preview mode."
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

func generateOTP() (string, error) {
	return randomDigits(6)
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
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	username := strings.TrimSpace(os.Getenv("SMTP_USER"))
	password := strings.TrimSpace(os.Getenv("SMTP_PASS"))
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))

	if host == "" || port == "" || from == "" {
		return fmt.Errorf("smtp not configured")
	}

	addr := host + ":" + port
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	var builder strings.Builder
	for key, value := range headers {
		builder.WriteString(key)
		builder.WriteString(": ")
		builder.WriteString(value)
		builder.WriteString("\r\n")
	}
	builder.WriteString("\r\n")
	builder.WriteString(body)

	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(builder.String()))
}
