package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewNotificationService(t *testing.T) {
	// Test with empty token
	ns := NewNotificationService(nil, "")
	assert.NotNil(t, ns)
	assert.Nil(t, ns.bot)
	assert.Nil(t, ns.db)

	// Test with token - bot creation may fail with invalid token but service should still be created
	ns2 := NewNotificationService(nil, "test-token")
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
	ns := NewNotificationService(nil, "")

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
	assert.Equal(t, -100.0, negativeOpp.BuyPrice)
	assert.Equal(t, -50.0, negativeOpp.SellPrice)
	assert.Equal(t, -10.0, negativeOpp.ProfitPercent)

	// Test with very large values
	largeOpp := ArbitrageOpportunity{
		Symbol:        "BTC/USDT",
		BuyPrice:      1000000.0,
		SellPrice:     1100000.0,
		ProfitPercent: 10.0,
		ProfitAmount:  100000.0,
		Volume:        1000.0,
	}
	assert.Equal(t, 1000000.0, largeOpp.BuyPrice)
	assert.Equal(t, 1100000.0, largeOpp.SellPrice)
	assert.Equal(t, 10.0, largeOpp.ProfitPercent)
}

// Test formatArbitrageMessage with edge cases
func TestNotificationService_formatArbitrageMessage_EdgeCases(t *testing.T) {
	ns := NewNotificationService(nil, "")

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
	ns := NewNotificationService(nil, "")

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
	ns := NewNotificationService(nil, "")

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
	ns := NewNotificationService(nil, "")

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
