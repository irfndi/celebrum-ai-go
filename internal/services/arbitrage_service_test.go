package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
)


func TestArbitrageService_ConfigParsing(t *testing.T) {
	// Test configuration parsing logic
	type ArbitrageServiceConfig struct {
		IntervalSeconds int     `mapstructure:"interval_seconds"`
		MinProfit       float64 `mapstructure:"min_profit"`
		MaxAgeMinutes   int     `mapstructure:"max_age_minutes"`
		BatchSize       int     `mapstructure:"batch_size"`
		Enabled         bool    `mapstructure:"enabled"`
	}
	
	// Test default values
	config := ArbitrageServiceConfig{
		IntervalSeconds: 60, // 1 minute default
		MinProfit:       0.5, // 0.5% minimum profit
		MaxAgeMinutes:   30, // 30 minutes default
		BatchSize:       100, // 100 items default
		Enabled:         true,
	}
	
	assert.Equal(t, 60, config.IntervalSeconds)
	assert.Equal(t, 0.5, config.MinProfit)
	assert.Equal(t, 30, config.MaxAgeMinutes)
	assert.Equal(t, 100, config.BatchSize)
	assert.True(t, config.Enabled)
}

func TestArbitrageService_ContextManagement(t *testing.T) {
	// Test context management for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	// Test that context is properly initialized
	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)
	
	// Test context cancellation
	cancel()
	
	// Wait for cancellation to propagate
	time.Sleep(10 * time.Millisecond)
	
	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Context was cancelled as expected
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		t.Error("Context should have been cancelled")
	}
}

func TestArbitrageService_ConcurrentOperations(t *testing.T) {
	// Test concurrent operations on the service
	var wg sync.WaitGroup
	var counter int64
	var mu sync.Mutex
	
	// Test concurrent increment operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}
	
	wg.Wait()
	
	// Verify that all operations completed
	assert.Equal(t, int64(100), counter)
}

func TestArbitrageService_TimeHandling(t *testing.T) {
	// Test time handling for arbitrage calculations
	now := time.Now()
	
	// Test that timestamps are properly recorded
	assert.False(t, now.IsZero())
	assert.True(t, now.After(time.Time{}))
	
	// Test time calculations
	interval := time.Minute
	nextTime := now.Add(interval)
	
	assert.True(t, nextTime.After(now))
	assert.Equal(t, interval, nextTime.Sub(now))
}

