package ccxt

import (
	"time"

	"github.com/shopspring/decimal"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime,omitempty"`
	Version   string    `json:"version,omitempty"`
}

// ErrorResponse represents an error response from the CCXT service
type ErrorResponse struct {
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// ExchangeInfo represents information about a supported exchange
type ExchangeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Countries   string `json:"countries,omitempty"`
	RateLimit   int    `json:"rateLimit"`
	HasCORS     bool   `json:"hasCORS"`
	HasSpot     bool   `json:"hasSpot"`
	HasFutures  bool   `json:"hasFutures"`
	HasMargin   bool   `json:"hasMargin"`
	HasOption   bool   `json:"hasOption"`
	Sandbox     bool   `json:"sandbox"`
	Status      string `json:"status"`
}

// ExchangesResponse represents the response from /api/exchanges
type ExchangesResponse struct {
	Exchanges []ExchangeInfo `json:"exchanges"`
	Count     int            `json:"count"`
	Timestamp time.Time      `json:"timestamp"`
}

// Ticker represents ticker data from an exchange
type Ticker struct {
	Symbol    string          `json:"symbol"`
	Bid       decimal.Decimal `json:"bid"`
	BidVolume decimal.Decimal `json:"bidVolume"`
	Ask       decimal.Decimal `json:"ask"`
	AskVolume decimal.Decimal `json:"askVolume"`
	Last      decimal.Decimal `json:"last"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Open      decimal.Decimal `json:"open"`
	Close     decimal.Decimal `json:"close"`
	Volume    decimal.Decimal `json:"volume"`
	QuoteVolume decimal.Decimal `json:"quoteVolume"`
	VWAP      decimal.Decimal `json:"vwap"`
	Change    decimal.Decimal `json:"change"`
	Percentage decimal.Decimal `json:"percentage"`
	Timestamp time.Time       `json:"timestamp"`
	Datetime  string          `json:"datetime"`
}

// TickerResponse represents the response from /api/ticker/{exchange}/{symbol}
type TickerResponse struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Ticker    Ticker    `json:"ticker"`
	Timestamp time.Time `json:"timestamp"`
}

// TickersRequest represents the request body for /api/tickers
type TickersRequest struct {
	Symbols   []string `json:"symbols"`
	Exchanges []string `json:"exchanges"`
}

// TickersResponse represents the response from /api/tickers
type TickersResponse struct {
	Tickers   []TickerData `json:"tickers"`
	Count     int          `json:"count"`
	Timestamp time.Time    `json:"timestamp"`
}

// TickerData represents ticker data with exchange information
type TickerData struct {
	Exchange string `json:"exchange"`
	Ticker   Ticker `json:"ticker"`
}

// OrderBookEntry represents a single order book entry (bid or ask)
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
	Datetime  string            `json:"datetime"`
	Nonce     int64             `json:"nonce,omitempty"`
}

// OrderBookResponse represents the response from /api/orderbook/{exchange}/{symbol}
type OrderBookResponse struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	OrderBook OrderBook `json:"orderbook"`
	Timestamp time.Time `json:"timestamp"`
}

// Trade represents a single trade
type Trade struct {
	ID        string          `json:"id"`
	Order     string          `json:"order,omitempty"`
	Symbol    string          `json:"symbol"`
	Side      string          `json:"side"` // 'buy' or 'sell'
	Amount    decimal.Decimal `json:"amount"`
	Price     decimal.Decimal `json:"price"`
	Cost      decimal.Decimal `json:"cost"`
	Fee       *TradeFee       `json:"fee,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Datetime  string          `json:"datetime"`
}

// TradeFee represents trading fee information
type TradeFee struct {
	Currency string          `json:"currency"`
	Cost     decimal.Decimal `json:"cost"`
	Rate     decimal.Decimal `json:"rate,omitempty"`
}

// TradesResponse represents the response from /api/trades/{exchange}/{symbol}
type TradesResponse struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Trades    []Trade   `json:"trades"`
	Count     int       `json:"count"`
	Timestamp time.Time `json:"timestamp"`
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

// OHLCVResponse represents the response from /api/ohlcv/{exchange}/{symbol}
type OHLCVResponse struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Timeframe string    `json:"timeframe"`
	OHLCV     []OHLCV   `json:"ohlcv"`
	Count     int       `json:"count"`
	Timestamp time.Time `json:"timestamp"`
}

// Market represents a trading pair/market
type Market struct {
	ID       string          `json:"id"`
	Symbol   string          `json:"symbol"`
	Base     string          `json:"base"`
	Quote    string          `json:"quote"`
	Settle   string          `json:"settle,omitempty"`
	Type     string          `json:"type"` // 'spot', 'future', 'option', etc.
	Spot     bool            `json:"spot"`
	Margin   bool            `json:"margin"`
	Future   bool            `json:"future"`
	Option   bool            `json:"option"`
	Active   bool            `json:"active"`
	Contract bool            `json:"contract"`
	Linear   bool            `json:"linear,omitempty"`
	Inverse  bool            `json:"inverse,omitempty"`
	Taker    decimal.Decimal `json:"taker,omitempty"`
	Maker    decimal.Decimal `json:"maker,omitempty"`
	ContractSize decimal.Decimal `json:"contractSize,omitempty"`
	Expiry   time.Time       `json:"expiry,omitempty"`
	ExpiryDatetime string    `json:"expiryDatetime,omitempty"`
	Strike   decimal.Decimal `json:"strike,omitempty"`
	OptionType string        `json:"optionType,omitempty"`
	Precision *MarketPrecision `json:"precision,omitempty"`
	Limits   *MarketLimits   `json:"limits,omitempty"`
	Info     interface{}     `json:"info,omitempty"`
}

// MarketPrecision represents precision information for a market
type MarketPrecision struct {
	Amount int `json:"amount"`
	Price  int `json:"price"`
}

// MarketLimits represents trading limits for a market
type MarketLimits struct {
	Amount *LimitRange `json:"amount,omitempty"`
	Price  *LimitRange `json:"price,omitempty"`
	Cost   *LimitRange `json:"cost,omitempty"`
}

// LimitRange represents min/max limits
type LimitRange struct {
	Min decimal.Decimal `json:"min,omitempty"`
	Max decimal.Decimal `json:"max,omitempty"`
}

// MarketsResponse represents the response from /api/markets/{exchange}
type MarketsResponse struct {
	Exchange  string    `json:"exchange"`
	Symbols   []string  `json:"symbols"`
	Markets   []Market  `json:"markets,omitempty"`
	Count     int       `json:"count"`
	Timestamp time.Time `json:"timestamp"`
}