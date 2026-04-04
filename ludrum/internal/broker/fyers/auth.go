package fyers

import (
	"context"
	"fmt"
	"time"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
)

type OAuthClient interface {
	LoginURL() string
	ExchangeAuthCode(ctx context.Context, authCode string) (*TokenSet, error)
}

type Client struct {
	config AuthConfig
	raw    *fyersgosdk.Client
}

func NewClient(config AuthConfig) *Client {
	return &Client{
		config: config,
		raw:    fyersgosdk.SetClientData(config.AppID, config.SecretID, config.RedirectURL),
	}
}

func (c *Client) LoginURL() string {
	return c.raw.GetLoginURL()
}

func (c *Client) ExchangeAuthCode(ctx context.Context, authCode string) (*TokenSet, error) {
	type result struct {
		token map[string]interface{}
		err   error
	}

	ch := make(chan result, 1)
	go func() {
		tokenJSON, err := c.raw.GenerateAccessToken(authCode, c.raw)
		if err != nil {
			ch <- result{err: err}
			return
		}

		parsed, err := parseTokenResponse(tokenJSON)
		ch <- result{token: parsed, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-ch:
		if out.err != nil {
			return nil, out.err
		}
		return mapTokenSet(out.token), nil
	}
}

func parseTokenResponse(raw string) (map[string]interface{}, error) {
	token, err := decodeMap(raw)
	if err != nil {
		return nil, fmt.Errorf("parse fyers token response: %w", err)
	}
	return token, nil
}

func mapTokenSet(raw map[string]interface{}) *TokenSet {
	set := &TokenSet{
		AccessToken:  stringValue(raw["access_token"]),
		RefreshToken: stringValue(raw["refresh_token"]),
		TokenType:    defaultString(stringValue(raw["token_type"]), "Bearer"),
		Raw:          raw,
	}

	if expiresAt := parseExpiry(raw["expires_at"]); expiresAt != nil {
		set.ExpiresAt = expiresAt
	}

	return set
}

func parseExpiry(value interface{}) *time.Time {
	switch v := value.(type) {
	case float64:
		t := time.Unix(int64(v), 0).UTC()
		return &t
	case int64:
		t := time.Unix(v, 0).UTC()
		return &t
	case string:
		if v == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			t := parsed.UTC()
			return &t
		}
	}
	return nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
