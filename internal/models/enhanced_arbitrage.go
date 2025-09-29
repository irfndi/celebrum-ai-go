package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// EnhancedArbitrageOpportunity represents an arbitrage opportunity with price ranges and volume data
type EnhancedArbitrageOpportunity struct {
	ID                  string               `json:"id" db:"id"`
	Symbol              string               `json:"symbol"`
	BuyPriceRange       PriceRange           `json:"buy_price_range"`
	SellPriceRange      PriceRange           `json:"sell_price_range"`
	ProfitRange         ProfitRange          `json:"profit_range"`
	BuyExchanges        []ExchangePrice      `json:"buy_exchanges"`
	SellExchanges       []ExchangePrice      `json:"sell_exchanges"`
	MinVolume           decimal.Decimal      `json:"min_volume"`
	TotalVolume         decimal.Decimal      `json:"total_volume"`
	ValidityDuration    time.Duration        `json:"validity_duration"`
	DetectedAt          time.Time            `json:"detected_at"`
	ExpiresAt           time.Time            `json:"expires_at"`
	QualityScore        decimal.Decimal      `json:"quality_score"`
	VolumeWeightedPrice VolumeWeightedPrices `json:"volume_weighted_price"`
}

// ExchangePrice represents price and volume data from a specific exchange
type ExchangePrice struct {
	ExchangeID   int             `json:"exchange_id"`
	ExchangeName string          `json:"exchange_name"`
	Price        decimal.Decimal `json:"price"`
	Volume       decimal.Decimal `json:"volume"`
	Spread       decimal.Decimal `json:"spread"`
	Reliability  decimal.Decimal `json:"reliability"` // 0-1 score
}

// VolumeWeightedPrices represents volume-weighted average prices
type VolumeWeightedPrices struct {
	BuyVWAP  decimal.Decimal `json:"buy_vwap"`
	SellVWAP decimal.Decimal `json:"sell_vwap"`
}

// ArbitrageAggregationInput represents input for enhanced arbitrage aggregation
type ArbitrageAggregationInput struct {
	Opportunities []ArbitrageOpportunity `json:"opportunities"`
	MinVolume     decimal.Decimal        `json:"min_volume"`
	MaxSpread     decimal.Decimal        `json:"max_spread"`
	BaseAmount    decimal.Decimal        `json:"base_amount"` // For profit calculation
}

// ArbitrageQualityMetrics represents quality assessment metrics for arbitrage opportunities
type ArbitrageQualityMetrics struct {
	VolumeScore     decimal.Decimal `json:"volume_score"`
	SpreadScore     decimal.Decimal `json:"spread_score"`
	ExchangeScore   decimal.Decimal `json:"exchange_score"`
	LiquidityScore  decimal.Decimal `json:"liquidity_score"`
	OverallScore    decimal.Decimal `json:"overall_score"`
	IsAcceptable    bool            `json:"is_acceptable"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
}
