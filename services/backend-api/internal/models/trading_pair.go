package models

import (
	"time"
)

// TradingPair represents a specific trading pair (e.g., BTC/USD) on a specific exchange.
// It links the base and quote currencies and tracks the active status of the pair.
type TradingPair struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	ExchangeID    int       `json:"exchange_id" gorm:"not null;index"`
	Symbol        string    `json:"symbol" gorm:"size:50;not null"`
	BaseCurrency  string    `json:"base_currency" gorm:"size:10;not null"`
	QuoteCurrency string    `json:"quote_currency" gorm:"size:10;not null"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Exchange Exchange `json:"exchange" gorm:"foreignKey:ExchangeID;references:ID"`
}

// TableName returns the database table name for the TradingPair model.
func (TradingPair) TableName() string {
	return "trading_pairs"
}

// String returns the string representation of the trading pair, typically its symbol.
func (tp *TradingPair) String() string {
	return tp.Symbol
}

// GetFullSymbol returns a standardized symbol string in the format "BASE/QUOTE".
func (tp *TradingPair) GetFullSymbol() string {
	return tp.BaseCurrency + "/" + tp.QuoteCurrency
}
