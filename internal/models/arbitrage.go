package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ArbitrageOpportunity represents a detected arbitrage opportunity
type ArbitrageOpportunity struct {
	ID               string          `json:"id" db:"id"`
	TradingPairID    int             `json:"trading_pair_id" db:"trading_pair_id"`
	BuyExchangeID    int             `json:"buy_exchange_id" db:"buy_exchange_id"`
	SellExchangeID   int             `json:"sell_exchange_id" db:"sell_exchange_id"`
	BuyPrice         decimal.Decimal `json:"buy_price" db:"buy_price"`
	SellPrice        decimal.Decimal `json:"sell_price" db:"sell_price"`
	ProfitPercentage decimal.Decimal `json:"profit_percentage" db:"profit_percentage"`
	DetectedAt       time.Time       `json:"detected_at" db:"detected_at"`
	ExpiresAt        time.Time       `json:"expires_at" db:"expires_at"`
	TradingPair      *TradingPair    `json:"trading_pair,omitempty"`
	BuyExchange      *Exchange       `json:"buy_exchange,omitempty"`
	SellExchange     *Exchange       `json:"sell_exchange,omitempty"`
}

// ArbitrageOpportunityRequest represents request parameters for arbitrage opportunities
type ArbitrageOpportunityRequest struct {
	MinProfit decimal.Decimal `json:"min_profit" form:"min_profit"`
	Symbol    string          `json:"symbol" form:"symbol"`
	Limit     int             `json:"limit" form:"limit"`
}

// ArbitrageOpportunityResponse represents arbitrage opportunity for API responses
type ArbitrageOpportunityResponse struct {
	ID               string          `json:"id"`
	Symbol           string          `json:"symbol"`
	BuyExchange      string          `json:"buy_exchange"`
	SellExchange     string          `json:"sell_exchange"`
	BuyPrice         decimal.Decimal `json:"buy_price"`
	SellPrice        decimal.Decimal `json:"sell_price"`
	ProfitPercentage decimal.Decimal `json:"profit_percentage"`
	DetectedAt       time.Time       `json:"detected_at"`
	ExpiresAt        time.Time       `json:"expires_at"`
}

// ArbitrageOpportunitiesResponse represents the response for arbitrage opportunities list
type ArbitrageOpportunitiesResponse struct {
	Opportunities []ArbitrageOpportunityResponse `json:"opportunities"`
	Count         int                            `json:"count"`
	Timestamp     time.Time                      `json:"timestamp"`
}
