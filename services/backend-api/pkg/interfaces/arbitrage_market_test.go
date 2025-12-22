package interfaces

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestArbitrageOpportunityResponse_GetSymbol tests the GetSymbol method
func TestArbitrageOpportunityResponse_GetSymbol(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{Symbol: "BTC/USDT"}
	assert.Equal(t, "BTC/USDT", ao.GetSymbol())
}

// TestArbitrageOpportunityResponse_GetBuyExchange tests the GetBuyExchange method
func TestArbitrageOpportunityResponse_GetBuyExchange(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{BuyExchange: "binance"}
	assert.Equal(t, "binance", ao.GetBuyExchange())
}

// TestArbitrageOpportunityResponse_GetSellExchange tests the GetSellExchange method
func TestArbitrageOpportunityResponse_GetSellExchange(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{SellExchange: "kraken"}
	assert.Equal(t, "kraken", ao.GetSellExchange())
}

// TestArbitrageOpportunityResponse_GetBuyPrice tests the GetBuyPrice method
func TestArbitrageOpportunityResponse_GetBuyPrice(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{BuyPrice: decimal.NewFromFloat(45000.50)}
	assert.True(t, ao.GetBuyPrice().Equal(decimal.NewFromFloat(45000.50)))
}

// TestArbitrageOpportunityResponse_GetSellPrice tests the GetSellPrice method
func TestArbitrageOpportunityResponse_GetSellPrice(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{SellPrice: decimal.NewFromFloat(45100.75)}
	assert.True(t, ao.GetSellPrice().Equal(decimal.NewFromFloat(45100.75)))
}

// TestArbitrageOpportunityResponse_GetProfitPercentage tests the GetProfitPercentage method
func TestArbitrageOpportunityResponse_GetProfitPercentage(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{ProfitPercentage: decimal.NewFromFloat(0.22)}
	assert.True(t, ao.GetProfitPercentage().Equal(decimal.NewFromFloat(0.22)))
}

// TestArbitrageOpportunityResponse_GetDetectedAt tests the GetDetectedAt method
func TestArbitrageOpportunityResponse_GetDetectedAt(t *testing.T) {
	now := time.Now()
	ao := &ArbitrageOpportunityResponse{DetectedAt: now}
	assert.Equal(t, now, ao.GetDetectedAt())
}

// TestArbitrageOpportunityResponse_GetExpiresAt tests the GetExpiresAt method
func TestArbitrageOpportunityResponse_GetExpiresAt(t *testing.T) {
	expires := time.Now().Add(5 * time.Minute)
	ao := &ArbitrageOpportunityResponse{ExpiresAt: expires}
	assert.Equal(t, expires, ao.GetExpiresAt())
}

// TestArbitrageOpportunityResponse_ImplementsInterface verifies interface implementation
func TestArbitrageOpportunityResponse_ImplementsInterface(t *testing.T) {
	// This test verifies that ArbitrageOpportunityResponse implements ArbitrageOpportunityInterface
	var _ ArbitrageOpportunityInterface = &ArbitrageOpportunityResponse{}

	now := time.Now()
	expires := now.Add(5 * time.Minute)

	ao := &ArbitrageOpportunityResponse{
		Symbol:           "ETH/USDT",
		BuyExchange:      "coinbase",
		SellExchange:     "binance",
		BuyPrice:         decimal.NewFromFloat(3000),
		SellPrice:        decimal.NewFromFloat(3010),
		ProfitPercentage: decimal.NewFromFloat(0.33),
		DetectedAt:       now,
		ExpiresAt:        expires,
	}

	// Test via interface
	var iface ArbitrageOpportunityInterface = ao
	assert.Equal(t, "ETH/USDT", iface.GetSymbol())
	assert.Equal(t, "coinbase", iface.GetBuyExchange())
	assert.Equal(t, "binance", iface.GetSellExchange())
	assert.True(t, iface.GetBuyPrice().Equal(decimal.NewFromFloat(3000)))
	assert.True(t, iface.GetSellPrice().Equal(decimal.NewFromFloat(3010)))
	assert.True(t, iface.GetProfitPercentage().Equal(decimal.NewFromFloat(0.33)))
	assert.Equal(t, now, iface.GetDetectedAt())
	assert.Equal(t, expires, iface.GetExpiresAt())
}

