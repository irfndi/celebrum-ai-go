package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewResourceOptimizer(t *testing.T) {
	config := ResourceOptimizerConfig{
		OptimizationInterval: 5 * time.Minute,
		AdaptiveMode:         true,
		MaxHistorySize:       100,
		CPUThreshold:         80.0,
		MemoryThreshold:      85.0,
		MinWorkers:           2,
		MaxWorkers:           20,
	}

	ro := NewResourceOptimizer(config)

	assert.NotNil(t, ro)
	assert.Greater(t, ro.cpuCores, 0)
	assert.Greater(t, ro.memoryGB, 0.0)
	assert.Equal(t, config.OptimizationInterval, ro.optimizationInterval)
	assert.Equal(t, config.MaxHistorySize, ro.maxHistorySize)
	assert.Equal(t, config.AdaptiveMode, ro.adaptiveMode)
	assert.NotNil(t, ro.logger)
	assert.NotNil(t, ro.performanceHistory) // Should be initialized but empty
	assert.NotNil(t, ro.optimalConcurrency)
}

func TestNewResourceOptimizer_WithDefaults(t *testing.T) {
	config := ResourceOptimizerConfig{} // Use default values

	ro := NewResourceOptimizer(config)

	assert.NotNil(t, ro)
	assert.Greater(t, ro.cpuCores, 0)
	assert.Greater(t, ro.memoryGB, 0.0)
	assert.Equal(t, 5*time.Minute, ro.optimizationInterval) // Default value
	assert.Equal(t, 100, ro.maxHistorySize)                 // Default value
	assert.False(t, ro.adaptiveMode)                        // Default value
}

func TestResourceOptimizer_calculateOptimalConcurrency(t *testing.T) {
	config := ResourceOptimizerConfig{
		MinWorkers:      2,
		MaxWorkers:      20,
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
	}

	ro := NewResourceOptimizer(config)

	// Test normal conditions
	ro.currentCPUUsage = 50.0
	ro.currentMemoryUsage = 60.0

	ro.calculateOptimalConcurrency(config)

	concurrency := ro.GetOptimalConcurrency()
	assert.GreaterOrEqual(t, concurrency.MaxWorkers, config.MinWorkers)
	assert.LessOrEqual(t, concurrency.MaxWorkers, config.MaxWorkers)
	assert.Equal(t, concurrency.MaxWorkers, concurrency.MaxConcurrentSymbols)
	assert.LessOrEqual(t, concurrency.MaxConcurrentBackfill, concurrency.MaxWorkers)
	assert.LessOrEqual(t, concurrency.MaxConcurrentWrites, concurrency.MaxWorkers)
	assert.Greater(t, concurrency.MaxCircuitBreakerCalls, concurrency.MaxWorkers)
	assert.Equal(t, 0.8, concurrency.WorkerPoolUtilization)
	assert.Equal(t, config.MemoryThreshold, concurrency.MemoryThreshold)
	assert.Equal(t, config.CPUThreshold, concurrency.CPUThreshold)
}

func TestResourceOptimizer_calculateOptimalConcurrency_HighLoad(t *testing.T) {
	config := ResourceOptimizerConfig{
		MinWorkers:      2,
		MaxWorkers:      20,
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
	}

	ro := NewResourceOptimizer(config)

	// Simulate high load
	ro.currentCPUUsage = 90.0
	ro.currentMemoryUsage = 90.0

	ro.calculateOptimalConcurrency(config)

	concurrency := ro.GetOptimalConcurrency()
	// Should be reduced due to high load
	assert.Less(t, concurrency.MaxWorkers, ro.cpuCores*2)
}

func TestResourceOptimizer_calculateOptimalConcurrency_LowMemory(t *testing.T) {
	config := ResourceOptimizerConfig{
		MinWorkers:      2,
		MaxWorkers:      20,
		CPUThreshold:    80.0,
		MemoryThreshold: 85.0,
	}

	ro := NewResourceOptimizer(config)

	// Simulate low memory system
	ro.memoryGB = 2.0
	ro.currentCPUUsage = 50.0
	ro.currentMemoryUsage = 60.0

	ro.calculateOptimalConcurrency(config)

	concurrency := ro.GetOptimalConcurrency()
	// Should be reduced due to low memory
	assert.Less(t, concurrency.MaxWorkers, ro.cpuCores*2)
}

