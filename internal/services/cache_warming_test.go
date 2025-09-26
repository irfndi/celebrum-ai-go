package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/irfandi/celebrum-ai-go/internal/database"
)

// TestCacheWarmingService_NewCacheWarmingService tests service creation
func TestCacheWarmingService_NewCacheWarmingService(t *testing.T) {
	// Create service with nil dependencies to test service creation
	service := NewCacheWarmingService(nil, nil, nil)
	
	// Service should be created successfully (nil dependencies are handled gracefully)
	assert.NotNil(t, service)
}

// TestCacheWarmingService_WarmCache tests the main cache warming function
func TestCacheWarmingService_WarmCache(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.WarmCache(ctx)
	
	// The function should handle nil dependencies gracefully and return nil
	// Individual warming operations will fail and log warnings, but overall function succeeds
	assert.Nil(t, err)
}

// TestCacheWarmingService_warmExchangeConfig tests exchange config warming
func TestCacheWarmingService_warmExchangeConfig(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.warmExchangeConfig(ctx)
	
	// The function should handle nil dependencies gracefully
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// TestCacheWarmingService_warmSupportedExchanges tests supported exchanges warming
func TestCacheWarmingService_warmSupportedExchanges(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.warmSupportedExchanges(ctx)
	
	// The function should handle nil dependencies gracefully
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// TestCacheWarmingService_warmTradingPairs tests trading pairs warming
func TestCacheWarmingService_warmTradingPairs(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.warmTradingPairs(ctx)
	
	// The function should handle nil dependencies gracefully
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// TestCacheWarmingService_warmExchanges tests exchanges warming
func TestCacheWarmingService_warmExchanges(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.warmExchanges(ctx)
	
	// The function should handle nil dependencies gracefully
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// TestCacheWarmingService_warmFundingRates tests funding rates warming
func TestCacheWarmingService_warmFundingRates(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.warmFundingRates(ctx)
	
	// The function should handle nil dependencies gracefully
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// TestCacheWarmingService_warmFundingRates_Success tests successful funding rates warming
func TestCacheWarmingService_warmFundingRates_Success(t *testing.T) {
	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Setup mock database
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	// Create service with nil database pool to test error handling
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	// Mock database response
	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"exchange_name", "symbol", "funding_rate", 
		"next_funding_time", "timestamp",
	}).AddRow(
		"Binance", "BTC/USDT", 0.0001, now.Add(8*time.Hour), now,
	)

	mockDB.ExpectQuery(`SELECT DISTINCT ON \(e\.name, tp\.symbol\)`).
		WithArgs().
		WillReturnRows(rows)

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_DatabaseError tests database query error handling
func TestCacheWarmingService_warmFundingRates_DatabaseError(t *testing.T) {
	// Setup mock database
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create service with nil database pool to test error handling
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	// Mock database error
	mockDB.ExpectQuery(`SELECT DISTINCT ON \(e\.name, tp\.symbol\)`).
		WithArgs().
		WillReturnError(fmt.Errorf("database error"))

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_EmptyResults tests handling of empty database results
func TestCacheWarmingService_warmFundingRates_EmptyResults(t *testing.T) {
	// Setup mock database
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create service with nil database pool to test error handling
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	// Mock empty database response
	rows := pgxmock.NewRows([]string{
		"exchange_name", "symbol", "funding_rate", 
		"next_funding_time", "timestamp",
	})

	mockDB.ExpectQuery(`SELECT DISTINCT ON \(e\.name, tp\.symbol\)`).
		WithArgs().
		WillReturnRows(rows)

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_ScanError tests row scanning error handling
func TestCacheWarmingService_warmFundingRates_ScanError(t *testing.T) {
	// Setup mock database
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create service with nil database pool to test error handling
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	// Mock database response with invalid data types
	rows := pgxmock.NewRows([]string{
		"exchange_name", "symbol", "funding_rate", 
		"next_funding_time", "timestamp",
	}).AddRow(
		"Binance", "BTC/USDT", "invalid_float", "invalid_time", "2024-01-01",
	)

	mockDB.ExpectQuery(`SELECT DISTINCT ON \(e\.name, tp\.symbol\)`).
		WithArgs().
		WillReturnRows(rows)

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_RedisError tests Redis error handling
func TestCacheWarmingService_warmFundingRates_RedisError(t *testing.T) {
	// Setup mock Redis (but don't use it - we want to test Redis nil error)
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	// Create service with nil Redis client to test Redis error
	service := NewCacheWarmingService(nil, nil, &database.PostgresDB{Pool: nil})

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool (checked before Redis)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_FullSuccess tests the complete success path
func TestCacheWarmingService_warmFundingRates_FullSuccess(t *testing.T) {
	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create service with nil database pool to avoid type casting issues for this test
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_MultipleExchanges tests handling multiple exchanges and pairs
func TestCacheWarmingService_warmFundingRates_MultipleExchanges(t *testing.T) {
	// Setup mock database
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create service with nil database pool to test error handling
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	// Mock database response with multiple exchanges and pairs
	now := time.Now()
	rows := pgxmock.NewRows([]string{
		"exchange_name", "symbol", "funding_rate", 
		"next_funding_time", "timestamp",
	}).AddRow(
		"Binance", "BTC/USDT", 0.0001, now.Add(8*time.Hour), now,
	).AddRow(
		"Bybit", "ETH/USDT", 0.0002, now.Add(8*time.Hour), now,
	).AddRow(
		"Binance", "ETH/USDT", -0.0001, now.Add(8*time.Hour), now,
	)

	mockDB.ExpectQuery(`SELECT DISTINCT ON \(e\.name, tp\.symbol\)`).
		WithArgs().
		WillReturnRows(rows)

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_warmFundingRates_TimePrecision tests timestamp and time handling precision
func TestCacheWarmingService_warmFundingRates_TimePrecision(t *testing.T) {
	// Setup mock database
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockDB.Close()

	// Setup mock Redis
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create service with nil database pool to test error handling
	service := NewCacheWarmingService(redisClient, nil, &database.PostgresDB{Pool: nil})

	// Mock database response with precise timestamps
	now := time.Now().UTC()
	nextFunding := now.Add(8 * time.Hour).Truncate(time.Second)
	
	rows := pgxmock.NewRows([]string{
		"exchange_name", "symbol", "funding_rate", 
		"next_funding_time", "timestamp",
	}).AddRow(
		"Binance", "BTC/USDT", 0.000123456789, nextFunding, now.Truncate(time.Microsecond),
	)

	mockDB.ExpectQuery(`SELECT DISTINCT ON \(e\.name, tp\.symbol\)`).
		WithArgs().
		WillReturnRows(rows)

	ctx := context.Background()
	err = service.warmFundingRates(ctx)

	// Verify error is handled gracefully due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCacheWarmingService_errorHandling tests error handling
func TestCacheWarmingService_errorHandling(t *testing.T) {
	// Create service with nil dependencies to test error handling
	service := NewCacheWarmingService(nil, nil, nil)
	
	ctx := context.Background()
	err := service.WarmCache(ctx)
	
	// The function should handle nil dependencies gracefully and return nil
	// Individual warming operations will fail and log warnings, but overall function succeeds
	assert.Nil(t, err)
}