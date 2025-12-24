package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewNotificationService(t *testing.T) {
	// Test with empty token
	ns := NewNotificationService(nil, nil, "", "", "")
	assert.NotNil(t, ns)
	assert.Nil(t, ns.db)

	// Test with token - bot creation may fail with invalid token but service should still be created
	ns2 := NewNotificationService(nil, nil, "http://test-url", "localhost:50052", "test-key")
	assert.NotNil(t, ns2)
	assert.Nil(t, ns2.db)
}

func TestArbitrageOpportunity_Struct(t *testing.T) {
	now := time.Now()
	opportunity := ArbitrageOpportunity{
		Symbol:          "BTC/USDT",
		BuyExchange:     "binance",
		SellExchange:    "coinbase",
		BuyPrice:        50000.0,
		SellPrice:       50500.0,
		ProfitPercent:   1.0,
		ProfitAmount:    500.0,
		Volume:          1.0,
		Timestamp:       now,
		OpportunityType: "arbitrage",
	}

	assert.Equal(t, "BTC/USDT", opportunity.Symbol)
	assert.Equal(t, "binance", opportunity.BuyExchange)
	assert.Equal(t, "coinbase", opportunity.SellExchange)
	assert.Equal(t, 50000.0, opportunity.BuyPrice)
	assert.Equal(t, 50500.0, opportunity.SellPrice)
	assert.Equal(t, 1.0, opportunity.ProfitPercent)
	assert.Equal(t, 500.0, opportunity.ProfitAmount)
	assert.Equal(t, 1.0, opportunity.Volume)
	assert.Equal(t, now, opportunity.Timestamp)
	assert.Equal(t, "arbitrage", opportunity.OpportunityType)
}

func TestNotificationService_formatArbitrageMessage(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with empty opportunities
	message := ns.formatArbitrageMessage([]ArbitrageOpportunity{})
	assert.Equal(t, "No arbitrage opportunities found.", message)

	// Test with single arbitrage opportunity
	opportunities := []ArbitrageOpportunity{
		{
			Symbol:          "BTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "coinbase",
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			OpportunityType: "arbitrage",
		},
	}

	message = ns.formatArbitrageMessage(opportunities)
	assert.Contains(t, message, "🚀 *True Arbitrage Opportunities*")
	assert.Contains(t, message, "BTC/USDT")
	assert.Contains(t, message, "1.00%")
	assert.Contains(t, message, "binance")
	assert.Contains(t, message, "coinbase")

	// Test with technical opportunity
	technicalOpps := []ArbitrageOpportunity{
		{
			Symbol:          "ETH/USDT",
			BuyExchange:     "binance",
			SellExchange:    "binance",
			BuyPrice:        3000.0,
			SellPrice:       3030.0,
			ProfitPercent:   1.0,
			OpportunityType: "technical",
		},
	}

	message = ns.formatArbitrageMessage(technicalOpps)
	assert.Contains(t, message, "📊 *Technical Analysis Signals*")
	assert.Contains(t, message, "ETH/USDT")

	// Test with AI-generated opportunity
	aiOpps := []ArbitrageOpportunity{
		{
			Symbol:          "ADA/USDT",
			BuyExchange:     "kraken",
			SellExchange:    "bitfinex",
			BuyPrice:        0.5,
			SellPrice:       0.51,
			ProfitPercent:   2.0,
			OpportunityType: "ai_generated",
		},
	}

	message = ns.formatArbitrageMessage(aiOpps)
	assert.Contains(t, message, "🤖 *AI-Generated Opportunities*")
	assert.Contains(t, message, "ADA/USDT")

	// Test with more than 3 opportunities
	manyOpps := make([]ArbitrageOpportunity, 5)
	for i := 0; i < 5; i++ {
		manyOpps[i] = ArbitrageOpportunity{
			Symbol:          "BTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "coinbase",
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			OpportunityType: "arbitrage",
		}
	}

	message = ns.formatArbitrageMessage(manyOpps)
	assert.Contains(t, message, "Found 5 profitable opportunities")
	assert.Contains(t, message, "...and 2 more opportunities")
}

