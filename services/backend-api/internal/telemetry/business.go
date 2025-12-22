package telemetry

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
)

// BusinessTracer provides utilities for tracing business logic operations using Sentry.
// It allows detailed tracking of domain-specific activities like arbitrage detection and signal processing.
type BusinessTracer struct{}

// NewBusinessTracer creates a new instance of BusinessTracer.
//
// Returns:
//   - A pointer to an initialized BusinessTracer.
func NewBusinessTracer() *BusinessTracer {
	return &BusinessTracer{}
}

// TraceArbitrageDetection starts a span for tracing the detection of arbitrage opportunities.
//
// Parameters:
//   - ctx: The context to attach the span to.
//   - symbol: The trading symbol being analyzed.
//   - exchanges: A list of exchanges involved in the detection.
//
// Returns:
//   - A context containing the new span.
//   - The created Sentry span.
func (bt *BusinessTracer) TraceArbitrageDetection(ctx context.Context, symbol string, exchanges []string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "arbitrage_detection")
	span.SetTag("symbol", symbol)
	span.SetData("exchanges", exchanges)
	return ctx, span
}

// RecordArbitrageOpportunity adds details of a found arbitrage opportunity to an existing span.
//
// Parameters:
//   - span: The Sentry span to update.
//   - opportunity: The details of the arbitrage opportunity.
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

// TraceSignalProcessing starts a span for tracing the processing of a trading signal.
//
// Parameters:
//   - ctx: The context to attach the span to.
//   - signalType: The type of signal being processed.
//   - symbol: The symbol associated with the signal.
//
// Returns:
//   - A context containing the new span.
//   - The created Sentry span.
func (bt *BusinessTracer) TraceSignalProcessing(ctx context.Context, signalType string, symbol string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "signal_processing")
	span.SetTag("signal_type", signalType)
	span.SetTag("symbol", symbol)
	return ctx, span
}

// RecordSignalMetrics records metrics related to signal processing onto a span.
//
// Parameters:
//   - span: The Sentry span to update.
//   - metrics: The signal processing metrics to record.
func (bt *BusinessTracer) RecordSignalMetrics(span *sentry.Span, metrics SignalMetrics) {
	span.SetData("processed_count", metrics.ProcessedCount)
	span.SetData("valid_count", metrics.ValidCount)
	span.SetData("invalid_count", metrics.InvalidCount)
	span.SetData("average_strength", metrics.AverageStrength)
	span.SetData("processing_time_ms", metrics.ProcessingTime.Milliseconds())
	span.SetTag("quality_grade", metrics.QualityGrade)
}

// TraceTechnicalAnalysis starts a span for tracing technical analysis calculations.
//
// Parameters:
//   - ctx: The context to attach the span to.
//   - indicator: The name of the technical indicator being calculated.
//   - symbol: The symbol being analyzed.
//   - timeframe: The timeframe of the analysis.
//
// Returns:
//   - A context containing the new span.
//   - The created Sentry span.
func (bt *BusinessTracer) TraceTechnicalAnalysis(ctx context.Context, indicator string, symbol string, timeframe string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "technical_analysis")
	span.SetTag("indicator", indicator)
	span.SetTag("symbol", symbol)
	span.SetTag("timeframe", timeframe)
	return ctx, span
}

// RecordTechnicalAnalysisResult adds the results of a technical analysis to a span.
//
// Parameters:
//   - span: The Sentry span to update.
//   - result: The result of the analysis.
func (bt *BusinessTracer) RecordTechnicalAnalysisResult(span *sentry.Span, result TechnicalAnalysisResult) {
	span.SetData("value", result.Value)
	span.SetTag("signal_direction", result.SignalDirection)
	span.SetData("confidence", result.Confidence)
	span.SetData("data_points", result.DataPoints)
	span.SetData("is_valid", result.IsValid)
}

