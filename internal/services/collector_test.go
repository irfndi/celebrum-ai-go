package services

import (
	"context"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCCXTService is a mock implementation of CCXTService for testing
type MockCCXTService struct {
	mock.Mock
}

func (m *MockCCXTService) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockCCXTService) IsHealthy(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func (m *MockCCXTService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCCXTService) GetSupportedExchanges() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockCCXTService) GetExchangeInfo(exchangeID string) (ccxt.ExchangeInfo, bool) {
	args := m.Called(exchangeID)
	return args.Get(0).(ccxt.ExchangeInfo), args.Bool(1)
}

func (m *MockCCXTService) FetchMarketData(ctx context.Context, exchanges []string, symbols []string) ([]models.MarketPrice, error) {
	args := m.Called(ctx, exchanges, symbols)
	return args.Get(0).([]models.MarketPrice), args.Error(1)
}

func (m *MockCCXTService) FetchSingleTicker(ctx context.Context, exchange, symbol string) (*models.MarketPrice, error) {
	args := m.Called(ctx, exchange, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MarketPrice), args.Error(1)
}

func (m *MockCCXTService) FetchOrderBook(ctx context.Context, exchange, symbol string, limit int) (*ccxt.OrderBookResponse, error) {
	args := m.Called(ctx, exchange, symbol, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.OrderBookResponse), args.Error(1)
}

func (m *MockCCXTService) FetchOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*ccxt.OHLCVResponse, error) {
	args := m.Called(ctx, exchange, symbol, timeframe, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.OHLCVResponse), args.Error(1)
}

func (m *MockCCXTService) FetchTrades(ctx context.Context, exchange, symbol string, limit int) (*ccxt.TradesResponse, error) {
	args := m.Called(ctx, exchange, symbol, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.TradesResponse), args.Error(1)
}

func (m *MockCCXTService) FetchMarkets(ctx context.Context, exchange string) (*ccxt.MarketsResponse, error) {
	args := m.Called(ctx, exchange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.MarketsResponse), args.Error(1)
}

func (m *MockCCXTService) CalculateArbitrageOpportunities(ctx context.Context, exchanges []string, symbols []string, minProfitPercent decimal.Decimal) ([]models.ArbitrageOpportunityResponse, error) {
	args := m.Called(ctx, exchanges, symbols, minProfitPercent)
	return args.Get(0).([]models.ArbitrageOpportunityResponse), args.Error(1)
}

func (m *MockCCXTService) FetchFundingRate(ctx context.Context, exchange, symbol string) (*ccxt.FundingRate, error) {
	args := m.Called(ctx, exchange, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.FundingRate), args.Error(1)
}

func (m *MockCCXTService) FetchFundingRates(ctx context.Context, exchange string, symbols []string) ([]ccxt.FundingRate, error) {
	args := m.Called(ctx, exchange, symbols)
	return args.Get(0).([]ccxt.FundingRate), args.Error(1)
}

func (m *MockCCXTService) FetchAllFundingRates(ctx context.Context, exchange string) ([]ccxt.FundingRate, error) {
	args := m.Called(ctx, exchange)
	return args.Get(0).([]ccxt.FundingRate), args.Error(1)
}

func (m *MockCCXTService) CalculateFundingRateArbitrage(ctx context.Context, symbols []string, exchanges []string, minProfit float64) ([]ccxt.FundingArbitrageOpportunity, error) {
	args := m.Called(ctx, symbols, exchanges, minProfit)
	return args.Get(0).([]ccxt.FundingArbitrageOpportunity), args.Error(1)
}

func (m *MockCCXTService) GetServiceURL() string {
	args := m.Called()
	return args.String(0)
}

// Exchange management methods
func (m *MockCCXTService) GetExchangeConfig(ctx context.Context) (*ccxt.ExchangeConfigResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.ExchangeConfigResponse), args.Error(1)
}

func (m *MockCCXTService) AddExchangeToBlacklist(ctx context.Context, exchange string) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx, exchange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) RemoveExchangeFromBlacklist(ctx context.Context, exchange string) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx, exchange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) RefreshExchanges(ctx context.Context) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) AddExchange(ctx context.Context, exchange string) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx, exchange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func TestNewCollectorService(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config, nil)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.workers)
	assert.Equal(t, 300, collector.collectorConfig.IntervalSeconds)
	assert.Equal(t, 5, collector.collectorConfig.MaxErrors)
}

