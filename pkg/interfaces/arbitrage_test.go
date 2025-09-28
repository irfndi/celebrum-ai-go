package interfaces

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestArbitrageOpportunityResponse_Interface(t *testing.T) {
	// Test data
	detectedAt := time.Now()
	expiresAt := detectedAt.Add(30 * time.Minute)
	
	arbitrage := &ArbitrageOpportunityResponse{
		Symbol:           "BTC/USDT",
		BuyExchange:      "binance",
		SellExchange:     "coinbase",
		BuyPrice:         decimal.NewFromFloat(50000.0),
		SellPrice:        decimal.NewFromFloat(50100.0),
		ProfitPercentage: decimal.NewFromFloat(0.2),
		DetectedAt:       detectedAt,
		ExpiresAt:        expiresAt,
	}

	// Test interface implementation
	assert.Implements(t, (*ArbitrageOpportunityInterface)(nil), arbitrage)

	// Test all interface methods
	assert.Equal(t, "BTC/USDT", arbitrage.GetSymbol())
	assert.Equal(t, "binance", arbitrage.GetBuyExchange())
	assert.Equal(t, "coinbase", arbitrage.GetSellExchange())
	assert.Equal(t, decimal.NewFromFloat(50000.0), arbitrage.GetBuyPrice())
	assert.Equal(t, decimal.NewFromFloat(50100.0), arbitrage.GetSellPrice())
	assert.Equal(t, decimal.NewFromFloat(0.2), arbitrage.GetProfitPercentage())
	assert.Equal(t, detectedAt, arbitrage.GetDetectedAt())
	assert.Equal(t, expiresAt, arbitrage.GetExpiresAt())
}

func TestMarketPrice_Interface(t *testing.T) {
	// Test data
	timestamp := time.Now()
	
	marketPrice := &MarketPrice{
		ExchangeID:   1,
		ExchangeName: "binance",
		Symbol:       "BTC/USDT",
		Price:        decimal.NewFromFloat(50000.0),
		Volume:       decimal.NewFromFloat(1000.0),
		Timestamp:    timestamp,
	}

	// Test interface implementation
	assert.Implements(t, (*MarketPriceInterface)(nil), marketPrice)

	// Test all interface methods
	assert.Equal(t, 50000.0, marketPrice.GetPrice())
	assert.Equal(t, 1000.0, marketPrice.GetVolume())
	assert.Equal(t, timestamp, marketPrice.GetTimestamp())
	assert.Equal(t, "binance", marketPrice.GetExchangeName())
	assert.Equal(t, "BTC/USDT", marketPrice.GetSymbol())
}

func TestMarketPrice_InexactFloat64Conversion(t *testing.T) {
	// Test decimal to float64 conversion precision
	marketPrice := &MarketPrice{
		ExchangeID:   1,
		ExchangeName: "binance",
		Symbol:       "BTC/USDT",
		Price:        decimal.NewFromFloat(12345.6789),
		Volume:       decimal.NewFromFloat(9876.54321),
		Timestamp:    time.Now(),
	}

	// Test that the float64 conversion is within acceptable precision
	priceFloat := marketPrice.GetPrice()
	volumeFloat := marketPrice.GetVolume()
	
	// Allow for small precision differences due to decimal to float64 conversion
	assert.InDelta(t, 12345.6789, priceFloat, 0.0001)
	assert.InDelta(t, 9876.54321, volumeFloat, 0.0001)
}

func TestArbitrageOpportunityResponse_EmptyValues(t *testing.T) {
	// Test with empty/zero values
	arbitrage := &ArbitrageOpportunityResponse{
		Symbol:           "",
		BuyExchange:      "",
		SellExchange:     "",
		BuyPrice:         decimal.Zero,
		SellPrice:        decimal.Zero,
		ProfitPercentage: decimal.Zero,
		DetectedAt:       time.Time{},
		ExpiresAt:        time.Time{},
	}

	// Test interface methods with empty values
	assert.Equal(t, "", arbitrage.GetSymbol())
	assert.Equal(t, "", arbitrage.GetBuyExchange())
	assert.Equal(t, "", arbitrage.GetSellExchange())
	assert.Equal(t, decimal.Zero, arbitrage.GetBuyPrice())
	assert.Equal(t, decimal.Zero, arbitrage.GetSellPrice())
	assert.Equal(t, decimal.Zero, arbitrage.GetProfitPercentage())
	assert.Equal(t, time.Time{}, arbitrage.GetDetectedAt())
	assert.Equal(t, time.Time{}, arbitrage.GetExpiresAt())
}

