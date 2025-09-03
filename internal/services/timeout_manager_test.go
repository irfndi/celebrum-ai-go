package services

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestTimeoutManager_NewTimeoutManager tests timeout manager creation
func TestTimeoutManager_NewTimeoutManager(t *testing.T) {
	logger := logrus.New()
	
	// Test with config
	config := DefaultTimeoutConfig()
	tm := NewTimeoutManager(config, logger)
	
	assert.NotNil(t, tm)
	assert.Equal(t, config, tm.config)
	assert.Equal(t, logger, tm.logger)
	assert.NotNil(t, tm.activeContexts)
	assert.Equal(t, 30*time.Second, tm.defaultTimeout)
}

// TestTimeoutManager_DefaultTimeoutConfig tests the default timeout configuration
func TestTimeoutManager_DefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, 10*time.Second, config.APICall)
	assert.Equal(t, 5*time.Second, config.DatabaseQuery)
	assert.Equal(t, 2*time.Second, config.RedisOperation)
	assert.Equal(t, 15*time.Second, config.ConcurrentOp)
	assert.Equal(t, 3*time.Second, config.HealthCheck)
	assert.Equal(t, 60*time.Second, config.Backfill)
	assert.Equal(t, 20*time.Second, config.SymbolFetch)
	assert.Equal(t, 8*time.Second, config.MarketData)
}

// TestTimeoutManager_CreateOperationContext tests operation context creation
func TestTimeoutManager_CreateOperationContext(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	opCtx := tm.CreateOperationContext("test", "op1")
	
	assert.NotNil(t, opCtx)
	assert.Equal(t, "op1", opCtx.OperationID)
	assert.NotNil(t, opCtx.Ctx)
	assert.NotNil(t, opCtx.Cancel)
	assert.True(t, time.Since(opCtx.StartTime) < time.Second)
}

// TestTimeoutManager_CreateOperationContextWithParent tests operation context creation with parent
func TestTimeoutManager_CreateOperationContextWithParent(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()
	
	opCtx := tm.CreateOperationContextWithParent(parentCtx, "test", "op1")
	
	assert.NotNil(t, opCtx)
	assert.Equal(t, "op1", opCtx.OperationID)
	assert.NotNil(t, opCtx.Ctx)
	assert.NotNil(t, opCtx.Cancel)
	
	// Test that parent cancellation affects child
	parentCancel()
	select {
	case <-opCtx.Ctx.Done():
		// Context should be cancelled when parent is cancelled
	default:
		t.Error("Child context should be cancelled when parent is cancelled")
	}
}

// TestTimeoutManager_CreateOperationContextWithCustomTimeout tests custom timeout creation
func TestTimeoutManager_CreateOperationContextWithCustomTimeout(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	opCtx := tm.CreateOperationContextWithCustomTimeout("op1", 5*time.Second)
	
	assert.NotNil(t, opCtx)
	assert.Equal(t, "op1", opCtx.OperationID)
	assert.Equal(t, 5*time.Second, opCtx.Timeout)
	assert.NotNil(t, opCtx.Ctx)
	assert.NotNil(t, opCtx.Cancel)
}

// TestTimeoutManager_CompleteOperation tests operation completion
func TestTimeoutManager_CompleteOperation(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	opCtx := tm.CreateOperationContext("test", "op1")
	
	// Complete the operation
	tm.CompleteOperation("op1")
	
	// Verify context is cancelled
	select {
	case <-opCtx.Ctx.Done():
		// Context should be cancelled
	default:
		t.Error("Context should be cancelled after completion")
	}
}

// TestTimeoutManager_CancelOperation tests operation cancellation
func TestTimeoutManager_CancelOperation(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	opCtx := tm.CreateOperationContext("test", "op1")
	
	// Cancel the operation
	tm.CancelOperation("op1")
	
	// Verify context is cancelled
	select {
	case <-opCtx.Ctx.Done():
		// Context should be cancelled
	default:
		t.Error("Context should be cancelled after cancellation")
	}
}

