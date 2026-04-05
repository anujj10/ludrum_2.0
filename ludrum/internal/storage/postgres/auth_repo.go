package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type BetaUser struct {
	ID           int64     `json:"id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	TradingStyle string    `json:"trading_style"`
	ClientID     string    `json:"client_id"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

var (
	ErrInvalidCredentials = errors.New("invalid client id or password")
	ErrInvalidOTP         = errors.New("invalid or expired otp")
	ErrSessionNotFound    = errors.New("session not found")
)

type AuthOverviewUser struct {
	UserID                 int64      `json:"user_id"`
	FullName               string     `json:"full_name"`
	Email                  string     `json:"email"`
	ClientID               string     `json:"client_id"`
	Status                 string     `json:"status"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	LastCredentialSentAt   *time.Time `json:"last_credential_sent_at,omitempty"`
	ActiveSessionCount     int64      `json:"active_session_count"`
	PendingOTPCount        int64      `json:"pending_otp_count"`
	LastSessionSeenAt      *time.Time `json:"last_session_seen_at,omitempty"`
}

type AuthOverview struct {
	TotalUsers             int64              `json:"total_users"`
	ActiveSessions         int64              `json:"active_sessions"`
	PendingOTPs            int64              `json:"pending_otps"`
	CredentialsIssuedToday int64              `json:"credentials_issued_today"`
	Users                  []AuthOverviewUser `json:"users"`
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
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

func randomString(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789@#$%*"
	var builder strings.Builder
	for range length {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(alphabet[n.Int64()])
	}
	return builder.String(), nil
}

func generateClientID() (string, error) {
	digits, err := randomDigits(6)
	if err != nil {
		return "", err
	}
	return "IDX-" + digits, nil
}

func CreateOrRotateBetaUser(ctx context.Context, fullName, email, phone, tradingStyle string) (*BetaUser, string, error) {
	fullName = strings.TrimSpace(fullName)
	email = strings.ToLower(strings.TrimSpace(email))
	phone = strings.TrimSpace(phone)
	tradingStyle = strings.TrimSpace(tradingStyle)

	if fullName == "" || email == "" || phone == "" {
		return nil, "", errors.New("name, email, and phone are required")
	}

	tx, err := DB.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback(ctx)

	var existingID int64
	err = tx.QueryRow(ctx, `SELECT id FROM beta_users WHERE email = $1`, email).Scan(&existingID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, "", err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		existingID = 0
	}

	if existingID == 0 {
		var conflictID int64
		err = tx.QueryRow(ctx, `SELECT id FROM beta_users WHERE phone = $1`, phone).Scan(&conflictID)
		if err == nil && conflictID != 0 {
			return nil, "", errors.New("phone number already registered with another account")
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, "", err
		}
	}

	clientID := ""
	password, err := randomString(12)
	if err != nil {
		return nil, "", err
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	passwordHash := string(passwordHashBytes)

	if existingID == 0 {
		for attempts := 0; attempts < 5; attempts++ {
			clientID, err = generateClientID()
			if err != nil {
				return nil, "", err
			}

			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM beta_users WHERE client_id = $1)`, clientID).Scan(&exists); err != nil {
				return nil, "", err
			}
			if !exists {
				break
			}
			clientID = ""
		}

		if clientID == "" {
			return nil, "", errors.New("failed to generate unique client id")
		}

		user, err := scanBetaUserRow(tx.QueryRow(
			ctx,
			`
			INSERT INTO beta_users (full_name, email, phone, trading_style, client_id, password_hash, status, last_credential_sent_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW())
			RETURNING id, full_name, email, phone, trading_style, client_id, status, created_at, updated_at
			`,
			fullName, email, phone, tradingStyle, clientID, passwordHash,
		))
		if err != nil {
			return nil, "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, "", err
		}
		return user, password, nil
	} else {
		user, err := scanBetaUserRow(tx.QueryRow(
			ctx,
			`
			UPDATE beta_users
			SET full_name = $2,
			    phone = $3,
			    trading_style = $4,
			    password_hash = $5,
			    status = 'active',
			    updated_at = NOW(),
			    last_credential_sent_at = NOW()
			WHERE id = $1
			RETURNING id, full_name, email, phone, trading_style, client_id, status, created_at, updated_at
			`,
			existingID, fullName, phone, tradingStyle, passwordHash,
		))
		if err != nil {
			return nil, "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, "", err
		}
		return user, password, nil
	}
}

