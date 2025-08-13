package services

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TimeoutConfig defines timeout settings for different operation types
type TimeoutConfig struct {
	APICall        time.Duration
	DatabaseQuery  time.Duration
	RedisOperation time.Duration
	ConcurrentOp   time.Duration
	HealthCheck    time.Duration
	Backfill       time.Duration
	SymbolFetch    time.Duration
	MarketData     time.Duration
}

// TimeoutManager manages timeouts for concurrent operations
type TimeoutManager struct {
	config         *TimeoutConfig
	logger         *logrus.Logger
	activeContexts map[string]context.CancelFunc
	mu             sync.RWMutex
	defaultTimeout time.Duration
}

// OperationContext wraps a context with timeout and cancellation
type OperationContext struct {
	Ctx         context.Context
	Cancel      context.CancelFunc
	OperationID string
	StartTime   time.Time
	Timeout     time.Duration
}

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(config *TimeoutConfig, logger *logrus.Logger) *TimeoutManager {
	if config == nil {
		config = DefaultTimeoutConfig()
	}

	return &TimeoutManager{
		config:         config,
		logger:         logger,
		activeContexts: make(map[string]context.CancelFunc),
		defaultTimeout: 30 * time.Second,
	}
}

// DefaultTimeoutConfig returns default timeout configuration
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		APICall:        10 * time.Second,
		DatabaseQuery:  5 * time.Second,
		RedisOperation: 2 * time.Second,
		ConcurrentOp:   15 * time.Second,
		HealthCheck:    3 * time.Second,
		Backfill:       60 * time.Second,
		SymbolFetch:    20 * time.Second,
		MarketData:     8 * time.Second,
	}
}

// CreateOperationContext creates a new operation context with timeout
func (tm *TimeoutManager) CreateOperationContext(operationType string, operationID string) *OperationContext {
	timeout := tm.getTimeoutForOperation(operationType)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	tm.mu.Lock()
	tm.activeContexts[operationID] = cancel
	tm.mu.Unlock()

	return &OperationContext{
		Ctx:         ctx,
		Cancel:      cancel,
		OperationID: operationID,
		StartTime:   time.Now(),
		Timeout:     timeout,
	}
}

// CreateOperationContextWithParent creates a new operation context with a parent context
func (tm *TimeoutManager) CreateOperationContextWithParent(parent context.Context, operationType string, operationID string) *OperationContext {
	timeout := tm.getTimeoutForOperation(operationType)
	ctx, cancel := context.WithTimeout(parent, timeout)

	tm.mu.Lock()
	tm.activeContexts[operationID] = cancel
	tm.mu.Unlock()

	return &OperationContext{
		Ctx:         ctx,
		Cancel:      cancel,
		OperationID: operationID,
		StartTime:   time.Now(),
		Timeout:     timeout,
	}
}

// CreateOperationContextWithCustomTimeout creates a context with custom timeout
func (tm *TimeoutManager) CreateOperationContextWithCustomTimeout(operationID string, timeout time.Duration) *OperationContext {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	tm.mu.Lock()
	tm.activeContexts[operationID] = cancel
	tm.mu.Unlock()

	return &OperationContext{
		Ctx:         ctx,
		Cancel:      cancel,
		OperationID: operationID,
		StartTime:   time.Now(),
		Timeout:     timeout,
	}
}

// getTimeoutForOperation returns the appropriate timeout for an operation type
func (tm *TimeoutManager) getTimeoutForOperation(operationType string) time.Duration {
	switch operationType {
	case "api_call":
		return tm.config.APICall
	case "database_query":
		return tm.config.DatabaseQuery
	case "redis_operation":
		return tm.config.RedisOperation
	case "concurrent_op":
		return tm.config.ConcurrentOp
	case "health_check":
		return tm.config.HealthCheck
	case "backfill":
		return tm.config.Backfill
	case "symbol_fetch":
		return tm.config.SymbolFetch
	case "market_data":
		return tm.config.MarketData
	default:
		return tm.defaultTimeout
	}
}

// CompleteOperation marks an operation as complete and cleans up resources
func (tm *TimeoutManager) CompleteOperation(operationID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if cancel, exists := tm.activeContexts[operationID]; exists {
		cancel()
		delete(tm.activeContexts, operationID)
	}
}

// CancelOperation cancels a specific operation
func (tm *TimeoutManager) CancelOperation(operationID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if cancel, exists := tm.activeContexts[operationID]; exists {
		cancel()
		delete(tm.activeContexts, operationID)
		tm.logger.WithField("operation_id", operationID).Info("Operation cancelled")
	}
}