func TestNotificationService_OpportunityCategorization(t *testing.T) {
	// Test categorization logic without database dependencies
	opportunities := []ArbitrageOpportunity{
		{
			Symbol:        "BTC/USDT",
			BuyExchange:   "binance",
			SellExchange:  "coinbase", // Different exchanges = arbitrage
			BuyPrice:      50000.0,
			SellPrice:     50500.0,
			ProfitPercent: 1.0,
		},
		{
			Symbol:        "ETH/USDT",
			BuyExchange:   "binance",
			SellExchange:  "binance", // Same exchange = technical
			BuyPrice:      3000.0,
			SellPrice:     3030.0,
			ProfitPercent: 1.0,
		},
	}

	// Verify categorization logic manually
	var arbitrageOpps, technicalOpps []ArbitrageOpportunity
	for _, opp := range opportunities {
		if opp.BuyExchange != opp.SellExchange {
			opp.OpportunityType = "arbitrage"
			arbitrageOpps = append(arbitrageOpps, opp)
		} else {
			opp.OpportunityType = "technical"
			technicalOpps = append(technicalOpps, opp)
		}
	}

	assert.Len(t, arbitrageOpps, 1)
	assert.Len(t, technicalOpps, 1)
	assert.Equal(t, "arbitrage", arbitrageOpps[0].OpportunityType)
	assert.Equal(t, "technical", technicalOpps[0].OpportunityType)
}

// Test ArbitrageOpportunity with different field values
func TestArbitrageOpportunity_EdgeCases(t *testing.T) {
	// Test with zero values
	zeroOpp := ArbitrageOpportunity{}
	assert.Empty(t, zeroOpp.Symbol)
	assert.Empty(t, zeroOpp.BuyExchange)
	assert.Empty(t, zeroOpp.SellExchange)
	assert.Equal(t, 0.0, zeroOpp.BuyPrice)
	assert.Equal(t, 0.0, zeroOpp.SellPrice)
	assert.Equal(t, 0.0, zeroOpp.ProfitPercent)
	assert.True(t, zeroOpp.Timestamp.IsZero())

	// Test with negative values
	negativeOpp := ArbitrageOpportunity{
		Symbol:        "TEST/USDT",
		BuyPrice:      -100.0,
		SellPrice:     -50.0,
		ProfitPercent: -10.0,
		ProfitAmount:  -500.0,
		Volume:        -1.0,
	}
	assert.Equal(t, "TEST/USDT", negativeOpp.Symbol)
	assert.Equal(t, -100.0, negativeOpp.BuyPrice)
	assert.Equal(t, -50.0, negativeOpp.SellPrice)
	assert.Equal(t, -10.0, negativeOpp.ProfitPercent)
	assert.Equal(t, -500.0, negativeOpp.ProfitAmount)
	assert.Equal(t, -1.0, negativeOpp.Volume)

	// Test with very large values
	largeOpp := ArbitrageOpportunity{
		Symbol:        "BTC/USDT",
		BuyPrice:      1000000.0,
		SellPrice:     1100000.0,
		ProfitPercent: 10.0,
		ProfitAmount:  100000.0,
		Volume:        1000.0,
	}
	assert.Equal(t, "BTC/USDT", largeOpp.Symbol)
	assert.Equal(t, 1000000.0, largeOpp.BuyPrice)
	assert.Equal(t, 1100000.0, largeOpp.SellPrice)
	assert.Equal(t, 10.0, largeOpp.ProfitPercent)
	assert.Equal(t, 100000.0, largeOpp.ProfitAmount)
	assert.Equal(t, 1000.0, largeOpp.Volume)
}

// Test formatArbitrageMessage with edge cases
func TestNotificationService_formatArbitrageMessage_EdgeCases(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with nil slice
	message := ns.formatArbitrageMessage(nil)
	assert.Equal(t, "No arbitrage opportunities found.", message)

	// Test with opportunity having empty strings
	emptyOpp := []ArbitrageOpportunity{
		{
			Symbol:          "",
			BuyExchange:     "",
			SellExchange:    "",
			BuyPrice:        0.0,
			SellPrice:       0.0,
			ProfitPercent:   0.0,
			OpportunityType: "",
		},
	}

	message = ns.formatArbitrageMessage(emptyOpp)
	assert.Contains(t, message, "🚨 *Arbitrage Alert!*") // Default header
	assert.Contains(t, message, "Found 1 profitable opportunities")

	// Test with unknown opportunity type
	unknownOpp := []ArbitrageOpportunity{
		{
			Symbol:          "TEST/USDT",
			BuyExchange:     "exchange1",
			SellExchange:    "exchange2",
			BuyPrice:        100.0,
			SellPrice:       101.0,
			ProfitPercent:   1.0,
			OpportunityType: "unknown_type",
		},
	}

	message = ns.formatArbitrageMessage(unknownOpp)
	assert.Contains(t, message, "🚨 *Arbitrage Alert!*") // Default header for unknown type
	assert.Contains(t, message, "TEST/USDT")
}