func TestCollectorService_Start(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	// Mock the Initialize and GetSupportedExchanges calls
	mockCCXT.On("Initialize", mock.Anything).Return(nil)
	mockCCXT.On("GetSupportedExchanges").Return([]string{}) // Return empty slice to avoid database operations

	collector := NewCollectorService(nil, mockCCXT, config, nil)

	// Test that the service can start without errors
	err := collector.Start()
	assert.NoError(t, err)

	// Clean up: stop the collector to prevent it from running indefinitely
	collector.Stop()

	mockCCXT.AssertExpectations(t)
}

func TestCollectorService_Stop(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config, nil)

	// Test that the service can be stopped without errors
	collector.Stop()

	// Verify that the context is cancelled
	select {
	case <-collector.ctx.Done():
		// Context was cancelled as expected
	default:
		t.Error("Expected context to be cancelled")
	}
}

func TestCollectorService_GetWorkerStatus(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config, nil)

	// Get worker status (should be empty initially)
	status := collector.GetWorkerStatus()
	assert.NotNil(t, status)
	assert.Len(t, status, 0) // No workers created yet
}

func TestCollectorService_IsHealthy(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config, nil)

	// Test with no workers (should be unhealthy)
	assert.False(t, collector.IsHealthy())

	// Add a running worker
	collector.workers["binance"] = &Worker{
		Exchange:  "binance",
		IsRunning: true,
	}

	// Test with one running worker (should be healthy)
	assert.True(t, collector.IsHealthy())

	// Add a stopped worker
	collector.workers["coinbase"] = &Worker{
		Exchange:  "coinbase",
		IsRunning: false,
	}

	// Test with 50% workers running (should be healthy)
	assert.True(t, collector.IsHealthy())

	// Stop the first worker
	collector.workers["binance"].IsRunning = false

	// Test with 0% workers running (should be unhealthy)
	assert.False(t, collector.IsHealthy())
}

func TestCollectorService_RestartWorker(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config, nil)

	// Test restarting non-existent worker
	err := collector.RestartWorker("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worker for exchange nonexistent not found")

	// Add a worker with errors
	collector.workers["binance"] = &Worker{
		Exchange:   "binance",
		IsRunning:  false,
		ErrorCount: 5,
		Interval:   60 * time.Second,
		MaxErrors:  5,
	}

	// Test restarting existing worker
	err = collector.RestartWorker("binance")
	assert.NoError(t, err)
	assert.Equal(t, 0, collector.workers["binance"].ErrorCount)
	assert.True(t, collector.workers["binance"].IsRunning)
}

func TestWorker_Struct(t *testing.T) {
	worker := &Worker{
		Exchange:   "binance",
		Symbols:    []string{"BTC/USDT", "ETH/USDT"},
		Interval:   60 * time.Second,
		LastUpdate: time.Now(),
		IsRunning:  true,
		ErrorCount: 0,
		MaxErrors:  5,
	}

	assert.Equal(t, "binance", worker.Exchange)
	assert.Len(t, worker.Symbols, 2)
	assert.Equal(t, 60*time.Second, worker.Interval)
	assert.True(t, worker.IsRunning)
	assert.Equal(t, 0, worker.ErrorCount)
	assert.Equal(t, 5, worker.MaxErrors)
}

func TestCollectorConfig_Struct(t *testing.T) {
	config := CollectorConfig{
		IntervalSeconds: 30,
		MaxErrors:       10,
	}

	assert.Equal(t, 30, config.IntervalSeconds)
	assert.Equal(t, 10, config.MaxErrors)
}

