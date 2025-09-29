package models

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFundingRate_Struct(t *testing.T) {
	now := time.Now()
	markPrice := decimal.NewFromFloat(50000.0)
	indexPrice := decimal.NewFromFloat(49900.0)
	nextFundingTime := now.Add(8 * time.Hour)

	fundingRate := FundingRate{
		ID:              1,
		ExchangeID:      1,
		TradingPairID:   1,
		FundingRate:     decimal.NewFromFloat(0.0001),
		FundingTime:     now,
		NextFundingTime: &nextFundingTime,
		MarkPrice:       &markPrice,
		IndexPrice:      &indexPrice,
		Timestamp:       now,
		CreatedAt:       now,
		ExchangeName:    "Binance",
		Symbol:          "BTC/USDT",
		BaseCurrency:    "BTC",
		QuoteCurrency:   "USDT",
	}

	assert.Equal(t, int64(1), fundingRate.ID)
	assert.Equal(t, 1, fundingRate.ExchangeID)
	assert.Equal(t, 1, fundingRate.TradingPairID)
	assert.Equal(t, decimal.NewFromFloat(0.0001), fundingRate.FundingRate)
	assert.Equal(t, now, fundingRate.FundingTime)
	assert.Equal(t, &nextFundingTime, fundingRate.NextFundingTime)
	assert.Equal(t, &markPrice, fundingRate.MarkPrice)
	assert.Equal(t, &indexPrice, fundingRate.IndexPrice)
	assert.Equal(t, now, fundingRate.Timestamp)
	assert.Equal(t, now, fundingRate.CreatedAt)
	assert.Equal(t, "Binance", fundingRate.ExchangeName)
	assert.Equal(t, "BTC/USDT", fundingRate.Symbol)
	assert.Equal(t, "BTC", fundingRate.BaseCurrency)
	assert.Equal(t, "USDT", fundingRate.QuoteCurrency)
}

func TestFundingArbitrageOpportunity_Struct(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(8 * time.Hour)
	longMarkPrice := decimal.NewFromFloat(50000.0)
	shortMarkPrice := decimal.NewFromFloat(50100.0)
	priceDifference := decimal.NewFromFloat(100.0)
	priceDifferencePercentage := decimal.NewFromFloat(0.2)

	opportunity := FundingArbitrageOpportunity{
		ID:                        1,
		TradingPairID:             1,
		LongExchangeID:            1,
		ShortExchangeID:           2,
		LongFundingRate:           decimal.NewFromFloat(0.0001),
		ShortFundingRate:          decimal.NewFromFloat(-0.0002),
		NetFundingRate:            decimal.NewFromFloat(0.0003),
		EstimatedProfit8h:         decimal.NewFromFloat(0.024),
		EstimatedProfitDaily:      decimal.NewFromFloat(0.072),
		EstimatedProfitPercentage: decimal.NewFromFloat(0.072),
		LongMarkPrice:             &longMarkPrice,
		ShortMarkPrice:            &shortMarkPrice,
		PriceDifference:           &priceDifference,
		PriceDifferencePercentage: &priceDifferencePercentage,
		RiskScore:                 decimal.NewFromFloat(1.5),
		IsActive:                  true,
		DetectedAt:                now,
		ExpiresAt:                 &expiresAt,
		CreatedAt:                 now,
		Symbol:                    "BTC/USDT",
		BaseCurrency:              "BTC",
		QuoteCurrency:             "USDT",
		LongExchangeName:          "Binance",
		ShortExchangeName:         "Bybit",
	}

	assert.Equal(t, int64(1), opportunity.ID)
	assert.Equal(t, 1, opportunity.TradingPairID)
	assert.Equal(t, 1, opportunity.LongExchangeID)
	assert.Equal(t, 2, opportunity.ShortExchangeID)
	assert.Equal(t, decimal.NewFromFloat(0.0001), opportunity.LongFundingRate)
	assert.Equal(t, decimal.NewFromFloat(-0.0002), opportunity.ShortFundingRate)
	assert.Equal(t, decimal.NewFromFloat(0.0003), opportunity.NetFundingRate)
	assert.Equal(t, decimal.NewFromFloat(0.024), opportunity.EstimatedProfit8h)
	assert.Equal(t, decimal.NewFromFloat(0.072), opportunity.EstimatedProfitDaily)
	assert.Equal(t, decimal.NewFromFloat(0.072), opportunity.EstimatedProfitPercentage)
	assert.Equal(t, &longMarkPrice, opportunity.LongMarkPrice)
	assert.Equal(t, &shortMarkPrice, opportunity.ShortMarkPrice)
	assert.Equal(t, &priceDifference, opportunity.PriceDifference)
	assert.Equal(t, &priceDifferencePercentage, opportunity.PriceDifferencePercentage)
	assert.Equal(t, decimal.NewFromFloat(1.5), opportunity.RiskScore)
	assert.True(t, opportunity.IsActive)
	assert.Equal(t, now, opportunity.DetectedAt)
	assert.Equal(t, &expiresAt, opportunity.ExpiresAt)
	assert.Equal(t, now, opportunity.CreatedAt)
	assert.Equal(t, "BTC/USDT", opportunity.Symbol)
	assert.Equal(t, "BTC", opportunity.BaseCurrency)
	assert.Equal(t, "USDT", opportunity.QuoteCurrency)
	assert.Equal(t, "Binance", opportunity.LongExchangeName)
	assert.Equal(t, "Bybit", opportunity.ShortExchangeName)
}

