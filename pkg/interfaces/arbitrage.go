package interfaces

import (
	"github.com/shopspring/decimal"
	"time"
)

// ArbitrageOpportunityResponse represents an arbitrage opportunity
type ArbitrageOpportunityResponse struct {
	Symbol           string          `json:"symbol"`
	BuyExchange      string          `json:"buy_exchange"`
	SellExchange     string          `json:"sell_exchange"`
	BuyPrice         decimal.Decimal `json:"buy_price"`
	SellPrice        decimal.Decimal `json:"sell_price"`
	ProfitPercentage decimal.Decimal `json:"profit_percentage"`
	DetectedAt       time.Time       `json:"detected_at"`
	ExpiresAt        time.Time       `json:"expires_at"`
}

// ArbitrageOpportunityInterface defines the interface for arbitrage opportunity data
type ArbitrageOpportunityInterface interface {
	GetSymbol() string
	GetBuyExchange() string
	GetSellExchange() string
	GetBuyPrice() decimal.Decimal
	GetSellPrice() decimal.Decimal
	GetProfitPercentage() decimal.Decimal
	GetDetectedAt() time.Time
	GetExpiresAt() time.Time
}

// GetSymbol returns the symbol
func (ao *ArbitrageOpportunityResponse) GetSymbol() string {
	return ao.Symbol
}

// GetBuyExchange returns the buy exchange
func (ao *ArbitrageOpportunityResponse) GetBuyExchange() string {
	return ao.BuyExchange
}

// GetSellExchange returns the sell exchange
func (ao *ArbitrageOpportunityResponse) GetSellExchange() string {
	return ao.SellExchange
}

// GetBuyPrice returns the buy price
func (ao *ArbitrageOpportunityResponse) GetBuyPrice() decimal.Decimal {
	return ao.BuyPrice
}

// GetSellPrice returns the sell price
func (ao *ArbitrageOpportunityResponse) GetSellPrice() decimal.Decimal {
	return ao.SellPrice
}

// GetProfitPercentage returns the profit percentage
func (ao *ArbitrageOpportunityResponse) GetProfitPercentage() decimal.Decimal {
	return ao.ProfitPercentage
}

// GetDetectedAt returns the detection time
func (ao *ArbitrageOpportunityResponse) GetDetectedAt() time.Time {
	return ao.DetectedAt
}

// GetExpiresAt returns the expiration time
func (ao *ArbitrageOpportunityResponse) GetExpiresAt() time.Time {
	return ao.ExpiresAt
}
