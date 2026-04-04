package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type FyersAccount struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	BrokerUserID    string     `json:"broker_user_id"`
	AppID           string     `json:"app_id"`
	RedirectURI     string     `json:"redirect_uri"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastConnectedAt *time.Time `json:"last_connected_at,omitempty"`
}

type FyersToken struct {
	ID                     int64      `json:"id"`
	AccountID              int64      `json:"account_id"`
	AccessTokenEncrypted   string     `json:"-"`
	RefreshTokenEncrypted  string     `json:"-"`
	TokenType              string     `json:"token_type"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	RefreshedAt            *time.Time `json:"refreshed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type UserRuntimeStatus struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	AccountID       int64      `json:"account_id"`
	RuntimeState    string     `json:"runtime_state"`
	LastWSConnectAt *time.Time `json:"last_ws_connect_at,omitempty"`
	LastTickAt      *time.Time `json:"last_tick_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

var ErrFyersAccountNotFound = errors.New("fyers account not found")

func UpsertFyersAccount(ctx context.Context, userID int64, appID, redirectURI, brokerUserID, status string) (*FyersAccount, error) {
	appID = strings.TrimSpace(appID)
	redirectURI = strings.TrimSpace(redirectURI)
	brokerUserID = strings.TrimSpace(brokerUserID)
	status = normalizeFyersStatus(status)

	if userID == 0 || appID == "" || redirectURI == "" {
		return nil, errors.New("user id, app id, and redirect uri are required")
	}

	row := DB.QueryRow(
		ctx,
		`
		INSERT INTO fyers_accounts (user_id, broker_user_id, app_id, redirect_uri, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, app_id)
		DO UPDATE SET
			broker_user_id = EXCLUDED.broker_user_id,
			redirect_uri = EXCLUDED.redirect_uri,
			status = EXCLUDED.status,
			updated_at = NOW()
		RETURNING id, user_id, broker_user_id, app_id, redirect_uri, status, created_at, updated_at, last_connected_at
		`,
		userID,
		nullIfEmpty(brokerUserID),
		appID,
		redirectURI,
		status,
	)

	return scanFyersAccount(row)
}

func GetFyersAccountByUserID(ctx context.Context, userID int64) (*FyersAccount, error) {
	row := DB.QueryRow(
		ctx,
		`
		SELECT id, user_id, broker_user_id, app_id, redirect_uri, status, created_at, updated_at, last_connected_at
		FROM fyers_accounts
		WHERE user_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
		`,
		userID,
	)

	account, err := scanFyersAccount(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFyersAccountNotFound
		}
		return nil, err
	}
	return account, nil
}

func UpsertFyersToken(ctx context.Context, accountID int64, accessTokenEncrypted, refreshTokenEncrypted, tokenType string, expiresAt *time.Time) (*FyersToken, error) {
	accessTokenEncrypted = strings.TrimSpace(accessTokenEncrypted)
	refreshTokenEncrypted = strings.TrimSpace(refreshTokenEncrypted)
	tokenType = strings.TrimSpace(tokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}
	if accountID == 0 || accessTokenEncrypted == "" {
		return nil, errors.New("account id and access token are required")
	}

	row := DB.QueryRow(
		ctx,
		`
		INSERT INTO fyers_tokens (account_id, access_token_encrypted, refresh_token_encrypted, token_type, expires_at, refreshed_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (account_id)
		DO UPDATE SET
			access_token_encrypted = EXCLUDED.access_token_encrypted,
			refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
			token_type = EXCLUDED.token_type,
			expires_at = EXCLUDED.expires_at,
			refreshed_at = NOW(),
			updated_at = NOW()
		RETURNING id, account_id, access_token_encrypted, refresh_token_encrypted, token_type, expires_at, refreshed_at, created_at, updated_at
		`,
		accountID,
		accessTokenEncrypted,
		nullIfEmpty(refreshTokenEncrypted),
		tokenType,
		expiresAt,
	)

	return scanFyersToken(row)
}

func GetFyersTokenByAccountID(ctx context.Context, accountID int64) (*FyersToken, error) {
	row := DB.QueryRow(
		ctx,
		`
		SELECT id, account_id, access_token_encrypted, refresh_token_encrypted, token_type, expires_at, refreshed_at, created_at, updated_at
		FROM fyers_tokens
		WHERE account_id = $1
		`,
		accountID,
	)

	token, err := scanFyersToken(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFyersAccountNotFound
		}
		return nil, err
	}
	return token, nil
}

func UpsertUserRuntimeStatus(ctx context.Context, userID, accountID int64, runtimeState string, lastWSConnectAt, lastTickAt *time.Time, lastError string) (*UserRuntimeStatus, error) {
	if userID == 0 || accountID == 0 {
		return nil, errors.New("user id and account id are required")
	}

	row := DB.QueryRow(
		ctx,
		`
		INSERT INTO user_runtime_status (user_id, account_id, runtime_state, last_ws_connect_at, last_tick_at, last_error, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, account_id)
		DO UPDATE SET
			runtime_state = EXCLUDED.runtime_state,
			last_ws_connect_at = EXCLUDED.last_ws_connect_at,
			last_tick_at = EXCLUDED.last_tick_at,
			last_error = EXCLUDED.last_error,
			updated_at = NOW()
		RETURNING id, user_id, account_id, runtime_state, last_ws_connect_at, last_tick_at, last_error, updated_at
		`,
		userID,
		accountID,
		strings.TrimSpace(runtimeState),
		lastWSConnectAt,
		lastTickAt,
		nullIfEmpty(lastError),
	)

	return scanUserRuntimeStatus(row)
}

func nullIfEmpty(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func normalizeFyersStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return "pending"
	}
	return status
}

func scanFyersAccount(row rowScanner) (*FyersAccount, error) {
	account := &FyersAccount{}
	if err := row.Scan(
		&account.ID,
		&account.UserID,
		&account.BrokerUserID,
		&account.AppID,
		&account.RedirectURI,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.LastConnectedAt,
	); err != nil {
		return nil, err
	}
	return account, nil
}

func scanFyersToken(row rowScanner) (*FyersToken, error) {
	token := &FyersToken{}
	if err := row.Scan(
		&token.ID,
		&token.AccountID,
		&token.AccessTokenEncrypted,
		&token.RefreshTokenEncrypted,
		&token.TokenType,
		&token.ExpiresAt,
		&token.RefreshedAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return token, nil
}

func scanUserRuntimeStatus(row rowScanner) (*UserRuntimeStatus, error) {
	status := &UserRuntimeStatus{}
	if err := row.Scan(
		&status.ID,
		&status.UserID,
		&status.AccountID,
		&status.RuntimeState,
		&status.LastWSConnectAt,
		&status.LastTickAt,
		&status.LastError,
		&status.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return status, nil
}