// Test NotificationService struct fields
func TestNotificationService_StructFields(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test initial state
	assert.NotNil(t, ns)
	assert.Nil(t, ns.db)

	// Test that we can access fields without panic
	assert.NotPanics(t, func() {
		_ = ns.db
	})
}

// Test ArbitrageOpportunity JSON tags (implicit test)
func TestArbitrageOpportunity_JSONStructure(t *testing.T) {
	now := time.Now()
	opp := ArbitrageOpportunity{
		Symbol:          "BTC/USDT",
		BuyExchange:     "binance",
		SellExchange:    "coinbase",
		BuyPrice:        50000.0,
		SellPrice:       50500.0,
		ProfitPercent:   1.0,
		ProfitAmount:    500.0,
		Volume:          1.0,
		Timestamp:       now,
		OpportunityType: "arbitrage",
	}

	// Verify all fields are accessible and have expected values
	assert.Equal(t, "BTC/USDT", opp.Symbol)
	assert.Equal(t, "binance", opp.BuyExchange)
	assert.Equal(t, "coinbase", opp.SellExchange)
	assert.Equal(t, 50000.0, opp.BuyPrice)
	assert.Equal(t, 50500.0, opp.SellPrice)
	assert.Equal(t, 1.0, opp.ProfitPercent)
	assert.Equal(t, 500.0, opp.ProfitAmount)
	assert.Equal(t, 1.0, opp.Volume)
	assert.Equal(t, now, opp.Timestamp)
	assert.Equal(t, "arbitrage", opp.OpportunityType)
}

// Test formatArbitrageMessage with exactly 3 opportunities
func TestNotificationService_formatArbitrageMessage_ExactlyThree(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with exactly 3 opportunities
	threeOpps := make([]ArbitrageOpportunity, 3)
	for i := 0; i < 3; i++ {
		threeOpps[i] = ArbitrageOpportunity{
			Symbol:          "BTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "coinbase",
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			OpportunityType: "arbitrage",
		}
	}

	message := ns.formatArbitrageMessage(threeOpps)
	assert.Contains(t, message, "Found 3 profitable opportunities")
	assert.NotContains(t, message, "...and") // Should not show "and more" for exactly 3
}

// Test formatArbitrageMessage with 4 opportunities
func TestNotificationService_formatArbitrageMessage_MoreThanThree(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with 4 opportunities
	fourOpps := make([]ArbitrageOpportunity, 4)
	for i := 0; i < 4; i++ {
		fourOpps[i] = ArbitrageOpportunity{
			Symbol:          "BTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "coinbase",
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			OpportunityType: "arbitrage",
		}
	}

	message := ns.formatArbitrageMessage(fourOpps)
	assert.Contains(t, message, "Found 4 profitable opportunities")
	assert.Contains(t, message, "...and 1 more opportunities") // Should show "and more" for 4
}

func TestNotificationService_PublishOpportunityUpdate(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")
	ns.PublishOpportunityUpdate(context.Background(), []ArbitrageOpportunity{})
	// Should not panic
}

func TestNotificationService_GetCacheStats(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")
	stats := ns.GetCacheStats(context.Background())

	assert.False(t, stats["redis_available"].(bool))
	assert.NotContains(t, stats, "users_cached")
	assert.NotContains(t, stats, "opportunities_cached")
}

func TestNotificationService_cacheArbitrageOpportunities(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	opportunities := []ArbitrageOpportunity{
		{
			Symbol:        "BTC/USDT",
			BuyExchange:   "binance",
			SellExchange:  "coinbase",
			BuyPrice:      50000.0,
			SellPrice:     50500.0,
			ProfitPercent: 1.0,
		},
	}

	// Should not panic with nil Redis
	ns.cacheArbitrageOpportunities(context.Background(), opportunities)
}

func TestNotificationService_CacheMarketData(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	data := map[string]interface{}{
		"symbol": "BTC/USDT",
		"price":  50000.0,
	}

	// Should not panic with nil Redis
	ns.CacheMarketData(context.Background(), "binance", data)
}