// CancelAllOperations cancels all active operations
func (tm *TimeoutManager) CancelAllOperations() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for operationID, cancel := range tm.activeContexts {
		cancel()
		tm.logger.WithField("operation_id", operationID).Info("Operation cancelled during shutdown")
	}

	tm.activeContexts = make(map[string]context.CancelFunc)
}

// GetActiveOperationCount returns the number of active operations
func (tm *TimeoutManager) GetActiveOperationCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.activeContexts)
}

// GetActiveOperations returns a list of active operation IDs
func (tm *TimeoutManager) GetActiveOperations() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	operations := make([]string, 0, len(tm.activeContexts))
	for operationID := range tm.activeContexts {
		operations = append(operations, operationID)
	}

	return operations
}

// ExecuteWithTimeout executes a function with timeout handling
func (tm *TimeoutManager) ExecuteWithTimeout(
	operationType string,
	operationID string,
	operation func(ctx context.Context) (interface{}, error),
) (interface{}, error) {
	opCtx := tm.CreateOperationContext(operationType, operationID)
	defer tm.CompleteOperation(operationID)

	// Create a channel to receive the result
	resultChan := make(chan struct {
		data interface{}
		err  error
	}, 1)

	// Execute the operation in a goroutine
	go func() {
		data, err := operation(opCtx.Ctx)
		resultChan <- struct {
			data interface{}
			err  error
		}{data: data, err: err}
	}()

	// Wait for either completion or timeout
	select {
	case result := <-resultChan:
		duration := time.Since(opCtx.StartTime)
		tm.logger.WithFields(logrus.Fields{
			"operation_type": operationType,
			"operation_id":   operationID,
			"duration":       duration,
			"success":        result.err == nil,
		}).Debug("Operation completed")
		return result.data, result.err

	case <-opCtx.Ctx.Done():
		duration := time.Since(opCtx.StartTime)
		tm.logger.WithFields(logrus.Fields{
			"operation_type": operationType,
			"operation_id":   operationID,
			"duration":       duration,
			"timeout":        opCtx.Timeout,
		}).Warn("Operation timed out")
		return nil, opCtx.Ctx.Err()
	}
}

// ExecuteWithTimeoutAndFallback executes a function with timeout and fallback
func (tm *TimeoutManager) ExecuteWithTimeoutAndFallback(
	operationType string,
	operationID string,
	operation func(ctx context.Context) (interface{}, error),
	fallback func() (interface{}, error),
) (interface{}, error) {
	result, err := tm.ExecuteWithTimeout(operationType, operationID, operation)
	if err != nil && fallback != nil {
		tm.logger.WithFields(logrus.Fields{
			"operation_type": operationType,
			"operation_id":   operationID,
			"error":          err.Error(),
		}).Info("Executing fallback operation")

		return fallback()
	}

	return result, err
}

// MonitorOperationHealth monitors the health of active operations
func (tm *TimeoutManager) MonitorOperationHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		activeCount := tm.GetActiveOperationCount()
		tm.logger.WithField("active_operations", activeCount).Debug("Operation health check")

		// Log warning if too many operations are active
		if activeCount > 100 {
			tm.logger.WithField("active_operations", activeCount).Warn("High number of active operations detected")
		}
	}
}

// UpdateTimeoutConfig updates the timeout configuration
func (tm *TimeoutManager) UpdateTimeoutConfig(config *TimeoutConfig) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.config = config
	tm.logger.Info("Timeout configuration updated")
}

// GetTimeoutConfig returns the current timeout configuration
func (tm *TimeoutManager) GetTimeoutConfig() *TimeoutConfig {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.config
}

// IsOperationActive checks if an operation is currently active
func (tm *TimeoutManager) IsOperationActive(operationID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	_, exists := tm.activeContexts[operationID]
	return exists
}

// GetOperationStats returns statistics about operations
func (tm *TimeoutManager) GetOperationStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return map[string]interface{}{
		"active_operations": len(tm.activeContexts),
		"timeout_config":    tm.config,
		"default_timeout":   tm.defaultTimeout,
	}
}

// Shutdown gracefully shuts down the timeout manager
func (tm *TimeoutManager) Shutdown() {
	tm.logger.Info("Shutting down timeout manager")
	tm.CancelAllOperations()
	tm.logger.Info("Timeout manager shutdown complete")
}