func TestMarketPrice_EmptyValues(t *testing.T) {
	// Test with empty/zero values
	marketPrice := &MarketPrice{
		ExchangeID:   0,
		ExchangeName: "",
		Symbol:       "",
		Price:        decimal.Zero,
		Volume:       decimal.Zero,
		Timestamp:    time.Time{},
	}

	// Test interface methods with empty values
	assert.Equal(t, 0.0, marketPrice.GetPrice())
	assert.Equal(t, 0.0, marketPrice.GetVolume())
	assert.Equal(t, time.Time{}, marketPrice.GetTimestamp())
	assert.Equal(t, "", marketPrice.GetExchangeName())
	assert.Equal(t, "", marketPrice.GetSymbol())
}

func TestArbitrageOpportunityResponse_NegativeProfit(t *testing.T) {
	// Test with negative profit (loss scenario)
	arbitrage := &ArbitrageOpportunityResponse{
		Symbol:           "BTC/USDT",
		BuyExchange:      "binance",
		SellExchange:     "coinbase",
		BuyPrice:         decimal.NewFromFloat(50100.0),
		SellPrice:        decimal.NewFromFloat(50000.0),
		ProfitPercentage: decimal.NewFromFloat(-0.2),
		DetectedAt:       time.Now(),
		ExpiresAt:        time.Now().Add(30 * time.Minute),
	}

	assert.Equal(t, decimal.NewFromFloat(-0.2), arbitrage.GetProfitPercentage())
	assert.Equal(t, "BTC/USDT", arbitrage.GetSymbol())
	assert.Equal(t, "binance", arbitrage.GetBuyExchange())
	assert.Equal(t, "coinbase", arbitrage.GetSellExchange())
}

func TestArbitrageOpportunityResponse_Timing(t *testing.T) {
	// Test timing relationships
	detectedAt := time.Now()
	expiresAt := detectedAt.Add(1 * time.Hour)
	
	arbitrage := &ArbitrageOpportunityResponse{
		Symbol:           "BTC/USDT",
		BuyExchange:      "binance",
		SellExchange:     "coinbase",
		BuyPrice:         decimal.NewFromFloat(50000.0),
		SellPrice:        decimal.NewFromFloat(50100.0),
		ProfitPercentage: decimal.NewFromFloat(0.2),
		DetectedAt:       detectedAt,
		ExpiresAt:        expiresAt,
	}

	// Verify timing relationships
	assert.True(t, arbitrage.GetExpiresAt().After(arbitrage.GetDetectedAt()))
	assert.Equal(t, detectedAt, arbitrage.GetDetectedAt())
	assert.Equal(t, expiresAt, arbitrage.GetExpiresAt())
}

func TestMarketPrice_ConcurrentAccess(t *testing.T) {
	// Test concurrent access to market price data
	marketPrice := &MarketPrice{
		ExchangeID:   1,
		ExchangeName: "binance",
		Symbol:       "BTC/USDT",
		Price:        decimal.NewFromFloat(50000.0),
		Volume:       decimal.NewFromFloat(1000.0),
		Timestamp:    time.Now(),
	}

	// Concurrent access to test thread safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = marketPrice.GetPrice()
			_ = marketPrice.GetVolume()
			_ = marketPrice.GetTimestamp()
			_ = marketPrice.GetExchangeName()
			_ = marketPrice.GetSymbol()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify data integrity
	assert.Equal(t, 50000.0, marketPrice.GetPrice())
	assert.Equal(t, 1000.0, marketPrice.GetVolume())
	assert.Equal(t, "binance", marketPrice.GetExchangeName())
	assert.Equal(t, "BTC/USDT", marketPrice.GetSymbol())
}