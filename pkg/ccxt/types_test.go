package ccxt

import (
	"context"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Test Service struct initialization
func TestService_Struct(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Test initial state
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.supportedExchanges)
	assert.Empty(t, service.supportedExchanges)
	assert.True(t, service.lastUpdate.IsZero())
}

// Test GetSupportedExchanges with empty exchanges
func TestService_GetSupportedExchanges_Empty(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	exchanges := service.GetSupportedExchanges()

	assert.Empty(t, exchanges)
	assert.NotNil(t, exchanges) // Should return empty slice, not nil
}

// Test GetSupportedExchanges with populated exchanges
func TestService_GetSupportedExchanges_Populated(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Manually populate supported exchanges for testing
	service.mu.Lock()
	service.supportedExchanges["binance"] = ExchangeInfo{ID: "binance", Name: "Binance"}
	service.supportedExchanges["coinbase"] = ExchangeInfo{ID: "coinbase", Name: "Coinbase"}
	service.mu.Unlock()

	exchanges := service.GetSupportedExchanges()

	assert.Len(t, exchanges, 2)
	assert.Contains(t, exchanges, "binance")
	assert.Contains(t, exchanges, "coinbase")
}

// Test GetExchangeInfo with existing exchange
func TestService_GetExchangeInfo_Exists(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Manually populate supported exchanges for testing
	binanceInfo := ExchangeInfo{
		ID:        "binance",
		Name:      "Binance",
		Countries: []string{"MT"},
	}

	service.mu.Lock()
	service.supportedExchanges["binance"] = binanceInfo
	service.mu.Unlock()

	info, exists := service.GetExchangeInfo("binance")

	assert.True(t, exists)
	assert.Equal(t, binanceInfo.ID, info.ID)
	assert.Equal(t, binanceInfo.Name, info.Name)
	assert.Equal(t, binanceInfo.Countries, info.Countries)
}

// Test GetExchangeInfo with non-existing exchange
func TestService_GetExchangeInfo_NotExists(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	info, exists := service.GetExchangeInfo("nonexistent")

	assert.False(t, exists)
	assert.Empty(t, info.ID)
	assert.Empty(t, info.Name)
}

// Test FetchMarketData with empty parameters
func TestService_FetchMarketData_EmptyExchanges(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	ctx := context.Background()

	// Test with empty exchanges
	data, err := service.FetchMarketData(ctx, []string{}, []string{"BTC/USDT"})

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "exchanges and symbols cannot be empty")
}

func TestService_FetchMarketData_EmptySymbols(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	ctx := context.Background()

	// Test with empty symbols
	data, err := service.FetchMarketData(ctx, []string{"binance"}, []string{})

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "exchanges and symbols cannot be empty")
}

// Test CalculateArbitrageOpportunities with empty parameters
func TestService_CalculateArbitrageOpportunities_EmptyExchanges(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	ctx := context.Background()
	minProfit := decimal.NewFromFloat(1.0)

	// Test with empty exchanges
	opportunities, err := service.CalculateArbitrageOpportunities(ctx, []string{}, []string{"BTC/USDT"}, minProfit)

	assert.Error(t, err)
	assert.Nil(t, opportunities)
	assert.Contains(t, err.Error(), "exchanges and symbols cannot be empty")
}

func TestService_CalculateArbitrageOpportunities_EmptySymbols(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	ctx := context.Background()
	minProfit := decimal.NewFromFloat(1.0)

	// Test with empty symbols
	opportunities, err := service.CalculateArbitrageOpportunities(ctx, []string{"binance"}, []string{}, minProfit)

	assert.Error(t, err)
	assert.Nil(t, opportunities)
	assert.Contains(t, err.Error(), "exchanges and symbols cannot be empty")
}

// Test Service mutex operations
func TestService_ConcurrentAccess(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Test concurrent read/write operations
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		service.mu.Lock()
		service.supportedExchanges["test"] = ExchangeInfo{ID: "test", Name: "Test"}
		service.lastUpdate = time.Now()
		service.mu.Unlock()
		done <- true
	}()

	// Reader goroutine
	go func() {
		exchanges := service.GetSupportedExchanges()
		_ = exchanges // Use the result to avoid unused variable
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify the exchange was added
	info, exists := service.GetExchangeInfo("test")
	assert.True(t, exists)
	assert.Equal(t, "test", info.ID)
}

// Test Service lastUpdate field
func TestService_LastUpdate(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Initially should be zero
	assert.True(t, service.lastUpdate.IsZero())

	// Manually update lastUpdate
	now := time.Now()
	service.mu.Lock()
	service.lastUpdate = now
	service.mu.Unlock()

	// Verify update
	service.mu.RLock()
	updateTime := service.lastUpdate
	service.mu.RUnlock()

	assert.Equal(t, now.Unix(), updateTime.Unix())
}

