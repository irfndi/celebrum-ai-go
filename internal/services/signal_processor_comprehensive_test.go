package services

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/irfandi/celebrum-ai-go/internal/models"
)

// TestNewSignalProcessor tests the NewSignalProcessor constructor
func TestNewSignalProcessor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	// Create a minimal signal processor with nil dependencies
	// This tests the constructor initialization
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)

	require.NotNil(t, sp)
	assert.NotNil(t, sp.config)
	assert.NotNil(t, sp.metrics)
	assert.NotNil(t, sp.logger)
	assert.NotNil(t, sp.ctx)
	assert.NotNil(t, sp.cancel)

	// Verify default config values
	assert.Equal(t, 100, sp.config.BatchSize)
	assert.Equal(t, 5*time.Minute, sp.config.ProcessingInterval)
	assert.Equal(t, 4, sp.config.MaxConcurrentBatch)
	assert.Equal(t, 24*time.Hour, sp.config.SignalTTL)
	assert.Equal(t, 0.7, sp.config.QualityThreshold)
	assert.True(t, sp.config.NotificationEnabled)
	assert.Equal(t, 3, sp.config.RetryAttempts)
	assert.Equal(t, 30*time.Second, sp.config.RetryDelay)

	// Clean up
	sp.cancel()
}

// TestSignalProcessor_IsRunning tests the IsRunning method
func TestSignalProcessor_IsRunning(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)

	// Initially not running
	assert.False(t, sp.IsRunning())

	// Clean up
	sp.cancel()
}

// TestSignalProcessor_GetMetrics tests the GetMetrics method
func TestSignalProcessor_GetMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)

	metrics := sp.GetMetrics()
	require.NotNil(t, metrics)

	// Verify initial metric values
	assert.Equal(t, int64(0), metrics.TotalSignalsProcessed)
	assert.Equal(t, int64(0), metrics.SuccessfulSignals)
	assert.Equal(t, int64(0), metrics.FailedSignals)
	assert.Equal(t, int64(0), metrics.QualityFilteredSignals)
	assert.Equal(t, int64(0), metrics.NotificationsSent)
	assert.Equal(t, float64(0), metrics.AverageProcessingTime)
	assert.Equal(t, float64(0), metrics.ErrorRate)
	assert.Equal(t, float64(0), metrics.ThroughputPerMinute)

	// Clean up
	sp.cancel()
}

// TestSignalProcessor_StopNotRunning tests Stop when processor is not running
func TestSignalProcessor_StopNotRunning(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)

	// Try to stop when not running
	err := sp.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signal processor is not running")

	// Clean up
	sp.cancel()
}

// TestSignalProcessor_IsRetryableError tests the isRetryableError method
func TestSignalProcessor_IsRetryableError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("network error occurred"),
			expected: true,
		},
		{
			name:     "temporary error",
			err:      errors.New("temporary failure"),
			expected: true,
		},
		{
			name:     "database connection error",
			err:      errors.New("database connection reset"),
			expected: true,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "too many requests",
			err:      errors.New("too many requests"),
			expected: true,
		},
		{
			name:     "context cancelled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "validation error",
			err:      errors.New("validation failed"),
			expected: false,
		},
		{
			name:     "invalid input",
			err:      errors.New("invalid input"),
			expected: false,
		},
		{
			name:     "malformed data",
			err:      errors.New("malformed request"),
			expected: false,
		},
		{
			name:     "unknown error",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.isRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSignalProcessor_CalculateMarketVolatility tests the calculateMarketVolatility method
func TestSignalProcessor_CalculateMarketVolatility(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		data     models.MarketData
		expected float64
		minVal   float64
		maxVal   float64
	}{
		{
			name:   "zero price returns default volatility",
			data:   models.MarketData{LastPrice: decimal.Zero},
			minVal: 0.02,
			maxVal: 0.02,
		},
		{
			name: "with bid-ask spread",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(100),
				Bid:       decimal.NewFromFloat(99),
				Ask:       decimal.NewFromFloat(101),
			},
			minVal: 0.005, // Minimum volatility
			maxVal: 0.5,   // Should be reasonable
		},
		{
			name: "with 24h high-low range",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(100),
				High24h:   decimal.NewFromFloat(110),
				Low24h:    decimal.NewFromFloat(90),
			},
			minVal: 0.01,
			maxVal: 0.1,
		},
		{
			name: "only last price, returns default",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(50000),
			},
			minVal: 0.02,
			maxVal: 0.02,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.calculateMarketVolatility(tt.data)
			assert.GreaterOrEqual(t, result, tt.minVal)
			assert.LessOrEqual(t, result, tt.maxVal)
		})
	}
}