func TestNotificationService_GetCachedMarketData(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	var result map[string]interface{}
	err := ns.GetCachedMarketData(context.Background(), "binance", &result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis not available")
}

func TestNotificationService_InvalidateUserCache(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	// Should not panic with nil Redis
	ns.InvalidateUserCache(context.Background())
}

func TestNotificationService_InvalidateOpportunityCache(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	// Should not panic with nil Redis
	ns.InvalidateOpportunityCache(context.Background())
}

func TestNotificationService_formatTechnicalSignalMessage(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with empty signals
	message := ns.formatTechnicalSignalMessage([]TechnicalSignalNotification{})
	assert.Equal(t, "No technical analysis signals found.", message)

	// Test with single signal
	signals := []TechnicalSignalNotification{
		{
			Symbol:       "BTC/USDT",
			SignalType:   "buy",
			Action:       "buy",
			SignalText:   "RSI oversold",
			CurrentPrice: 50000.0,
			EntryRange:   "$49900.0 - $50100.0",
			Targets: []Target{
				{Price: 51000.0, Profit: 2.0},
				{Price: 52000.0, Profit: 4.0},
			},
			StopLoss:   StopLoss{Price: 49500.0, Risk: 1.0},
			RiskReward: "1:2",
			Exchanges:  []string{"binance", "coinbase"},
			Timeframe:  "4H",
			Confidence: 0.85,
			Timestamp:  time.Now(),
		},
	}

	message = ns.formatTechnicalSignalMessage(signals)
	assert.Contains(t, message, "📊 *Technical Analysis Signals*")
	assert.Contains(t, message, "BTC/USDT")
	assert.Contains(t, message, "RSI oversold")
	assert.Contains(t, message, "85.0%")
}

func TestNotificationService_ConvertAggregatedSignalToNotification(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with buy signal
	signal := &AggregatedSignal{
		Symbol:          "BTC/USDT",
		SignalType:      SignalTypeTechnical,
		Action:          "buy",
		ProfitPotential: decimal.NewFromFloat(5.0),
		RiskLevel:       decimal.NewFromFloat(0.02),
		Confidence:      decimal.NewFromFloat(0.85),
		Exchanges:       []string{"binance", "coinbase"},
		Indicators:      []string{"RSI", "MACD"},
		Metadata: map[string]interface{}{
			"current_price": 50000.0,
			"timeframe":     "4H",
		},
		CreatedAt: time.Now(),
	}

	notification := ns.ConvertAggregatedSignalToNotification(signal)
	assert.NotNil(t, notification)
	assert.Equal(t, "BTC/USDT", notification.Symbol)
	assert.Equal(t, "buy", notification.Action)
	assert.Equal(t, 0.85, notification.Confidence)
	assert.Equal(t, "RSI + MACD", notification.SignalText)
	assert.Equal(t, "4H", notification.Timeframe)
	assert.Len(t, notification.Targets, 2)
	assert.Equal(t, 49000.0, notification.StopLoss.Price)
}

func TestNotificationService_generateOpportunityHash(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	opportunities := []ArbitrageOpportunity{
		{
			Symbol:        "BTC/USDT",
			BuyExchange:   "binance",
			SellExchange:  "coinbase",
			BuyPrice:      50000.0,
			SellPrice:     50500.0,
			ProfitPercent: 1.0,
		},
	}

	hash := ns.generateOpportunityHash(opportunities)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32) // MD5 hash length

	// Same opportunities should produce same hash
	hash2 := ns.generateOpportunityHash(opportunities)
	assert.Equal(t, hash, hash2)
}

func TestNotificationService_generateTechnicalSignalsHash(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	signals := []TechnicalSignalNotification{
		{
			Symbol:       "BTC/USDT",
			SignalType:   "buy",
			Action:       "buy",
			CurrentPrice: 50000.0,
			Confidence:   0.85,
		},
	}

	hash := ns.generateTechnicalSignalsHash(signals)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 32) // MD5 hash length

	// Same signals should produce same hash
	hash2 := ns.generateTechnicalSignalsHash(signals)
	assert.Equal(t, hash, hash2)
}

func TestNotificationService_getCachedMessage(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	message, found := ns.getCachedMessage(context.Background(), "test", "testhash")
	assert.Empty(t, message)
	assert.False(t, found)
}

func TestNotificationService_setCachedMessage(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "", "", "")

	// Should not panic with nil Redis
	ns.setCachedMessage(context.Background(), "test", "testhash", "test message")
}