// Test validateMarketData function
func TestCollectorService_ValidateMarketData(t *testing.T) {
	collector := &CollectorService{}

	tests := []struct {
		name     string
		ticker   *models.MarketPrice
		exchange string
		symbol   string
		wantErr  bool
	}{
		{
			name: "valid data",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(50000.0),
				Volume:    decimal.NewFromFloat(1000.0),
				Timestamp: time.Now(),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  false,
		},
		{
			name: "zero price",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(0),
				Volume:    decimal.NewFromFloat(1000.0),
				Timestamp: time.Now(),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  true,
		},
		{
			name: "negative price",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(-100.0),
				Volume:    decimal.NewFromFloat(1000.0),
				Timestamp: time.Now(),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  true,
		},
		{
			name: "extremely high price",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(20000000.0),
				Volume:    decimal.NewFromFloat(1000.0),
				Timestamp: time.Now(),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  true,
		},
		{
			name: "negative volume",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(50000.0),
				Volume:    decimal.NewFromFloat(-100.0),
				Timestamp: time.Now(),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  true,
		},
		{
			name: "future timestamp",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(50000.0),
				Volume:    decimal.NewFromFloat(1000.0),
				Timestamp: time.Now().Add(2 * time.Hour),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  true,
		},
		{
			name: "old timestamp",
			ticker: &models.MarketPrice{
				Price:     decimal.NewFromFloat(50000.0),
				Volume:    decimal.NewFromFloat(1000.0),
				Timestamp: time.Now().Add(-25 * time.Hour),
			},
			exchange: "binance",
			symbol:   "BTC/USDT",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.validateMarketData(tt.ticker, tt.exchange, tt.symbol)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMarketData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test parseSymbol function
func TestCollectorService_ParseSymbol(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}
	collector := NewCollectorService(nil, mockCCXT, config, nil)

	tests := []struct {
		name          string
		symbol        string
		expectedBase  string
		expectedQuote string
	}{
		{
			name:          "standard slash format",
			symbol:        "BTC/USDT",
			expectedBase:  "BTC",
			expectedQuote: "USDT",
		},
		{
			name:          "futures format with settlement",
			symbol:        "BTC/USDT:USDT",
			expectedBase:  "BTC",
			expectedQuote: "USDT",
		},
		{
			name:          "concatenated format USDT",
			symbol:        "BTCUSDT",
			expectedBase:  "BTC",
			expectedQuote: "USDT",
		},
		{
			name:          "concatenated format USD",
			symbol:        "BTCUSD",
			expectedBase:  "BTC",
			expectedQuote: "USD",
		},
		{
			name:          "invalid symbol",
			symbol:        "INVALID",
			expectedBase:  "",
			expectedQuote: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, quote := collector.parseSymbol(tt.symbol)
			assert.Equal(t, tt.expectedBase, base)
			assert.Equal(t, tt.expectedQuote, quote)
		})
	}
}

// Test isOptionsContract function
func TestCollectorService_IsOptionsContract(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}
	collector := NewCollectorService(nil, mockCCXT, config, nil)

	tests := []struct {
		name     string
		symbol   string
		expected bool
	}{
		{
			name:     "call option",
			symbol:   "SOLUSDT:USDT-250815-180-C",
			expected: true,
		},
		{
			name:     "put option",
			symbol:   "BTC-25DEC20-20000-P",
			expected: true,
		},
		{
			name:     "regular spot pair",
			symbol:   "BTC/USDT",
			expected: false,
		},
		{
			name:     "futures pair",
			symbol:   "BTC/USDT:USDT",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.isOptionsContract(tt.symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test isInvalidSymbolFormat function
func TestCollectorService_IsInvalidSymbolFormat(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}
	collector := NewCollectorService(nil, mockCCXT, config, nil)

	tests := []struct {
		name     string
		symbol   string
		expected bool
	}{
		{
			name:     "valid short symbol",
			symbol:   "BTC/USDT",
			expected: false,
		},
		{
			name:     "too long symbol",
			symbol:   "VERYLONGSYMBOLNAMETHATEXCEEDSLIMIT",
			expected: true,
		},
		{
			name:     "multiple colons",
			symbol:   "BTC:USDT:USD:EUR",
			expected: true,
		},
		{
			name:     "complex derivative",
			symbol:   "BTC_USDT-PERP_SWAP",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.isInvalidSymbolFormat(tt.symbol)
			assert.Equal(t, tt.expected, result)
		})
	}
}
