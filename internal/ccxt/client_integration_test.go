package ccxt_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/ccxt"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	client := ccxt.NewClient(cfg)
	assert.NotNil(t, client)
	assert.Equal(t, cfg.ServiceURL, client.BaseURL())
	assert.NotNil(t, client.HTTPClient)
}

func TestClient_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   interface{}
		expectError    bool
	}{
		{
			name:           "successful health check",
			responseStatus: http.StatusOK,
			responseBody: ccxt.HealthResponse{
				Status:    "ok",
				Timestamp: time.Now().Format(time.RFC3339),
				Version:   "1.0.0",
			},
			expectError: false,
		},
		{
			name:           "server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   ccxt.ErrorResponse{Error: "Internal server error"},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/health", r.URL.Path)
				assert.Equal(t, "GET", r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				if err := json.NewEncoder(w).Encode(tt.responseBody); err != nil {
					t.Errorf("Failed to encode response: %v", err)
				}
			}))
			defer server.Close()

			cfg := &config.CCXTConfig{
				ServiceURL: server.URL,
				Timeout:    30,
			}
			client := ccxt.NewClient(cfg)

			ctx := context.Background()
			resp, err := client.HealthCheck(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, "ok", resp.Status)
			}
		})
	}
}

func TestClient_GetExchanges(t *testing.T) {
	expectedExchanges := []ccxt.ExchangeInfo{
		{
			ID:        "binance",
			Name:      "Binance",
			Countries: []string{"MT"},
			URLs:      map[string]interface{}{"api": "https://api.binance.com"},
		},
		{
			ID:        "coinbase",
			Name:      "Coinbase Pro",
			Countries: []string{"US"},
			URLs:      map[string]interface{}{"api": "https://api.pro.coinbase.com"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/exchanges", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.ExchangesResponse{
			Exchanges: expectedExchanges,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetExchanges(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Exchanges, 2)
	assert.Equal(t, "binance", resp.Exchanges[0].ID)
	assert.Equal(t, "coinbase", resp.Exchanges[1].ID)
}

func TestClient_GetTicker(t *testing.T) {
	expectedTicker := ccxt.Ticker{
		Symbol:    "BTC/USDT",
		Timestamp: ccxt.UnixTimestamp(time.Now()),
		High:      decimal.NewFromFloat(45000.0),
		Low:       decimal.NewFromFloat(43000.0),
		Bid:       decimal.NewFromFloat(44500.0),
		Ask:       decimal.NewFromFloat(44550.0),
		Last:      decimal.NewFromFloat(44525.0),
		Close:     decimal.NewFromFloat(44525.0),
		Volume:    decimal.NewFromFloat(1234.56),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/ticker/binance/BTCUSDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.TickerResponse{
			Ticker: expectedTicker,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetTicker(ctx, "binance", "BTC/USDT")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "BTC/USDT", resp.Ticker.Symbol)
	assert.True(t, resp.Ticker.Bid.Equal(decimal.NewFromFloat(44500.0)))
	assert.True(t, resp.Ticker.Ask.Equal(decimal.NewFromFloat(44550.0)))
}

func TestClient_GetTickers(t *testing.T) {
	request := &ccxt.TickersRequest{
		Exchanges: []string{"binance", "coinbase"},
		Symbols:   []string{"BTC/USDT", "ETH/USDT"},
	}

	expectedTickers := []ccxt.TickerData{
		{
			Exchange: "binance",
			Ticker: ccxt.Ticker{
				Symbol:    "BTC/USDT",
				Bid:       decimal.NewFromFloat(44500.0),
				Ask:       decimal.NewFromFloat(44550.0),
				Last:      decimal.NewFromFloat(44525.0),
				Timestamp: ccxt.UnixTimestamp(time.Now()),
			},
		},
		{
			Exchange: "coinbase",
			Ticker: ccxt.Ticker{
				Symbol:    "BTC/USDT",
				Bid:       decimal.NewFromFloat(44480.0),
				Ask:       decimal.NewFromFloat(44530.0),
				Last:      decimal.NewFromFloat(44505.0),
				Timestamp: ccxt.UnixTimestamp(time.Now()),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tickers", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Verify request body
		var reqBody ccxt.TickersRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, request.Exchanges, reqBody.Exchanges)
		assert.Equal(t, request.Symbols, reqBody.Symbols)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.TickersResponse{
			Tickers:   expectedTickers,
			Timestamp: time.Now().Format(time.RFC3339),
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetTickers(ctx, request)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Tickers, 2)
	assert.Equal(t, "binance", resp.Tickers[0].Exchange)
	assert.Equal(t, "coinbase", resp.Tickers[1].Exchange)
}

func TestClient_GetOrderBook(t *testing.T) {
	expectedOrderBook := ccxt.OrderBook{
		Symbol:    "BTC/USDT",
		Timestamp: time.Now(),
		Nonce:     12345,
		Bids: []ccxt.OrderBookEntry{
			{Price: decimal.NewFromFloat(44500.0), Amount: decimal.NewFromFloat(1.5)},
			{Price: decimal.NewFromFloat(44499.0), Amount: decimal.NewFromFloat(2.0)},
		},
		Asks: []ccxt.OrderBookEntry{
			{Price: decimal.NewFromFloat(44550.0), Amount: decimal.NewFromFloat(1.2)},
			{Price: decimal.NewFromFloat(44551.0), Amount: decimal.NewFromFloat(1.8)},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/orderbook/binance/BTCUSDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.OrderBookResponse{
			OrderBook: expectedOrderBook,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetOrderBook(ctx, "binance", "BTC/USDT", 10)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "BTC/USDT", resp.OrderBook.Symbol)
	assert.Len(t, resp.OrderBook.Bids, 2)
	assert.Len(t, resp.OrderBook.Asks, 2)
}

func TestClient_GetTrades(t *testing.T) {
	expectedTrades := []ccxt.Trade{
		{
			ID:        "12345",
			Timestamp: time.Now(),
			Symbol:    "BTC/USDT",
			Side:      "buy",
			Amount:    decimal.NewFromFloat(1.5),
			Price:     decimal.NewFromFloat(44500.0),
			Cost:      decimal.NewFromFloat(66750.0),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/trades/binance/BTCUSDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.TradesResponse{
			Trades: expectedTrades,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetTrades(ctx, "binance", "BTC/USDT", 50)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Trades, 1)
	assert.Equal(t, "12345", resp.Trades[0].ID)
	assert.Equal(t, "buy", resp.Trades[0].Side)
}

func TestClient_GetOHLCV(t *testing.T) {
	expectedOHLCV := []ccxt.OHLCV{
		{
			Timestamp: time.Now(),
			Open:      decimal.NewFromFloat(44000.0),
			High:      decimal.NewFromFloat(45000.0),
			Low:       decimal.NewFromFloat(43500.0),
			Close:     decimal.NewFromFloat(44500.0),
			Volume:    decimal.NewFromFloat(1234.56),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/ohlcv/binance/BTCUSDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "1h", r.URL.Query().Get("timeframe"))
		assert.Equal(t, "100", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.OHLCVResponse{
			OHLCV: expectedOHLCV,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetOHLCV(ctx, "binance", "BTC/USDT", "1h", 100)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.OHLCV, 1)
	assert.True(t, resp.OHLCV[0].Open.Equal(decimal.NewFromFloat(44000.0)))
	assert.True(t, resp.OHLCV[0].Close.Equal(decimal.NewFromFloat(44500.0)))
}

func TestClient_GetMarkets(t *testing.T) {
	expectedSymbols := []string{"BTC/USDT", "ETH/USDT"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/markets/binance", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.MarketsResponse{
			Exchange:  "binance",
			Symbols:   expectedSymbols,
			Count:     len(expectedSymbols),
			Timestamp: time.Now().Format(time.RFC3339),
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetMarkets(ctx, "binance")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Symbols, 2)
	assert.Equal(t, "binance", resp.Exchange)
	assert.Contains(t, resp.Symbols, "BTC/USDT")
}

func TestClient_GetFundingRate(t *testing.T) {
	expectedFundingRate := ccxt.FundingRate{
		Symbol:      "BTC/USDT",
		FundingRate: 0.0001,
		Timestamp:   ccxt.UnixTimestamp(time.Now()),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/funding-rate/binance/BTCUSDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedFundingRate); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetFundingRate(ctx, "binance", "BTC/USDT")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "BTC/USDT", resp.Symbol)
	assert.Equal(t, 0.0001, resp.FundingRate)
}

func TestClient_GetFundingRates(t *testing.T) {
	t.Run("with symbols", func(t *testing.T) {
		expectedRates := []ccxt.FundingRate{
			{
				Symbol:      "BTC/USDT",
				FundingRate: 0.0001,
				Timestamp:   ccxt.UnixTimestamp(time.Now()),
			},
			{
				Symbol:      "ETH/USDT",
				FundingRate: 0.0002,
				Timestamp:   ccxt.UnixTimestamp(time.Now()),
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/funding-rates/binance", r.URL.Path)
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.RawQuery, "symbols=")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(ccxt.FundingRateResponse{
				FundingRates: expectedRates,
			}); err != nil {
				t.Errorf("Failed to encode response: %v", err)
			}
		}))
		defer server.Close()

		cfg := &config.CCXTConfig{
			ServiceURL: server.URL,
			Timeout:    30,
		}
		client := ccxt.NewClient(cfg)

		ctx := context.Background()
		resp, err := client.GetFundingRates(ctx, "binance", []string{"BTC/USDT", "ETH/USDT"})

		require.NoError(t, err)
		assert.Len(t, resp, 2)
		assert.Equal(t, "BTC/USDT", resp[0].Symbol)
		assert.Equal(t, "ETH/USDT", resp[1].Symbol)
	})

	t.Run("empty symbols", func(t *testing.T) {
		cfg := &config.CCXTConfig{
			ServiceURL: "http://localhost:3001",
			Timeout:    30,
		}
		client := ccxt.NewClient(cfg)

		ctx := context.Background()
		resp, err := client.GetFundingRates(ctx, "binance", []string{})

		require.NoError(t, err)
		assert.Empty(t, resp)
	})
}

func TestClient_GetAllFundingRates(t *testing.T) {
	expectedRates := []ccxt.FundingRate{
		{
			Symbol:      "BTC/USDT",
			FundingRate: 0.0001,
			Timestamp:   ccxt.UnixTimestamp(time.Now()),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/funding-rates/binance", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Empty(t, r.URL.RawQuery)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ccxt.FundingRateResponse{
			FundingRates: expectedRates,
		}); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	ctx := context.Background()
	resp, err := client.GetAllFundingRates(ctx, "binance")

	require.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, "BTC/USDT", resp[0].Symbol)
}

func TestClient_GetExchangeConfig(t *testing.T) {
	expectedConfig := ccxt.ExchangeConfigResponse{
		ActiveExchanges:    []string{"binance", "coinbase"},
		AvailableExchanges: []string{"binance", "coinbase", "ftx"},
		Config: ccxt.ExchangeConfig{
			BlacklistedExchanges: []string{"ftx"},
			PriorityExchanges:    []string{"binance"},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/admin/exchanges/config", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedConfig); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{ServiceURL: server.URL, Timeout: 30}
	client := ccxt.NewClient(cfg)
	ctx := context.Background()
	resp, err := client.GetExchangeConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.ActiveExchanges, 2)
	assert.Len(t, resp.Config.BlacklistedExchanges, 1)
	assert.Contains(t, resp.ActiveExchanges, "binance")
}

func TestClient_AddExchangeToBlacklist(t *testing.T) {
	expectedResponse := ccxt.ExchangeManagementResponse{
		Message:   "Exchange added to blacklist",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/admin/exchanges/blacklist/ftx", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{ServiceURL: server.URL, Timeout: 30}
	client := ccxt.NewClient(cfg)
	ctx := context.Background()
	resp, err := client.AddExchangeToBlacklist(ctx, "ftx")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Exchange added to blacklist", resp.Message)
}

func TestClient_RemoveExchangeFromBlacklist(t *testing.T) {
	expectedResponse := ccxt.ExchangeManagementResponse{
		Message:   "Exchange removed from blacklist",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/admin/exchanges/blacklist/ftx", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{ServiceURL: server.URL, Timeout: 30}
	client := ccxt.NewClient(cfg)
	ctx := context.Background()
	resp, err := client.RemoveExchangeFromBlacklist(ctx, "ftx")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Exchange removed from blacklist", resp.Message)
}

func TestClient_RefreshExchanges(t *testing.T) {
	expectedResponse := ccxt.ExchangeManagementResponse{
		Message:   "Exchanges refreshed successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/admin/exchanges/refresh", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{ServiceURL: server.URL, Timeout: 30}
	client := ccxt.NewClient(cfg)
	ctx := context.Background()
	resp, err := client.RefreshExchanges(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Exchanges refreshed successfully", resp.Message)
}

func TestClient_AddExchange(t *testing.T) {
	expectedResponse := ccxt.ExchangeManagementResponse{
		Message:   "Exchange added successfully",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/admin/exchanges/add/kraken", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(expectedResponse); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{ServiceURL: server.URL, Timeout: 30}
	client := ccxt.NewClient(cfg)
	ctx := context.Background()
	resp, err := client.AddExchange(ctx, "kraken")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Exchange added successfully", resp.Message)
}

func TestClient_FormatSymbolForExchange(t *testing.T) {
	tests := []struct {
		name     string
		exchange string
		symbol   string
		expected string
	}{
		{"binance", "binance", "BTC/USDT", "/api/ticker/binance/BTCUSDT"},
		{"coinbase", "coinbase", "BTC/USDT", "/api/ticker/coinbase/BTC-USDT"},
		{"coinbasepro", "coinbasepro", "BTC/USDT", "/api/ticker/coinbasepro/BTC-USDT"},
		{"kraken", "kraken", "BTC/USDT", "/api/ticker/kraken/BTC/USDT"},
		{"okx", "okx", "BTC/USDT", "/api/ticker/okx/BTC/USDT"},
		{"unknown", "unknown", "BTC/USDT", "/api/ticker/unknown/BTCUSDT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The URL path will be decoded by the HTTP server, so BTC%2FUSDT becomes BTC/USDT
				assert.Equal(t, tt.expected, r.URL.Path)
				w.WriteHeader(http.StatusOK)
				if _, err := fmt.Fprint(w, `{"symbol":"BTC/USDT","price":50000}`); err != nil {
					t.Errorf("Failed to write response: %v", err)
				}
			}))
			defer server.Close()

			cfg := &config.CCXTConfig{
				ServiceURL: server.URL,
				Timeout:    30,
			}
			client := ccxt.NewClient(cfg)

			ctx := context.Background()
			_, err := client.GetTicker(ctx, tt.exchange, tt.symbol)
			require.NoError(t, err)
		})
	}
}

func TestClient_Close(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	err := client.Close()
	assert.NoError(t, err)
}