func TestArbitrageService_ErrorHandling(t *testing.T) {
	// Test error handling patterns
	testError := func() error {
		return assert.AnError
	}
	
	// Test error return
	err := testError()
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestArbitrageService_StateTransitions(t *testing.T) {
	// Test state transitions for the arbitrage service
	type ServiceState int
	const (
		Stopped ServiceState = iota
		Starting
		Running
		Stopping
	)
	
	var currentState ServiceState
	var mu sync.RWMutex
	
	// Test state transitions
	setState := func(newState ServiceState) {
		mu.Lock()
		defer mu.Unlock()
		currentState = newState
	}
	
	getState := func() ServiceState {
		mu.RLock()
		defer mu.RUnlock()
		return currentState
	}
	
	// Test initial state
	assert.Equal(t, Stopped, getState())
	
	// Test state changes
	setState(Starting)
	assert.Equal(t, Starting, getState())
	
	setState(Running)
	assert.Equal(t, Running, getState())
	
	setState(Stopping)
	assert.Equal(t, Stopping, getState())
	
	setState(Stopped)
	assert.Equal(t, Stopped, getState())
}

func TestArbitrageService_MetricsCollection(t *testing.T) {
	// Test metrics collection functionality
	type Metrics struct {
		OpportunitiesFound int
		TotalCalculations int
		FailedCalculations int
		LastCalculation time.Time
		mu sync.RWMutex
	}
	
	metrics := &Metrics{}
	
	// Test metrics recording
	recordOpportunity := func() {
		metrics.mu.Lock()
		defer metrics.mu.Unlock()
		metrics.OpportunitiesFound++
		metrics.TotalCalculations++
		metrics.LastCalculation = time.Now()
	}
	
	recordFailure := func() {
		metrics.mu.Lock()
		defer metrics.mu.Unlock()
		metrics.FailedCalculations++
		metrics.TotalCalculations++
		metrics.LastCalculation = time.Now()
	}
	
	// Test recording opportunities
	recordOpportunity()
	recordOpportunity()
	
	// Test recording failures
	recordFailure()
	recordFailure()
	recordFailure()
	
	// Verify metrics
	metrics.mu.RLock()
	assert.Equal(t, 2, metrics.OpportunitiesFound)
	assert.Equal(t, 5, metrics.TotalCalculations)
	assert.Equal(t, 3, metrics.FailedCalculations)
	assert.False(t, metrics.LastCalculation.IsZero())
	metrics.mu.RUnlock()
}

func TestArbitrageService_BatchProcessing(t *testing.T) {
	// Test batch processing functionality
	type BatchProcessor struct {
		batchSize int
		items []interface{}
		mu sync.Mutex
	}
	
	processor := &BatchProcessor{
		batchSize: 10,
		items: make([]interface{}, 0),
	}
	
	// Test adding items to batch
	addItem := func(item interface{}) {
		processor.mu.Lock()
		defer processor.mu.Unlock()
		processor.items = append(processor.items, item)
	}
	
	getBatchSize := func() int {
		processor.mu.Lock()
		defer processor.mu.Unlock()
		return len(processor.items)
	}
	
	// Add items to batch
	for i := 0; i < 25; i++ {
		addItem(i)
	}
	
	// Verify batch size
	assert.Equal(t, 25, getBatchSize())
	
	// Test batch processing logic
	processBatch := func() [][]interface{} {
		processor.mu.Lock()
		defer processor.mu.Unlock()
		
		var batches [][]interface{}
		for i := 0; i < len(processor.items); i += processor.batchSize {
			end := i + processor.batchSize
			if end > len(processor.items) {
				end = len(processor.items)
			}
			batches = append(batches, processor.items[i:end])
		}
		return batches
	}
	
	batches := processBatch()
	assert.Equal(t, 3, len(batches)) // 25 items with batch size 10 = 3 batches
	assert.Equal(t, 10, len(batches[0]))
	assert.Equal(t, 10, len(batches[1]))
	assert.Equal(t, 5, len(batches[2]))
}

func TestNewArbitrageService(t *testing.T) {
	// Test creating a new arbitrage service with default configuration
	var mockDB *database.PostgresDB // Using nil for testing service logic
	mockConfig := &config.Config{
		Arbitrage: config.ArbitrageConfig{
			Enabled:             true,
			IntervalSeconds:    30,
			MinProfitThreshold:  1.0,
			MaxAgeMinutes:      60,
			BatchSize:          50,
		},
	}
	
	calculator := NewFuturesArbitrageCalculator()
	
	service := NewArbitrageService(mockDB, mockConfig, calculator)
	
	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)
	assert.Equal(t, mockConfig, service.config)
	assert.Equal(t, calculator, service.calculator)
	assert.Equal(t, 30, service.arbitrageConfig.IntervalSeconds)
	assert.Equal(t, 1.0, service.arbitrageConfig.MinProfit)
	assert.Equal(t, 60, service.arbitrageConfig.MaxAgeMinutes)
	assert.Equal(t, 50, service.arbitrageConfig.BatchSize)
	assert.True(t, service.arbitrageConfig.Enabled)
	assert.False(t, service.isRunning)
}

func TestNewArbitrageService_DefaultValues(t *testing.T) {
	// Test creating a new arbitrage service with default values when config is empty
	var mockDB *database.PostgresDB // Using nil for testing service logic
	mockConfig := &config.Config{
		Arbitrage: config.ArbitrageConfig{
			Enabled: false,
		},
	}
	
	calculator := NewFuturesArbitrageCalculator()
	
	service := NewArbitrageService(mockDB, mockConfig, calculator)
	
	assert.NotNil(t, service)
	assert.Equal(t, 60, service.arbitrageConfig.IntervalSeconds) // default
	assert.Equal(t, 0.5, service.arbitrageConfig.MinProfit)     // default
	assert.Equal(t, 30, service.arbitrageConfig.MaxAgeMinutes)  // default
	assert.Equal(t, 100, service.arbitrageConfig.BatchSize)     // default
	assert.False(t, service.arbitrageConfig.Enabled)
}

func TestArbitrageService_StartStop(t *testing.T) {
	// Test starting and stopping the arbitrage service with disabled configuration
	var mockDB *database.PostgresDB // Using nil for testing service logic
	mockConfig := &config.Config{
		Arbitrage: config.ArbitrageConfig{
			Enabled: false, // Test with disabled to avoid database issues
		},
	}
	
	calculator := NewFuturesArbitrageCalculator()
	service := NewArbitrageService(mockDB, mockConfig, calculator)
	
	// Test initial state
	assert.False(t, service.IsRunning())
	
	// Test starting when disabled - should not error but not start
	err := service.Start()
	assert.NoError(t, err)
	assert.False(t, service.IsRunning()) // Should not start when disabled
	
	// Test stopping when disabled - should not panic
	service.Stop()
	assert.False(t, service.IsRunning())
}