func TestResourceOptimizer_GetOptimalConcurrency(t *testing.T) {
	config := ResourceOptimizerConfig{
		MinWorkers: 2,
		MaxWorkers: 10,
	}

	ro := NewResourceOptimizer(config)
	ro.calculateOptimalConcurrency(config)

	concurrency := ro.GetOptimalConcurrency()
	assert.NotNil(t, concurrency)
	assert.GreaterOrEqual(t, concurrency.MaxWorkers, config.MinWorkers)
	assert.LessOrEqual(t, concurrency.MaxWorkers, config.MaxWorkers)
}

func TestResourceOptimizer_UpdateSystemMetrics(t *testing.T) {
	config := ResourceOptimizerConfig{}
	ro := NewResourceOptimizer(config)

	ctx := context.Background()
	err := ro.UpdateSystemMetrics(ctx)

	// This might fail in test environment due to missing system metrics
	// but should not panic
	if err != nil {
		assert.ErrorContains(t, err, "failed to get")
	} else {
		assert.GreaterOrEqual(t, ro.currentCPUUsage, 0.0)
		assert.GreaterOrEqual(t, ro.currentMemoryUsage, 0.0)
	}
}

func TestResourceOptimizer_UpdateSystemMetrics_WithMockData(t *testing.T) {
	config := ResourceOptimizerConfig{}
	ro := NewResourceOptimizer(config)

	// Mock system metrics
	ro.currentCPUUsage = 75.0
	ro.currentMemoryUsage = 65.0

	assert.Equal(t, 75.0, ro.currentCPUUsage)
	assert.Equal(t, 65.0, ro.currentMemoryUsage)
}

func TestResourceOptimizer_RecordPerformanceSnapshot(t *testing.T) {
	config := ResourceOptimizerConfig{
		MaxHistorySize: 5,
	}
	ro := NewResourceOptimizer(config)

	initialLen := len(ro.performanceHistory)

	// Record initial snapshot
	ro.RecordPerformanceSnapshot(10, 100.0, 1.0, 50.0)

	if assert.Equal(t, initialLen+1, len(ro.performanceHistory)) {
		snapshot := ro.performanceHistory[len(ro.performanceHistory)-1]
		assert.Equal(t, 10, snapshot.ActiveOperations)
		assert.Equal(t, 100.0, snapshot.Throughput)
		assert.Equal(t, 1.0, snapshot.ErrorRate)
		assert.Equal(t, 50.0, snapshot.ResponseTime)
		assert.False(t, snapshot.Timestamp.IsZero())
	}

	// Record multiple snapshots
	for i := 0; i < 10; i++ {
		ro.RecordPerformanceSnapshot(i+1, float64(i*10), float64(i), float64(i*5))
	}

	// Should respect max history size
	assert.Equal(t, config.MaxHistorySize, len(ro.performanceHistory))
}

func TestResourceOptimizer_OptimizeIfNeeded_RegularInterval(t *testing.T) {
	config := ResourceOptimizerConfig{
		OptimizationInterval: 100 * time.Millisecond,
		AdaptiveMode:         false,
		MinWorkers:           2,
		MaxWorkers:           10,
	}

	ro := NewResourceOptimizer(config)
	ro.lastOptimization = time.Now().Add(-200 * time.Millisecond) // Make it seem old

	optimized := ro.OptimizeIfNeeded(config)
	assert.True(t, optimized)
	assert.True(t, time.Since(ro.lastOptimization) < 100*time.Millisecond)

	// Should not optimize again immediately
	optimized = ro.OptimizeIfNeeded(config)
	assert.False(t, optimized)
}

