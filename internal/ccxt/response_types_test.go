package ccxt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test UnixTimestamp type
func TestUnixTimestamp_UnmarshalJSON(t *testing.T) {
	// Test valid timestamp
	jsonData := []byte("1640995200000") // 2022-01-01 00:00:00 UTC

	var ut UnixTimestamp
	err := ut.UnmarshalJSON(jsonData)
	require.NoError(t, err)

	expected := time.Unix(1640995200, 0)
	assert.Equal(t, expected, ut.Time())
}

func TestUnixTimestamp_UnmarshalJSON_InvalidData(t *testing.T) {
	// Test invalid JSON
	invalidJSON := []byte("invalid")

	var ut UnixTimestamp
	err := ut.UnmarshalJSON(invalidJSON)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal timestamp")
}

func TestUnixTimestamp_MarshalJSON(t *testing.T) {
	// Test marshaling
	t1 := time.Unix(1640995200, 0)
	ut := UnixTimestamp(t1)

	jsonData, err := ut.MarshalJSON()
	require.NoError(t, err)

	expected := "1640995200000"
	assert.Equal(t, expected, string(jsonData))
}

func TestUnixTimestamp_Time(t *testing.T) {
	// Test Time() method
	t1 := time.Unix(1640995200, 500000000) // with nanoseconds
	ut := UnixTimestamp(t1)

	result := ut.Time()
	assert.Equal(t, t1, result)
}

// Test HealthResponse struct
func TestHealthResponse_Struct(t *testing.T) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: "2022-01-01T00:00:00Z",
		Service:   "ccxt",
		Version:   "1.0.0",
	}

	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "2022-01-01T00:00:00Z", response.Timestamp)
	assert.Equal(t, "ccxt", response.Service)
	assert.Equal(t, "1.0.0", response.Version)
}

func TestHealthResponse_JSONMarshaling(t *testing.T) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: "2022-01-01T00:00:00Z",
		Service:   "ccxt",
		Version:   "1.0.0",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "ok")
	assert.Contains(t, string(jsonData), "ccxt")

	// Test JSON unmarshaling
	var unmarshaled HealthResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.Status, unmarshaled.Status)
	assert.Equal(t, response.Service, unmarshaled.Service)
}

// Test ExchangeInfo struct
func TestExchangeInfo_Struct(t *testing.T) {
	exchange := ExchangeInfo{
		ID:        "binance",
		Name:      "Binance",
		Countries: []string{"MT"},
		URLs: map[string]interface{}{
			"api": "https://api.binance.com",
		},
	}

	assert.Equal(t, "binance", exchange.ID)
	assert.Equal(t, "Binance", exchange.Name)
	assert.Equal(t, []string{"MT"}, exchange.Countries)
	assert.Equal(t, "https://api.binance.com", exchange.URLs["api"])
}

func TestExchangeInfo_JSONMarshaling(t *testing.T) {
	exchange := ExchangeInfo{
		ID:        "binance",
		Name:      "Binance",
		Countries: []string{"MT"},
		URLs: map[string]interface{}{
			"api": "https://api.binance.com",
		},
	}

	jsonData, err := json.Marshal(exchange)
	require.NoError(t, err)

	var unmarshaled ExchangeInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, exchange.ID, unmarshaled.ID)
	assert.Equal(t, exchange.Name, unmarshaled.Name)
}

// Test ExchangesResponse struct
func TestExchangesResponse_Struct(t *testing.T) {
	exchanges := []ExchangeInfo{
		{ID: "binance", Name: "Binance"},
		{ID: "coinbase", Name: "Coinbase"},
	}

	response := ExchangesResponse{
		Exchanges: exchanges,
	}

	assert.Len(t, response.Exchanges, 2)
	assert.Equal(t, "binance", response.Exchanges[0].ID)
	assert.Equal(t, "coinbase", response.Exchanges[1].ID)
}

// Test Ticker struct
func TestTicker_Struct(t *testing.T) {
	ticker := Ticker{
		Symbol:     "BTC/USDT",
		Last:       decimal.NewFromFloat(50000.0),
		Bid:        decimal.NewFromFloat(49999.0),
		Ask:        decimal.NewFromFloat(50001.0),
		Volume:     decimal.NewFromFloat(1000.0),
		High:       decimal.NewFromFloat(51000.0),
		Low:        decimal.NewFromFloat(49000.0),
		Open:       decimal.NewFromFloat(49500.0),
		Close:      decimal.NewFromFloat(50000.0),
		Change:     decimal.NewFromFloat(500.0),
		Percentage: decimal.NewFromFloat(1.01),
		Timestamp:  UnixTimestamp(time.Now()),
	}

	assert.Equal(t, "BTC/USDT", ticker.Symbol)
	assert.True(t, ticker.Last.Equal(decimal.NewFromFloat(50000.0)))
	assert.True(t, ticker.Bid.Equal(decimal.NewFromFloat(49999.0)))
	assert.True(t, ticker.Ask.Equal(decimal.NewFromFloat(50001.0)))
}