func TestFundingArbitrageOpportunityResponse_Struct(t *testing.T) {
	detectedAt := "2025-01-01T00:00:00Z"
	expiresAt := "2025-01-01T08:00:00Z"
	longMarkPrice := 50000.0
	shortMarkPrice := 50100.0
	priceDifference := 100.0
	priceDifferencePercentage := 0.2

	response := FundingArbitrageOpportunityResponse{
		ID:                        1,
		Symbol:                    "BTC/USDT",
		BaseCurrency:              "BTC",
		QuoteCurrency:             "USDT",
		LongExchange:              "Binance",
		ShortExchange:             "Bybit",
		LongFundingRate:           0.0001,
		ShortFundingRate:          -0.0002,
		NetFundingRate:            0.0003,
		EstimatedProfit8h:         0.024,
		EstimatedProfitDaily:      0.072,
		EstimatedProfitPercentage: 0.072,
		LongMarkPrice:             &longMarkPrice,
		ShortMarkPrice:            &shortMarkPrice,
		PriceDifference:           &priceDifference,
		PriceDifferencePercentage: &priceDifferencePercentage,
		RiskScore:                 1.5,
		DetectedAt:                detectedAt,
		ExpiresAt:                 &expiresAt,
	}

	assert.Equal(t, int64(1), response.ID)
	assert.Equal(t, "BTC/USDT", response.Symbol)
	assert.Equal(t, "BTC", response.BaseCurrency)
	assert.Equal(t, "USDT", response.QuoteCurrency)
	assert.Equal(t, "Binance", response.LongExchange)
	assert.Equal(t, "Bybit", response.ShortExchange)
	assert.Equal(t, 0.0001, response.LongFundingRate)
	assert.Equal(t, -0.0002, response.ShortFundingRate)
	assert.Equal(t, 0.0003, response.NetFundingRate)
	assert.Equal(t, 0.024, response.EstimatedProfit8h)
	assert.Equal(t, 0.072, response.EstimatedProfitDaily)
	assert.Equal(t, 0.072, response.EstimatedProfitPercentage)
	assert.Equal(t, &longMarkPrice, response.LongMarkPrice)
	assert.Equal(t, &shortMarkPrice, response.ShortMarkPrice)
	assert.Equal(t, &priceDifference, response.PriceDifference)
	assert.Equal(t, &priceDifferencePercentage, response.PriceDifferencePercentage)
	assert.Equal(t, 1.5, response.RiskScore)
	assert.Equal(t, detectedAt, response.DetectedAt)
	assert.Equal(t, &expiresAt, response.ExpiresAt)
}

