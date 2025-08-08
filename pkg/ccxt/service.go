package ccxt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	service := NewService(cfg)
	assert.NotNil(t, service)
	assert.NotNil(t, service.client)
}

func TestService_Initialize(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   interface{}
		expectError    bool
	}{
		{
			name:           "successful initialization",
			responseStatus: http.StatusOK,
			responseBody: HealthResponse{
				Status:    "ok",
				Timestamp: time.Now(),
				Version:   "1.0.0",
			},
			expectError: false,
		},
		{
			name:           "service unavailable",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   ErrorResponse{Error: "Service unavailable"},
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			cfg := &config.CCXTConfig{
				ServiceURL: server.URL,
				Timeout:    30,
			}
			service := NewService(cfg)

			ctx := context.Background()
			err := service.Initialize(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetSupportedExchanges(t *testing.T) {
	expectedExchanges := []ExchangeInfo{
		{ID: "binance", Name: "Binance", Status: "ok"},
		{ID: "coinbase", Name: "Coinbase Pro", Status: "ok"},
		{ID: "kraken", Name: "Kraken", Status: "ok"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
			return
		}

		if r.URL.Path == "/api/exchanges" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ExchangesResponse{
				Exchanges: expectedExchanges,
				Count:     len(expectedExchanges),
				Timestamp: time.Now(),
			})
			return
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	service := NewService(cfg)

	// Initialize service first
	ctx := context.Background()
	err := service.Initialize(ctx)
	require.NoError(t, err)

	exchanges := service.GetSupportedExchanges()
	require.NotEmpty(t, exchanges)

	assert.Len(t, exchanges, 3)
	assert.Contains(t, exchanges, "binance")
	assert.Contains(t, exchanges, "coinbase")
	assert.Contains(t, exchanges, "kraken")

}

func TestService_FetchSingleTicker(t *testing.T) {
	expectedTicker := Ticker{
		Symbol:    "BTC/USDT",
		Timestamp: time.Now(),
		Datetime:  time.Now().Format(time.RFC3339),
		Bid:       decimal.NewFromFloat(44500.0),
		Ask:       decimal.NewFromFloat(44550.0),
		Last:      decimal.NewFromFloat(44525.0),
		Volume:    decimal.NewFromFloat(1234.56),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Timestamp: time.Now()})
			return
		}

		if r.URL.Path == "/api/exchanges" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ExchangesResponse{
				Exchanges: []ExchangeInfo{{ID: "binance", Name: "Binance", Status: "ok"}},
				Count:     1,
				Timestamp: time.Now(),
			})
			return
		}

		if r.URL.Path == "/api/ticker/binance/BTC/USDT" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TickerResponse{
				Exchange:  "binance",
				Symbol:    "BTC/USDT",
				Ticker:    expectedTicker,
				Timestamp: time.Now(),
			})
			return
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	service := NewService(cfg)

	// Initialize service first
	ctx := context.Background()
	err := service.Initialize(ctx)
	require.NoError(t, err)

	marketPrice, err := service.FetchSingleTicker(ctx, "binance", "BTC/USDT")
	require.NoError(t, err)
	require.NotNil(t, marketPrice)

	assert.Equal(t, "BTC/USDT", marketPrice.Symbol)
	assert.True(t, marketPrice.Price.Equal(decimal.NewFromFloat(44525.0)))
	assert.True(t, marketPrice.Volume.Equal(decimal.NewFromFloat(1234.56)))
}

func TestService_FetchMarketData(t *testing.T) {
	expectedTickers := []TickerData{
		{
			Exchange: "binance",
			Ticker: Ticker{
				Symbol: "BTC/USDT",
				Bid:    decimal.NewFromFloat(44500.0),
				Ask:    decimal.NewFromFloat(44550.0),
				Last:   decimal.NewFromFloat(44525.0),
				Timestamp: time.Now(),
			},
		},
		{
			Exchange: "binance",
			Ticker: Ticker{
				Symbol: "ETH/USDT",
				Bid:    decimal.NewFromFloat(2800.0),
				Ask:    decimal.NewFromFloat(2805.0),
				Last:   decimal.NewFromFloat(2802.5),
				Timestamp: time.Now(),
			},
		},
		{
			Exchange: "coinbase",
			Ticker: Ticker{
				Symbol: "BTC/USDT",
				Bid:    decimal.NewFromFloat(44480.0),
				Ask:    decimal.NewFromFloat(44530.0),
				Last:   decimal.NewFromFloat(44505.0),
				Timestamp: time.Now(),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Timestamp: time.Now()})
			return
		}

		if r.URL.Path == "/api/exchanges" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ExchangesResponse{
				Exchanges: []ExchangeInfo{
					{ID: "binance", Name: "Binance", Status: "ok"},
					{ID: "coinbase", Name: "Coinbase", Status: "ok"},
				},
				Count:     2,
				Timestamp: time.Now(),
			})
			return
		}

		if r.URL.Path == "/api/tickers" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TickersResponse{
				Tickers:   expectedTickers,
				Count:     len(expectedTickers),
				Timestamp: time.Now(),
			})
			return
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	service := NewService(cfg)

	// Initialize service first
	ctx := context.Background()
	err := service.Initialize(ctx)
	require.NoError(t, err)

	marketPrices, err := service.FetchMarketData(ctx, []string{"binance", "coinbase"}, []string{"BTC/USDT", "ETH/USDT"})
	require.NoError(t, err)
	require.NotEmpty(t, marketPrices)

	// Should have 3 market prices: binance BTC/USDT, binance ETH/USDT, coinbase BTC/USDT
	assert.Len(t, marketPrices, 3)

	// Check that we have the expected symbols and exchanges
	symbolExchangePairs := make(map[string]bool)
	for _, mp := range marketPrices {
		key := mp.Symbol + "@" + mp.ExchangeName
		symbolExchangePairs[key] = true
	}

	assert.True(t, symbolExchangePairs["BTC/USDT@binance"])
	assert.True(t, symbolExchangePairs["ETH/USDT@binance"])
	assert.True(t, symbolExchangePairs["BTC/USDT@coinbase"])
}

