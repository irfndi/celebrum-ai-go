package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// BusinessTracer provides business logic tracing utilities
type BusinessTracer struct {
	tracer trace.Tracer
}

// NewBusinessTracer creates a new business tracer
func NewBusinessTracer() *BusinessTracer {
	return &BusinessTracer{
		tracer: GetBusinessTracer(),
	}
}

// TraceArbitrageDetection traces arbitrage opportunity detection
func (bt *BusinessTracer) TraceArbitrageDetection(ctx context.Context, symbol string, exchanges []string) (context.Context, trace.Span) {
	ctx, span := bt.tracer.Start(ctx, "arbitrage_detection")
	span.SetAttributes(
		StringAttribute("symbol", symbol),
		StringSliceAttribute("exchanges", exchanges),
	)
	return ctx, span
}

// RecordArbitrageOpportunity records details of a found arbitrage opportunity
func (bt *BusinessTracer) RecordArbitrageOpportunity(span trace.Span, opportunity ArbitrageOpportunity) {
	span.SetAttributes(
		StringAttribute("buy_exchange", opportunity.BuyExchange),
		StringAttribute("sell_exchange", opportunity.SellExchange),
		Float64Attribute("buy_price", opportunity.BuyPrice),
		Float64Attribute("sell_price", opportunity.SellPrice),
		Float64Attribute("profit_percentage", opportunity.ProfitPercentage),
		Float64Attribute("profit_amount", opportunity.ProfitAmount),
		Float64Attribute("volume", opportunity.Volume),
		StringAttribute("type", opportunity.Type),
		StringAttribute("quality", opportunity.Quality),
		Float64Attribute("confidence_score", opportunity.ConfidenceScore),
	)
}

// TraceSignalProcessing traces trading signal processing
func (bt *BusinessTracer) TraceSignalProcessing(ctx context.Context, signalType string, symbol string) (context.Context, trace.Span) {
	ctx, span := bt.tracer.Start(ctx, "signal_processing")
	span.SetAttributes(
		StringAttribute("signal_type", signalType),
		StringAttribute("symbol", symbol),
	)
	return ctx, span
}

// RecordSignalMetrics records signal processing metrics
func (bt *BusinessTracer) RecordSignalMetrics(span trace.Span, metrics SignalMetrics) {
	span.SetAttributes(
		Int64Attribute("processed_count", int64(metrics.ProcessedCount)),
		Int64Attribute("valid_count", int64(metrics.ValidCount)),
		Int64Attribute("invalid_count", int64(metrics.InvalidCount)),
		Float64Attribute("average_strength", metrics.AverageStrength),
		Int64Attribute("processing_time_ms", metrics.ProcessingTime.Milliseconds()),
		StringAttribute("quality_grade", metrics.QualityGrade),
	)
}

// TraceTechnicalAnalysis traces technical analysis operations
func (bt *BusinessTracer) TraceTechnicalAnalysis(ctx context.Context, indicator string, symbol string, timeframe string) (context.Context, trace.Span) {
	ctx, span := bt.tracer.Start(ctx, "technical_analysis")
	span.SetAttributes(
		StringAttribute("indicator", indicator),
		StringAttribute("symbol", symbol),
		StringAttribute("timeframe", timeframe),
	)
	return ctx, span
}

// RecordTechnicalAnalysisResult records technical analysis results
func (bt *BusinessTracer) RecordTechnicalAnalysisResult(span trace.Span, result TechnicalAnalysisResult) {
	span.SetAttributes(
		Float64Attribute("value", result.Value),
		StringAttribute("signal_direction", result.SignalDirection),
		Float64Attribute("confidence", result.Confidence),
		Int64Attribute("data_points", int64(result.DataPoints)),
		BoolAttribute("is_valid", result.IsValid),
	)
}

// TraceMarketDataCollection traces market data collection
func (bt *BusinessTracer) TraceMarketDataCollection(ctx context.Context, exchange string, symbols []string) (context.Context, trace.Span) {
	ctx, span := bt.tracer.Start(ctx, "market_data_collection")
	span.SetAttributes(
		StringAttribute("exchange", exchange),
		StringSliceAttribute("symbols", symbols),
	)
	return ctx, span
}

// RecordMarketDataMetrics records market data collection metrics
func (bt *BusinessTracer) RecordMarketDataMetrics(span trace.Span, metrics MarketDataMetrics) {
	span.SetAttributes(
		Int64Attribute("collected_count", int64(metrics.CollectedCount)),
		Int64Attribute("failed_count", int64(metrics.FailedCount)),
		Int64Attribute("collection_time_ms", metrics.CollectionTime.Milliseconds()),
		Float64Attribute("success_rate", metrics.SuccessRate),
		StringAttribute("data_quality", metrics.DataQuality),
	)
}

// TraceRiskAssessment traces risk assessment operations
func (bt *BusinessTracer) TraceRiskAssessment(ctx context.Context, assessmentType string, symbol string) (context.Context, trace.Span) {
	ctx, span := bt.tracer.Start(ctx, "risk_assessment")
	span.SetAttributes(
		StringAttribute("assessment_type", assessmentType),
		StringAttribute("symbol", symbol),
	)
	return ctx, span
}

// RecordRiskMetrics records risk assessment metrics
func (bt *BusinessTracer) RecordRiskMetrics(span trace.Span, metrics RiskMetrics) {
	span.SetAttributes(
		Float64Attribute("risk_score", metrics.RiskScore),
		StringAttribute("risk_level", metrics.RiskLevel),
		Float64Attribute("volatility", metrics.Volatility),
		Float64Attribute("max_drawdown", metrics.MaxDrawdown),
		BoolAttribute("is_acceptable", metrics.IsAcceptable),
	)
}

// TraceNotification traces notification sending
func (bt *BusinessTracer) TraceNotification(ctx context.Context, notificationType string, channel string) (context.Context, trace.Span) {
	ctx, span := bt.tracer.Start(ctx, "notification")
	span.SetAttributes(
		StringAttribute("notification_type", notificationType),
		StringAttribute("channel", channel),
	)
	return ctx, span
}

// RecordNotificationResult records notification sending result
func (bt *BusinessTracer) RecordNotificationResult(span trace.Span, success bool, recipientCount int, err error) {
	span.SetAttributes(
		BoolAttribute("success", success),
		Int64Attribute("recipient_count", int64(recipientCount)),
	)
	if err != nil {
		span.SetAttributes(StringAttribute("error", err.Error()))
	}
}

// Business domain types (keeping the same structure)
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