// TestTimeoutManager_CancelAllOperations tests cancelling all operations
func TestTimeoutManager_CancelAllOperations(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Create multiple operations
	opCtx1 := tm.CreateOperationContext("test", "op1")
	opCtx2 := tm.CreateOperationContext("test", "op2")
	
	// Cancel all operations
	tm.CancelAllOperations()
	
	// Verify all contexts are cancelled
	select {
	case <-opCtx1.Ctx.Done():
		// Context should be cancelled
	default:
		t.Error("Context op1 should be cancelled")
	}
	
	select {
	case <-opCtx2.Ctx.Done():
		// Context should be cancelled
	default:
		t.Error("Context op2 should be cancelled")
	}
}

// TestTimeoutManager_GetActiveOperationCount tests active operation counting
func TestTimeoutManager_GetActiveOperationCount(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Initially no active operations
	count := tm.GetActiveOperationCount()
	assert.Equal(t, 0, count)
	
	// Create an operation
	opCtx := tm.CreateOperationContext("test", "op1")
	defer opCtx.Cancel()
	
	// Should have one active operation
	count = tm.GetActiveOperationCount()
	assert.Equal(t, 1, count)
}

// TestTimeoutManager_GetActiveOperations tests getting active operations
func TestTimeoutManager_GetActiveOperations(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Create an operation
	opCtx := tm.CreateOperationContext("test", "op1")
	defer opCtx.Cancel()
	
	// Get active operations
	operations := tm.GetActiveOperations()
	assert.Contains(t, operations, "op1")
}

// TestTimeoutManager_ExecuteWithTimeout tests timeout execution
func TestTimeoutManager_ExecuteWithTimeout(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	result, err := tm.ExecuteWithTimeout("test", "op1", func(ctx context.Context) (interface{}, error) {
		return "success", nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

// TestTimeoutManager_ExecuteWithTimeoutAndFallback tests timeout execution with fallback
func TestTimeoutManager_ExecuteWithTimeoutAndFallback(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Test successful execution
	result, err := tm.ExecuteWithTimeoutAndFallback("test", "op1", func(ctx context.Context) (interface{}, error) {
		return "success", nil
	}, func() (interface{}, error) {
		return "fallback", nil
	})
	
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

// TestTimeoutManager_MonitorOperationHealth tests operation health monitoring
func TestTimeoutManager_MonitorOperationHealth(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Monitor operation health (this function starts a goroutine, so we just test it doesn't panic)
	go tm.MonitorOperationHealth()
	time.Sleep(10 * time.Millisecond) // Give it time to start
}

// TestTimeoutManager_UpdateTimeoutConfig tests timeout configuration updates
func TestTimeoutManager_UpdateTimeoutConfig(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	newConfig := &TimeoutConfig{
		APICall: 20 * time.Second,
	}
	
	tm.UpdateTimeoutConfig(newConfig)
	
	// Verify config was updated
	retrievedConfig := tm.GetTimeoutConfig()
	assert.Equal(t, 20*time.Second, retrievedConfig.APICall)
}

// TestTimeoutManager_IsOperationActive tests operation active status
func TestTimeoutManager_IsOperationActive(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Test with non-existent operation
	isActive := tm.IsOperationActive("nonexistent")
	assert.False(t, isActive)
	
	// Create an operation
	opCtx := tm.CreateOperationContext("test", "op1")
	defer opCtx.Cancel()
	
	// Test with active operation
	isActive = tm.IsOperationActive("op1")
	assert.True(t, isActive)
}

// TestTimeoutManager_GetOperationStats tests operation statistics
func TestTimeoutManager_GetOperationStats(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Create an operation
	opCtx := tm.CreateOperationContext("test", "op1")
	defer opCtx.Cancel()
	
	// Get operation stats
	stats := tm.GetOperationStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "active_operations")
	assert.Contains(t, stats, "timeout_config")
}

// TestTimeoutManager_Shutdown tests graceful shutdown
func TestTimeoutManager_Shutdown(t *testing.T) {
	tm := NewTimeoutManager(nil, logrus.New())
	
	// Create an operation
	opCtx := tm.CreateOperationContext("test", "op1")
	
	// Shutdown timeout manager
	tm.Shutdown()
	
	// Verify all operations are cancelled
	select {
	case <-opCtx.Ctx.Done():
		// Context should be cancelled
	default:
		t.Error("Context should be cancelled after shutdown")
	}
}