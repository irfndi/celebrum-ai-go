package services

import (
	"context"
	"testing"

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

func TestNewCollectorService(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.workers)
	assert.Equal(t, 60, collector.collectorConfig.IntervalSeconds)
	assert.Equal(t, 5, collector.collectorConfig.MaxErrors)
}

func TestCollectorService_Start(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	// Mock the Initialize and GetSupportedExchanges calls
	mockCCXT.On("Initialize", mock.Anything).Return(nil)
	mockCCXT.On("GetSupportedExchanges").Return([]string{}) // Return empty slice to avoid database operations

	collector := NewCollectorService(nil, mockCCXT, config)

	// Test that the service can start without errors
	err := collector.Start()
	assert.NoError(t, err)

	mockCCXT.AssertExpectations(t)
}

func TestCollectorService_Stop(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	config := &config.Config{}

	collector := NewCollectorService(nil, mockCCXT, config)

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

	collector := NewCollectorService(nil, mockCCXT, config)

	// Get worker status (should be empty initially)
	status := collector.GetWorkerStatus()
	assert.NotNil(t, status)
	assert.Len(t, status, 0) // No workers created yet
}
