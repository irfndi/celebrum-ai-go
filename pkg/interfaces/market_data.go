package interfaces

import (
	"time"
	"github.com/shopspring/decimal"
)

// MarketPrice represents current market price for API responses
type MarketPrice struct {
	ExchangeID   int             `json:"exchange_id"`
	ExchangeName string          `json:"exchange_name"`
	Symbol       string          `json:"symbol"`
	Price        decimal.Decimal `json:"price"`
	Volume       decimal.Decimal `json:"volume"`
	Timestamp    time.Time       `json:"timestamp"`
}

// MarketPriceInterface defines the interface for market price data
type MarketPriceInterface interface {
	GetPrice() float64
	GetVolume() float64
	GetTimestamp() time.Time
	GetExchangeName() string
	GetSymbol() string
}

// GetPrice returns the price as float64
func (mp *MarketPrice) GetPrice() float64 {
	return mp.Price.InexactFloat64()
}

// GetVolume returns the volume as float64
func (mp *MarketPrice) GetVolume() float64 {
	return mp.Volume.InexactFloat64()
}

// GetTimestamp returns the timestamp
func (mp *MarketPrice) GetTimestamp() time.Time {
	return mp.Timestamp
}

// GetExchangeName returns the exchange name
func (mp *MarketPrice) GetExchangeName() string {
	return mp.ExchangeName
}

// GetSymbol returns the symbol
func (mp *MarketPrice) GetSymbol() string {
	return mp.Symbol
}