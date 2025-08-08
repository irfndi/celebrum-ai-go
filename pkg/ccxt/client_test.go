package ccxt_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
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
	assert.Equal(t, cfg.ServiceURL, client.BaseURL)
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
				json.NewEncoder(w).Encode(tt.responseBody)
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
		json.NewEncoder(w).Encode(ccxt.ExchangesResponse{
			Exchanges: expectedExchanges,
		})
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
		Timestamp: time.Now(),
		High:      decimal.NewFromFloat(45000.0),
		Low:       decimal.NewFromFloat(43000.0),
		Bid:       decimal.NewFromFloat(44500.0),
		Ask:       decimal.NewFromFloat(44550.0),
		Last:      decimal.NewFromFloat(44525.0),
		Close:     decimal.NewFromFloat(44525.0),
		Volume:    decimal.NewFromFloat(1234.56),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/ticker/binance/BTC/USDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ccxt.TickerResponse{
			Ticker: expectedTicker,
		})
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
				Symbol: "BTC/USDT",
				Bid:    decimal.NewFromFloat(44500.0),
				Ask:    decimal.NewFromFloat(44550.0),
				Last:   decimal.NewFromFloat(44525.0),
				Timestamp: time.Now(),
			},
		},
		{
			Exchange: "coinbase",
			Ticker: ccxt.Ticker{
				Symbol: "BTC/USDT",
				Bid:    decimal.NewFromFloat(44480.0),
				Ask:    decimal.NewFromFloat(44530.0),
				Last:   decimal.NewFromFloat(44505.0),
				Timestamp: time.Now(),
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
		json.NewEncoder(w).Encode(ccxt.TickersResponse{
			Tickers:   expectedTickers,
			Timestamp: time.Now().Format(time.RFC3339),
		})
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
		assert.Equal(t, "/api/orderbook/binance/BTC/USDT", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ccxt.OrderBookResponse{
			OrderBook: expectedOrderBook,
		})
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

func TestClient_Close(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}
	client := ccxt.NewClient(cfg)

	err := client.Close()
	assert.NoError(t, err)
}