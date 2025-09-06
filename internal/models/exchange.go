package models

import (
	"time"
)

// Exchange represents a cryptocurrency exchange
type Exchange struct {
	ID        int        `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	APIURL    string     `json:"api_url" db:"api_url"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	Priority  int        `json:"priority" db:"priority"`
	LastPing  *time.Time `json:"last_ping" db:"last_ping"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// CCXTExchange represents CCXT-specific exchange configuration
type CCXTExchange struct {
	ID               int        `json:"id" db:"id"`
	ExchangeID       int        `json:"exchange_id" db:"exchange_id"`
	CCXTID           string     `json:"ccxt_id" db:"ccxt_id"`
	IsTestnet        bool       `json:"is_testnet" db:"is_testnet"`
	APIKeyRequired   bool       `json:"api_key_required" db:"api_key_required"`
	RateLimit        int        `json:"rate_limit" db:"rate_limit"`
	HasFutures       bool       `json:"has_futures" db:"has_futures"`
	WebsocketEnabled bool       `json:"websocket_enabled" db:"websocket_enabled"`
	LastHealthCheck  *time.Time `json:"last_health_check" db:"last_health_check"`
	Status           string     `json:"status" db:"status"`
	Exchange         *Exchange  `json:"exchange,omitempty"`
}

// ExchangeInfo represents exchange information for API responses
type ExchangeInfo struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	CCXTID           string `json:"ccxt_id"`
	IsActive         bool   `json:"is_active"`
	Priority         int    `json:"priority"`
	HasFutures       bool   `json:"has_futures"`
	WebsocketEnabled bool   `json:"websocket_enabled"`
	Status           string `json:"status"`
}