// Test Service with different config values
func TestService_DifferentConfigs(t *testing.T) {
	testCases := []struct {
		name     string
		config   *config.CCXTConfig
		expected bool
	}{
		{
			name: "valid config",
			config: &config.CCXTConfig{
				ServiceURL: "http://localhost:3001",
				Timeout:    30,
			},
			expected: true,
		},
		{
			name: "different URL",
			config: &config.CCXTConfig{
				ServiceURL: "http://example.com:8080",
				Timeout:    60,
			},
			expected: true,
		},
		{
			name: "zero timeout",
			config: &config.CCXTConfig{
				ServiceURL: "http://localhost:3001",
				Timeout:    0,
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := logrus.New()
			service := NewService(tc.config, logger)
			assert.NotNil(t, service)
			assert.NotNil(t, service.client)
			assert.Equal(t, tc.expected, service != nil)
		})
	}
}

// Test Service supportedExchanges map operations
func TestService_SupportedExchangesOperations(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Test adding multiple exchanges
	exchanges := map[string]ExchangeInfo{
		"binance":  {ID: "binance", Name: "Binance", Countries: []string{"MT"}},
		"coinbase": {ID: "coinbase", Name: "Coinbase", Countries: []string{"US"}},
		"kraken":   {ID: "kraken", Name: "Kraken", Countries: []string{"US"}},
	}

	service.mu.Lock()
	for id, info := range exchanges {
		service.supportedExchanges[id] = info
	}
	service.mu.Unlock()

	// Test GetSupportedExchanges returns all exchanges
	supportedIDs := service.GetSupportedExchanges()
	assert.Len(t, supportedIDs, 3)
	assert.Contains(t, supportedIDs, "binance")
	assert.Contains(t, supportedIDs, "coinbase")
	assert.Contains(t, supportedIDs, "kraken")

	// Test GetExchangeInfo for each exchange
	for id, expectedInfo := range exchanges {
		info, exists := service.GetExchangeInfo(id)
		assert.True(t, exists, "Exchange %s should exist", id)
		assert.Equal(t, expectedInfo.ID, info.ID)
		assert.Equal(t, expectedInfo.Name, info.Name)
		assert.Equal(t, expectedInfo.Countries, info.Countries)
	}
}

// Test Service edge cases
func TestService_EdgeCases(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Test GetExchangeInfo with empty string
	info, exists := service.GetExchangeInfo("")
	assert.False(t, exists)
	assert.Empty(t, info.ID)

	// Test GetSupportedExchanges multiple times
	exchanges1 := service.GetSupportedExchanges()
	exchanges2 := service.GetSupportedExchanges()
	assert.Equal(t, exchanges1, exchanges2)

	// Test that GetSupportedExchanges is consistent
	assert.Equal(t, len(exchanges1), len(exchanges2))
}

// Test Service initialization state
func TestService_InitializationState(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)

	// Test initial state
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.supportedExchanges)
	assert.Empty(t, service.supportedExchanges)
	assert.True(t, service.lastUpdate.IsZero())

	// Test that supportedExchanges is properly initialized as a map
	assert.IsType(t, make(map[string]ExchangeInfo), service.supportedExchanges)

	// Test that we can write to the map without panic
	assert.NotPanics(t, func() {
		service.mu.Lock()
		service.supportedExchanges["test"] = ExchangeInfo{ID: "test"}
		service.mu.Unlock()
	})
}

// Test Service method parameter validation
func TestService_ParameterValidation(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	ctx := context.Background()

	// Test FetchMarketData with context.TODO() (should not panic)
	assert.NotPanics(t, func() {
		_, _ = service.FetchMarketData(context.TODO(), []string{"binance"}, []string{"BTC/USDT"})
		// May or may not error depending on service availability, but should not panic
	})

	// Test CalculateArbitrageOpportunities with context.TODO() (should not panic)
	assert.NotPanics(t, func() {
		_, _ = service.CalculateArbitrageOpportunities(context.TODO(), []string{"binance"}, []string{"BTC/USDT"}, decimal.NewFromFloat(1.0))
		// May or may not error depending on service availability, but should not panic
	})

	// Test with valid context but invalid parameters
	_, err := service.FetchMarketData(ctx, nil, []string{"BTC/USDT"})
	assert.Error(t, err)

	_, err = service.FetchMarketData(ctx, []string{"binance"}, nil)
	assert.Error(t, err)
}

// Test Service concurrent safety
func TestService_ConcurrentSafety(t *testing.T) {
	cfg := &config.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	service := NewService(cfg, logger)
	numGoroutines := 10
	done := make(chan bool, numGoroutines*2)

	// Start multiple readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				_ = service.GetSupportedExchanges()
				_, _ = service.GetExchangeInfo("test")
			}
			done <- true
		}(i)
	}

	// Start multiple writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				service.mu.Lock()
				service.supportedExchanges["test"] = ExchangeInfo{ID: "test"}
				service.lastUpdate = time.Now()
				service.mu.Unlock()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	// Verify final state
	info, exists := service.GetExchangeInfo("test")
	assert.True(t, exists)
	assert.Equal(t, "test", info.ID)
}