// TestMarketPrice_GetPrice tests the GetPrice method
func TestMarketPrice_GetPrice(t *testing.T) {
	mp := &MarketPrice{Price: decimal.NewFromFloat(50000.25)}
	assert.InDelta(t, 50000.25, mp.GetPrice(), 0.01)
}

// TestMarketPrice_GetVolume tests the GetVolume method
func TestMarketPrice_GetVolume(t *testing.T) {
	mp := &MarketPrice{Volume: decimal.NewFromFloat(1000000.5)}
	assert.InDelta(t, 1000000.5, mp.GetVolume(), 0.01)
}

// TestMarketPrice_GetTimestamp tests the GetTimestamp method
func TestMarketPrice_GetTimestamp(t *testing.T) {
	now := time.Now()
	mp := &MarketPrice{Timestamp: now}
	assert.Equal(t, now, mp.GetTimestamp())
}

// TestMarketPrice_GetExchangeName tests the GetExchangeName method
func TestMarketPrice_GetExchangeName(t *testing.T) {
	mp := &MarketPrice{ExchangeName: "kraken"}
	assert.Equal(t, "kraken", mp.GetExchangeName())
}

// TestMarketPrice_GetSymbol tests the GetSymbol method
func TestMarketPrice_GetSymbol(t *testing.T) {
	mp := &MarketPrice{Symbol: "XRP/USDT"}
	assert.Equal(t, "XRP/USDT", mp.GetSymbol())
}

// TestMarketPrice_ImplementsInterface verifies interface implementation
func TestMarketPrice_ImplementsInterface(t *testing.T) {
	// This test verifies that MarketPrice implements MarketPriceInterface
	var _ MarketPriceInterface = &MarketPrice{}

	now := time.Now()
	mp := &MarketPrice{
		ExchangeID:   1,
		ExchangeName: "binance",
		Symbol:       "BTC/USDT",
		Price:        decimal.NewFromFloat(45000),
		Volume:       decimal.NewFromFloat(500000),
		Timestamp:    now,
	}

	// Test via interface
	var iface MarketPriceInterface = mp
	assert.InDelta(t, 45000, iface.GetPrice(), 0.01)
	assert.InDelta(t, 500000, iface.GetVolume(), 0.01)
	assert.Equal(t, now, iface.GetTimestamp())
	assert.Equal(t, "binance", iface.GetExchangeName())
	assert.Equal(t, "BTC/USDT", iface.GetSymbol())
}

// TestMarketPrice_ZeroValues tests behavior with zero values
func TestMarketPrice_ZeroValues(t *testing.T) {
	mp := &MarketPrice{}

	assert.Equal(t, float64(0), mp.GetPrice())
	assert.Equal(t, float64(0), mp.GetVolume())
	assert.True(t, mp.GetTimestamp().IsZero())
	assert.Equal(t, "", mp.GetExchangeName())
	assert.Equal(t, "", mp.GetSymbol())
}

// TestArbitrageOpportunityResponse_ZeroValues tests behavior with zero values
func TestArbitrageOpportunityResponse_ZeroValues(t *testing.T) {
	ao := &ArbitrageOpportunityResponse{}

	assert.Equal(t, "", ao.GetSymbol())
	assert.Equal(t, "", ao.GetBuyExchange())
	assert.Equal(t, "", ao.GetSellExchange())
	assert.True(t, ao.GetBuyPrice().IsZero())
	assert.True(t, ao.GetSellPrice().IsZero())
	assert.True(t, ao.GetProfitPercentage().IsZero())
	assert.True(t, ao.GetDetectedAt().IsZero())
	assert.True(t, ao.GetExpiresAt().IsZero())
}
