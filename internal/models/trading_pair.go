package models

import (
	"time"
)

// TradingPair represents a trading pair in the system
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

// TableName returns the table name for TradingPair
func (TradingPair) TableName() string {
	return "trading_pairs"
}

// String returns a string representation of the trading pair
func (tp *TradingPair) String() string {
	return tp.Symbol
}

// GetFullSymbol returns the full symbol representation
func (tp *TradingPair) GetFullSymbol() string {
	return tp.BaseCurrency + "/" + tp.QuoteCurrency
}