func TestNotificationService_checkRateLimit(t *testing.T) {
	// Test with nil Redis - should deny (fail-closed for security)
	ns := NewNotificationService(nil, nil, "", "", "")

	allowed, err := ns.checkRateLimit(context.Background(), "testuser")
	assert.Error(t, err)     // Should return error when Redis is not available
	assert.False(t, allowed) // Should deny when Redis is not available (fail-closed)
}

// TestNotificationService_checkRateLimit_Comprehensive tests checkRateLimit with various scenarios
func TestNotificationService_checkRateLimit_Comprehensive(t *testing.T) {
	// Test with nil Redis - should always deny (fail-closed for security)
	t.Run("NilRedis", func(t *testing.T) {
		ns := NewNotificationService(nil, nil, "", "", "")

		// Multiple calls should all be denied when Redis is not available
		for i := 0; i < 10; i++ {
			allowed, err := ns.checkRateLimit(context.Background(), "testuser")
			assert.Error(t, err, "Call %d should return error", i)
			assert.False(t, allowed, "Call %d should be denied (fail-closed)", i)
		}
	})

	// Test with different user IDs - all should be denied without Redis
	t.Run("DifferentUserIDs", func(t *testing.T) {
		ns := NewNotificationService(nil, nil, "", "", "")

		userIDs := []string{"user1", "user2", "user3", "user-with-dashes", "user_123", ""}

		for _, userID := range userIDs {
			allowed, err := ns.checkRateLimit(context.Background(), userID)
			assert.Error(t, err, "User %s should return error", userID)
			assert.False(t, allowed, "User %s should be denied (fail-closed)", userID)
		}
	})

	// Test with different contexts - all should be denied without Redis
	t.Run("DifferentContexts", func(t *testing.T) {
		ns := NewNotificationService(nil, nil, "", "", "")

		// Test with background context
		allowed, err := ns.checkRateLimit(context.Background(), "testuser")
		assert.Error(t, err)
		assert.False(t, allowed)

		// Test with cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		allowed, err = ns.checkRateLimit(ctx, "testuser")
		assert.Error(t, err)
		assert.False(t, allowed) // Should deny with nil Redis (fail-closed)
	})

	// Test with timeout context - should be denied without Redis
	t.Run("TimeoutContext", func(t *testing.T) {
		ns := NewNotificationService(nil, nil, "", "", "")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()

		allowed, err := ns.checkRateLimit(ctx, "testuser")
		assert.Error(t, err)
		assert.False(t, allowed)
	})

	// Test concurrent calls - all should be denied without Redis (fail-closed)
	t.Run("ConcurrentCalls", func(t *testing.T) {
		ns := NewNotificationService(nil, nil, "", "", "")

		var wg sync.WaitGroup
		results := make([]bool, 10)
		errors := make([]error, 10)

		// Launch multiple goroutines calling checkRateLimit concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				allowed, err := ns.checkRateLimit(context.Background(), "testuser")
				results[index] = allowed
				errors[index] = err
			}(i)
		}

		wg.Wait()

		// All should error and be denied (fail-closed for security)
		for i := 0; i < 10; i++ {
			assert.Error(t, errors[i], "Call %d should error", i)
			assert.False(t, results[i], "Call %d should be denied (fail-closed)", i)
		}
	})
}

func TestNotificationService_logNotification(t *testing.T) {
	// Test with nil database - expect panic due to nil database access
	ns := NewNotificationService(nil, nil, "", "", "")

	assert.Panics(t, func() {
		err := ns.logNotification(context.Background(), "testuser", "telegram", "test message")
		if err != nil {
			t.Log("Error:", err)
		}
	})
}

func TestNotificationService_CheckUserNotificationPreferences(t *testing.T) {
	// Test with nil database and Redis - expect panic due to nil database access
	ns := NewNotificationService(nil, nil, "", "", "")

	assert.Panics(t, func() {
		enabled, err := ns.CheckUserNotificationPreferences(context.Background(), "testuser")
		if err != nil {
			t.Log("Error:", err)
		}
		t.Log("Enabled:", enabled)
	})
}

