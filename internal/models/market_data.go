package models

import (
	"github.com/shopspring/decimal"
	"time"
)

// MarketData represents real-time market data from exchanges
type MarketData struct {
	ID            string          `json:"id" db:"id"`
	ExchangeID    int             `json:"exchange_id" db:"exchange_id"`
	TradingPairID int             `json:"trading_pair_id" db:"trading_pair_id"`
	Price         decimal.Decimal `json:"price" db:"price"`
	Volume        decimal.Decimal `json:"volume" db:"volume"`
	Timestamp     time.Time       `json:"timestamp" db:"timestamp"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	Exchange      *Exchange       `json:"exchange,omitempty"`
	TradingPair   *TradingPair    `json:"trading_pair,omitempty"`
}

// MarketPrice represents current market price for API responses
type MarketPrice struct {
	ExchangeID   int             `json:"exchange_id"`
	ExchangeName string          `json:"exchange_name"`
	Symbol       string          `json:"symbol"`
	Price        decimal.Decimal `json:"price"`
	Volume       decimal.Decimal `json:"volume"`
	Timestamp    time.Time       `json:"timestamp"`
}

// TickerData represents real-time ticker information from CCXT
type TickerData struct {
	Symbol    string          `json:"symbol"`
	Bid       decimal.Decimal `json:"bid"`
	Ask       decimal.Decimal `json:"ask"`
	Last      decimal.Decimal `json:"last"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Volume    decimal.Decimal `json:"volume"`
	Timestamp time.Time       `json:"timestamp"`
}

// OrderBookData represents order book information from CCXT
type OrderBookData struct {
	Symbol    string              `json:"symbol"`
	Bids      [][]decimal.Decimal `json:"bids"`
	Asks      [][]decimal.Decimal `json:"asks"`
	Timestamp time.Time           `json:"timestamp"`
}

// MarketDataRequest represents request parameters for market data
type MarketDataRequest struct {
	Symbols  []string `json:"symbols" form:"symbols"`
	Exchange string   `json:"exchange" form:"exchange"`
	Limit    int      `json:"limit" form:"limit"`
}