// TraceMarketDataCollection starts a span for tracing the collection of market data.
//
// Parameters:
//   - ctx: The context to attach the span to.
//   - exchange: The exchange from which data is being collected.
//   - symbols: The list of symbols being collected.
//
// Returns:
//   - A context containing the new span.
//   - The created Sentry span.
func (bt *BusinessTracer) TraceMarketDataCollection(ctx context.Context, exchange string, symbols []string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "market_data_collection")
	span.SetTag("exchange", exchange)
	span.SetData("symbols", symbols)
	return ctx, span
}

// RecordMarketDataMetrics records metrics related to market data collection onto a span.
//
// Parameters:
//   - span: The Sentry span to update.
//   - metrics: The collection metrics to record.
func (bt *BusinessTracer) RecordMarketDataMetrics(span *sentry.Span, metrics MarketDataMetrics) {
	span.SetData("collected_count", metrics.CollectedCount)
	span.SetData("failed_count", metrics.FailedCount)
	span.SetData("collection_time_ms", metrics.CollectionTime.Milliseconds())
	span.SetData("success_rate", metrics.SuccessRate)
	span.SetTag("data_quality", metrics.DataQuality)
}

// TraceRiskAssessment starts a span for tracing risk assessment operations.
//
// Parameters:
//   - ctx: The context to attach the span to.
//   - assessmentType: The type of risk assessment being performed.
//   - symbol: The symbol being assessed.
//
// Returns:
//   - A context containing the new span.
//   - The created Sentry span.
func (bt *BusinessTracer) TraceRiskAssessment(ctx context.Context, assessmentType string, symbol string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "risk_assessment")
	span.SetTag("assessment_type", assessmentType)
	span.SetTag("symbol", symbol)
	return ctx, span
}

// RecordRiskMetrics records risk assessment metrics onto a span.
//
// Parameters:
//   - span: The Sentry span to update.
//   - metrics: The risk metrics to record.
func (bt *BusinessTracer) RecordRiskMetrics(span *sentry.Span, metrics RiskMetrics) {
	span.SetData("risk_score", metrics.RiskScore)
	span.SetTag("risk_level", metrics.RiskLevel)
	span.SetData("volatility", metrics.Volatility)
	span.SetData("max_drawdown", metrics.MaxDrawdown)
	span.SetData("is_acceptable", metrics.IsAcceptable)
}

// TraceNotification starts a span for tracing notification delivery.
//
// Parameters:
//   - ctx: The context to attach the span to.
//   - notificationType: The type of notification being sent.
//   - channel: The delivery channel (e.g., "telegram", "email").
//
// Returns:
//   - A context containing the new span.
//   - The created Sentry span.
func (bt *BusinessTracer) TraceNotification(ctx context.Context, notificationType string, channel string) (context.Context, *sentry.Span) {
	span := sentry.StartSpan(ctx, "notification")
	span.SetTag("notification_type", notificationType)
	span.SetTag("channel", channel)
	return ctx, span
}

// RecordNotificationResult records the outcome of a notification attempt onto a span.
//
// Parameters:
//   - span: The Sentry span to update.
//   - success: Whether the notification was sent successfully.
//   - recipientCount: The number of recipients.
//   - err: Any error that occurred during sending.
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

// ArbitrageOpportunity defines the structure for tracking arbitrage opportunity details in telemetry.
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

// SignalMetrics defines the structure for tracking signal processing statistics in telemetry.
type SignalMetrics struct {
	ProcessedCount  int
	ValidCount      int
	InvalidCount    int
	AverageStrength float64
	ProcessingTime  time.Duration
	QualityGrade    string
}

// TechnicalAnalysisResult defines the structure for tracking technical analysis outcomes in telemetry.
type TechnicalAnalysisResult struct {
	Value           float64
	SignalDirection string
	Confidence      float64
	DataPoints      int
	IsValid         bool
}

// MarketDataMetrics defines the structure for tracking market data collection statistics in telemetry.
type MarketDataMetrics struct {
	CollectedCount int
	FailedCount    int
	CollectionTime time.Duration
	SuccessRate    float64
	DataQuality    string
}

// RiskMetrics defines the structure for tracking risk assessment results in telemetry.
type RiskMetrics struct {
	RiskScore    float64
	RiskLevel    string
	Volatility   float64
	MaxDrawdown  float64
	IsAcceptable bool
}
