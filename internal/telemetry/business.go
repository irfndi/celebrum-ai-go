package telemetry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
)

// BusinessTracer provides business logic tracing utilities
type BusinessTracer struct{}

// NewBusinessTracer creates a new business tracer
func NewBusinessTracer() *BusinessTracer {
	return &BusinessTracer{}
}

// TraceArbitrageDetection traces arbitrage opportunity detection
func (bt *BusinessTracer) TraceArbitrageDetection(ctx context.Context, symbol string, exchanges []string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "arbitrage_detection")
	span.SetTag("symbol", symbol)
	span.SetData("exchanges", exchanges)
	return ctx, span
}

// RecordArbitrageOpportunity records details of a found arbitrage opportunity
func (bt *BusinessTracer) RecordArbitrageOpportunity(span *sentry.Span, opportunity ArbitrageOpportunity) {
	span.SetTag("buy_exchange", opportunity.BuyExchange)
	span.SetTag("sell_exchange", opportunity.SellExchange)
	span.SetData("buy_price", opportunity.BuyPrice)
	span.SetData("sell_price", opportunity.SellPrice)
	span.SetData("profit_percentage", opportunity.ProfitPercentage)
	span.SetData("profit_amount", opportunity.ProfitAmount)
	span.SetData("volume", opportunity.Volume)
	span.SetTag("type", opportunity.Type)
	span.SetTag("quality", opportunity.Quality)
	span.SetData("confidence_score", opportunity.ConfidenceScore)
}

// TraceSignalProcessing traces trading signal processing
func (bt *BusinessTracer) TraceSignalProcessing(ctx context.Context, signalType string, symbol string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "signal_processing")
	span.SetTag("signal_type", signalType)
	span.SetTag("symbol", symbol)
	return ctx, span
}

// RecordSignalMetrics records signal processing metrics
func (bt *BusinessTracer) RecordSignalMetrics(span *sentry.Span, metrics SignalMetrics) {
	span.SetData("processed_count", metrics.ProcessedCount)
	span.SetData("valid_count", metrics.ValidCount)
	span.SetData("invalid_count", metrics.InvalidCount)
	span.SetData("average_strength", metrics.AverageStrength)
	span.SetData("processing_time_ms", metrics.ProcessingTime.Milliseconds())
	span.SetTag("quality_grade", metrics.QualityGrade)
}

// TraceTechnicalAnalysis traces technical analysis operations
func (bt *BusinessTracer) TraceTechnicalAnalysis(ctx context.Context, indicator string, symbol string, timeframe string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "technical_analysis")
	span.SetTag("indicator", indicator)
	span.SetTag("symbol", symbol)
	span.SetTag("timeframe", timeframe)
	return ctx, span
}

// RecordTechnicalAnalysisResult records technical analysis results
func (bt *BusinessTracer) RecordTechnicalAnalysisResult(span *sentry.Span, result TechnicalAnalysisResult) {
	span.SetData("value", result.Value)
	span.SetTag("signal_direction", result.SignalDirection)
	span.SetData("confidence", result.Confidence)
	span.SetData("data_points", result.DataPoints)
	span.SetData("is_valid", result.IsValid)
}

// TraceMarketDataCollection traces market data collection
func (bt *BusinessTracer) TraceMarketDataCollection(ctx context.Context, exchange string, symbols []string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "market_data_collection")
	span.SetTag("exchange", exchange)
	span.SetData("symbols", symbols)
	return ctx, span
}

// RecordMarketDataMetrics records market data collection metrics
func (bt *BusinessTracer) RecordMarketDataMetrics(span *sentry.Span, metrics MarketDataMetrics) {
	span.SetData("collected_count", metrics.CollectedCount)
	span.SetData("failed_count", metrics.FailedCount)
	span.SetData("collection_time_ms", metrics.CollectionTime.Milliseconds())
	span.SetData("success_rate", metrics.SuccessRate)
	span.SetTag("data_quality", metrics.DataQuality)
}

// TraceRiskAssessment traces risk assessment operations
func (bt *BusinessTracer) TraceRiskAssessment(ctx context.Context, assessmentType string, symbol string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "risk_assessment")
	span.SetTag("assessment_type", assessmentType)
	span.SetTag("symbol", symbol)
	return ctx, span
}

// RecordRiskMetrics records risk assessment metrics
func (bt *BusinessTracer) RecordRiskMetrics(span *sentry.Span, metrics RiskMetrics) {
	span.SetData("risk_score", metrics.RiskScore)
	span.SetTag("risk_level", metrics.RiskLevel)
	span.SetData("volatility", metrics.Volatility)
	span.SetData("max_drawdown", metrics.MaxDrawdown)
	span.SetData("is_acceptable", metrics.IsAcceptable)
}

// TraceNotification traces notification sending
func (bt *BusinessTracer) TraceNotification(ctx context.Context, notificationType string, channel string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "notification")
	span.SetTag("notification_type", notificationType)
	span.SetTag("channel", channel)
	return ctx, span
}

// RecordNotificationResult records notification sending result
func (bt *BusinessTracer) RecordNotificationResult(span *sentry.Span, success bool, recipientCount int, err error) {
	span.SetData("success", success)
	span.SetData("recipient_count", recipientCount)
	if err != nil {
		span.SetTag("error", err.Error())
		span.Status = sentry.SpanStatusInternalError
	} else {
		span.Status = sentry.SpanStatusOK
	}
}

// Business domain types
type ArbitrageOpportunity struct {
	BuyExchange      string
	SellExchange     string
	BuyPrice         float64
	SellPrice        float64
	ProfitPercentage float64
	ProfitAmount     float64
	Volume           float64
	Type             string
	Quality          string
	ConfidenceScore  float64
}

type SignalMetrics struct {
	ProcessedCount  int
	ValidCount      int
	InvalidCount    int
	AverageStrength float64
	ProcessingTime  time.Duration
	QualityGrade    string
}

type TechnicalAnalysisResult struct {
	Value           float64
	SignalDirection string
	Confidence      float64
	DataPoints      int
	IsValid         bool
}

type MarketDataMetrics struct {
	CollectedCount int
	FailedCount    int
	CollectionTime time.Duration
	SuccessRate    float64
	DataQuality    string
}

type RiskMetrics struct {
	RiskScore    float64
	RiskLevel    string
	Volatility   float64
	MaxDrawdown  float64
	IsAcceptable bool
}
