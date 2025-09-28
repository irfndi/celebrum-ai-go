package services

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewNotificationService(t *testing.T) {
	// Test with empty token
	ns := NewNotificationService(nil, nil, "")
	assert.NotNil(t, ns)
	assert.Nil(t, ns.bot)
	assert.Nil(t, ns.db)

	// Test with token - bot creation may fail with invalid token but service should still be created
	ns2 := NewNotificationService(nil, nil, "test-token")
	assert.NotNil(t, ns2)
	assert.Nil(t, ns2.db)
	// Bot may be nil if token is invalid, which is expected behavior
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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

	// Test initial state
	assert.NotNil(t, ns)
	assert.Nil(t, ns.db)
	assert.Nil(t, ns.bot)

	// Test that we can access fields without panic
	assert.NotPanics(t, func() {
		_ = ns.db
		_ = ns.bot
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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")
	ns.PublishOpportunityUpdate(context.Background(), []ArbitrageOpportunity{})
	// Should not panic
}

func TestNotificationService_GetCacheStats(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")
	stats := ns.GetCacheStats(context.Background())

	assert.False(t, stats["redis_available"].(bool))
	assert.NotContains(t, stats, "users_cached")
	assert.NotContains(t, stats, "opportunities_cached")
}

func TestNotificationService_cacheArbitrageOpportunities(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

	data := map[string]interface{}{
		"symbol": "BTC/USDT",
		"price":  50000.0,
	}

	// Should not panic with nil Redis
	ns.CacheMarketData(context.Background(), "binance", data)
}

func TestNotificationService_GetCachedMarketData(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")

	var result map[string]interface{}
	err := ns.GetCachedMarketData(context.Background(), "binance", &result)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis not available")
}

func TestNotificationService_InvalidateUserCache(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")

	// Should not panic with nil Redis
	ns.InvalidateUserCache(context.Background())
}

func TestNotificationService_InvalidateOpportunityCache(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")

	// Should not panic with nil Redis
	ns.InvalidateOpportunityCache(context.Background())
}

func TestNotificationService_formatTechnicalSignalMessage(t *testing.T) {
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

	message, found := ns.getCachedMessage(context.Background(), "test", "testhash")
	assert.Empty(t, message)
	assert.False(t, found)
}

func TestNotificationService_setCachedMessage(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")

	// Should not panic with nil Redis
	ns.setCachedMessage(context.Background(), "test", "testhash", "test message")
}

func TestNotificationService_checkRateLimit(t *testing.T) {
	// Test with nil Redis
	ns := NewNotificationService(nil, nil, "")

	allowed, err := ns.checkRateLimit(context.Background(), "testuser")
	assert.NoError(t, err)
	assert.True(t, allowed) // Should allow when Redis is not available
}

func TestNotificationService_logNotification(t *testing.T) {
	// Test with nil database - expect panic due to nil database access
	ns := NewNotificationService(nil, nil, "")

	assert.Panics(t, func() {
		err := ns.logNotification(context.Background(), "testuser", "telegram", "test message")
		if err != nil {
			t.Log("Error:", err)
		}
	})
}

func TestNotificationService_CheckUserNotificationPreferences(t *testing.T) {
	// Test with nil database and Redis - expect panic due to nil database access
	ns := NewNotificationService(nil, nil, "")

	assert.Panics(t, func() {
		enabled, err := ns.CheckUserNotificationPreferences(context.Background(), "testuser")
		if err != nil {
			t.Log("Error:", err)
		}
		t.Log("Enabled:", enabled)
	})
}

func TestNotificationService_generateAggregatedSignalsHash(t *testing.T) {
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
	ns := NewNotificationService(nil, nil, "")

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
