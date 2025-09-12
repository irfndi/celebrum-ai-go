package ccxt

import (
	"context"
	"testing"
	"time"

		"github.com/irfandi/celebrum-ai-go/pkg/interfaces"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	cfg := &interfaces.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()
	service := NewService(cfg, logger, blacklistCache)
	require.NotNil(t, service)
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.supportedExchanges)
}

func TestService_Initialize(t *testing.T) {
	// This test requires a running CCXT service
	t.Skip("Skipping integration test - requires running CCXT service")

	cfg := &interfaces.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()
	service := NewService(cfg, logger, blacklistCache)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := service.Initialize(ctx)
	assert.NoError(t, err)

	exchanges := service.GetSupportedExchanges()
	assert.NotEmpty(t, exchanges)
}

func TestService_GetSupportedExchanges(t *testing.T) {
	cfg := &interfaces.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()
	service := NewService(cfg, logger, blacklistCache)

	// Before initialization, should return empty slice
	exchanges := service.GetSupportedExchanges()
	assert.Empty(t, exchanges)
}

func TestService_FetchSingleTicker(t *testing.T) {
	// This test requires a running CCXT service
	t.Skip("Skipping integration test - requires running CCXT service")

	cfg := &interfaces.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()
	service := NewService(cfg, logger, blacklistCache)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize service first
	err := service.Initialize(ctx)
	require.NoError(t, err)

	// Test fetching ticker
	marketPrice, err := service.FetchSingleTicker(ctx, "binance", "BTC/USDT")
	assert.NoError(t, err)
	assert.NotNil(t, marketPrice)
	assert.Equal(t, "binance", marketPrice.GetExchangeName())
	assert.Equal(t, "BTC/USDT", marketPrice.GetSymbol())
	assert.True(t, marketPrice.GetPrice() > 0)
}

func TestService_FetchMarketData(t *testing.T) {
	// This test requires a running CCXT service
	t.Skip("Skipping integration test - requires running CCXT service")

	cfg := &interfaces.CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	logger := logrus.New()
	blacklistCache := interfaces.NewInMemoryBlacklistCache()
	service := NewService(cfg, logger, blacklistCache)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize service first
	err := service.Initialize(ctx)
	require.NoError(t, err)

	// Test fetching market data
	exchanges := []string{"binance", "bybit"}
	symbols := []string{"BTC/USDT", "ETH/USDT"}

	marketData, err := service.FetchMarketData(ctx, exchanges, symbols)
	assert.NoError(t, err)
	assert.NotEmpty(t, marketData)

	// Should have data for each exchange-symbol combination
	for _, data := range marketData {
		assert.Contains(t, exchanges, data.GetExchangeName())
		assert.Contains(t, symbols, data.GetSymbol())
		assert.True(t, data.GetPrice() > 0)
	}
}