func TestTicker_JSONMarshaling(t *testing.T) {
	ticker := Ticker{
		Symbol: "BTC/USDT",
		Last:   decimal.NewFromFloat(50000.0),
		Bid:    decimal.NewFromFloat(49999.0),
		Ask:    decimal.NewFromFloat(50001.0),
	}

	jsonData, err := json.Marshal(ticker)
	require.NoError(t, err)

	var unmarshaled Ticker
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, ticker.Symbol, unmarshaled.Symbol)
	assert.True(t, ticker.Last.Equal(unmarshaled.Last))
}

// Test TickerResponse struct
func TestTickerResponse_Struct(t *testing.T) {
	ticker := Ticker{
		Symbol: "BTC/USDT",
		Last:   decimal.NewFromFloat(50000.0),
	}

	response := TickerResponse{
		Exchange:  "binance",
		Symbol:    "BTC/USDT",
		Ticker:    ticker,
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Equal(t, "binance", response.Exchange)
	assert.Equal(t, "BTC/USDT", response.Symbol)
	assert.Equal(t, ticker.Symbol, response.Ticker.Symbol)
}

// Test TickerData struct
func TestTickerData_Struct(t *testing.T) {
	ticker := Ticker{
		Symbol: "BTC/USDT",
		Last:   decimal.NewFromFloat(50000.0),
	}

	data := TickerData{
		Exchange: "binance",
		Ticker:   ticker,
	}

	assert.Equal(t, "binance", data.Exchange)
	assert.Equal(t, ticker.Symbol, data.Ticker.Symbol)
}

// Test TickersRequest struct
func TestTickersRequest_Struct(t *testing.T) {
	request := TickersRequest{
		Symbols:   []string{"BTC/USDT", "ETH/USDT"},
		Exchanges: []string{"binance", "coinbase"},
	}

	assert.Len(t, request.Symbols, 2)
	assert.Len(t, request.Exchanges, 2)
	assert.Contains(t, request.Symbols, "BTC/USDT")
	assert.Contains(t, request.Exchanges, "binance")
}

func TestTickersRequest_JSONMarshaling(t *testing.T) {
	request := TickersRequest{
		Symbols:   []string{"BTC/USDT"},
		Exchanges: []string{"binance"},
	}

	jsonData, err := json.Marshal(request)
	require.NoError(t, err)

	var unmarshaled TickersRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, request.Symbols, unmarshaled.Symbols)
	assert.Equal(t, request.Exchanges, unmarshaled.Exchanges)
}