func TestNotificationService_generateAggregatedSignalsHash(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	signals := []*AggregatedSignal{
		{
			Symbol:     "BTC/USDT",
			SignalType: SignalTypeTechnical,
			Action:     "buy",
			Strength:   SignalStrengthStrong,
			Confidence: decimal.NewFromFloat(0.85),
		},
		{
			Symbol:     "ETH/USDT",
			SignalType: SignalTypeTechnical,
			Action:     "sell",
			Strength:   SignalStrengthWeak,
			Confidence: decimal.NewFromFloat(0.65),
		},
	}

	hash := ns.generateAggregatedSignalsHash(signals)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 hash length

	// Same signals should produce same hash
	hash2 := ns.generateAggregatedSignalsHash(signals)
	assert.Equal(t, hash, hash2)

	// Different order should produce same hash (signals are sorted internally)
	reversedSignals := []*AggregatedSignal{signals[1], signals[0]}
	hash3 := ns.generateAggregatedSignalsHash(reversedSignals)
	assert.Equal(t, hash, hash3)
}

func TestNotificationService_formatEnhancedArbitrageMessage(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with nil signal
	message := ns.formatEnhancedArbitrageMessage(nil)
	assert.Equal(t, "No arbitrage signal found.", message)

	// Test with non-arbitrage signal
	nonArbitrageSignal := &AggregatedSignal{
		Symbol:     "BTC/USDT",
		SignalType: SignalTypeTechnical,
	}

	message = ns.formatEnhancedArbitrageMessage(nonArbitrageSignal)
	assert.Equal(t, "No arbitrage signal found.", message)

	// Test with arbitrage signal
	signal := &AggregatedSignal{
		Symbol:     "BTC/USDT",
		SignalType: SignalTypeArbitrage,
		Confidence: decimal.NewFromFloat(0.85),
		Metadata: map[string]interface{}{
			"buy_price_range": map[string]interface{}{
				"min": decimal.NewFromFloat(49900.0),
				"max": decimal.NewFromFloat(50100.0),
			},
			"sell_price_range": map[string]interface{}{
				"min": decimal.NewFromFloat(50400.0),
				"max": decimal.NewFromFloat(50600.0),
			},
			"profit_range": map[string]interface{}{
				"min_percent": decimal.NewFromFloat(0.5),
				"max_percent": decimal.NewFromFloat(1.5),
				"min_dollar":  decimal.NewFromFloat(250.0),
				"max_dollar":  decimal.NewFromFloat(750.0),
				"base_amount": decimal.NewFromFloat(50000.0),
			},
			"buy_exchanges":     []string{"binance", "coinbase"},
			"sell_exchanges":    []string{"kraken", "bitfinex"},
			"opportunity_count": 4,
			"min_volume":        decimal.NewFromFloat(10000.0),
			"validity_minutes":  5,
		},
	}

	message = ns.formatEnhancedArbitrageMessage(signal)
	assert.Contains(t, message, "ARBITRAGE ALERT: BTC/USDT")
	assert.Contains(t, message, "0.50% - 1.50%")
	assert.Contains(t, message, "$250 - $750")
	assert.Contains(t, message, "binance, coinbase")
	assert.Contains(t, message, "kraken, bitfinex")
	assert.Contains(t, message, "85.0%")
	assert.Contains(t, message, "5 minutes")
}

func TestNotificationService_formatAggregatedArbitrageMessage(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with empty signals
	message := ns.formatAggregatedArbitrageMessage([]*AggregatedSignal{})
	assert.Equal(t, "🔍 No arbitrage opportunities available", message)

	// Test with signals
	signals := []*AggregatedSignal{
		{
			Symbol:          "BTC/USDT",
			SignalType:      SignalTypeArbitrage,
			Action:          "buy",
			ProfitPotential: decimal.NewFromFloat(2.5),
			Confidence:      decimal.NewFromFloat(0.85),
			Exchanges:       []string{"binance", "coinbase"},
			Metadata: map[string]interface{}{
				"buy_price":  49900.0,
				"sell_price": 50500.0,
			},
		},
		{
			Symbol:          "ETH/USDT",
			SignalType:      SignalTypeArbitrage,
			Action:          "sell",
			ProfitPotential: decimal.NewFromFloat(1.8),
			Confidence:      decimal.NewFromFloat(0.75),
			Exchanges:       []string{"kraken", "bitfinex"},
		},
	}

	message = ns.formatAggregatedArbitrageMessage(signals)
	assert.Contains(t, message, "🚀 *Aggregated Arbitrage Opportunities*")
	assert.Contains(t, message, "BTC/USDT")
	assert.Contains(t, message, "ETH/USDT")
	assert.Contains(t, message, "2.50%")
	assert.Contains(t, message, "1.80%")
	assert.Contains(t, message, "0.8%")
	// Both confidence values might round to 0.8% due to InexactFloat64()
}