func scanBetaUserRow(row rowScanner) (*BetaUser, error) {
	user := &BetaUser{}
	err := row.Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &user.TradingStyle, &user.ClientID, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

type txLike interface {
	QueryRow(context.Context, string, ...interface{}) rowScanner
}

type rowScanner interface {
	Scan(...interface{}) error
}

func AuthenticateClient(ctx context.Context, clientID, password string) (*BetaUser, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	var user BetaUser
	var passwordHash string
	err := DB.QueryRow(
		ctx,
		`SELECT id, full_name, email, phone, trading_style, client_id, status, created_at, updated_at, password_hash
		 FROM beta_users WHERE client_id = $1 AND status = 'active'`,
		clientID,
	).Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &user.TradingStyle, &user.ClientID, &user.Status, &user.CreatedAt, &user.UpdatedAt, &passwordHash)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

func GetBetaUserByClientID(ctx context.Context, clientID string) (*BetaUser, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, ErrInvalidCredentials
	}

	var user BetaUser
	err := DB.QueryRow(
		ctx,
		`SELECT id, full_name, email, phone, trading_style, client_id, status, created_at, updated_at
		 FROM beta_users WHERE client_id = $1`,
		clientID,
	).Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &user.TradingStyle, &user.ClientID, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

func SaveEmailOTP(ctx context.Context, userID int64, otp string, expiresAt time.Time) error {
	_, err := DB.Exec(
		ctx,
		`INSERT INTO email_otp_codes (user_id, code_hash, expires_at) VALUES ($1, $2, $3)`,
		userID,
		hashString(otp),
		expiresAt.UTC(),
	)
	return err
}

func VerifyEmailOTP(ctx context.Context, clientID, otp string) (*BetaUser, error) {
	var user BetaUser
	err := DB.QueryRow(
		ctx,
		`SELECT id, full_name, email, phone, trading_style, client_id, status, created_at, updated_at
		 FROM beta_users WHERE client_id = $1 AND status = 'active'`,
		strings.TrimSpace(clientID),
	).Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &user.TradingStyle, &user.ClientID, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrInvalidOTP
	}

	var otpID int64
	var codeHash string
	var expiresAt time.Time
	err = DB.QueryRow(
		ctx,
		`
		SELECT id, code_hash, expires_at
		FROM email_otp_codes
		WHERE user_id = $1
		  AND consumed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
		`,
		user.ID,
	).Scan(&otpID, &codeHash, &expiresAt)
	if err != nil {
		return nil, ErrInvalidOTP
	}

	if time.Now().After(expiresAt) || hashString(strings.TrimSpace(otp)) != codeHash {
		return nil, ErrInvalidOTP
	}

	if _, err := DB.Exec(ctx, `UPDATE email_otp_codes SET consumed_at = NOW() WHERE id = $1`, otpID); err != nil {
		return nil, err
	}

	return &user, nil
}

func CreateSession(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	token, err := randomString(48)
	if err != nil {
		return "", err
	}

	_, err = DB.Exec(
		ctx,
		`INSERT INTO auth_sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID,
		hashString(token),
		time.Now().Add(ttl).UTC(),
	)
	if err != nil {
		return "", err
	}

	return token, nil
}

func GetUserBySessionToken(ctx context.Context, token string) (*BetaUser, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrSessionNotFound
	}

	user := &BetaUser{}
	err := DB.QueryRow(
		ctx,
		`
		SELECT u.id, u.full_name, u.email, u.phone, u.trading_style, u.client_id, u.status, u.created_at, u.updated_at
		FROM auth_sessions s
		JOIN beta_users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.expires_at > NOW()
		  AND u.status = 'active'
		`,
		hashString(token),
	).Scan(&user.ID, &user.FullName, &user.Email, &user.Phone, &user.TradingStyle, &user.ClientID, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	_, _ = DB.Exec(ctx, `UPDATE auth_sessions SET last_seen_at = NOW() WHERE token_hash = $1`, hashString(token))

	return user, nil
}

func DeleteSession(ctx context.Context, token string) error {
	_, err := DB.Exec(ctx, `DELETE FROM auth_sessions WHERE token_hash = $1`, hashString(strings.TrimSpace(token)))
	return err
}

func GetAuthOverview(ctx context.Context, limit int) (*AuthOverview, error) {
	if limit <= 0 {
		limit = 10
	}

	overview := &AuthOverview{}

	if err := DB.QueryRow(ctx, `SELECT COUNT(*) FROM beta_users`).Scan(&overview.TotalUsers); err != nil {
		return nil, err
	}
	if err := DB.QueryRow(ctx, `SELECT COUNT(*) FROM auth_sessions WHERE expires_at > NOW()`).Scan(&overview.ActiveSessions); err != nil {
		return nil, err
	}
	if err := DB.QueryRow(ctx, `SELECT COUNT(*) FROM email_otp_codes WHERE consumed_at IS NULL AND expires_at > NOW()`).Scan(&overview.PendingOTPs); err != nil {
		return nil, err
	}
	if err := DB.QueryRow(ctx, `SELECT COUNT(*) FROM beta_users WHERE last_credential_sent_at >= date_trunc('day', NOW() AT TIME ZONE 'Asia/Kolkata') AT TIME ZONE 'Asia/Kolkata'`).Scan(&overview.CredentialsIssuedToday); err != nil {
		return nil, err
	}

	rows, err := DB.Query(
		ctx,
		`
		SELECT
			u.id,
			u.full_name,
			u.email,
			u.client_id,
			u.status,
			u.created_at,
			u.updated_at,
			u.last_credential_sent_at,
			COALESCE(session_counts.active_session_count, 0),
			COALESCE(otp_counts.pending_otp_count, 0),
			session_counts.last_session_seen_at
		FROM beta_users u
		LEFT JOIN (
			SELECT
				user_id,
				COUNT(*) FILTER (WHERE expires_at > NOW()) AS active_session_count,
				MAX(last_seen_at) FILTER (WHERE expires_at > NOW()) AS last_session_seen_at
			FROM auth_sessions
			GROUP BY user_id
		) session_counts ON session_counts.user_id = u.id
		LEFT JOIN (
			SELECT
				user_id,
				COUNT(*) FILTER (WHERE consumed_at IS NULL AND expires_at > NOW()) AS pending_otp_count
			FROM email_otp_codes
			GROUP BY user_id
		) otp_counts ON otp_counts.user_id = u.id
		ORDER BY u.updated_at DESC
		LIMIT $1
		`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	overview.Users = make([]AuthOverviewUser, 0, limit)
	for rows.Next() {
		var entry AuthOverviewUser
		if err := rows.Scan(
			&entry.UserID,
			&entry.FullName,
			&entry.Email,
			&entry.ClientID,
			&entry.Status,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.LastCredentialSentAt,
			&entry.ActiveSessionCount,
			&entry.PendingOTPCount,
			&entry.LastSessionSeenAt,
		); err != nil {
			return nil, err
		}
		overview.Users = append(overview.Users, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return overview, nil
}