// TestSignalProcessor_CalculateMarketTrend tests the calculateMarketTrend method
func TestSignalProcessor_CalculateMarketTrend(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		data     models.MarketData
		expected string
	}{
		{
			name:     "zero last price returns neutral",
			data:     models.MarketData{LastPrice: decimal.Zero},
			expected: "neutral",
		},
		{
			name: "zero high24h returns neutral",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(100),
				High24h:   decimal.Zero,
				Low24h:    decimal.NewFromFloat(90),
			},
			expected: "neutral",
		},
		{
			name: "zero low24h returns neutral",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(100),
				High24h:   decimal.NewFromFloat(110),
				Low24h:    decimal.Zero,
			},
			expected: "neutral",
		},
		{
			name: "price at high = bullish",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(108),
				High24h:   decimal.NewFromFloat(110),
				Low24h:    decimal.NewFromFloat(90),
			},
			expected: "bullish",
		},
		{
			name: "price at low = bearish",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(92),
				High24h:   decimal.NewFromFloat(110),
				Low24h:    decimal.NewFromFloat(90),
			},
			expected: "bearish",
		},
		{
			name: "price in middle = neutral",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(100),
				High24h:   decimal.NewFromFloat(110),
				Low24h:    decimal.NewFromFloat(90),
			},
			expected: "neutral",
		},
		{
			name: "same high and low = neutral",
			data: models.MarketData{
				LastPrice: decimal.NewFromFloat(100),
				High24h:   decimal.NewFromFloat(100),
				Low24h:    decimal.NewFromFloat(100),
			},
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.calculateMarketTrend(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSignalProcessor_FallbackQualityAssessment tests the fallbackQualityAssessment method
func TestSignalProcessor_FallbackQualityAssessment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		signal   *AggregatedSignal
		minVal   float64
		maxVal   float64
	}{
		{
			name: "BUY action with high confidence",
			signal: &AggregatedSignal{
				Action:     "BUY",
				Confidence: decimal.NewFromFloat(0.9),
			},
			minVal: 0.7, // 0.5 + 0.1 (BUY) + 0.2 (high confidence) = 0.8
			maxVal: 0.9,
		},
		{
			name: "SELL action with high confidence",
			signal: &AggregatedSignal{
				Action:     "SELL",
				Confidence: decimal.NewFromFloat(0.85),
			},
			minVal: 0.7,
			maxVal: 0.9,
		},
		{
			name: "HOLD action with medium confidence",
			signal: &AggregatedSignal{
				Action:     "HOLD",
				Confidence: decimal.NewFromFloat(0.6),
			},
			minVal: 0.3,
			maxVal: 0.5,
		},
		{
			name: "BUY action with low confidence",
			signal: &AggregatedSignal{
				Action:     "BUY",
				Confidence: decimal.NewFromFloat(0.3),
			},
			minVal: 0.3,
			maxVal: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.fallbackQualityAssessment(tt.signal)
			assert.GreaterOrEqual(t, result, tt.minVal)
			assert.LessOrEqual(t, result, tt.maxVal)
		})
	}
}