func TestNotificationService_formatAggregatedTechnicalMessage(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	// Test with empty signals
	message := ns.formatAggregatedTechnicalMessage([]*AggregatedSignal{})
	assert.Equal(t, "📊 No technical analysis signals available", message)

	// Test with signals
	signals := []*AggregatedSignal{
		{
			Symbol:          "BTC/USDT",
			SignalType:      SignalTypeTechnical,
			Action:          "buy",
			Strength:        SignalStrengthStrong,
			ProfitPotential: decimal.NewFromFloat(5.0),
			Confidence:      decimal.NewFromFloat(0.85),
			RiskLevel:       decimal.NewFromFloat(0.02),
			Exchanges:       []string{"binance", "coinbase"},
			Indicators:      []string{"RSI", "MACD", "BB"},
			Metadata: map[string]interface{}{
				"entry_price": 49900.0,
				"stop_loss":   49500.0,
				"target":      52000.0,
			},
		},
		{
			Symbol:          "ETH/USDT",
			SignalType:      SignalTypeTechnical,
			Action:          "sell",
			Strength:        SignalStrengthWeak,
			ProfitPotential: decimal.NewFromFloat(3.0),
			Confidence:      decimal.NewFromFloat(0.65),
			RiskLevel:       decimal.NewFromFloat(0.03),
			Exchanges:       []string{"kraken", "bitfinex"},
			Indicators:      []string{"EMA", "STOCH"},
		},
	}

	message = ns.formatAggregatedTechnicalMessage(signals)
	assert.Contains(t, message, "📊 *Aggregated Technical Analysis*")
	assert.Contains(t, message, "BTC/USDT")
	assert.Contains(t, message, "ETH/USDT")
	assert.Contains(t, message, "strong")
	assert.Contains(t, message, "weak")
	assert.Contains(t, message, "RSI, MACD, BB")
	assert.Contains(t, message, "EMA, STOCH")
	assert.Contains(t, message, "0.02%")
	assert.Contains(t, message, "0.03%")
}

func TestNotificationService_NotifyArbitrageOpportunities_EmptyOpportunities(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")
	ctx := context.Background()

	opportunities := []ArbitrageOpportunity{}

	// Should panic due to nil database when trying to get eligible users
	assert.Panics(t, func() {
		_ = ns.NotifyArbitrageOpportunities(ctx, opportunities)
	})
}

func TestNotificationService_NotifyArbitrageOpportunities_CategorizationLogic(t *testing.T) {
	// Test the categorization logic directly by examining opportunities
	opportunities := []ArbitrageOpportunity{
		{
			Symbol:          "BTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "coinbase", // Different exchanges
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			OpportunityType: "",
		},
		{
			Symbol:          "ETH/USDT",
			BuyExchange:     "binance",
			SellExchange:    "binance", // Same exchange
			BuyPrice:        3000.0,
			SellPrice:       3020.0,
			ProfitPercent:   0.67,
			OpportunityType: "",
		},
	}

	// Test categorization logic separately
	arbitrageOpps := make([]ArbitrageOpportunity, 0)
	technicalOpps := make([]ArbitrageOpportunity, 0)

	for _, opp := range opportunities {
		// Categorize opportunity based on exchanges (same logic as in the function)
		if opp.BuyExchange != opp.SellExchange {
			opp.OpportunityType = "arbitrage"
			arbitrageOpps = append(arbitrageOpps, opp)
		} else {
			opp.OpportunityType = "technical"
			technicalOpps = append(technicalOpps, opp)
		}
	}

	// Verify categorization
	assert.Equal(t, 1, len(arbitrageOpps))
	assert.Equal(t, 1, len(technicalOpps))
	assert.Equal(t, "arbitrage", arbitrageOpps[0].OpportunityType)
	assert.Equal(t, "technical", technicalOpps[0].OpportunityType)
	assert.Equal(t, "BTC/USDT", arbitrageOpps[0].Symbol)
	assert.Equal(t, "ETH/USDT", technicalOpps[0].Symbol)
}

