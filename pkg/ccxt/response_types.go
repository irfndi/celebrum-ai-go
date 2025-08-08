package ccxt

import (
	"time"

	"github.com/shopspring/decimal"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	Version   string `json:"version"`
}

// ExchangeInfo represents information about a cryptocurrency exchange
type ExchangeInfo struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Countries []string               `json:"countries"`
	URLs      map[string]interface{} `json:"urls"`
}

// ExchangesResponse represents the response from the exchanges endpoint
type ExchangesResponse struct {
	Exchanges []ExchangeInfo `json:"exchanges"`
}

// Ticker represents ticker data from an exchange
type Ticker struct {
	Symbol    string          `json:"symbol"`
	Last      decimal.Decimal `json:"last"`
	Bid       decimal.Decimal `json:"bid"`
	Ask       decimal.Decimal `json:"ask"`
	Volume    decimal.Decimal `json:"volume"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	Change    decimal.Decimal `json:"change"`
	Percentage decimal.Decimal `json:"percentage"`
	Timestamp time.Time       `json:"timestamp"`
}

// TickerResponse represents the response from the ticker endpoint
type TickerResponse struct {
	Exchange  string `json:"exchange"`
	Symbol    string `json:"symbol"`
	Ticker    Ticker `json:"ticker"`
	Timestamp string `json:"timestamp"`
}

// TickerData represents ticker data with exchange information
type TickerData struct {
	Exchange string `json:"exchange"`
	Ticker   Ticker `json:"ticker"`
}

// TickersRequest represents the request for multiple tickers
type TickersRequest struct {
	Symbols   []string `json:"symbols"`
	Exchanges []string `json:"exchanges,omitempty"`
}

// TickersResponse represents the response from the tickers endpoint
type TickersResponse struct {
	Tickers   []TickerData `json:"tickers"`
	Timestamp string       `json:"timestamp"`
}

// OrderBookEntry represents a single order book entry
type OrderBookEntry struct {
	Price  decimal.Decimal `json:"price"`
	Amount decimal.Decimal `json:"amount"`
}

// OrderBook represents order book data
type OrderBook struct {
	Symbol    string            `json:"symbol"`
	Bids      []OrderBookEntry  `json:"bids"`
	Asks      []OrderBookEntry  `json:"asks"`
	Timestamp time.Time         `json:"timestamp"`
	Nonce     int64             `json:"nonce"`
}

// OrderBookResponse represents the response from the order book endpoint
type OrderBookResponse struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	OrderBook OrderBook `json:"orderbook"`
	Timestamp string    `json:"timestamp"`
}

// OHLCV represents OHLCV (candlestick) data
type OHLCV struct {
	Timestamp time.Time       `json:"timestamp"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`
}

// OHLCVResponse represents the response from the OHLCV endpoint
type OHLCVResponse struct {
	Exchange  string   `json:"exchange"`
	Symbol    string   `json:"symbol"`
	Timeframe string   `json:"timeframe"`
	OHLCV     []OHLCV  `json:"ohlcv"`
	Timestamp string   `json:"timestamp"`
}

// Trade represents a single trade
type Trade struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Symbol    string          `json:"symbol"`
	Side      string          `json:"side"` // 'buy' or 'sell'
	Amount    decimal.Decimal `json:"amount"`
	Price     decimal.Decimal `json:"price"`
	Cost      decimal.Decimal `json:"cost"`
}

// TradesResponse represents the response from the trades endpoint
type TradesResponse struct {
	Exchange  string  `json:"exchange"`
	Symbol    string  `json:"symbol"`
	Trades    []Trade `json:"trades"`
	Timestamp string  `json:"timestamp"`
}

// MarketsResponse represents the response from the markets endpoint
type MarketsResponse struct {
	Exchange  string   `json:"exchange"`
	Symbols   []string `json:"symbols"`
	Count     int      `json:"count"`
	Timestamp string   `json:"timestamp"`
}

// ErrorResponse represents an error response from the CCXT service
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}