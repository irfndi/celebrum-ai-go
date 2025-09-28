package ccxt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/pkg/interfaces"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newTestServiceWithServer(t *testing.T, handler http.Handler) (*Service, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	cfg := &interfaces.CCXTConfig{ServiceURL: server.URL, Timeout: 5}
	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()

	service := NewService(cfg, logger, blacklistCache)
	service.client.HTTPClient = server.Client()
	service.client.BaseURL = server.URL

	cleanup := func() {
		server.Close()
	}

	return service, cleanup
}

func TestService_AddExchange_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/admin/exchanges/add/binance", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		resp := ExchangeManagementResponse{Message: "Exchange added"}
		assert.NoError(t, json.NewEncoder(w).Encode(resp))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	response, err := service.AddExchange(context.Background(), "binance")
	assert.NoError(t, err)
	assert.Equal(t, "Exchange added", response.Message)
}

func TestService_Initialize_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/exchanges", func(w http.ResponseWriter, r *http.Request) {
		resp := ExchangesResponse{Exchanges: []ExchangeInfo{{ID: "binance", Name: "Binance"}, {ID: "bybit", Name: "Bybit"}}}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(resp))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	err := service.Initialize(context.Background())
	assert.NoError(t, err)

	exchanges := service.GetSupportedExchanges()
	assert.ElementsMatch(t, []string{"binance", "bybit"}, exchanges)
	assert.False(t, service.lastUpdate.IsZero())
}

