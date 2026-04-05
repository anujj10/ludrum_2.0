package fyers

import "time"

type AuthConfig struct {
	AppID       string
	SecretID    string
	RedirectURL string
}

type TokenSet struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    *time.Time
	Raw          map[string]interface{}
}

type MarketSubscription struct {
	Symbols  []string
	DataType string
}

type RuntimeConfig struct {
	UserID          int64
	AccountID       int64
	AppID           string
	BrokerUserID    string
	AccessToken     string
	TrackedSymbols  []string
	OptionChainRoot string
	StrikeCount     int
	PollInterval    time.Duration
}