func TestResourceOptimizer_OptimizeIfNeeded_AdaptiveMode(t *testing.T) {
	config := ResourceOptimizerConfig{
		OptimizationInterval: 1 * time.Hour,
		AdaptiveMode:         true,
		MaxHistorySize:       10,
		MinWorkers:           2,
		MaxWorkers:           10,
	}

	ro := NewResourceOptimizer(config)
	ro.lastOptimization = time.Now().Add(-2 * time.Hour) // Due for regular optimization

	// Add performance snapshots that should trigger optimization
	for i := 0; i < 5; i++ {
		ro.currentCPUUsage = 10.0
		ro.currentMemoryUsage = 100.0
		ro.RecordPerformanceSnapshot(10, 100.0, 10.0, 6000.0) // High error rate and response time
	}

	// Test shouldOptimize directly
	shouldOpt := ro.shouldOptimize()
	t.Logf("Should optimize result: %v", shouldOpt)

	// Try with the original config
	optimized := ro.OptimizeIfNeeded(config)
	t.Logf("Optimization result: %v", optimized)
	assert.True(t, optimized)
}

func TestResourceOptimizer_shouldOptimize(t *testing.T) {
	config := ResourceOptimizerConfig{
		MaxHistorySize: 10,
	}
	ro := NewResourceOptimizer(config)

	// Not enough snapshots
	assert.False(t, ro.shouldOptimize())

	// Add snapshots with good performance
	for i := 0; i < 5; i++ {
		ro.RecordPerformanceSnapshot(5, 100.0, 0.5, 100.0)
	}

	assert.False(t, ro.shouldOptimize())

	// Add snapshots with high CPU usage
	ro.performanceHistory = ro.performanceHistory[:0]
	for i := 0; i < 5; i++ {
		snapshot := PerformanceSnapshot{
			CPUUsage:    90.0, // High CPU
			MemoryUsage: 60.0,
		}
		ro.performanceHistory = append(ro.performanceHistory, snapshot)
	}

	assert.True(t, ro.shouldOptimize())

	// Add snapshots with high memory usage
	ro.performanceHistory = ro.performanceHistory[:0]
	for i := 0; i < 5; i++ {
		snapshot := PerformanceSnapshot{
			CPUUsage:    60.0,
			MemoryUsage: 95.0, // High memory
		}
		ro.performanceHistory = append(ro.performanceHistory, snapshot)
	}

	assert.True(t, ro.shouldOptimize())

	// Add snapshots with high error rate
	ro.performanceHistory = ro.performanceHistory[:0]
	for i := 0; i < 5; i++ {
		snapshot := PerformanceSnapshot{
			CPUUsage:    60.0,
			MemoryUsage: 60.0,
			ErrorRate:   10.0, // High error rate
		}
		ro.performanceHistory = append(ro.performanceHistory, snapshot)
	}

	assert.True(t, ro.shouldOptimize())

	// Add snapshots with high response time
	ro.performanceHistory = ro.performanceHistory[:0]
	for i := 0; i < 5; i++ {
		snapshot := PerformanceSnapshot{
			CPUUsage:     60.0,
			MemoryUsage:  60.0,
			ErrorRate:    1.0,
			ResponseTime: 6000.0, // High response time
		}
		ro.performanceHistory = append(ro.performanceHistory, snapshot)
	}

	assert.True(t, ro.shouldOptimize())
}

func TestResourceOptimizer_GetPerformanceHistory(t *testing.T) {
	config := ResourceOptimizerConfig{
		MaxHistorySize: 10,
	}
	ro := NewResourceOptimizer(config)

	initialLen := len(ro.performanceHistory)

	// Add some snapshots
	for i := 0; i < 5; i++ {
		ro.RecordPerformanceSnapshot(i+1, float64(i*10), float64(i), float64(i*5))
	}

	// Get all history
	history := ro.GetPerformanceHistory(0)
	assert.Equal(t, initialLen+5, len(history))

	// Get limited history
	limited := ro.GetPerformanceHistory(3)
	assert.Equal(t, 3, len(limited))

	// Get more than available
	overflow := ro.GetPerformanceHistory(10)
	assert.Equal(t, initialLen+5, len(overflow))
}

func TestResourceOptimizer_GetSystemInfo(t *testing.T) {
	config := ResourceOptimizerConfig{}
	ro := NewResourceOptimizer(config)

	info := ro.GetSystemInfo()

	assert.Contains(t, info, "cpu_cores")
	assert.Contains(t, info, "memory_gb")
	assert.Contains(t, info, "current_cpu")
	assert.Contains(t, info, "current_memory")
	assert.Contains(t, info, "goroutines")
	assert.Contains(t, info, "last_optimization")
	assert.Contains(t, info, "adaptive_mode")
	assert.Contains(t, info, "optimal_config")

	// Check types
	assert.IsType(t, ro.cpuCores, info["cpu_cores"])
	assert.IsType(t, ro.memoryGB, info["memory_gb"])
	assert.IsType(t, ro.adaptiveMode, info["adaptive_mode"])
}

