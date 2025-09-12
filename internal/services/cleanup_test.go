package services

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	"github.com/irfandi/celebrum-ai-go/internal/telemetry"
)

// TestNewCleanupService tests the NewCleanupService function
func TestNewCleanupService(t *testing.T) {
	// Initialize telemetry for testing to avoid nil logger
	config := telemetry.DefaultConfig()
	config.Enabled = false // Disable OTLP for tests to avoid connection issues
	err := telemetry.InitTelemetry(*config)
	assert.NoError(t, err)

	// Create mock database
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Test creating cleanup service (ResourceManager and PerformanceMonitor are not used)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to avoid type casting issues
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Verify service is created
	assert.NotNil(t, service)
	assert.NotNil(t, service.ctx)
	assert.NotNil(t, service.cancel)
	assert.Nil(t, service.db.Pool) // Pool should be nil as set above
	assert.Equal(t, errorRecoveryManager, service.errorRecoveryManager)
	assert.Nil(t, service.resourceManager) // Should be nil as passed
	assert.Nil(t, service.performanceMonitor) // Should be nil as passed
	assert.NotNil(t, service.logger)
}

// TestCleanupService_Start tests the Start method
func TestCleanupService_Start(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test configuration
	cleanupConfig := config.CleanupConfig{
		MarketData: config.CleanupDataConfig{
			RetentionHours: 36,
			DeletionHours:  12,
		},
		FundingRates: config.CleanupDataConfig{
			RetentionHours: 36,
			DeletionHours:  12,
		},
		ArbitrageOpportunities: config.CleanupArbitrageConfig{
			RetentionHours: 72,
		},
		IntervalMinutes:    60,
		EnableSmartCleanup: true,
	}

	// Test starting the service - should not panic even with nil database
	assert.NotPanics(t, func() {
		service.Start(cleanupConfig)
	})

	// Wait a moment for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Test stopping the service
	assert.NotPanics(t, func() {
		service.Stop()
	})

	// Wait for graceful shutdown
	time.Sleep(10 * time.Millisecond)
}

// TestCleanupService_RunCleanup tests the RunCleanup method
func TestCleanupService_RunCleanup(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test configuration
	cleanupConfig := config.CleanupConfig{
		MarketData: config.CleanupDataConfig{
			RetentionHours: 36,
			DeletionHours:  12,
		},
		FundingRates: config.CleanupDataConfig{
			RetentionHours: 36,
			DeletionHours:  12,
		},
		ArbitrageOpportunities: config.CleanupArbitrageConfig{
			RetentionHours: 72,
		},
		IntervalMinutes:    60,
		EnableSmartCleanup: true,
	}

	// Test running cleanup with nil database - should handle errors gracefully
	err := service.RunCleanup(cleanupConfig)
	// We expect an error due to nil database pool
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}


// TestCleanupService_GetDataStats tests the GetDataStats method
func TestCleanupService_GetDataStats(t *testing.T) {
	// Create mock database
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service (ResourceManager and PerformanceMonitor are not used)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test getting data stats with nil database
	ctx := context.Background()
	stats, err := service.GetDataStats(ctx)
	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_GetDataStats_WithError tests GetDataStats with database error
func TestCleanupService_GetDataStats_WithError(t *testing.T) {
	// Create mock database
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service (ResourceManager and PerformanceMonitor are not used)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test getting data stats with nil database
	ctx := context.Background()
	stats, err := service.GetDataStats(ctx)
	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_CleanupMarketDataSmart tests the cleanupMarketDataSmart method
func TestCleanupService_CleanupMarketDataSmart(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test cleanup market data with nil database - should handle errors gracefully
	ctx := context.Background()
	err := service.cleanupMarketDataSmart(ctx, 36, 12)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_CleanupFundingRatesSmart tests the cleanupFundingRatesSmart method
func TestCleanupService_CleanupFundingRatesSmart(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test cleanup funding rates with nil database - should handle errors gracefully
	ctx := context.Background()
	err := service.cleanupFundingRatesSmart(ctx, 36, 12)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_CleanupArbitrageOpportunities tests the cleanupArbitrageOpportunities method
func TestCleanupService_CleanupArbitrageOpportunities(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test cleanup arbitrage opportunities with nil database - should handle errors gracefully
	ctx := context.Background()
	err := service.cleanupArbitrageOpportunities(ctx, 72)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_CleanupFundingArbitrageOpportunities tests the cleanupFundingArbitrageOpportunities method
func TestCleanupService_CleanupFundingArbitrageOpportunities(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test cleanup funding arbitrage opportunities with nil database - should handle errors gracefully
	ctx := context.Background()
	err := service.cleanupFundingArbitrageOpportunities(ctx, 72)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_CleanupMarketDataSmart_WithError tests cleanupMarketDataSmart with database error
func TestCleanupService_CleanupMarketDataSmart_WithError(t *testing.T) {
	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service with nil database (tests error handling paths)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to test error handling
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test cleanup market data with nil database - should handle errors gracefully
	ctx := context.Background()
	err := service.cleanupMarketDataSmart(ctx, 36, 12)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database pool is not available")
}

// TestCleanupService_Stop tests the Stop method
func TestCleanupService_Stop(t *testing.T) {
	// Create mock database
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service (ResourceManager and PerformanceMonitor are not used)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to avoid type casting issues
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test stopping the service
	assert.NotPanics(t, func() {
		service.Stop()
	})
}

// TestCleanupService_ContextCancellation tests context cancellation handling
func TestCleanupService_ContextCancellation(t *testing.T) {
	// Create mock database
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	// Create real ErrorRecoveryManager for testing
	errorRecoveryManager := NewErrorRecoveryManager(logrus.New())

	// Create cleanup service (ResourceManager and PerformanceMonitor are not used)
	service := NewCleanupService(
		&database.PostgresDB{Pool: nil}, // Use nil Pool to avoid type casting issues
		errorRecoveryManager,
		nil, // ResourceManager not used by CleanupService
		nil, // PerformanceMonitor not used by CleanupService
	)

	// Test configuration
	cleanupConfig := config.CleanupConfig{
		MarketData: config.CleanupDataConfig{
			RetentionHours: 36,
			DeletionHours:  12,
		},
		FundingRates: config.CleanupDataConfig{
			RetentionHours: 36,
			DeletionHours:  12,
		},
		ArbitrageOpportunities: config.CleanupArbitrageConfig{
			RetentionHours: 72,
		},
		IntervalMinutes:    60,
		EnableSmartCleanup: true,
	}

	// Start the service
	service.Start(cleanupConfig)

	// Wait a moment for goroutines to start
	time.Sleep(10 * time.Millisecond)

	// Test stopping the service
	service.Stop()

	// Wait for graceful shutdown
	time.Sleep(10 * time.Millisecond)
}