func TestService_FetchSingleTicker_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ticker/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		response := TickerResponse{
			Exchange: "binance",
			Symbol:   "BTC/USDT",
			Ticker: Ticker{
				Symbol:    "BTC/USDT",
				Last:      decimal.NewFromInt(50000),
				Volume:    decimal.NewFromInt(1000),
				Timestamp: UnixTimestamp(ts),
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	ticker, err := service.FetchSingleTicker(context.Background(), "binance", "BTC/USDT")
	assert.NoError(t, err)
	if assert.IsType(t, &interfaces.MarketPrice{}, ticker) {
		mp := ticker.(*interfaces.MarketPrice)
		assert.Equal(t, "binance", mp.ExchangeName)
		assert.Equal(t, "BTC/USDT", mp.Symbol)
		assert.Equal(t, decimal.NewFromInt(50000), mp.Price)
	}
}

func TestService_HealthChecks(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ticker/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		response := TickerResponse{
			Exchange: "binance",
			Symbol:   "BTC/USDT",
			Ticker: Ticker{
				Symbol:    "BTC/USDT",
				Last:      decimal.NewFromInt(51000),
				Volume:    decimal.NewFromInt(500),
				Timestamp: UnixTimestamp(ts),
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	healthy, err := service.GetExchangeHealth(context.Background(), "binance")
	assert.NoError(t, err)
	assert.True(t, healthy)
	assert.True(t, service.IsHealthy(context.Background()))
}

func TestService_GetServiceURLAndClose(t *testing.T) {
	service, cleanup := newTestServiceWithServer(t, http.NewServeMux())
	defer cleanup()

	assert.NotEmpty(t, service.GetServiceURL())
	assert.NoError(t, service.Close())
}

func TestService_FetchOrderBook_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/orderbook/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := OrderBookResponse{
			Exchange: "binance",
			Symbol:   "BTC/USDT",
			OrderBook: OrderBook{
				Symbol: "BTC/USDT",
				Bids: []OrderBookEntry{
					{Price: decimal.NewFromInt(50000), Amount: decimal.NewFromFloat(1.5)},
				},
				Asks: []OrderBookEntry{
					{Price: decimal.NewFromInt(50010), Amount: decimal.NewFromFloat(0.75)},
				},
				Timestamp: ts,
				Nonce:     1,
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	orderBook, err := service.FetchOrderBook(context.Background(), "binance", "BTC/USDT", 10)
	assert.NoError(t, err)
	assert.Equal(t, "binance", orderBook.Exchange)
	assert.Equal(t, "BTC/USDT", orderBook.Symbol)
	assert.Len(t, orderBook.OrderBook.Bids, 1)
	assert.Len(t, orderBook.OrderBook.Asks, 1)
}

func TestService_FetchOrderBook_Blacklisted(t *testing.T) {
	cfg := &interfaces.CCXTConfig{ServiceURL: "http://example.com", Timeout: 5}
	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()

	service := NewService(cfg, logger, blacklistCache)
	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}
	service.blacklistCache.Add("BTC/USDT", "maintenance", time.Minute)

	orderBook, err := service.FetchOrderBook(context.Background(), "binance", "BTC/USDT", 5)
	assert.Error(t, err)
	assert.Nil(t, orderBook)
	assert.Contains(t, err.Error(), "blacklisted")
}

func TestService_FetchOHLCV_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ohlcv/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "limit=2&timeframe=1h", r.URL.RawQuery)

		response := OHLCVResponse{
			Exchange:  "binance",
			Symbol:    "BTC/USDT",
			Timeframe: "1h",
			OHLCV: []OHLCV{
				{
					Timestamp: ts,
					Open:      decimal.NewFromInt(50000),
					High:      decimal.NewFromInt(51000),
					Low:       decimal.NewFromInt(49000),
					Close:     decimal.NewFromInt(50500),
					Volume:    decimal.NewFromInt(100),
				},
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	ohlcv, err := service.FetchOHLCV(context.Background(), "binance", "BTC/USDT", "1h", 2)
	assert.NoError(t, err)
	assert.Equal(t, "1h", ohlcv.Timeframe)
	assert.Len(t, ohlcv.OHLCV, 1)
}

func TestService_FetchTrades_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/trades/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "limit=3", r.URL.RawQuery)

		response := TradesResponse{
			Exchange: "binance",
			Symbol:   "BTC/USDT",
			Trades: []Trade{
				{
					ID:        "1",
					Timestamp: ts,
					Symbol:    "BTC/USDT",
					Side:      "buy",
					Amount:    decimal.NewFromFloat(0.5),
					Price:     decimal.NewFromInt(50000),
					Cost:      decimal.NewFromFloat(25000),
				},
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	trades, err := service.FetchTrades(context.Background(), "binance", "BTC/USDT", 3)
	assert.NoError(t, err)
	assert.Equal(t, "BTC/USDT", trades.Symbol)
	assert.Len(t, trades.Trades, 1)
}

func TestService_FetchMarkets_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/markets/binance", func(w http.ResponseWriter, r *http.Request) {
		response := MarketsResponse{
			Exchange:  "binance",
			Symbols:   []string{"BTC/USDT", "ETH/USDT"},
			Count:     2,
			Timestamp: time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	markets, err := service.FetchMarkets(context.Background(), "binance")
	assert.NoError(t, err)
	assert.Equal(t, 2, markets.Count)
	assert.Contains(t, markets.Symbols, "BTC/USDT")
}

func TestService_CalculateArbitrageOpportunities_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ticker/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		response := TickerResponse{
			Exchange: "binance",
			Symbol:   "BTC/USDT",
			Ticker: Ticker{
				Symbol:    "BTC/USDT",
				Last:      decimal.NewFromInt(100),
				Volume:    decimal.NewFromInt(10),
				Timestamp: UnixTimestamp(ts),
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	mux.HandleFunc("/api/ticker/bybit/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		response := TickerResponse{
			Exchange: "bybit",
			Symbol:   "BTC/USDT",
			Ticker: Ticker{
				Symbol:    "BTC/USDT",
				Last:      decimal.NewFromInt(110),
				Volume:    decimal.NewFromInt(8),
				Timestamp: UnixTimestamp(ts),
			},
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}
	service.supportedExchanges["bybit"] = ExchangeInfo{ID: "bybit"}

	opportunities, err := service.CalculateArbitrageOpportunities(
		context.Background(),
		[]string{"binance", "bybit"},
		[]string{"BTC/USDT"},
		decimal.NewFromFloat(5),
	)

	assert.NoError(t, err)
	if assert.Len(t, opportunities, 1) {
		assert.Equal(t, "BTC/USDT", opportunities[0].GetSymbol())
		assert.Equal(t, "binance", opportunities[0].GetBuyExchange())
		assert.Equal(t, "bybit", opportunities[0].GetSellExchange())
	}
}

func TestService_FetchFundingRate_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/funding-rate/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		response := FundingRate{
			Symbol:           "BTC/USDT",
			FundingRate:      0.03,
			FundingTimestamp: UnixTimestamp(ts),
			NextFundingTime:  UnixTimestamp(ts.Add(8 * time.Hour)),
			MarkPrice:        50000,
			IndexPrice:       49900,
			Timestamp:        UnixTimestamp(ts),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	rate, err := service.FetchFundingRate(context.Background(), "binance", "BTC/USDT")
	assert.NoError(t, err)
	assert.Equal(t, "BTC/USDT", rate.Symbol)
	assert.Equal(t, 0.03, rate.FundingRate)
}

func TestService_FetchFundingRates_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/funding-rates/binance", func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "symbols=BTCUSDT%2CETHUSDT")

		response := FundingRateResponse{
			Exchange: "binance",
			FundingRates: []FundingRate{
				{
					Symbol:      "BTC/USDT",
					FundingRate: 0.02,
					Timestamp:   UnixTimestamp(ts),
				},
				{
					Symbol:      "ETH/USDT",
					FundingRate: 0.01,
					Timestamp:   UnixTimestamp(ts),
				},
			},
			Count:     2,
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	rates, err := service.FetchFundingRates(context.Background(), "binance", []string{"BTC/USDT", "ETH/USDT"})
	assert.NoError(t, err)
	assert.Len(t, rates, 2)
}

func TestService_FetchFundingRates_Blacklisted(t *testing.T) {
	cfg := &interfaces.CCXTConfig{ServiceURL: "http://example.com", Timeout: 5}
	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()

	service := NewService(cfg, logger, blacklistCache)
	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}
	service.blacklistCache.Add("BTC/USDT", "suspended", time.Minute)

	rates, err := service.FetchFundingRates(context.Background(), "binance", []string{"BTC/USDT"})
	assert.Error(t, err)
	assert.Nil(t, rates)
	assert.Contains(t, err.Error(), "blacklisted")
}

func TestService_FetchAllFundingRates_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/funding-rates/binance", func(w http.ResponseWriter, r *http.Request) {
		response := FundingRateResponse{
			Exchange: "binance",
			FundingRates: []FundingRate{
				{Symbol: "BTC/USDT", FundingRate: 0.02, Timestamp: UnixTimestamp(ts)},
			},
			Count:     1,
			Timestamp: ts.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}

	rates, err := service.FetchAllFundingRates(context.Background(), "binance")
	assert.NoError(t, err)
	assert.Len(t, rates, 1)
}

func TestService_CalculateFundingRateArbitrage_Success(t *testing.T) {
	ts := time.Now()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/funding-rate/binance/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		rate := FundingRate{
			Symbol:      "BTC/USDT",
			FundingRate: 0.02,
			Timestamp:   UnixTimestamp(ts),
			MarkPrice:   50000,
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(rate))
	})

	mux.HandleFunc("/api/funding-rate/bybit/BTCUSDT", func(w http.ResponseWriter, r *http.Request) {
		rate := FundingRate{
			Symbol:      "BTC/USDT",
			FundingRate: 0.05,
			Timestamp:   UnixTimestamp(ts),
			MarkPrice:   50100,
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(rate))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance"}
	service.supportedExchanges["bybit"] = ExchangeInfo{ID: "bybit"}

	opportunities, err := service.CalculateFundingRateArbitrage(
		context.Background(),
		[]string{"BTC/USDT"},
		[]string{"binance", "bybit"},
		0.05,
	)

	assert.NoError(t, err)
	if assert.Len(t, opportunities, 1) {
		assert.Equal(t, "BTC/USDT", opportunities[0].Symbol)
		assert.Equal(t, "binance", opportunities[0].LongExchange)
		assert.Equal(t, "bybit", opportunities[0].ShortExchange)
	}
}

func TestService_ExchangeManagementEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/admin/exchanges/config", func(w http.ResponseWriter, r *http.Request) {
		resp := ExchangeConfigResponse{
			Config: ExchangeConfig{
				BlacklistedExchanges: []string{"bitmex"},
				PriorityExchanges:    []string{"binance"},
			},
			ActiveExchanges:    []string{"binance"},
			AvailableExchanges: []string{"bybit"},
			Timestamp:          time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(resp))
	})

	mux.HandleFunc("/api/admin/exchanges/blacklist/binance", func(w http.ResponseWriter, r *http.Request) {
		response := ExchangeManagementResponse{Message: "updated"}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	mux.HandleFunc("/api/admin/exchanges/refresh", func(w http.ResponseWriter, r *http.Request) {
		response := ExchangeManagementResponse{Message: "refreshed"}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	})

	service, cleanup := newTestServiceWithServer(t, mux)
	defer cleanup()

	cfgResp, err := service.GetExchangeConfig(context.Background())
	assert.NoError(t, err)
	assert.Contains(t, cfgResp.ActiveExchanges, "binance")

	addResp, err := service.AddExchangeToBlacklist(context.Background(), "binance")
	assert.NoError(t, err)
	assert.Equal(t, "updated", addResp.Message)

	removeResp, err := service.RemoveExchangeFromBlacklist(context.Background(), "binance")
	assert.NoError(t, err)
	assert.Equal(t, "updated", removeResp.Message)

	refreshResp, err := service.RefreshExchanges(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "refreshed", refreshResp.Message)
}