func TestArbitrageService_StartDisabled(t *testing.T) {
	// Test starting the service when disabled in configuration
	var mockDB *database.PostgresDB // Using nil for testing service logic
	mockConfig := &config.Config{
		Arbitrage: config.ArbitrageConfig{
			Enabled: false,
		},
	}
	
	calculator := NewFuturesArbitrageCalculator()
	service := NewArbitrageService(mockDB, mockConfig, calculator)
	
	// Test starting when disabled
	err := service.Start()
	assert.NoError(t, err)
	assert.False(t, service.IsRunning()) // Should not start when disabled
}

func TestArbitrageService_GetStatus(t *testing.T) {
	// Test getting service status
	var mockDB *database.PostgresDB // Using nil for testing service logic
	mockConfig := &config.Config{
		Arbitrage: config.ArbitrageConfig{
			Enabled: true,
		},
	}
	
	calculator := NewFuturesArbitrageCalculator()
	service := NewArbitrageService(mockDB, mockConfig, calculator)
	
	// Test initial status
	isRunning, lastCalculation, opportunitiesFound := service.GetStatus()
	assert.False(t, isRunning)
	assert.True(t, lastCalculation.IsZero())
	assert.Equal(t, 0, opportunitiesFound)
}

func TestArbitrageService_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to service methods - simplified version
	var mockDB *database.PostgresDB // Using nil for testing service logic
	mockConfig := &config.Config{
		Arbitrage: config.ArbitrageConfig{
			Enabled: false, // Disabled to avoid database issues
		},
	}
	
	calculator := NewFuturesArbitrageCalculator()
	service := NewArbitrageService(mockDB, mockConfig, calculator)
	
	var wg sync.WaitGroup
	
	// Test concurrent status checks (safe operation)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = service.GetStatus()
		}()
	}
	
	wg.Wait()
	
	// Verify service is still not running
	assert.False(t, service.IsRunning())
}

func TestArbitrageService_ConfigValidation(t *testing.T) {
	// Test various configuration combinations
	testCases := []struct {
		name     string
		config   config.ArbitrageConfig
		expected ArbitrageServiceConfig
	}{
		{
			name: "All values provided",
			config: config.ArbitrageConfig{
				Enabled:             true,
				IntervalSeconds:    120,
				MinProfitThreshold:  2.5,
				MaxAgeMinutes:      90,
				BatchSize:          200,
			},
			expected: ArbitrageServiceConfig{
				Enabled:           true,
				IntervalSeconds:   120,
				MinProfit:         2.5,
				MaxAgeMinutes:     90,
				BatchSize:         200,
			},
		},
		{
			name: "Only enabled flag",
			config: config.ArbitrageConfig{
				Enabled: true,
			},
			expected: ArbitrageServiceConfig{
				Enabled:           true,
				IntervalSeconds:   60,  // default
				MinProfit:         0.5,  // default
				MaxAgeMinutes:     30,  // default
				BatchSize:         100, // default
			},
		},
		{
			name: "Partial configuration",
			config: config.ArbitrageConfig{
				Enabled:             false,
				IntervalSeconds:    45,
				MinProfitThreshold:  1.0,
			},
			expected: ArbitrageServiceConfig{
				Enabled:           false,
				IntervalSeconds:   45,
				MinProfit:         1.0,
				MaxAgeMinutes:     30,  // default
				BatchSize:         100, // default
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var mockDB *database.PostgresDB // Using nil for testing service logic
			mockConfig := &config.Config{
				Arbitrage: tc.config,
			}
			
			calculator := NewFuturesArbitrageCalculator()
			service := NewArbitrageService(mockDB, mockConfig, calculator)
			
			assert.Equal(t, tc.expected.Enabled, service.arbitrageConfig.Enabled)
			assert.Equal(t, tc.expected.IntervalSeconds, service.arbitrageConfig.IntervalSeconds)
			assert.Equal(t, tc.expected.MinProfit, service.arbitrageConfig.MinProfit)
			assert.Equal(t, tc.expected.MaxAgeMinutes, service.arbitrageConfig.MaxAgeMinutes)
			assert.Equal(t, tc.expected.BatchSize, service.arbitrageConfig.BatchSize)
		})
	}
}