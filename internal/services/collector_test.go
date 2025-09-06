package services

import (
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/cache"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/test/testmocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)


func TestNewCollectorService(t *testing.T) {
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}

	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.workers)
	assert.Equal(t, 300, collector.collectorConfig.IntervalSeconds)
	assert.Equal(t, 5, collector.collectorConfig.MaxErrors)
}

func TestCollectorService_Start(t *testing.T) {
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}

	// Mock the Initialize and GetSupportedExchanges calls
	mockCCXT.On("Initialize", mock.Anything).Return(nil)
	mockCCXT.On("GetSupportedExchanges").Return([]string{}) // Called once in initializeWorkersAsync

	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

	// Test that the service can start without errors
	err := collector.Start()
	assert.NoError(t, err)

	// Wait a bit for async initialization to start
	time.Sleep(100 * time.Millisecond)

	// Clean up: stop the collector to prevent it from running indefinitely
	collector.Stop()

	mockCCXT.AssertExpectations(t)
}

func TestCollectorService_Stop(t *testing.T) {
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}

	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

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
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}

	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

	// Get worker status (should be empty initially)
	status := collector.GetWorkerStatus()
	assert.NotNil(t, status)
	assert.Len(t, status, 0) // No workers created yet
}

func TestCollectorService_IsHealthy(t *testing.T) {
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}

	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

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
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}

	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

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
	now := time.Now()
	worker := &Worker{
		Exchange:   "binance",
		Symbols:    []string{"BTC/USDT", "ETH/USDT"},
		Interval:   60 * time.Second,
		LastUpdate: now,
		IsRunning:  true,
		ErrorCount: 0,
		MaxErrors:  5,
	}

	assert.Equal(t, "binance", worker.Exchange)
	assert.Len(t, worker.Symbols, 2)
	assert.Equal(t, 60*time.Second, worker.Interval)
	assert.Equal(t, now, worker.LastUpdate)
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
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}
	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

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
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}
	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

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
	mockCCXT := &testmocks.MockCCXTService{}
	config := &config.Config{}
	blacklistCache := cache.NewInMemoryBlacklistCache()
	collector := NewCollectorService(nil, mockCCXT, config, nil, blacklistCache)

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
			symbol:   "VERYLONGSYMBOLNAMETHATEXCEEDSLIMITANDISMORETHANFIFTYCHARACTERS",
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
