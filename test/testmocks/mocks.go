package testmocks

import (
	"context"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/internal/ccxt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
)

// MockCCXTService implements ccxt.CCXTService for testing
type MockCCXTService struct {
	mock.Mock
}

// MockCCXTClient implements ccxt.CCXTClient for testing
type MockCCXTClient struct {
	mock.Mock
}

// MockCollectorService implements services.CollectorService for testing
type MockCollectorService struct {
	mock.Mock
}

// MockCacheAnalyticsService implements services.CacheAnalyticsService for testing
type MockCacheAnalyticsService struct {
	mock.Mock
}

// MockRedisClient implements RedisInterface for testing
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

// Mock implementations for CCXTService interface methods
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

func (m *MockCCXTService) GetServiceURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCCXTService) GetSupportedExchanges() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockCCXTService) GetExchangeInfo(exchangeID string) (ccxt.ExchangeInfo, bool) {
	args := m.Called(exchangeID)
	return args.Get(0).(ccxt.ExchangeInfo), args.Bool(1)
}

func (m *MockCCXTService) GetExchangeConfig(ctx context.Context) (*ccxt.ExchangeConfigResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ccxt.ExchangeConfigResponse), args.Error(1)
}

func (m *MockCCXTService) AddExchangeToBlacklist(ctx context.Context, exchange string) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx, exchange)
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) RemoveExchangeFromBlacklist(ctx context.Context, exchange string) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx, exchange)
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) RefreshExchanges(ctx context.Context) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) AddExchange(ctx context.Context, exchange string) (*ccxt.ExchangeManagementResponse, error) {
	args := m.Called(ctx, exchange)
	return args.Get(0).(*ccxt.ExchangeManagementResponse), args.Error(1)
}

func (m *MockCCXTService) FetchMarketData(ctx context.Context, exchanges []string, symbols []string) ([]models.MarketPrice, error) {
	args := m.Called(ctx, exchanges, symbols)
	return args.Get(0).([]models.MarketPrice), args.Error(1)
}

func (m *MockCCXTService) FetchSingleTicker(ctx context.Context, exchange, symbol string) (*models.MarketPrice, error) {
	args := m.Called(ctx, exchange, symbol)
	return args.Get(0).(*models.MarketPrice), args.Error(1)
}

func (m *MockCCXTService) FetchOrderBook(ctx context.Context, exchange, symbol string, limit int) (*ccxt.OrderBookResponse, error) {
	args := m.Called(ctx, exchange, symbol, limit)
	return args.Get(0).(*ccxt.OrderBookResponse), args.Error(1)
}

func (m *MockCCXTService) FetchOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*ccxt.OHLCVResponse, error) {
	args := m.Called(ctx, exchange, symbol, timeframe, limit)
	return args.Get(0).(*ccxt.OHLCVResponse), args.Error(1)
}

func (m *MockCCXTService) FetchTrades(ctx context.Context, exchange, symbol string, limit int) (*ccxt.TradesResponse, error) {
	args := m.Called(ctx, exchange, symbol, limit)
	return args.Get(0).(*ccxt.TradesResponse), args.Error(1)
}

func (m *MockCCXTService) FetchMarkets(ctx context.Context, exchange string) (*ccxt.MarketsResponse, error) {
	args := m.Called(ctx, exchange)
	return args.Get(0).(*ccxt.MarketsResponse), args.Error(1)
}

func (m *MockCCXTService) FetchFundingRate(ctx context.Context, exchange, symbol string) (*ccxt.FundingRate, error) {
	args := m.Called(ctx, exchange, symbol)
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

func (m *MockCCXTService) CalculateArbitrageOpportunities(ctx context.Context, exchanges []string, symbols []string, minProfitPercent decimal.Decimal) ([]models.ArbitrageOpportunityResponse, error) {
	args := m.Called(ctx, exchanges, symbols, minProfitPercent)
	return args.Get(0).([]models.ArbitrageOpportunityResponse), args.Error(1)
}

func (m *MockCCXTService) CalculateFundingRateArbitrage(ctx context.Context, symbols []string, exchanges []string, minProfit float64) ([]ccxt.FundingArbitrageOpportunity, error) {
	args := m.Called(ctx, symbols, exchanges, minProfit)
	return args.Get(0).([]ccxt.FundingArbitrageOpportunity), args.Error(1)
}

// Mock implementations for CollectorService
func (m *MockCollectorService) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCollectorService) Stop() {
	m.Called()
}

func (m *MockCollectorService) RestartWorker(exchange string) error {
	args := m.Called(exchange)
	return args.Error(0)
}

// Mock implementations for CacheAnalyticsService
func (m *MockCacheAnalyticsService) GetCacheStats() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockCacheAnalyticsService) GetCacheStatsByCategory(category string) map[string]interface{} {
	args := m.Called(category)
	return args.Get(0).(map[string]interface{})
}

func (m *MockCacheAnalyticsService) GetCacheMetrics() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockCacheAnalyticsService) ResetCacheStats() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockCacheAnalyticsService) RecordCacheHit(category string) {
	m.Called(category)
}

func (m *MockCacheAnalyticsService) RecordCacheMiss(category string) {
	m.Called(category)
}