func TestFundingRateRequest_Struct(t *testing.T) {
	request := FundingRateRequest{
		Exchange: "Binance",
		Symbols:  []string{"BTC/USDT", "ETH/USDT"},
		Limit:    100,
	}

	assert.Equal(t, "Binance", request.Exchange)
	assert.Len(t, request.Symbols, 2)
	assert.Contains(t, request.Symbols, "BTC/USDT")
	assert.Contains(t, request.Symbols, "ETH/USDT")
	assert.Equal(t, 100, request.Limit)
}

func TestFundingRateResponse_Struct(t *testing.T) {
	fundingTime := "2025-01-01T00:00:00Z"
	nextFundingTime := "2025-01-01T08:00:00Z"
	markPrice := 50000.0
	indexPrice := 49900.0
	timestamp := "2025-01-01T00:00:00Z"

	response := FundingRateResponse{
		Exchange:        "Binance",
		Symbol:          "BTC/USDT",
		BaseCurrency:    "BTC",
		QuoteCurrency:   "USDT",
		FundingRate:     0.0001,
		FundingTime:     fundingTime,
		NextFundingTime: &nextFundingTime,
		MarkPrice:       &markPrice,
		IndexPrice:      &indexPrice,
		Timestamp:       timestamp,
	}

	assert.Equal(t, "Binance", response.Exchange)
	assert.Equal(t, "BTC/USDT", response.Symbol)
	assert.Equal(t, "BTC", response.BaseCurrency)
	assert.Equal(t, "USDT", response.QuoteCurrency)
	assert.Equal(t, 0.0001, response.FundingRate)
	assert.Equal(t, fundingTime, response.FundingTime)
	assert.Equal(t, &nextFundingTime, response.NextFundingTime)
	assert.Equal(t, &markPrice, response.MarkPrice)
	assert.Equal(t, &indexPrice, response.IndexPrice)
	assert.Equal(t, timestamp, response.Timestamp)
}

func TestFundingArbitrageRequest_Struct(t *testing.T) {
	request := FundingArbitrageRequest{
		Symbols:   []string{"BTC/USDT", "ETH/USDT"},
		Exchanges: []string{"Binance", "Bybit"},
		MinProfit: 0.05,
		MaxRisk:   2.0,
		Limit:     50,
		Page:      1,
	}

	assert.Len(t, request.Symbols, 2)
	assert.Contains(t, request.Symbols, "BTC/USDT")
	assert.Contains(t, request.Symbols, "ETH/USDT")
	assert.Len(t, request.Exchanges, 2)
	assert.Contains(t, request.Exchanges, "Binance")
	assert.Contains(t, request.Exchanges, "Bybit")
	assert.Equal(t, 0.05, request.MinProfit)
	assert.Equal(t, 2.0, request.MaxRisk)
	assert.Equal(t, 50, request.Limit)
	assert.Equal(t, 1, request.Page)
}

func TestFundingArbitrageRequest_DefaultValues(t *testing.T) {
	request := FundingArbitrageRequest{}

	assert.Empty(t, request.Symbols)
	assert.Empty(t, request.Exchanges)
	assert.Equal(t, 0.0, request.MinProfit)
	assert.Equal(t, 0.0, request.MaxRisk)
	assert.Equal(t, 0, request.Limit)
	assert.Equal(t, 0, request.Page)
}

func TestFundingRateRequest_DefaultValues(t *testing.T) {
	request := FundingRateRequest{
		Exchange: "Binance",
	}

	assert.Equal(t, "Binance", request.Exchange)
	assert.Empty(t, request.Symbols)
	assert.Equal(t, 0, request.Limit)
}