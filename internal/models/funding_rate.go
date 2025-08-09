package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// FundingRate represents funding rate data from an exchange
type FundingRate struct {
	ID              int64            `json:"id" db:"id"`
	ExchangeID      int              `json:"exchange_id" db:"exchange_id"`
	TradingPairID   int              `json:"trading_pair_id" db:"trading_pair_id"`
	FundingRate     decimal.Decimal  `json:"funding_rate" db:"funding_rate"`
	FundingTime     time.Time        `json:"funding_time" db:"funding_time"`
	NextFundingTime *time.Time       `json:"next_funding_time" db:"next_funding_time"`
	MarkPrice       *decimal.Decimal `json:"mark_price" db:"mark_price"`
	IndexPrice      *decimal.Decimal `json:"index_price" db:"index_price"`
	Timestamp       time.Time        `json:"timestamp" db:"timestamp"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`

	// Joined fields
	ExchangeName  string `json:"exchange_name,omitempty" db:"exchange_name"`
	Symbol        string `json:"symbol,omitempty" db:"symbol"`
	BaseCurrency  string `json:"base_currency,omitempty" db:"base_currency"`
	QuoteCurrency string `json:"quote_currency,omitempty" db:"quote_currency"`
}

// FundingArbitrageOpportunity represents a funding rate arbitrage opportunity
type FundingArbitrageOpportunity struct {
	ID                        int64            `json:"id" db:"id"`
	TradingPairID             int              `json:"trading_pair_id" db:"trading_pair_id"`
	LongExchangeID            int              `json:"long_exchange_id" db:"long_exchange_id"`
	ShortExchangeID           int              `json:"short_exchange_id" db:"short_exchange_id"`
	LongFundingRate           decimal.Decimal  `json:"long_funding_rate" db:"long_funding_rate"`
	ShortFundingRate          decimal.Decimal  `json:"short_funding_rate" db:"short_funding_rate"`
	NetFundingRate            decimal.Decimal  `json:"net_funding_rate" db:"net_funding_rate"`
	EstimatedProfit8h         decimal.Decimal  `json:"estimated_profit_8h" db:"estimated_profit_8h"`
	EstimatedProfitDaily      decimal.Decimal  `json:"estimated_profit_daily" db:"estimated_profit_daily"`
	EstimatedProfitPercentage decimal.Decimal  `json:"estimated_profit_percentage" db:"estimated_profit_percentage"`
	LongMarkPrice             *decimal.Decimal `json:"long_mark_price" db:"long_mark_price"`
	ShortMarkPrice            *decimal.Decimal `json:"short_mark_price" db:"short_mark_price"`
	PriceDifference           *decimal.Decimal `json:"price_difference" db:"price_difference"`
	PriceDifferencePercentage *decimal.Decimal `json:"price_difference_percentage" db:"price_difference_percentage"`
	RiskScore                 decimal.Decimal  `json:"risk_score" db:"risk_score"`
	IsActive                  bool             `json:"is_active" db:"is_active"`
	DetectedAt                time.Time        `json:"detected_at" db:"detected_at"`
	ExpiresAt                 *time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt                 time.Time        `json:"created_at" db:"created_at"`

	// Joined fields
	Symbol            string `json:"symbol,omitempty" db:"symbol"`
	BaseCurrency      string `json:"base_currency,omitempty" db:"base_currency"`
	QuoteCurrency     string `json:"quote_currency,omitempty" db:"quote_currency"`
	LongExchangeName  string `json:"long_exchange_name,omitempty" db:"long_exchange_name"`
	ShortExchangeName string `json:"short_exchange_name,omitempty" db:"short_exchange_name"`
}

// FundingArbitrageOpportunityResponse represents the API response for funding arbitrage opportunities
type FundingArbitrageOpportunityResponse struct {
	ID                        int64    `json:"id"`
	Symbol                    string   `json:"symbol"`
	BaseCurrency              string   `json:"base_currency"`
	QuoteCurrency             string   `json:"quote_currency"`
	LongExchange              string   `json:"long_exchange"`
	ShortExchange             string   `json:"short_exchange"`
	LongFundingRate           float64  `json:"long_funding_rate"`
	ShortFundingRate          float64  `json:"short_funding_rate"`
	NetFundingRate            float64  `json:"net_funding_rate"`
	EstimatedProfit8h         float64  `json:"estimated_profit_8h"`
	EstimatedProfitDaily      float64  `json:"estimated_profit_daily"`
	EstimatedProfitPercentage float64  `json:"estimated_profit_percentage"`
	LongMarkPrice             *float64 `json:"long_mark_price"`
	ShortMarkPrice            *float64 `json:"short_mark_price"`
	PriceDifference           *float64 `json:"price_difference"`
	PriceDifferencePercentage *float64 `json:"price_difference_percentage"`
	RiskScore                 float64  `json:"risk_score"`
	DetectedAt                string   `json:"detected_at"`
	ExpiresAt                 *string  `json:"expires_at"`
}

// FundingRateRequest represents the request for funding rate data
type FundingRateRequest struct {
	Exchange string   `json:"exchange"`
	Symbols  []string `json:"symbols,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

// FundingRateResponse represents the API response for funding rate data
type FundingRateResponse struct {
	Exchange        string   `json:"exchange"`
	Symbol          string   `json:"symbol"`
	BaseCurrency    string   `json:"base_currency"`
	QuoteCurrency   string   `json:"quote_currency"`
	FundingRate     float64  `json:"funding_rate"`
	FundingTime     string   `json:"funding_time"`
	NextFundingTime *string  `json:"next_funding_time"`
	MarkPrice       *float64 `json:"mark_price"`
	IndexPrice      *float64 `json:"index_price"`
	Timestamp       string   `json:"timestamp"`
}

// FundingArbitrageRequest represents the request for funding arbitrage opportunities
type FundingArbitrageRequest struct {
	Symbols   []string `json:"symbols,omitempty"`
	Exchanges []string `json:"exchanges,omitempty"`
	MinProfit float64  `json:"min_profit,omitempty"`
	MaxRisk   float64  `json:"max_risk,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Page      int      `json:"page,omitempty"`
}
