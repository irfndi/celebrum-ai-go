package models

import (
	"time"

	"github.com/google/uuid"
)

// TradingPair represents a trading pair in the system
type TradingPair struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Symbol        string    `json:"symbol" gorm:"size:20;not null"`
	BaseCurrency  string    `json:"base_currency" gorm:"size:10;not null"`
	QuoteCurrency string    `json:"quote_currency" gorm:"size:10;not null"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
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
