package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCCXTService for testing
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

func TestMarketHandler_GetTicker(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create mocks
	mockCCXT := new(MockCCXTService)

	// Create handler with nil database (we'll mock the CCXT service)
	handler := NewMarketHandler(nil, mockCCXT, nil)

	// Setup route
	router.GET("/ticker/:exchange/:symbol", handler.GetTicker)

	// Mock expectations
	expectedTicker := &models.MarketPrice{
		ExchangeName: "binance",
		Symbol:       "BTCUSDT",
		Price:        decimal.NewFromFloat(50000.0),
		Volume:       decimal.NewFromFloat(1000.0),
	}
	// Mock IsHealthy to return true so the service tries to fetch from CCXT
	mockCCXT.On("IsHealthy", mock.Anything).Return(true)
	mockCCXT.On("FetchSingleTicker", mock.Anything, "binance", "BTCUSDT").Return(expectedTicker, nil)

	// Create request
	req, _ := http.NewRequest("GET", "/ticker/binance/BTCUSDT", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "binance", response["exchange"])
	assert.Equal(t, "BTCUSDT", response["symbol"])
	assert.Equal(t, "50000", response["price"])

	// Verify mock expectations
	mockCCXT.AssertExpectations(t)
}

func TestMarketHandler_GetWorkerStatus(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create mocks
	mockCCXT := new(MockCCXTService)

	// For testing worker status, we'll pass nil for the collector service
	// In a real scenario, this would be properly initialized
	handler := NewMarketHandler(nil, mockCCXT, nil)

	// Setup route
	router.GET("/workers/status", handler.GetWorkerStatus)

	// Create request
	req, _ := http.NewRequest("GET", "/workers/status", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Assertions - Since collector service is nil, we expect an error response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}
