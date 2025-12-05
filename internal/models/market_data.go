package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// MarketData represents real-time market data from exchanges
type MarketData struct {
	ID                  string          `json:"id" db:"id"`
	ExchangeID          int             `json:"exchange_id" db:"exchange_id"`
	TradingPairID       int             `json:"trading_pair_id" db:"trading_pair_id"`
	Bid                 decimal.Decimal `json:"bid" db:"bid"`
	BidVolume           decimal.Decimal `json:"bid_volume" db:"bid_volume"`
	Ask                 decimal.Decimal `json:"ask" db:"ask"`
	AskVolume           decimal.Decimal `json:"ask_volume" db:"ask_volume"`
	LastPrice           decimal.Decimal `json:"last_price" db:"last_price"`
	High24h             decimal.Decimal `json:"high_24h" db:"high_24h"`
	Low24h              decimal.Decimal `json:"low_24h" db:"low_24h"`
	Open24h             decimal.Decimal `json:"open_24h" db:"open_24h"`
	Close24h            decimal.Decimal `json:"close_24h" db:"close_24h"`
	Volume24h           decimal.Decimal `json:"volume_24h" db:"volume_24h"`
	QuoteVolume24h      decimal.Decimal `json:"quote_volume_24h" db:"quote_volume_24h"`
	Vwap                decimal.Decimal `json:"vwap" db:"vwap"`
	Change24h           decimal.Decimal `json:"change_24h" db:"change_24h"`
	ChangePercentage24h decimal.Decimal `json:"change_percentage_24h" db:"change_percentage_24h"`
	Timestamp           time.Time       `json:"timestamp" db:"timestamp"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	Exchange            *Exchange       `json:"exchange,omitempty"`
	TradingPair         *TradingPair    `json:"trading_pair,omitempty"`
}

// MarketPrice represents current market price for API responses
type MarketPrice struct {
	ExchangeID   int             `json:"exchange_id"`
	ExchangeName string          `json:"exchange_name"`
	Symbol       string          `json:"symbol"`
	Bid          decimal.Decimal `json:"bid"`
	BidVolume    decimal.Decimal `json:"bid_volume"`
	Ask          decimal.Decimal `json:"ask"`
	AskVolume    decimal.Decimal `json:"ask_volume"`
	Price        decimal.Decimal `json:"price"` // Last traded price
	Volume       decimal.Decimal `json:"volume"`
	Timestamp    time.Time       `json:"timestamp"`
}

// GetPrice returns the price as float64
func (mp *MarketPrice) GetPrice() float64 {
	return mp.Price.InexactFloat64()
}

// GetVolume returns the volume as float64
func (mp *MarketPrice) GetVolume() float64 {
	return mp.Volume.InexactFloat64()
}

// GetTimestamp returns the timestamp
func (mp *MarketPrice) GetTimestamp() time.Time {
	return mp.Timestamp
}

// GetExchangeName returns the exchange name
func (mp *MarketPrice) GetExchangeName() string {
	return mp.ExchangeName
}

// GetSymbol returns the symbol
func (mp *MarketPrice) GetSymbol() string {
	return mp.Symbol
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
