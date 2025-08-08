package models

import (
	"time"
)

// TradingPair represents a cryptocurrency trading pair
type TradingPair struct {
	ID            int       `json:"id" db:"id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	BaseCurrency  string    `json:"base_currency" db:"base_currency"`
	QuoteCurrency string    `json:"quote_currency" db:"quote_currency"`
	IsFutures     bool      `json:"is_futures" db:"is_futures"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// TradingPairRequest represents request payload for trading pair operations
type TradingPairRequest struct {
	Symbol        string `json:"symbol" binding:"required"`
	BaseCurrency  string `json:"base_currency" binding:"required"`
	QuoteCurrency string `json:"quote_currency" binding:"required"`
	IsFutures     bool   `json:"is_futures"`
}

// TradingPairResponse represents trading pair information for API responses
type TradingPairResponse struct {
	ID            int    `json:"id"`
	Symbol        string `json:"symbol"`
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	IsFutures     bool   `json:"is_futures"`
}
