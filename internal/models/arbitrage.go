package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// PriceRange represents a price range from multiple exchanges
type PriceRange struct {
	Min decimal.Decimal `json:"min"`
	Max decimal.Decimal `json:"max"`
	Avg decimal.Decimal `json:"avg"`
}

// ProfitRange represents profit range based on best/worst price combinations
type ProfitRange struct {
	MinPercentage decimal.Decimal `json:"min_percentage"`
	MaxPercentage decimal.Decimal `json:"max_percentage"`
	MinAmount     decimal.Decimal `json:"min_amount"`
	MaxAmount     decimal.Decimal `json:"max_amount"`
	BaseAmount    decimal.Decimal `json:"base_amount"` // Amount used for calculation (e.g., $20,000)
}

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

	// Enhanced fields for price ranges and multiple exchanges
	BuyPriceRange   *PriceRange     `json:"buy_price_range,omitempty"`
	SellPriceRange  *PriceRange     `json:"sell_price_range,omitempty"`
	ProfitRange     *ProfitRange    `json:"profit_range,omitempty"`
	BuyExchanges    []string        `json:"buy_exchanges,omitempty"`
	SellExchanges   []string        `json:"sell_exchanges,omitempty"`
	MinVolume       decimal.Decimal `json:"min_volume,omitempty"`
	EstimatedVolume decimal.Decimal `json:"estimated_volume,omitempty"`
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

	// Enhanced fields for price ranges and multiple exchanges
	BuyPriceRange   *PriceRange     `json:"buy_price_range,omitempty"`
	SellPriceRange  *PriceRange     `json:"sell_price_range,omitempty"`
	ProfitRange     *ProfitRange    `json:"profit_range,omitempty"`
	BuyExchanges    []string        `json:"buy_exchanges,omitempty"`
	SellExchanges   []string        `json:"sell_exchanges,omitempty"`
	MinVolume       decimal.Decimal `json:"min_volume,omitempty"`
	ValidForMinutes int             `json:"valid_for_minutes,omitempty"`
}

// ArbitrageOpportunitiesResponse represents the response for arbitrage opportunities list
type ArbitrageOpportunitiesResponse struct {
	Opportunities []ArbitrageOpportunityResponse `json:"opportunities"`
	Count         int                            `json:"count"`
	Timestamp     time.Time                      `json:"timestamp"`
}
