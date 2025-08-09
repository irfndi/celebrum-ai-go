package ccxt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// UnixTimestamp is a custom type that can unmarshal Unix timestamps (in milliseconds)
type UnixTimestamp time.Time

// UnmarshalJSON implements json.Unmarshaler for UnixTimestamp
func (ut *UnixTimestamp) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}
	*ut = UnixTimestamp(time.Unix(timestamp/1000, (timestamp%1000)*1000000))
	return nil
}

// MarshalJSON implements json.Marshaler for UnixTimestamp
func (ut UnixTimestamp) MarshalJSON() ([]byte, error) {
	t := time.Time(ut)
	return json.Marshal(t.Unix() * 1000)
}

// Time returns the underlying time.Time
func (ut UnixTimestamp) Time() time.Time {
	return time.Time(ut)
}

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
	Symbol     string          `json:"symbol"`
	Last       decimal.Decimal `json:"last"`
	Bid        decimal.Decimal `json:"bid"`
	Ask        decimal.Decimal `json:"ask"`
	Volume     decimal.Decimal `json:"volume"`
	High       decimal.Decimal `json:"high"`
	Low        decimal.Decimal `json:"low"`
	Open       decimal.Decimal `json:"open"`
	Close      decimal.Decimal `json:"close"`
	Change     decimal.Decimal `json:"change"`
	Percentage decimal.Decimal `json:"percentage"`
	Timestamp  UnixTimestamp   `json:"timestamp"`
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
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookEntry `json:"bids"`
	Asks      []OrderBookEntry `json:"asks"`
	Timestamp time.Time        `json:"timestamp"`
	Nonce     int64            `json:"nonce"`
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
	Exchange  string  `json:"exchange"`
	Symbol    string  `json:"symbol"`
	Timeframe string  `json:"timeframe"`
	OHLCV     []OHLCV `json:"ohlcv"`
	Timestamp string  `json:"timestamp"`
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

// FundingRate represents funding rate data from an exchange
type FundingRate struct {
	Symbol           string        `json:"symbol"`
	FundingRate      float64       `json:"fundingRate"`
	FundingTimestamp UnixTimestamp `json:"fundingTimestamp"`
	NextFundingTime  UnixTimestamp `json:"nextFundingTime"`
	MarkPrice        float64       `json:"markPrice"`
	IndexPrice       float64       `json:"indexPrice"`
	Timestamp        UnixTimestamp `json:"timestamp"`
}

// FundingRateResponse represents the response from the funding rates endpoint
type FundingRateResponse struct {
	Exchange     string        `json:"exchange"`
	FundingRates []FundingRate `json:"fundingRates"`
	Count        int           `json:"count"`
	Timestamp    string        `json:"timestamp"`
}

// FundingArbitrageOpportunity represents a funding rate arbitrage opportunity
type FundingArbitrageOpportunity struct {
	Symbol                    string        `json:"symbol"`
	LongExchange              string        `json:"longExchange"`
	ShortExchange             string        `json:"shortExchange"`
	LongFundingRate           float64       `json:"longFundingRate"`
	ShortFundingRate          float64       `json:"shortFundingRate"`
	NetFundingRate            float64       `json:"netFundingRate"`
	EstimatedProfit8h         float64       `json:"estimatedProfit8h"`
	EstimatedProfitDaily      float64       `json:"estimatedProfitDaily"`
	EstimatedProfitPercentage float64       `json:"estimatedProfitPercentage"`
	LongMarkPrice             float64       `json:"longMarkPrice"`
	ShortMarkPrice            float64       `json:"shortMarkPrice"`
	PriceDifference           float64       `json:"priceDifference"`
	PriceDifferencePercentage float64       `json:"priceDifferencePercentage"`
	RiskScore                 float64       `json:"riskScore"`
	Timestamp                 UnixTimestamp `json:"timestamp"`
}

// ErrorResponse represents an error response from the CCXT service
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}