// TestSignalProcessor_ExtractVolumeFromMetadata tests extractVolumeFromMetadata method
func TestSignalProcessor_ExtractVolumeFromMetadata(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected float64
	}{
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: 0.0,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: 0.0,
		},
		{
			name:     "volume as float64",
			metadata: map[string]interface{}{"volume": 1000.5},
			expected: 1000.5,
		},
		{
			name:     "volume as string",
			metadata: map[string]interface{}{"volume": "2500.75"},
			expected: 2500.75,
		},
		{
			name:     "volume as invalid string",
			metadata: map[string]interface{}{"volume": "not-a-number"},
			expected: 0.0,
		},
		{
			name:     "volume as other type",
			metadata: map[string]interface{}{"volume": 123},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.extractVolumeFromMetadata(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSignalProcessor_ExtractIndicatorsFromMetadata tests extractIndicatorsFromMetadata method
func TestSignalProcessor_ExtractIndicatorsFromMetadata(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected map[string]float64
	}{
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: map[string]float64{},
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: map[string]float64{},
		},
		{
			name: "indicators as float64 values",
			metadata: map[string]interface{}{
				"indicators": map[string]interface{}{
					"rsi":  65.5,
					"macd": -0.5,
				},
			},
			expected: map[string]float64{"rsi": 65.5, "macd": -0.5},
		},
		{
			name: "indicators as string values",
			metadata: map[string]interface{}{
				"indicators": map[string]interface{}{
					"sma": "50.25",
					"ema": "48.75",
				},
			},
			expected: map[string]float64{"sma": 50.25, "ema": 48.75},
		},
		{
			name: "mixed indicators",
			metadata: map[string]interface{}{
				"indicators": map[string]interface{}{
					"rsi":     70.0,
					"sma":     "45.5",
					"invalid": "not-a-number",
				},
			},
			expected: map[string]float64{"rsi": 70.0, "sma": 45.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.extractIndicatorsFromMetadata(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSignalProcessor_ExtractSignalComponents tests extractSignalComponents method
func TestSignalProcessor_ExtractSignalComponents(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected []string
	}{
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: nil,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: nil,
		},
		{
			name: "components as string array",
			metadata: map[string]interface{}{
				"components": []interface{}{"RSI", "MACD", "SMA"},
			},
			expected: []string{"RSI", "MACD", "SMA"},
		},
		{
			name: "components with mixed types",
			metadata: map[string]interface{}{
				"components": []interface{}{"RSI", 123, "MACD"},
			},
			expected: []string{"RSI", "MACD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.extractSignalComponents(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSignalProcessor_ExtractSignalCount tests extractSignalCount method
func TestSignalProcessor_ExtractSignalCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected int
	}{
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: 1,
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: 1,
		},
		{
			name:     "signal_count as int",
			metadata: map[string]interface{}{"signal_count": 5},
			expected: 5,
		},
		{
			name:     "signal_count as float64",
			metadata: map[string]interface{}{"signal_count": 7.0},
			expected: 7,
		},
		{
			name:     "signal_count as other type",
			metadata: map[string]interface{}{"signal_count": "5"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sp.extractSignalCount(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSignalProcessor_ConvertToMarketDataSnapshot tests convertToMarketDataSnapshot method
func TestSignalProcessor_ConvertToMarketDataSnapshot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	t.Run("nil data returns nil", func(t *testing.T) {
		result := sp.convertToMarketDataSnapshot(nil)
		assert.Nil(t, result)
	})

	t.Run("valid data returns snapshot", func(t *testing.T) {
		now := time.Now()
		data := &models.MarketData{
			LastPrice: decimal.NewFromFloat(50000),
			Volume24h: decimal.NewFromFloat(1000000),
			Change24h: decimal.NewFromFloat(2.5),
			Ask:       decimal.NewFromFloat(50100),
			Bid:       decimal.NewFromFloat(49900),
			Timestamp: now,
		}

		result := sp.convertToMarketDataSnapshot(data)

		require.NotNil(t, result)
		assert.Equal(t, data.LastPrice, result.Price)
		assert.Equal(t, data.Volume24h, result.Volume24h)
		assert.Equal(t, data.Change24h, result.PriceChange24h)
		assert.Equal(t, data.Timestamp, result.LastTradeTime)
		// Spread should be Ask - Bid = 200
		assert.True(t, result.Spread.Equal(decimal.NewFromFloat(200)))
	})
}

// TestSignalProcessor_DeduplicateResults tests deduplicateResults method
func TestSignalProcessor_DeduplicateResults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	results := []ProcessingResult{
		{SignalID: "1", Symbol: "BTC/USDT", SignalType: SignalTypeArbitrage},
		{SignalID: "2", Symbol: "BTC/USDT", SignalType: SignalTypeArbitrage}, // Duplicate
		{SignalID: "3", Symbol: "ETH/USDT", SignalType: SignalTypeArbitrage},
		{SignalID: "4", Symbol: "BTC/USDT", SignalType: SignalTypeTechnical}, // Different type
		{SignalID: "5", Symbol: "ETH/USDT", SignalType: SignalTypeTechnical},
	}

	deduplicated := sp.deduplicateResults(results)

	// Should have 4 unique combinations (BTC/ARB, ETH/ARB, BTC/TECH, ETH/TECH)
	assert.Len(t, deduplicated, 4)

	// Verify first occurrence is kept
	assert.Equal(t, "1", deduplicated[0].SignalID)
}

// TestSignalProcessor_ApplyBatchQualityFiltering tests applyBatchQualityFiltering method
func TestSignalProcessor_ApplyBatchQualityFiltering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	results := []ProcessingResult{
		{SignalID: "1", QualityScore: 0.9, Error: nil},  // Pass
		{SignalID: "2", QualityScore: 0.5, Error: nil},  // Fail quality
		{SignalID: "3", QualityScore: 0.8, Error: nil},  // Pass
		{SignalID: "4", QualityScore: 0.7, Error: nil},  // Pass (at threshold)
		{SignalID: "5", QualityScore: 0.3, Error: errors.New("test error")}, // Pass (errors pass through)
	}

	filtered := sp.applyBatchQualityFiltering(results)

	// Should have 4 results (3 passed quality + 1 error)
	assert.Len(t, filtered, 4)
}

// TestSignalProcessor_PassesRateLimiting tests passesRateLimiting method
func TestSignalProcessor_PassesRateLimiting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	// First call should pass
	result1 := ProcessingResult{Symbol: "BTC/USDT", SignalType: SignalTypeArbitrage}
	assert.True(t, sp.passesRateLimiting(result1))

	// Immediate second call with same symbol/type should fail
	result2 := ProcessingResult{Symbol: "BTC/USDT", SignalType: SignalTypeArbitrage}
	assert.False(t, sp.passesRateLimiting(result2))

	// Different symbol should pass
	result3 := ProcessingResult{Symbol: "ETH/USDT", SignalType: SignalTypeArbitrage}
	assert.True(t, sp.passesRateLimiting(result3))

	// Same symbol but different type should pass
	result4 := ProcessingResult{Symbol: "BTC/USDT", SignalType: SignalTypeTechnical}
	assert.True(t, sp.passesRateLimiting(result4))
}

// TestSignalProcessor_CountSuccessful tests countSuccessful method
func TestSignalProcessor_CountSuccessful(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	results := []ProcessingResult{
		{SignalID: "1", Error: nil},
		{SignalID: "2", Error: errors.New("test error")},
		{SignalID: "3", Error: nil},
		{SignalID: "4", Error: errors.New("another error")},
		{SignalID: "5", Error: nil},
	}

	count := sp.countSuccessful(results)
	assert.Equal(t, 3, count)
}

// TestSignalProcessor_IncrementErrorCount tests incrementErrorCount method
func TestSignalProcessor_IncrementErrorCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	assert.Equal(t, 0, sp.errorCount)

	sp.incrementErrorCount()
	assert.Equal(t, 1, sp.errorCount)

	sp.incrementErrorCount()
	assert.Equal(t, 2, sp.errorCount)
}

// TestSignalProcessor_UpdateMetrics tests updateMetrics method
func TestSignalProcessor_UpdateMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	sp := NewSignalProcessor(nil, logger, nil, nil, nil, nil, nil, nil)
	defer sp.cancel()

	results := []ProcessingResult{
		{SignalID: "1", Error: nil, QualityScore: 0.9, NotificationSent: true},
		{SignalID: "2", Error: errors.New("test error"), QualityScore: 0.5},
		{SignalID: "3", Error: nil, QualityScore: 0.8, NotificationSent: true},
		{SignalID: "4", Error: nil, QualityScore: 0.5}, // Below threshold
	}

	processingTime := 100 * time.Millisecond
	sp.updateMetrics(results, processingTime)

	metrics := sp.GetMetrics()
	assert.Equal(t, int64(4), metrics.TotalSignalsProcessed)
	assert.Equal(t, int64(3), metrics.SuccessfulSignals)
	assert.Equal(t, int64(1), metrics.FailedSignals)
	assert.Equal(t, int64(2), metrics.QualityFilteredSignals)
	assert.Equal(t, int64(2), metrics.NotificationsSent)
	assert.NotZero(t, metrics.AverageProcessingTime)
}

// TestGetDefaultSignalProcessorConfig tests GetDefaultSignalProcessorConfig function
func TestGetDefaultSignalProcessorConfig(t *testing.T) {
	config := GetDefaultSignalProcessorConfig()

	require.NotNil(t, config)
	assert.Equal(t, 5*time.Minute, config.ProcessingInterval)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 4, config.MaxConcurrentBatch)
	assert.Equal(t, 24*time.Hour, config.SignalTTL)
	assert.Equal(t, 0.7, config.QualityThreshold)
	assert.True(t, config.NotificationEnabled)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 30*time.Second, config.RetryDelay)
}

// TestProcessingResult_Initialization tests ProcessingResult struct initialization
func TestProcessingResult_Initialization(t *testing.T) {
	result := ProcessingResult{
		SignalID:         "test-signal-123",
		SignalType:       SignalTypeArbitrage,
		Symbol:           "BTC/USDT",
		Processed:        true,
		QualityScore:     0.85,
		NotificationSent: false,
		Error:            nil,
		ProcessingTime:   time.Second * 2,
		Metadata:         map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test-signal-123", result.SignalID)
	assert.Equal(t, SignalTypeArbitrage, result.SignalType)
	assert.Equal(t, "BTC/USDT", result.Symbol)
	assert.True(t, result.Processed)
	assert.Equal(t, 0.85, result.QualityScore)
	assert.False(t, result.NotificationSent)
	assert.Nil(t, result.Error)
	assert.Equal(t, time.Second*2, result.ProcessingTime)
	assert.Equal(t, "value", result.Metadata["key"])
}