func TestService_CalculateArbitrageOpportunities(t *testing.T) {
	// Mock tickers with arbitrage opportunity
	expectedTickers := []TickerData{
		{
			Exchange: "binance",
			Ticker: Ticker{
				Symbol: "BTC/USDT",
				Bid:    decimal.NewFromFloat(44500.0), // Higher bid on binance
				Ask:    decimal.NewFromFloat(44550.0),
				Last:   decimal.NewFromFloat(44525.0),
				Timestamp: time.Now(),
			},
		},
		{
			Exchange: "coinbase",
			Ticker: Ticker{
				Symbol: "BTC/USDT",
				Bid:    decimal.NewFromFloat(44300.0),
				Ask:    decimal.NewFromFloat(44350.0), // Lower ask on coinbase
				Last:   decimal.NewFromFloat(44325.0),
				Timestamp: time.Now(),
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Timestamp: time.Now()})
			return
		}

		if r.URL.Path == "/api/exchanges" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ExchangesResponse{
				Exchanges: []ExchangeInfo{
					{ID: "binance", Name: "Binance", Status: "ok"},
					{ID: "coinbase", Name: "Coinbase", Status: "ok"},
				},
				Count:     2,
				Timestamp: time.Now(),
			})
			return
		}

		if r.URL.Path == "/api/tickers" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TickersResponse{
				Tickers:   expectedTickers,
				Count:     len(expectedTickers),
				Timestamp: time.Now(),
			})
			return
		}
	}))
	defer server.Close()

	cfg := &config.CCXTConfig{
		ServiceURL: server.URL,
		Timeout:    30,
	}
	service := NewService(cfg)

	// Initialize service first
	ctx := context.Background()
	err := service.Initialize(ctx)
	require.NoError(t, err)

	minProfitPercent := decimal.NewFromFloat(0.1) // 0.1%
	opportunities, err := service.CalculateArbitrageOpportunities(ctx, []string{"binance", "coinbase"}, []string{"BTC/USDT"}, minProfitPercent)
	require.NoError(t, err)
	require.NotEmpty(t, opportunities)

	// Should find arbitrage opportunity: buy on coinbase (44350), sell on binance (44500)
	assert.Len(t, opportunities, 1)
	opp := opportunities[0]
	assert.Equal(t, "BTC/USDT", opp.Symbol)
	assert.Equal(t, "coinbase", opp.BuyExchange)
	assert.Equal(t, "binance", opp.SellExchange)
	assert.True(t, opp.BuyPrice.Equal(decimal.NewFromFloat(44350.0)))
	assert.True(t, opp.SellPrice.Equal(decimal.NewFromFloat(44500.0)))

	// Calculate expected profit percentage: (44500 - 44350) / 44350 * 100 ≈ 0.338%
	expectedProfit := opp.SellPrice.Sub(opp.BuyPrice).Div(opp.BuyPrice).Mul(decimal.NewFromInt(100))
	assert.True(t, opp.ProfitPercentage.Equal(expectedProfit))
	assert.True(t, opp.ProfitPercentage.GreaterThan(minProfitPercent))
}

func TestService_IsHealthy(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   interface{}
		expected       bool
	}{
		{
			name:           "healthy service",
			responseStatus: http.StatusOK,
			responseBody:   HealthResponse{Status: "ok"},
			expected:       true,
		},
		{
			name:           "unhealthy service",
			responseStatus: http.StatusServiceUnavailable,
			responseBody:   ErrorResponse{Error: "Service unavailable"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			cfg := &config.CCXTConfig{
				ServiceURL: server.URL,
				Timeout:    30,
			}
			service := NewService(cfg)

			ctx := context.Background()
			result := service.IsHealthy(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestService_Close(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}
	service := NewService(cfg)

	err := service.Close()
	assert.NoError(t, err)
}