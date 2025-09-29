package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBusinessTracer(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)
	require.NotNil(t, bt.tracer)
}

func TestBusinessTracer_TraceArbitrageDetection(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	symbol := "BTC/USDT"
	exchanges := []string{"binance", "bybit"}

	_, span := bt.TraceArbitrageDetection(ctx, symbol, exchanges)
	require.NotNil(t, span)
	
	// End the span to avoid resource leaks
	span.End()
}

func TestBusinessTracer_RecordArbitrageOpportunity(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceArbitrageDetection(ctx, "BTC/USDT", []string{"binance", "bybit"})
	require.NotNil(t, span)

	opportunity := ArbitrageOpportunity{
		BuyExchange:      "binance",
		SellExchange:     "bybit",
		BuyPrice:         50000.0,
		SellPrice:        50100.0,
		ProfitPercentage: 0.2,
		ProfitAmount:     100.0,
		Volume:           1.0,
		Type:             "spot",
		Quality:          "high",
		ConfidenceScore:  0.95,
	}

	// This should not panic
	bt.RecordArbitrageOpportunity(span, opportunity)
	span.End()
}

func TestBusinessTracer_TraceSignalProcessing(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	signalType := "rsi"
	symbol := "BTC/USDT"

	_, span := bt.TraceSignalProcessing(ctx, signalType, symbol)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_RecordSignalMetrics(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceSignalProcessing(ctx, "rsi", "BTC/USDT")
	require.NotNil(t, span)

	metrics := SignalMetrics{
		ProcessedCount:  100,
		ValidCount:      85,
		InvalidCount:    15,
		AverageStrength: 0.75,
		ProcessingTime:  150 * time.Millisecond,
		QualityGrade:    "A",
	}

	// This should not panic
	bt.RecordSignalMetrics(span, metrics)
	span.End()
}

func TestBusinessTracer_TraceTechnicalAnalysis(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	indicator := "rsi"
	symbol := "BTC/USDT"
	timeframe := "1h"

	_, span := bt.TraceTechnicalAnalysis(ctx, indicator, symbol, timeframe)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_RecordTechnicalAnalysisResult(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceTechnicalAnalysis(ctx, "rsi", "BTC/USDT", "1h")
	require.NotNil(t, span)

	result := TechnicalAnalysisResult{
		Value:           65.5,
		SignalDirection: "bullish",
		Confidence:      0.8,
		DataPoints:      100,
		IsValid:         true,
	}

	// This should not panic
	bt.RecordTechnicalAnalysisResult(span, result)
	span.End()
}

func TestBusinessTracer_TraceMarketDataCollection(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	exchange := "binance"
	symbols := []string{"BTC/USDT", "ETH/USDT"}

	_, span := bt.TraceMarketDataCollection(ctx, exchange, symbols)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_RecordMarketDataMetrics(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceMarketDataCollection(ctx, "binance", []string{"BTC/USDT", "ETH/USDT"})
	require.NotNil(t, span)

	metrics := MarketDataMetrics{
		CollectedCount: 2,
		FailedCount:    0,
		CollectionTime: 100 * time.Millisecond,
		SuccessRate:    1.0,
		DataQuality:    "high",
	}

	// This should not panic
	bt.RecordMarketDataMetrics(span, metrics)
	span.End()
}

func TestBusinessTracer_TraceRiskAssessment(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	assessmentType := "volatility"
	symbol := "BTC/USDT"

	_, span := bt.TraceRiskAssessment(ctx, assessmentType, symbol)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_RecordRiskMetrics(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceRiskAssessment(ctx, "volatility", "BTC/USDT")
	require.NotNil(t, span)

	metrics := RiskMetrics{
		RiskScore:    3.5,
		RiskLevel:    "medium",
		Volatility:   0.15,
		MaxDrawdown:  0.08,
		IsAcceptable: true,
	}

	// This should not panic
	bt.RecordRiskMetrics(span, metrics)
	span.End()
}

func TestBusinessTracer_TraceNotification(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	notificationType := "arbitrage_opportunity"
	channel := "telegram"

	_, span := bt.TraceNotification(ctx, notificationType, channel)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_RecordNotificationResult(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceNotification(ctx, "arbitrage_opportunity", "telegram")
	require.NotNil(t, span)

	// Test successful notification
	bt.RecordNotificationResult(span, true, 5, nil)
	span.End()

	// Test failed notification
	_, span = bt.TraceNotification(ctx, "arbitrage_opportunity", "telegram")
	require.NotNil(t, span)
	
	testErr := assert.AnError
	bt.RecordNotificationResult(span, false, 0, testErr)
	span.End()
}

func TestBusinessTracer_TraceArbitrageDetectionEmptyExchanges(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	symbol := "BTC/USDT"
	exchanges := []string{}

	_, span := bt.TraceArbitrageDetection(ctx, symbol, exchanges)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_TraceMarketDataCollectionEmptySymbols(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	exchange := "binance"
	symbols := []string{}

	_, span := bt.TraceMarketDataCollection(ctx, exchange, symbols)
	require.NotNil(t, span)
	require.NotNil(t, span)
	
	span.End()
}

func TestBusinessTracer_RecordArbitrageOpportunityZeroValues(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceArbitrageDetection(ctx, "BTC/USDT", []string{"binance", "bybit"})
	require.NotNil(t, span)

	opportunity := ArbitrageOpportunity{
		BuyExchange:      "",
		SellExchange:     "",
		BuyPrice:         0.0,
		SellPrice:        0.0,
		ProfitPercentage: 0.0,
		ProfitAmount:     0.0,
		Volume:           0.0,
		Type:             "",
		Quality:          "",
		ConfidenceScore:  0.0,
	}

	// This should not panic even with zero values
	bt.RecordArbitrageOpportunity(span, opportunity)
	span.End()
}

func TestBusinessTracer_RecordSignalMetricsZeroValues(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceSignalProcessing(ctx, "rsi", "BTC/USDT")
	require.NotNil(t, span)

	metrics := SignalMetrics{
		ProcessedCount:  0,
		ValidCount:      0,
		InvalidCount:    0,
		AverageStrength: 0.0,
		ProcessingTime:  0,
		QualityGrade:    "",
	}

	// This should not panic even with zero values
	bt.RecordSignalMetrics(span, metrics)
	span.End()
}

func TestBusinessTracer_RecordRiskMetricsEdgeCases(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceRiskAssessment(ctx, "volatility", "BTC/USDT")
	require.NotNil(t, span)

	// Test with maximum values
	metrics := RiskMetrics{
		RiskScore:    10.0,
		RiskLevel:    "critical",
		Volatility:   1.0,
		MaxDrawdown:  1.0,
		IsAcceptable: false,
	}

	bt.RecordRiskMetrics(span, metrics)
	span.End()

	// Test with minimum values
	_, span = bt.TraceRiskAssessment(ctx, "volatility", "BTC/USDT")
	require.NotNil(t, span)

	metrics = RiskMetrics{
		RiskScore:    0.0,
		RiskLevel:    "low",
		Volatility:   0.0,
		MaxDrawdown:  0.0,
		IsAcceptable: true,
	}

	bt.RecordRiskMetrics(span, metrics)
	span.End()
}

func TestBusinessTracer_RecordTechnicalAnalysisResultEdgeCases(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceTechnicalAnalysis(ctx, "rsi", "BTC/USDT", "1h")
	require.NotNil(t, span)

	// Test with extreme values
	result := TechnicalAnalysisResult{
		Value:           100.0,
		SignalDirection: "overbought",
		Confidence:      1.0,
		DataPoints:      1000,
		IsValid:         true,
	}

	bt.RecordTechnicalAnalysisResult(span, result)
	span.End()

	// Test with invalid result
	_, span = bt.TraceTechnicalAnalysis(ctx, "rsi", "BTC/USDT", "1h")
	require.NotNil(t, span)

	result = TechnicalAnalysisResult{
		Value:           0.0,
		SignalDirection: "neutral",
		Confidence:      0.0,
		DataPoints:      0,
		IsValid:         false,
	}

	bt.RecordTechnicalAnalysisResult(span, result)
	span.End()
}

func TestBusinessTracer_RecordNotificationResultNilError(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx := context.Background()
	_, span := bt.TraceNotification(ctx, "arbitrage_opportunity", "telegram")
	require.NotNil(t, span)

	// Test with nil error
	bt.RecordNotificationResult(span, true, 10, nil)
	span.End()
}

func TestBusinessTracer_ContextCancellation(t *testing.T) {
	bt := NewBusinessTracer()
	require.NotNil(t, bt)

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel the context
	cancel()
	
	// The tracer should still work even with cancelled context
	_, span := bt.TraceArbitrageDetection(ctx, "BTC/USDT", []string{"binance", "bybit"})
	require.NotNil(t, span)
	
	span.End()
}