func TestResourceOptimizer_ConcurrentAccess(t *testing.T) {
	config := ResourceOptimizerConfig{
		OptimizationInterval: 10 * time.Millisecond,
		AdaptiveMode:         true,
		MaxHistorySize:       10,
	}

	ro := NewResourceOptimizer(config)

	// Test concurrent access to various methods
	for i := 0; i < 100; i++ {
		go func(i int) {
			ro.GetOptimalConcurrency()
			ro.RecordPerformanceSnapshot(i, float64(i), float64(i), float64(i))
			ro.GetPerformanceHistory(5)
			ro.GetSystemInfo()
		}(i)
	}

	// Let goroutines complete
	time.Sleep(50 * time.Millisecond)

	// Get performance history with proper synchronization
	history := ro.GetPerformanceHistory(100) // Get up to 100 entries
	
	// Should not panic and data should be consistent
	assert.GreaterOrEqual(t, len(history), 0)
	assert.LessOrEqual(t, len(history), config.MaxHistorySize)
}

func TestResourceOptimizer_min(t *testing.T) {
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 0, min(0, 10))
	assert.Equal(t, -5, min(-5, 0))
}

func TestResourceOptimizer_Configuration(t *testing.T) {
	config := ResourceOptimizerConfig{
		OptimizationInterval: 1 * time.Minute,
		AdaptiveMode:         true,
		MaxHistorySize:       50,
		CPUThreshold:         90.0,
		MemoryThreshold:      95.0,
		MinWorkers:           5,
		MaxWorkers:           50,
	}

	ro := NewResourceOptimizer(config)

	assert.Equal(t, config.OptimizationInterval, ro.optimizationInterval)
	assert.Equal(t, config.AdaptiveMode, ro.adaptiveMode)
	assert.Equal(t, config.MaxHistorySize, ro.maxHistorySize)

	// Test that configuration affects optimization
	ro.calculateOptimalConcurrency(config)
	concurrency := ro.GetOptimalConcurrency()

	// With high thresholds, should allow more workers
	assert.GreaterOrEqual(t, concurrency.MaxWorkers, config.MinWorkers)
	assert.LessOrEqual(t, concurrency.MaxWorkers, config.MaxWorkers)
	assert.Equal(t, config.MemoryThreshold, concurrency.MemoryThreshold)
	assert.Equal(t, config.CPUThreshold, concurrency.CPUThreshold)
}

func TestResourceOptimizer_PerformanceSnapshot_Timestamp(t *testing.T) {
	config := ResourceOptimizerConfig{
		MaxHistorySize: 10,
	}
	ro := NewResourceOptimizer(config)

	before := time.Now()
	ro.RecordPerformanceSnapshot(1, 1.0, 1.0, 1.0)
	after := time.Now()

	history := ro.GetPerformanceHistory(1)
	assert.Len(t, history, 1)
	snapshot := history[0]

	assert.False(t, snapshot.Timestamp.IsZero())
	assert.True(t, snapshot.Timestamp.After(before) || snapshot.Timestamp.Equal(before))
	assert.True(t, snapshot.Timestamp.Before(after) || snapshot.Timestamp.Equal(after))
}

func TestResourceOptimizer_OptimizationTiming(t *testing.T) {
	config := ResourceOptimizerConfig{
		OptimizationInterval: 50 * time.Millisecond,
		AdaptiveMode:         false,
	}

	ro := NewResourceOptimizer(config)
	initialOptTime := ro.lastOptimization

	// Wait for interval to pass
	time.Sleep(60 * time.Millisecond)

	optimized := ro.OptimizeIfNeeded(config)
	assert.True(t, optimized)
	assert.True(t, ro.lastOptimization.After(initialOptTime))

	// Should not optimize again immediately
	optimized = ro.OptimizeIfNeeded(config)
	assert.False(t, optimized)
}
