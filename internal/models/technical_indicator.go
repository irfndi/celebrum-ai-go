package models

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// TechnicalIndicator represents calculated technical analysis indicators
type TechnicalIndicator struct {
	ID            string          `json:"id" db:"id"`
	TradingPairID int             `json:"trading_pair_id" db:"trading_pair_id"`
	IndicatorType string          `json:"indicator_type" db:"indicator_type"`
	Timeframe     string          `json:"timeframe" db:"timeframe"`
	Values        json.RawMessage `json:"values" db:"values"`
	CalculatedAt  time.Time       `json:"calculated_at" db:"calculated_at"`
	TradingPair   *TradingPair    `json:"trading_pair,omitempty"`
}

// IndicatorData represents structured indicator calculation results
type IndicatorData struct {
	Symbol     string                 `json:"symbol"`
	Timeframe  string                 `json:"timeframe"`
	Indicators map[string]interface{} `json:"indicators"`
	Timestamp  time.Time              `json:"timestamp"`
}

// RSIData represents RSI indicator values
type RSIData struct {
	Value     decimal.Decimal `json:"value"`
	Signal    string          `json:"signal"` // "oversold", "overbought", "neutral"
	Timestamp time.Time       `json:"timestamp"`
}

// MACDData represents MACD indicator values
type MACDData struct {
	MACD      decimal.Decimal `json:"macd"`
	Signal    decimal.Decimal `json:"signal"`
	Histogram decimal.Decimal `json:"histogram"`
	Trend     string          `json:"trend"` // "bullish", "bearish", "neutral"
	Timestamp time.Time       `json:"timestamp"`
}

// SMAData represents Simple Moving Average values
type SMAData struct {
	Period    int             `json:"period"`
	Value     decimal.Decimal `json:"value"`
	Trend     string          `json:"trend"` // "up", "down", "sideways"
	Timestamp time.Time       `json:"timestamp"`
}

// Signal represents a trading signal generated from technical analysis
type Signal struct {
	Type       string          `json:"type"`     // "buy", "sell", "hold"
	Strength   string          `json:"strength"` // "strong", "weak", "neutral"
	Price      decimal.Decimal `json:"price"`
	Indicator  string          `json:"indicator"`  // Source indicator
	Confidence decimal.Decimal `json:"confidence"` // 0-100
	Timestamp  time.Time       `json:"timestamp"`
}

// TechnicalAnalysisRequest represents request for technical analysis
type TechnicalAnalysisRequest struct {
	Symbol     string   `json:"symbol" binding:"required"`
	Timeframe  string   `json:"timeframe" binding:"required"`
	Indicators []string `json:"indicators" binding:"required"`
	Period     int      `json:"period"`
}

// TechnicalAnalysisResponse represents technical analysis results
type TechnicalAnalysisResponse struct {
	Data      IndicatorData `json:"data"`
	Signals   []Signal      `json:"signals"`
	Timestamp time.Time     `json:"timestamp"`
}