// Test TickersResponse struct
func TestTickersResponse_Struct(t *testing.T) {
	tickers := []TickerData{
		{Exchange: "binance", Ticker: Ticker{Symbol: "BTC/USDT"}},
		{Exchange: "coinbase", Ticker: Ticker{Symbol: "ETH/USDT"}},
	}

	response := TickersResponse{
		Tickers:   tickers,
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Len(t, response.Tickers, 2)
	assert.Equal(t, "binance", response.Tickers[0].Exchange)
	assert.Equal(t, "coinbase", response.Tickers[1].Exchange)
}

// Test OrderBookEntry struct
func TestOrderBookEntry_Struct(t *testing.T) {
	entry := OrderBookEntry{
		Price:  decimal.NewFromFloat(50000.0),
		Amount: decimal.NewFromFloat(1.5),
	}

	assert.True(t, entry.Price.Equal(decimal.NewFromFloat(50000.0)))
	assert.True(t, entry.Amount.Equal(decimal.NewFromFloat(1.5)))
}

func TestOrderBookEntry_JSONMarshaling(t *testing.T) {
	entry := OrderBookEntry{
		Price:  decimal.NewFromFloat(50000.0),
		Amount: decimal.NewFromFloat(1.5),
	}

	jsonData, err := json.Marshal(entry)
	require.NoError(t, err)

	var unmarshaled OrderBookEntry
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.True(t, entry.Price.Equal(unmarshaled.Price))
	assert.True(t, entry.Amount.Equal(unmarshaled.Amount))
}

// Test OrderBook struct
func TestOrderBook_Struct(t *testing.T) {
	bids := []OrderBookEntry{
		{Price: decimal.NewFromFloat(49999.0), Amount: decimal.NewFromFloat(1.0)},
	}
	asks := []OrderBookEntry{
		{Price: decimal.NewFromFloat(50001.0), Amount: decimal.NewFromFloat(1.0)},
	}

	orderBook := OrderBook{
		Symbol:    "BTC/USDT",
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
		Nonce:     12345,
	}

	assert.Equal(t, "BTC/USDT", orderBook.Symbol)
	assert.Len(t, orderBook.Bids, 1)
	assert.Len(t, orderBook.Asks, 1)
	assert.Equal(t, int64(12345), orderBook.Nonce)
}

// Test OrderBookResponse struct
func TestOrderBookResponse_Struct(t *testing.T) {
	orderBook := OrderBook{
		Symbol: "BTC/USDT",
		Bids:   []OrderBookEntry{},
		Asks:   []OrderBookEntry{},
	}

	response := OrderBookResponse{
		Exchange:  "binance",
		Symbol:    "BTC/USDT",
		OrderBook: orderBook,
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Equal(t, "binance", response.Exchange)
	assert.Equal(t, "BTC/USDT", response.Symbol)
	assert.Equal(t, orderBook.Symbol, response.OrderBook.Symbol)
}

// Test OHLCV struct
func TestOHLCV_Struct(t *testing.T) {
	ohlcv := OHLCV{
		Timestamp: time.Now(),
		Open:      decimal.NewFromFloat(49000.0),
		High:      decimal.NewFromFloat(51000.0),
		Low:       decimal.NewFromFloat(48000.0),
		Close:     decimal.NewFromFloat(50000.0),
		Volume:    decimal.NewFromFloat(1000.0),
	}

	assert.True(t, ohlcv.Open.Equal(decimal.NewFromFloat(49000.0)))
	assert.True(t, ohlcv.High.Equal(decimal.NewFromFloat(51000.0)))
	assert.True(t, ohlcv.Low.Equal(decimal.NewFromFloat(48000.0)))
	assert.True(t, ohlcv.Close.Equal(decimal.NewFromFloat(50000.0)))
	assert.True(t, ohlcv.Volume.Equal(decimal.NewFromFloat(1000.0)))
}

func TestOHLCV_JSONMarshaling(t *testing.T) {
	ohlcv := OHLCV{
		Timestamp: time.Unix(1640995200, 0),
		Open:      decimal.NewFromFloat(49000.0),
		High:      decimal.NewFromFloat(51000.0),
		Low:       decimal.NewFromFloat(48000.0),
		Close:     decimal.NewFromFloat(50000.0),
		Volume:    decimal.NewFromFloat(1000.0),
	}

	jsonData, err := json.Marshal(ohlcv)
	require.NoError(t, err)

	var unmarshaled OHLCV
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.True(t, ohlcv.Open.Equal(unmarshaled.Open))
	assert.True(t, ohlcv.High.Equal(unmarshaled.High))
}

// Test OHLCVResponse struct
func TestOHLCVResponse_Struct(t *testing.T) {
	ohlcvData := []OHLCV{
		{Open: decimal.NewFromFloat(49000.0), Close: decimal.NewFromFloat(50000.0)},
	}

	response := OHLCVResponse{
		Exchange:  "binance",
		Symbol:    "BTC/USDT",
		Timeframe: "1h",
		OHLCV:     ohlcvData,
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Equal(t, "binance", response.Exchange)
	assert.Equal(t, "BTC/USDT", response.Symbol)
	assert.Equal(t, "1h", response.Timeframe)
	assert.Len(t, response.OHLCV, 1)
}

// Test Trade struct
func TestTrade_Struct(t *testing.T) {
	trade := Trade{
		ID:        "12345",
		Timestamp: time.Now(),
		Symbol:    "BTC/USDT",
		Side:      "buy",
		Amount:    decimal.NewFromFloat(1.0),
		Price:     decimal.NewFromFloat(50000.0),
		Cost:      decimal.NewFromFloat(50000.0),
	}

	assert.Equal(t, "12345", trade.ID)
	assert.Equal(t, "BTC/USDT", trade.Symbol)
	assert.Equal(t, "buy", trade.Side)
	assert.True(t, trade.Amount.Equal(decimal.NewFromFloat(1.0)))
	assert.True(t, trade.Price.Equal(decimal.NewFromFloat(50000.0)))
}

func TestTrade_JSONMarshaling(t *testing.T) {
	trade := Trade{
		ID:     "12345",
		Symbol: "BTC/USDT",
		Side:   "buy",
		Amount: decimal.NewFromFloat(1.0),
		Price:  decimal.NewFromFloat(50000.0),
	}

	jsonData, err := json.Marshal(trade)
	require.NoError(t, err)

	var unmarshaled Trade
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, trade.ID, unmarshaled.ID)
	assert.Equal(t, trade.Symbol, unmarshaled.Symbol)
	assert.True(t, trade.Amount.Equal(unmarshaled.Amount))
}

// Test TradesResponse struct
func TestTradesResponse_Struct(t *testing.T) {
	trades := []Trade{
		{ID: "1", Symbol: "BTC/USDT", Side: "buy"},
		{ID: "2", Symbol: "BTC/USDT", Side: "sell"},
	}

	response := TradesResponse{
		Exchange:  "binance",
		Symbol:    "BTC/USDT",
		Trades:    trades,
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Equal(t, "binance", response.Exchange)
	assert.Equal(t, "BTC/USDT", response.Symbol)
	assert.Len(t, response.Trades, 2)
	assert.Equal(t, "buy", response.Trades[0].Side)
	assert.Equal(t, "sell", response.Trades[1].Side)
}

// Test MarketsResponse struct
func TestMarketsResponse_Struct(t *testing.T) {
	symbols := []string{"BTC/USDT", "ETH/USDT", "ADA/USDT"}

	response := MarketsResponse{
		Exchange:  "binance",
		Symbols:   symbols,
		Count:     len(symbols),
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Equal(t, "binance", response.Exchange)
	assert.Len(t, response.Symbols, 3)
	assert.Equal(t, 3, response.Count)
	assert.Contains(t, response.Symbols, "BTC/USDT")
	assert.Contains(t, response.Symbols, "ETH/USDT")
}

func TestMarketsResponse_JSONMarshaling(t *testing.T) {
	response := MarketsResponse{
		Exchange: "binance",
		Symbols:  []string{"BTC/USDT", "ETH/USDT"},
		Count:    2,
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	var unmarshaled MarketsResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, response.Exchange, unmarshaled.Exchange)
	assert.Equal(t, response.Count, unmarshaled.Count)
	assert.Equal(t, response.Symbols, unmarshaled.Symbols)
}

// Test ErrorResponse struct
func TestErrorResponse_Struct(t *testing.T) {
	errorResp := ErrorResponse{
		Error:     "invalid_request",
		Message:   "Invalid symbol format",
		Timestamp: "2022-01-01T00:00:00Z",
	}

	assert.Equal(t, "invalid_request", errorResp.Error)
	assert.Equal(t, "Invalid symbol format", errorResp.Message)
	assert.Equal(t, "2022-01-01T00:00:00Z", errorResp.Timestamp)
}

func TestErrorResponse_JSONMarshaling(t *testing.T) {
	errorResp := ErrorResponse{
		Error:     "invalid_request",
		Message:   "Invalid symbol format",
		Timestamp: "2022-01-01T00:00:00Z",
	}

	jsonData, err := json.Marshal(errorResp)
	require.NoError(t, err)

	var unmarshaled ErrorResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, errorResp.Error, unmarshaled.Error)
	assert.Equal(t, errorResp.Message, unmarshaled.Message)
	assert.Equal(t, errorResp.Timestamp, unmarshaled.Timestamp)
}

// Test edge cases and error conditions
func TestUnixTimestamp_EdgeCases(t *testing.T) {
	// Test zero timestamp - should use current time instead of epoch
	zeroJSON := []byte("0")
	var ut UnixTimestamp
	before := time.Now()
	err := ut.UnmarshalJSON(zeroJSON)
	require.NoError(t, err)
	after := time.Now()
	// Verify the timestamp is between before and after (current time)
	assert.True(t, ut.Time().After(before.Add(-time.Second)) && ut.Time().Before(after.Add(time.Second)))

	// Test negative timestamp
	negativeJSON := []byte("-1000")
	err = ut.UnmarshalJSON(negativeJSON)
	require.NoError(t, err)
	assert.Equal(t, time.Unix(-1, 0), ut.Time())
}

func TestTicker_EmptyValues(t *testing.T) {
	// Test ticker with zero/empty values
	ticker := Ticker{
		Symbol: "",
		Last:   decimal.Zero,
		Bid:    decimal.Zero,
		Ask:    decimal.Zero,
	}

	assert.Empty(t, ticker.Symbol)
	assert.True(t, ticker.Last.IsZero())
	assert.True(t, ticker.Bid.IsZero())
	assert.True(t, ticker.Ask.IsZero())
}

func TestOrderBook_EmptyOrderBook(t *testing.T) {
	// Test empty order book
	orderBook := OrderBook{
		Symbol: "BTC/USDT",
		Bids:   []OrderBookEntry{},
		Asks:   []OrderBookEntry{},
	}

	assert.Equal(t, "BTC/USDT", orderBook.Symbol)
	assert.Empty(t, orderBook.Bids)
	assert.Empty(t, orderBook.Asks)
}

func TestTrade_SideValidation(t *testing.T) {
	// Test different trade sides
	buyTrade := Trade{Side: "buy"}
	sellTrade := Trade{Side: "sell"}

	assert.Equal(t, "buy", buyTrade.Side)
	assert.Equal(t, "sell", sellTrade.Side)
}