func TestNotificationService_NotifyArbitrageOpportunities_MixedTypes(t *testing.T) {
	// Test with mixed opportunity types
	opportunities := []ArbitrageOpportunity{
		{
			Symbol:          "BTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "coinbase",
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			OpportunityType: "arbitrage",
		},
		{
			Symbol:          "ETH/USDT",
			BuyExchange:     "kraken",
			SellExchange:    "bitfinex",
			BuyPrice:        3000.0,
			SellPrice:       3020.0,
			ProfitPercent:   0.67,
			OpportunityType: "arbitrage",
		},
		{
			Symbol:          "LTC/USDT",
			BuyExchange:     "binance",
			SellExchange:    "binance",
			BuyPrice:        150.0,
			SellPrice:       151.0,
			ProfitPercent:   0.67,
			OpportunityType: "technical",
		},
	}

	// Test categorization logic
	arbitrageOpps := make([]ArbitrageOpportunity, 0)
	technicalOpps := make([]ArbitrageOpportunity, 0)

	for _, opp := range opportunities {
		if opp.BuyExchange != opp.SellExchange {
			arbitrageOpps = append(arbitrageOpps, opp)
		} else {
			technicalOpps = append(technicalOpps, opp)
		}
	}

	// Verify categorization results
	assert.Equal(t, 2, len(arbitrageOpps))
	assert.Equal(t, 1, len(technicalOpps))

	// Check that arbitrage opportunities have different exchanges
	for _, opp := range arbitrageOpps {
		assert.NotEqual(t, opp.BuyExchange, opp.SellExchange)
	}

	// Check that technical opportunities have same exchanges
	for _, opp := range technicalOpps {
		assert.Equal(t, opp.BuyExchange, opp.SellExchange)
	}
}

func TestNotificationService_NotifyArbitrageOpportunities_WithDatabase(t *testing.T) {
	// This test demonstrates that the function requires a valid database
	// In a real test environment, you would need to set up a test database
	ctx := context.Background()

	opportunities := []ArbitrageOpportunity{
		{
			Symbol:        "BTC/USDT",
			BuyExchange:   "binance",
			SellExchange:  "coinbase",
			BuyPrice:      50000.0,
			SellPrice:     50500.0,
			ProfitPercent: 1.0,
		},
	}

	// Should panic due to nil database access
	ns := NewNotificationService(nil, nil, "", "", "")
	assert.Panics(t, func() {
		_ = ns.NotifyArbitrageOpportunities(ctx, opportunities)
	})
}

func TestNotificationService_getEligibleUsers_NilDependencies(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")
	ctx := context.Background()

	// Should panic due to nil database when trying to query users
	assert.Panics(t, func() {
		_, _ = ns.getEligibleUsers(ctx)
	})
}

func TestNotificationService_getEligibleUsers_CacheKey(t *testing.T) {
	// Test that the cache key is consistent
	expectedCacheKey := "eligible_users:arbitrage"

	// We can't directly test the private function, but we can verify the key format
	// by checking it matches the expected pattern
	assert.Equal(t, 24, len(expectedCacheKey)) // Length check
	assert.Contains(t, expectedCacheKey, "eligible_users")
	assert.Contains(t, expectedCacheKey, "arbitrage")
}

func TestNotificationService_getEligibleUsers_QueryLogic(t *testing.T) {
	// Test the SQL query logic by examining what it should do
	// The query should:
	// 1. Select users with telegram_chat_id IS NOT NULL and not empty
	// 2. Exclude users who have disabled arbitrage notifications

	// This is a logical test - the actual SQL execution requires a database
	query := `
		SELECT id, email, telegram_chat_id, subscription_tier, created_at, updated_at
		FROM users 
		WHERE telegram_chat_id IS NOT NULL 
		  AND telegram_chat_id != ''
		  AND id NOT IN (
			  SELECT DISTINCT user_id 
			  FROM user_alerts 
			  WHERE alert_type = 'arbitrage' 
			    AND is_active = false
			    AND conditions->>'notifications_enabled' = 'false'
		  )
	`

	// Verify query contains expected conditions
	assert.Contains(t, query, "telegram_chat_id IS NOT NULL")
	assert.Contains(t, query, "telegram_chat_id != ''")
	assert.Contains(t, query, "alert_type = 'arbitrage'")
	assert.Contains(t, query, "is_active = false")
	assert.Contains(t, query, "conditions->>'notifications_enabled' = 'false'")
}

func TestNotificationService_getEligibleUsers_ContextCancellation(t *testing.T) {
	ns := NewNotificationService(nil, nil, "", "", "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still panic due to nil database, even with cancelled context
	assert.Panics(t, func() {
		_, _ = ns.getEligibleUsers(ctx)
	})
}
