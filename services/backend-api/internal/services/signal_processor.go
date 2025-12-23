package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/shopspring/decimal"

	"github.com/irfandi/celebrum-ai-go/internal/models"
	"github.com/irfandi/celebrum-ai-go/internal/observability"
)

// SignalProcessorConfig holds configuration settings for the signal processor service.
type SignalProcessorConfig struct {
	BatchSize            int                  `json:"batch_size"`
	ProcessingInterval   time.Duration        `json:"processing_interval"`
	MaxConcurrentBatch   int                  `json:"max_concurrent_batch"`
	QualityThreshold     float64              `json:"quality_threshold"`
	DeduplicationWindow  time.Duration        `json:"deduplication_window"`
	RateLimitPerSecond   int                  `json:"rate_limit_per_second"`
	RetryAttempts        int                  `json:"retry_attempts"`
	RetryDelay           time.Duration        `json:"retry_delay"`
	TimeoutDuration      time.Duration        `json:"timeout_duration"`
	SignalTTL            time.Duration        `json:"signal_ttl"`
	NotificationEnabled  bool                 `json:"notification_enabled"`
	CircuitBreakerConfig CircuitBreakerConfig `json:"circuit_breaker"`
}

// SignalProcessor orchestrates the entire signal processing pipeline.
// It retrieves market data, generates signals, aggregates them, assesses quality, and triggers notifications.
type SignalProcessor struct {
	config              *SignalProcessorConfig
	db                  DBPool
	logger              *slog.Logger
	signalAggregator    SignalAggregatorInterface
	qualityScorer       SignalQualityScorerInterface
	technicalAnalysis   *TechnicalAnalysisService
	notificationService *NotificationService
	collectorService    *CollectorService
	circuitBreaker      *CircuitBreaker

	// Processing state
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	running        bool
	mu             sync.RWMutex
	metrics        *ProcessingMetrics
	lastRun        time.Time
	errorCount     int
	rateLimitCache map[string]time.Time
}

// ProcessingMetrics tracks performance statistics of the signal processing pipeline.
type ProcessingMetrics struct {
	TotalSignalsProcessed  int64     `json:"total_signals_processed"`
	SuccessfulSignals      int64     `json:"successful_signals"`
	FailedSignals          int64     `json:"failed_signals"`
	QualityFilteredSignals int64     `json:"quality_filtered_signals"`
	NotificationsSent      int64     `json:"notifications_sent"`
	AverageProcessingTime  float64   `json:"average_processing_time_ms"`
	LastProcessingTime     time.Time `json:"last_processing_time"`
	ErrorRate              float64   `json:"error_rate"`
	ThroughputPerMinute    float64   `json:"throughput_per_minute"`
}

// ProcessingResult represents the outcome of processing a single signal or batch.
type ProcessingResult struct {
	SignalID         string                 `json:"signal_id"`
	SignalType       SignalType             `json:"signal_type"`
	Symbol           string                 `json:"symbol"`
	Processed        bool                   `json:"processed"`
	QualityScore     float64                `json:"quality_score"`
	NotificationSent bool                   `json:"notification_sent"`
	Error            error                  `json:"error,omitempty"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NewSignalProcessor creates a new instance of SignalProcessor.
//
// Parameters:
//   - db: Connection pool to the database.
//   - logger: Logger instance.
//   - signalAggregator: Service for aggregating signals.
//   - qualityScorer: Service for scoring signal quality.
//   - technicalAnalysis: Service for technical analysis.
//   - notificationService: Service for sending notifications.
//   - collectorService: Service for collecting market data.
//   - circuitBreaker: Circuit breaker for fault tolerance.
//
// Returns:
//   - A pointer to the initialized SignalProcessor.
func NewSignalProcessor(
	db DBPool,
	logger *slog.Logger,
	signalAggregator SignalAggregatorInterface,
	qualityScorer SignalQualityScorerInterface,
	technicalAnalysis *TechnicalAnalysisService,
	notificationService *NotificationService,
	collectorService *CollectorService,
	circuitBreaker *CircuitBreaker,
) *SignalProcessor {
	config := &SignalProcessorConfig{
		BatchSize:           100,
		ProcessingInterval:  5 * time.Minute,
		MaxConcurrentBatch:  4,
		SignalTTL:           24 * time.Hour,
		QualityThreshold:    0.7,
		NotificationEnabled: true,
		RetryAttempts:       3,
		RetryDelay:          30 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SignalProcessor{
		config:              config,
		db:                  db,
		logger:              logger,
		signalAggregator:    signalAggregator,
		qualityScorer:       qualityScorer,
		technicalAnalysis:   technicalAnalysis,
		notificationService: notificationService,
		collectorService:    collectorService,
		circuitBreaker:      circuitBreaker,
		ctx:                 ctx,
		cancel:              cancel,
		metrics:             &ProcessingMetrics{},
	}
}

// Start begins the signal processing pipeline in a background goroutine.
//
// Returns:
//   - An error if the processor is already running.
func (sp *SignalProcessor) Start() error {
	ctx, span := observability.StartSpan(sp.ctx, observability.SpanOpSignalProcessing, "SignalProcessor.Start")
	defer observability.FinishSpan(span, nil)

	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.running {
		return fmt.Errorf("signal processor is already running")
	}

	sp.running = true
	sp.logger.Info("Starting signal processor",
		"interval", sp.config.ProcessingInterval,
		"batch_size", sp.config.BatchSize,
		"workers", sp.config.MaxConcurrentBatch)

	observability.AddBreadcrumbWithData(ctx, "signal_processor", "Starting signal processor", sentry.LevelInfo, map[string]interface{}{
		"interval":   sp.config.ProcessingInterval.String(),
		"batch_size": sp.config.BatchSize,
		"workers":    sp.config.MaxConcurrentBatch,
	})

	// Start the main processing loop
	sp.wg.Add(1)
	go sp.processingLoop()

	// Start metrics collection
	sp.wg.Add(1)
	go sp.metricsLoop()

	return nil
}

// Stop gracefully shuts down the signal processor, waiting for pending operations to complete.
//
// Returns:
//   - An error if the processor is not running.
func (sp *SignalProcessor) Stop() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.running {
		return fmt.Errorf("signal processor is not running")
	}

	sp.logger.Info("Stopping signal processor")
	sp.cancel()
	sp.wg.Wait()
	sp.running = false

	return nil
}

// IsRunning checks if the signal processor is currently active.
//
// Returns:
//   - True if running, false otherwise.
func (sp *SignalProcessor) IsRunning() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.running
}

// GetMetrics returns a snapshot of the current processing metrics.
//
// Returns:
//   - A pointer to the ProcessingMetrics struct.
func (sp *SignalProcessor) GetMetrics() *ProcessingMetrics {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.metrics
}

// processingLoop is the main loop that triggers signal processing at configured intervals.
func (sp *SignalProcessor) processingLoop() {
	defer sp.wg.Done()

	ticker := time.NewTicker(sp.config.ProcessingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sp.ctx.Done():
			sp.logger.Info("Processing loop stopped")
			return
		case <-ticker.C:
			if err := sp.processSignalBatch(); err != nil {
				sp.logger.Error("Error processing signal batch", "error", err)
				sp.incrementErrorCount()
			}
		}
	}
}

// processSignalBatch executes a single batch processing run with circuit breaker protection.
//
// Returns:
//   - An error if the batch processing fails or the circuit breaker is open.
func (sp *SignalProcessor) processSignalBatch() error {
	ctx, span := observability.StartSpanWithTags(sp.ctx, observability.SpanOpSignalProcessing, "SignalProcessor.processSignalBatch", map[string]string{
		"batch_size":        fmt.Sprintf("%d", sp.config.BatchSize),
		"max_concurrent":    fmt.Sprintf("%d", sp.config.MaxConcurrentBatch),
		"quality_threshold": fmt.Sprintf("%.2f", sp.config.QualityThreshold),
	})
	defer func() {
		observability.RecoverAndCapture(ctx, "processSignalBatch")
	}()

	startTime := time.Now()
	sp.lastRun = startTime

	// Use circuit breaker to protect against cascading failures
	err := sp.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		return sp.processSignalBatchWithRetry(ctx, startTime)
	})

	if err != nil {
		observability.CaptureExceptionWithContext(ctx, err, "signal_batch_processing", map[string]interface{}{
			"batch_size":     sp.config.BatchSize,
			"max_concurrent": sp.config.MaxConcurrentBatch,
			"duration_ms":    time.Since(startTime).Milliseconds(),
		})
		observability.FinishSpan(span, err)
		return err
	}

	span.SetData("duration_ms", time.Since(startTime).Milliseconds())
	observability.FinishSpan(span, nil)
	return nil
}

// processSignalBatchWithRetry attempts to process a signal batch with exponential backoff retries.
//
// Parameters:
//   - ctx: The context for the operation.
//   - startTime: The start time of the batch processing.
//
// Returns:
//   - An error if all retry attempts fail.
func (sp *SignalProcessor) processSignalBatchWithRetry(ctx context.Context, startTime time.Time) error {
	var lastErr error

	for attempt := 0; attempt < sp.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			sp.logger.Info("Retrying signal batch processing",
				"attempt", attempt+1,
				"max_attempts", sp.config.RetryAttempts,
				"last_error", lastErr)

			// Wait before retry with exponential backoff
			retryDelay := sp.config.RetryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
			}
		}

		err := sp.executeSignalBatch(ctx, startTime)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !sp.isRetryableError(err) {
			sp.logger.Error("Non-retryable error in signal processing", "error", err)
			return err
		}

		sp.logger.Warn("Retryable error in signal processing",
			"error", err,
			"attempt", attempt+1)
	}

	return fmt.Errorf("signal batch processing failed after %d attempts: %w", sp.config.RetryAttempts, lastErr)
}

// executeSignalBatch performs the core logic of fetching data, processing signals, and handling results.
//
// Parameters:
//   - ctx: The context for the operation.
//   - startTime: The start time of the batch execution.
//
// Returns:
//   - An error if any step of the batch execution fails.
func (sp *SignalProcessor) executeSignalBatch(ctx context.Context, startTime time.Time) error {
	// Stub logging for telemetry
	_ = fmt.Sprintf("Signal execution: timeout_duration=%s", sp.config.TimeoutDuration.String())

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, sp.config.TimeoutDuration)
	defer cancel()

	// Get market data for processing
	marketData, err := sp.getMarketDataForProcessingWithContext(timeoutCtx)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Failed to get market data: %v", err)
		return fmt.Errorf("failed to get market data: %w", err)
	}

	if len(marketData) == 0 {
		sp.logger.Debug("No market data available for processing")
		return nil
	}

	// Stub logging for market data count
	_ = fmt.Sprintf("Market data count: %d", len(marketData))

	// Process signals concurrently with timeout
	results, err := sp.processSignalsConcurrentlyWithContext(timeoutCtx, marketData)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Failed to process signals: %v", err)
		return fmt.Errorf("failed to process signals: %w", err)
	}

	// Store results and send notifications
	if err := sp.handleProcessingResultsWithContext(timeoutCtx, results); err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Failed to handle processing results: %v", err)
		return fmt.Errorf("failed to handle processing results: %w", err)
	}

	// Update metrics
	processingTime := time.Since(startTime)
	sp.updateMetrics(results, processingTime)

	// Add result tracking
	successfulCount := sp.countSuccessful(results)
	// Stub logging for results
	_ = fmt.Sprintf("Results tracking: total=%d, successful=%d, failed=%d, duration_ms=%d",
		len(results), successfulCount, len(results)-successfulCount, processingTime.Milliseconds())

	// Stub logging for results
	_ = fmt.Sprintf("Concurrent processing results: total=%d, successful=%d, failed=%d",
		len(results), successfulCount, len(results)-successfulCount)

	sp.logger.Info("Signal batch processed",
		"count", len(results),
		"duration", processingTime,
		"successful", successfulCount)

	return nil
}

// isRetryableError determines if a given error warrants a retry attempt.
func (sp *SignalProcessor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network-related errors are retryable
	if strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "network") ||
		strings.Contains(err.Error(), "temporary") {
		return true
	}

	// Database connection errors are retryable
	if strings.Contains(err.Error(), "database") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset") {
		return true
	}

	// Context cancellation and deadline exceeded are not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Rate limiting errors are retryable
	if strings.Contains(err.Error(), "rate limit") ||
		strings.Contains(err.Error(), "too many requests") {
		return true
	}

	// Validation errors are not retryable
	if strings.Contains(err.Error(), "validation") ||
		strings.Contains(err.Error(), "invalid") ||
		strings.Contains(err.Error(), "malformed") {
		return false
	}

	// Default to not retryable for unknown errors
	return false
}

// getMarketDataForProcessingWithContext retrieves market data respecting the provided context.
func (sp *SignalProcessor) getMarketDataForProcessingWithContext(ctx context.Context) ([]models.MarketData, error) {
	// Check context first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return sp.getMarketDataForProcessing()
}

// getMarketDataForProcessing retrieves recent market data for active trading pairs.
func (sp *SignalProcessor) getMarketDataForProcessing() ([]models.MarketData, error) {
	// Get active trading pairs
	tradingPairs, err := sp.getActiveTradingPairs()
	if err != nil {
		return nil, fmt.Errorf("failed to get trading pairs: %w", err)
	}

	var marketData []models.MarketData
	for _, pair := range tradingPairs {
		// Get recent market data for each pair
		data, err := sp.getRecentMarketDataFromDB(pair.Symbol, pair.Exchange.Name, time.Hour)
		if err != nil {
			sp.logger.Warn("Failed to get market data",
				"symbol", pair.Symbol,
				"exchange", pair.Exchange.Name,
				"error", err)
			continue
		}
		marketData = append(marketData, data...)
	}

	return marketData, nil
}

// processSignalsConcurrentlyWithContext processes signals concurrently, respecting the provided context.
func (sp *SignalProcessor) processSignalsConcurrentlyWithContext(ctx context.Context, marketData []models.MarketData) ([]ProcessingResult, error) {
	// Check context first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	results := sp.processSignalsConcurrently(marketData)
	return results, nil
}

// processSignalsConcurrently distributes market data processing across a pool of workers.
func (sp *SignalProcessor) processSignalsConcurrently(marketData []models.MarketData) []ProcessingResult {
	// Stub logging for telemetry
	workerCount := sp.config.MaxConcurrentBatch
	if workerCount <= 0 {
		workerCount = 4 // Default worker count
	}

	_ = fmt.Sprintf("Concurrent processing: worker_count=%d, market_data_count=%d",
		workerCount, len(marketData))

	jobs := make(chan models.MarketData, len(marketData))
	results := make(chan ProcessingResult, len(marketData))

	// Start workers
	var workerWg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		workerWg.Add(1)
		go sp.signalWorker(jobs, results, &workerWg)
	}

	// Send jobs
	for _, data := range marketData {
		jobs <- data
	}
	close(jobs)

	// Wait for workers to complete
	go func() {
		workerWg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []ProcessingResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Add result tracking
	successfulCount := 0
	for _, result := range allResults {
		if result.Error == nil {
			successfulCount++
		}
	}

	// Stub logging for concurrent results
	_ = fmt.Sprintf("All results: total=%d, successful=%d, failed=%d",
		len(allResults), successfulCount, len(allResults)-successfulCount)

	return allResults
}

// getRecentMarketDataFromDB retrieves market data for a specific symbol and exchange within a duration.
func (sp *SignalProcessor) getRecentMarketDataFromDB(symbol, exchange string, duration time.Duration) ([]models.MarketData, error) {
	since := time.Now().Add(-duration)

	query := `
		SELECT md.id, md.exchange_id, md.trading_pair_id, md.last_price, md.volume_24h, 
		       md.timestamp, md.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		JOIN exchanges e ON md.exchange_id = e.id
		WHERE tp.symbol = $1 AND e.ccxt_id = $2 AND md.timestamp >= $3
		ORDER BY md.timestamp DESC
		LIMIT 100
	`

	rows, err := sp.db.Query(context.Background(), query, symbol, exchange, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	var marketData []models.MarketData
	for rows.Next() {
		var data models.MarketData
		scanErr := rows.Scan(
			&data.ID,
			&data.ExchangeID,
			&data.TradingPairID,
			&data.LastPrice,
			&data.Volume24h,
			&data.Timestamp,
			&data.CreatedAt,
		)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan market data: %w", scanErr)
		}
		marketData = append(marketData, data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating market data rows: %w", err)
	}

	return marketData, nil
}

// getTradingPairSymbol retrieves the symbol string for a given trading pair ID.
func (sp *SignalProcessor) getTradingPairSymbol(tradingPairID int) (string, error) {
	var symbol string
	query := `SELECT symbol FROM trading_pairs WHERE id = $1`
	err := sp.db.QueryRow(context.Background(), query, tradingPairID).Scan(&symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get trading pair symbol: %w", err)
	}
	return symbol, nil
}

// getExchangeName retrieves the exchange name string for a given exchange ID.
func (sp *SignalProcessor) getExchangeName(exchangeID int) (string, error) {
	var name string
	query := `SELECT name FROM exchanges WHERE id = $1`
	err := sp.db.QueryRow(context.Background(), query, exchangeID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("failed to get exchange name: %w", err)
	}
	return name, nil
}

// signalWorker consumes market data jobs and produces processing results.
func (sp *SignalProcessor) signalWorker(jobs <-chan models.MarketData, results chan<- ProcessingResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for data := range jobs {
		result := sp.processSignal(data)
		results <- result
	}
}

// processSignal analyzes a single market data point to generate, aggregate, and score signals.
// It covers arbitrage, technical analysis, aggregation, and quality assessment.
func (sp *SignalProcessor) processSignal(data models.MarketData) ProcessingResult {
	// Stub logging for telemetry
	_ = fmt.Sprintf("Signal processing: trading_pair_id=%d, exchange_id=%d, price=%s, volume=%s",
		data.TradingPairID, data.ExchangeID, data.LastPrice.String(), data.Volume24h.String())

	startTime := time.Now()

	// Get trading pair symbol from database
	symbol, err := sp.getTradingPairSymbol(data.TradingPairID)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Failed to get trading pair symbol: %v", err)
		result := ProcessingResult{
			SignalID:       fmt.Sprintf("unknown_%d_%d", data.TradingPairID, time.Now().Unix()),
			Symbol:         "unknown",
			ProcessingTime: time.Since(startTime),
			Metadata:       make(map[string]interface{}),
			Error:          fmt.Errorf("failed to get trading pair symbol: %w", err),
		}
		return result
	}

	// Get exchange name from database
	exchangeName, err := sp.getExchangeName(data.ExchangeID)
	if err != nil {
		exchangeName = "unknown"
	}

	// Stub logging for symbol and exchange
	_ = fmt.Sprintf("Processing symbol: %s, exchange: %s", symbol, exchangeName)

	result := ProcessingResult{
		SignalID:       fmt.Sprintf("%s_%s_%d", symbol, exchangeName, time.Now().Unix()),
		Symbol:         symbol,
		ProcessingTime: 0,
		Metadata:       make(map[string]interface{}),
	}

	// Generate arbitrage signals
	arbitrageSignals, err := sp.generateArbitrageSignals(data)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Arbitrage signal generation failed: %v", err)
		result.Error = fmt.Errorf("arbitrage signal generation failed: %w", err)
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	// Generate technical analysis signals
	technicalSignals, err := sp.generateTechnicalSignals(data)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Technical signal generation failed: %v", err)
		result.Error = fmt.Errorf("technical signal generation failed: %w", err)
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	// Aggregate signals
	aggregatedSignal, err := sp.aggregateSignals(arbitrageSignals, technicalSignals, data)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Signal aggregation failed: %v", err)
		result.Error = fmt.Errorf("signal aggregation failed: %w", err)
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	// Assess signal quality
	qualityScore, err := sp.assessSignalQuality(aggregatedSignal, data)
	if err != nil {
		// Stub logging for error
		_ = fmt.Sprintf("Quality assessment failed: %v", err)
		result.Error = fmt.Errorf("quality assessment failed: %w", err)
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	result.QualityScore = qualityScore
	result.SignalType = aggregatedSignal.SignalType
	result.Processed = true
	result.ProcessingTime = time.Since(startTime)
	result.Metadata["aggregated_signal"] = aggregatedSignal

	// Stub logging for final results
	_ = fmt.Sprintf("Signal processing completed: arbitrage_count=%d, technical_count=%d, signal_type=%s, quality_score=%f, duration_ms=%d",
		len(arbitrageSignals), len(technicalSignals), string(aggregatedSignal.SignalType), qualityScore, result.ProcessingTime.Milliseconds())

	return result
}

// generateArbitrageSignals identifies potential arbitrage opportunities for a symbol.
func (sp *SignalProcessor) generateArbitrageSignals(data models.MarketData) ([]ArbitrageSignalInput, error) {
	// Get trading pair symbol
	symbol, err := sp.getTradingPairSymbol(data.TradingPairID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading pair symbol: %w", err)
	}

	// Get arbitrage opportunities for this symbol
	opportunities, err := sp.getArbitrageOpportunities(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get arbitrage opportunities: %w", err)
	}

	var signals []ArbitrageSignalInput
	if len(opportunities) > 0 {
		// Create a single ArbitrageSignalInput with all opportunities
		signal := ArbitrageSignalInput{
			Opportunities: opportunities,
			MinVolume:     decimal.NewFromFloat(1000.0),  // Default minimum volume
			BaseAmount:    decimal.NewFromFloat(10000.0), // Default base amount
		}
		signals = append(signals, signal)
	}

	return signals, nil
}

// getArbitrageOpportunities queries the database for active arbitrage opportunities for a symbol.
func (sp *SignalProcessor) getArbitrageOpportunities(symbol string) ([]models.ArbitrageOpportunity, error) {
	// Query the database for arbitrage opportunities
	query := `
		SELECT ao.id, ao.trading_pair_id, ao.buy_exchange_id, ao.sell_exchange_id, 
		       ao.buy_price, ao.sell_price, ao.profit_percentage, ao.detected_at, ao.expires_at
		FROM arbitrage_opportunities ao
		JOIN trading_pairs tp ON ao.trading_pair_id = tp.id
		WHERE tp.symbol = $1 AND ao.expires_at > NOW()
		ORDER BY ao.profit_percentage DESC
		LIMIT 10
	`

	rows, err := sp.db.Query(context.Background(), query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query arbitrage opportunities: %w", err)
	}
	defer rows.Close()

	var opportunities []models.ArbitrageOpportunity
	for rows.Next() {
		var opp models.ArbitrageOpportunity
		err := rows.Scan(&opp.ID, &opp.TradingPairID, &opp.BuyExchangeID, &opp.SellExchangeID,
			&opp.BuyPrice, &opp.SellPrice, &opp.ProfitPercentage, &opp.DetectedAt, &opp.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan arbitrage opportunity: %w", err)
		}
		opportunities = append(opportunities, opp)
	}

	return opportunities, nil
}

// generateTechnicalSignals prepares technical signal input from market data.
func (sp *SignalProcessor) generateTechnicalSignals(data models.MarketData) ([]TechnicalSignalInput, error) {
	// Get trading pair symbol and exchange name
	symbol, err := sp.getTradingPairSymbol(data.TradingPairID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trading pair symbol: %w", err)
	}

	exchangeName, err := sp.getExchangeName(data.ExchangeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange name: %w", err)
	}

	var signals []TechnicalSignalInput

	// Create technical signal input with price data
	signal := TechnicalSignalInput{
		Symbol:     symbol,
		Exchange:   exchangeName,
		Prices:     []decimal.Decimal{data.LastPrice},
		Volumes:    []decimal.Decimal{data.Volume24h},
		Timestamps: []time.Time{data.Timestamp},
	}

	signals = append(signals, signal)
	return signals, nil
}

// aggregateSignals combines separate arbitrage and technical signals into a unified signal.
func (sp *SignalProcessor) aggregateSignals(arbitrageSignals []ArbitrageSignalInput, technicalSignals []TechnicalSignalInput, data models.MarketData) (*AggregatedSignal, error) {
	// Use market data to enhance signal aggregation
	marketVolatility := sp.calculateMarketVolatility(data)
	marketTrend := sp.calculateMarketTrend(data)

	_ = fmt.Sprintf("Signal aggregation: arbitrage_count=%d, technical_count=%d, market_volatility=%.2f, market_trend=%s",
		len(arbitrageSignals), len(technicalSignals), marketVolatility, marketTrend)

	// Use the signal aggregator to combine signals
	if len(arbitrageSignals) > 0 {
		aggregatedSignals, err := sp.signalAggregator.AggregateArbitrageSignals(sp.ctx, arbitrageSignals[0])
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate arbitrage signals: %w", err)
		}
		if len(aggregatedSignals) > 0 {
			signal := aggregatedSignals[0]

			// If we have technical signals, incorporate them
			if len(technicalSignals) > 0 {
				// Enhance the aggregated signal with technical analysis
				for _, techSignal := range technicalSignals {
					// Add technical indicators to the aggregated signal
					if signal.Metadata == nil {
						signal.Metadata = make(map[string]interface{})
					}
					// Store technical signal metadata
					signal.Metadata["technical_symbol"] = techSignal.Symbol
					signal.Metadata["technical_exchange"] = techSignal.Exchange

					// Adjust confidence based on technical analysis alignment
					confidence, _ := signal.Confidence.Float64()
					newConfidence := confidence * 1.1 // Slight boost for technical alignment
					if newConfidence > 1.0 {
						newConfidence = 1.0
					}
					signal.Confidence = decimal.NewFromFloat(newConfidence)
				}
			}

			return signal, nil
		}
	}

	// If no arbitrage signals, try technical signals only
	if len(technicalSignals) > 0 {
		aggregatedSignals, err := sp.signalAggregator.AggregateTechnicalSignals(sp.ctx, technicalSignals[0])
		if err != nil {
			return nil, fmt.Errorf("failed to aggregate technical signals: %w", err)
		}
		if len(aggregatedSignals) > 0 {
			return aggregatedSignals[0], nil
		}
	}

	return nil, fmt.Errorf("no signals to aggregate")
}

// calculateMarketVolatility estimates volatility based on recent market data range or spread.
func (sp *SignalProcessor) calculateMarketVolatility(data models.MarketData) float64 {
	// Calculate volatility based on available market data
	if data.LastPrice.IsZero() {
		return 0.02 // Default 2% volatility
	}

	// Use bid-ask spread as a volatility indicator
	if !data.Bid.IsZero() && !data.Ask.IsZero() {
		spread := data.Ask.Sub(data.Bid).Div(data.LastPrice)
		spreadFloat, _ := spread.Float64()

		// Convert spread to volatility (higher spread = higher volatility)
		return math.Max(0.005, spreadFloat*10) // Minimum 0.5% volatility
	}

	// Use 24h high-low range as volatility indicator
	if !data.High24h.IsZero() && !data.Low24h.IsZero() {
		rangePercent := data.High24h.Sub(data.Low24h).Div(data.LastPrice)
		rangeFloat, _ := rangePercent.Float64()
		return math.Max(0.01, rangeFloat/4) // Scale down the range
	}

	return 0.02 // Default volatility
}

// calculateMarketTrend determines the market trend (bullish/bearish/neutral) from recent price action.
func (sp *SignalProcessor) calculateMarketTrend(data models.MarketData) string {
	if data.LastPrice.IsZero() || data.High24h.IsZero() || data.Low24h.IsZero() {
		return "neutral"
	}

	// Simple trend calculation using current price vs 24h range
	lastPriceFloat, _ := data.LastPrice.Float64()
	highFloat, _ := data.High24h.Float64()
	lowFloat, _ := data.Low24h.Float64()

	// Calculate position in 24h range
	rangeSize := highFloat - lowFloat
	if rangeSize == 0 {
		return "neutral"
	}

	position := (lastPriceFloat - lowFloat) / rangeSize

	// Determine trend based on position in range
	if position > 0.7 {
		return "bullish"
	} else if position < 0.3 {
		return "bearish"
	}
	return "neutral"
}

// assessSignalQuality evaluates the quality of an aggregated signal using the quality scorer service.
func (sp *SignalProcessor) assessSignalQuality(signal *AggregatedSignal, data models.MarketData) (float64, error) {
	// Use the comprehensive quality scorer for detailed assessment
	volumeFloat := sp.extractVolumeFromMetadata(signal.Metadata)
	indicatorsFloat := sp.extractIndicatorsFromMetadata(signal.Metadata)
	indicatorsInterface := make(map[string]interface{})
	for k, v := range indicatorsFloat {
		indicatorsInterface[k] = v
	}

	qualityInput := &SignalQualityInput{
		SignalType:       string(signal.SignalType),
		Symbol:           signal.Symbol,
		Exchanges:        signal.Exchanges,
		Volume:           decimal.NewFromFloat(volumeFloat),
		ProfitPotential:  signal.ProfitPotential,
		Confidence:       signal.Confidence,
		Timestamp:        signal.CreatedAt,
		Indicators:       indicatorsInterface,
		MarketData:       sp.convertToMarketDataSnapshot(&data),
		SignalComponents: sp.extractSignalComponents(signal.Metadata),
		SignalCount:      sp.extractSignalCount(signal.Metadata),
	}

	// Get comprehensive quality metrics
	qualityMetrics, err := sp.qualityScorer.AssessSignalQuality(sp.ctx, qualityInput)
	if err != nil {
		sp.logger.Warn("Failed to assess signal quality, using fallback",
			"signal_id", signal.ID,
			"error", err)
		return sp.fallbackQualityAssessment(signal), nil
	}

	// Apply additional filtering rules
	if !sp.passesAdvancedFiltering(signal, qualityMetrics) {
		sp.logger.Debug("Signal failed advanced filtering",
			"signal_id", signal.ID,
			"overall_score", qualityMetrics.OverallScore)
		return 0.0, nil
	}

	overallScore, _ := qualityMetrics.OverallScore.Float64()
	return overallScore, nil
}

// getActiveTradingPairs retrieves active trading pairs from the database.
func (sp *SignalProcessor) getActiveTradingPairs() ([]models.TradingPair, error) {
	query := `
		SELECT tp.id, tp.symbol, tp.is_active, tp.created_at, tp.updated_at, e.id as exchange_id, e.name as exchange_name
		FROM trading_pairs tp
		JOIN exchanges e ON tp.exchange_id = e.id
		WHERE tp.is_active = true
		ORDER BY tp.volume_24h DESC
		LIMIT $1
	`

	rows, err := sp.db.Query(context.Background(), query, sp.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to query trading pairs: %w", err)
	}
	defer rows.Close()

	var pairs []models.TradingPair
	for rows.Next() {
		var pair models.TradingPair
		var exchangeID int
		var exchangeName string
		err := rows.Scan(&pair.ID, &pair.Symbol, &pair.IsActive, &pair.CreatedAt, &pair.UpdatedAt, &exchangeID, &exchangeName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trading pair: %w", err)
		}
		pair.Exchange = models.Exchange{
			ID:   exchangeID,
			Name: exchangeName,
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}

// handleProcessingResultsWithContext processes and filters results, respecting the provided context.
func (sp *SignalProcessor) handleProcessingResultsWithContext(ctx context.Context, results []ProcessingResult) error {
	// Check context first
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return sp.handleProcessingResults(results)
}

// handleProcessingResults deduplicates, filters, stores, and notifies for the processed results.
func (sp *SignalProcessor) handleProcessingResults(results []ProcessingResult) error {
	// Apply deduplication across all results
	deduplicatedResults := sp.deduplicateResults(results)

	// Apply batch quality filtering
	filteredResults := sp.applyBatchQualityFiltering(deduplicatedResults)

	for _, result := range filteredResults {
		if result.Error != nil {
			sp.logger.Warn("Signal processing failed",
				"signal_id", result.SignalID,
				"symbol", result.Symbol,
				"error", result.Error)
			sp.metrics.FailedSignals++
			continue
		}

		// Check quality threshold
		if result.QualityScore < sp.config.QualityThreshold {
			sp.logger.Debug("Signal filtered due to low quality",
				"signal_id", result.SignalID,
				"quality_score", result.QualityScore,
				"threshold", sp.config.QualityThreshold)
			sp.metrics.QualityFilteredSignals++
			continue
		}

		// Apply rate limiting per symbol
		if !sp.passesRateLimiting(result) {
			sp.logger.Debug("Signal filtered due to rate limiting",
				"signal_id", result.SignalID,
				"symbol", result.Symbol)
			continue
		}

		// Store signal in database
		if err := sp.storeAggregatedSignal(result); err != nil {
			sp.logger.Error("Failed to store aggregated signal",
				"signal_id", result.SignalID,
				"error", err)
			sp.metrics.FailedSignals++
			continue
		}

		sp.metrics.SuccessfulSignals++

		// Send notification if enabled
		if sp.config.NotificationEnabled {
			if err := sp.sendSignalNotification(result); err != nil {
				sp.logger.Error("Failed to send signal notification",
					"signal_id", result.SignalID,
					"error", err)
			} else {
				result.NotificationSent = true
				sp.metrics.NotificationsSent++
			}
		}
	}

	return nil
}

// storeAggregatedSignal persists the aggregated signal to the database.
func (sp *SignalProcessor) storeAggregatedSignal(result ProcessingResult) error {
	aggregatedSignal, ok := result.Metadata["aggregated_signal"].(*AggregatedSignal)
	if !ok {
		return fmt.Errorf("invalid aggregated signal in result metadata")
	}

	query := `
		INSERT INTO aggregated_signals (
			id, signal_type, symbol, action, strength, confidence,
			profit_potential, risk_level, exchanges, indicators,
			metadata, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	expiresAt := time.Now().Add(sp.config.SignalTTL)
	indicatorsJSON, _ := json.Marshal(aggregatedSignal.Indicators)
	metadataJSON, _ := json.Marshal(aggregatedSignal.Metadata)
	exchangesJSON, _ := json.Marshal(aggregatedSignal.Exchanges)

	strengthStr := string(aggregatedSignal.Strength)
	confidenceFloat, _ := aggregatedSignal.Confidence.Float64()
	riskLevelFloat, _ := aggregatedSignal.RiskLevel.Float64()
	profitPotentialFloat, _ := aggregatedSignal.ProfitPotential.Float64()

	_, err := sp.db.Exec(context.Background(), query,
		result.SignalID,
		aggregatedSignal.SignalType,
		aggregatedSignal.Symbol,
		aggregatedSignal.Action,
		strengthStr,
		confidenceFloat,
		profitPotentialFloat,
		riskLevelFloat,
		exchangesJSON,
		indicatorsJSON,
		metadataJSON,
		time.Now(),
		expiresAt,
	)

	return err
}

// sendSignalNotification triggers a notification for high-quality signals.
func (sp *SignalProcessor) sendSignalNotification(result ProcessingResult) error {
	_, ok := result.Metadata["aggregated_signal"].(*AggregatedSignal)
	if !ok {
		return fmt.Errorf("invalid aggregated signal in result metadata")
	}

	// Convert to notification format and send (placeholder - implement based on notification service interface)
	// return sp.notificationService.NotifyAggregatedSignal(aggregatedSignal)
	return nil // Temporary placeholder
}

// metricsLoop periodically updates internal throughput metrics.
func (sp *SignalProcessor) metricsLoop() {
	defer sp.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sp.ctx.Done():
			return
		case <-ticker.C:
			sp.updateThroughputMetrics()
		}
	}
}

// Helper methods for enhanced signal quality filtering and deduplication

// extractVolumeFromMetadata parses volume information from signal metadata.
func (sp *SignalProcessor) extractVolumeFromMetadata(metadata map[string]interface{}) float64 {
	if volume, ok := metadata["volume"]; ok {
		if v, ok := volume.(float64); ok {
			return v
		}
		if v, ok := volume.(string); ok {
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed
			}
		}
	}
	return 0.0
}

// extractIndicatorsFromMetadata parses technical indicator values from signal metadata.
func (sp *SignalProcessor) extractIndicatorsFromMetadata(metadata map[string]interface{}) map[string]float64 {
	indicators := make(map[string]float64)

	if indicatorsData, ok := metadata["indicators"]; ok {
		if indicatorsMap, ok := indicatorsData.(map[string]interface{}); ok {
			for key, value := range indicatorsMap {
				if v, ok := value.(float64); ok {
					indicators[key] = v
				} else if v, ok := value.(string); ok {
					if parsed, err := strconv.ParseFloat(v, 64); err == nil {
						indicators[key] = parsed
					}
				}
			}
		}
	}

	return indicators
}

// convertToMarketDataSnapshot converts internal MarketData to the snapshot format used by the quality scorer.
func (sp *SignalProcessor) convertToMarketDataSnapshot(data *models.MarketData) *MarketDataSnapshot {
	if data == nil {
		return nil
	}

	return &MarketDataSnapshot{
		Price:          data.LastPrice,
		Volume24h:      data.Volume24h,
		PriceChange24h: data.Change24h,
		Volatility:     decimal.Zero, // Calculate if needed
		Spread:         data.Ask.Sub(data.Bid),
		OrderBookDepth: decimal.Zero, // Calculate if needed
		LastTradeTime:  data.Timestamp,
	}
}

// extractSignalComponents parses the list of signal components from metadata.
func (sp *SignalProcessor) extractSignalComponents(metadata map[string]interface{}) []string {
	var components []string

	if componentsData, ok := metadata["components"]; ok {
		if componentsList, ok := componentsData.([]interface{}); ok {
			for _, component := range componentsList {
				if comp, ok := component.(string); ok {
					components = append(components, comp)
				}
			}
		}
	}

	return components
}

// extractSignalCount parses the count of signals contributing to an aggregation from metadata.
func (sp *SignalProcessor) extractSignalCount(metadata map[string]interface{}) int {
	if count, ok := metadata["signal_count"]; ok {
		if c, ok := count.(int); ok {
			return c
		}
		if c, ok := count.(float64); ok {
			return int(c)
		}
	}
	return 1
}

// fallbackQualityAssessment provides a simplified quality score when the detailed scorer fails.
func (sp *SignalProcessor) fallbackQualityAssessment(signal *AggregatedSignal) float64 {
	baseQuality := 0.5

	// Adjust based on signal action
	switch signal.Action {
	case "BUY", "SELL":
		baseQuality += 0.1
	default:
		baseQuality -= 0.1
	}

	// Adjust based on confidence
	confidence, _ := signal.Confidence.Float64()
	if confidence > 0.8 {
		baseQuality += 0.2
	} else if confidence < 0.5 {
		baseQuality -= 0.2
	}

	// Ensure quality is between 0 and 1
	if baseQuality > 1.0 {
		baseQuality = 1.0
	}
	if baseQuality < 0.0 {
		baseQuality = 0.0
	}

	return baseQuality
}

// passesAdvancedFiltering applies additional checks on signal metrics to filter low-quality signals.
func (sp *SignalProcessor) passesAdvancedFiltering(signal *AggregatedSignal, metrics *SignalQualityMetrics) bool {
	// Check minimum exchange score
	exchangeScore, _ := metrics.ExchangeScore.Float64()
	if exchangeScore < 0.3 {
		return false
	}

	// Check minimum volume score
	volumeScore, _ := metrics.VolumeScore.Float64()
	if volumeScore < 0.2 {
		return false
	}

	// Check data freshness
	dataFreshnessScore, _ := metrics.DataFreshnessScore.Float64()
	if dataFreshnessScore < 0.5 {
		return false
	}

	// Check if signal is too old
	if time.Since(signal.CreatedAt) > 30*time.Minute {
		return false
	}

	// Check profit potential threshold
	profitPotential, _ := signal.ProfitPotential.Float64()
	if profitPotential < 0.01 { // Less than 1% profit potential
		return false
	}

	return true
}

// deduplicateResults removes processing results that are duplicates based on symbol and signal type.
func (sp *SignalProcessor) deduplicateResults(results []ProcessingResult) []ProcessingResult {
	seen := make(map[string]bool)
	var deduplicated []ProcessingResult

	for _, result := range results {
		key := fmt.Sprintf("%s_%s", result.Symbol, result.SignalType)
		if !seen[key] {
			seen[key] = true
			deduplicated = append(deduplicated, result)
		}
	}

	return deduplicated
}

// applyBatchQualityFiltering filters out results that do not meet the quality threshold from a batch.
func (sp *SignalProcessor) applyBatchQualityFiltering(results []ProcessingResult) []ProcessingResult {
	var filtered []ProcessingResult

	for _, result := range results {
		// Skip results with errors
		if result.Error != nil {
			filtered = append(filtered, result)
			continue
		}

		// Apply quality threshold
		if result.QualityScore >= sp.config.QualityThreshold {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// passesRateLimiting checks if a signal should be processed based on rate limiting rules.
func (sp *SignalProcessor) passesRateLimiting(result ProcessingResult) bool {
	// Implement actual rate limiting logic
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Initialize rate limiting map if not exists
	if sp.rateLimitCache == nil {
		sp.rateLimitCache = make(map[string]time.Time)
	}

	// Create rate limit key based on symbol and signal type
	rateLimitKey := fmt.Sprintf("%s:%s", result.Symbol, result.SignalType)

	// Check if this symbol/signal type was processed recently
	if lastProcessedTime, exists := sp.rateLimitCache[rateLimitKey]; exists {
		// Calculate minimum time between signals (1 minute for same symbol/type)
		minInterval := 1 * time.Minute

		if time.Since(lastProcessedTime) < minInterval {
			return false // Rate limited
		}
	}

	// Update the last processed time
	sp.rateLimitCache[rateLimitKey] = time.Now()

	// Clean up old entries (older than 5 minutes)
	cleanupThreshold := 5 * time.Minute
	for key, timestamp := range sp.rateLimitCache {
		if time.Since(timestamp) > cleanupThreshold {
			delete(sp.rateLimitCache, key)
		}
	}

	return true
}

// Helper methods for metrics and processing...

// incrementErrorCount increases the internal error counter thread-safely.
func (sp *SignalProcessor) incrementErrorCount() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.errorCount++
}

// updateMetrics updates the processing metrics with results from a batch execution.
func (sp *SignalProcessor) updateMetrics(results []ProcessingResult, processingTime time.Duration) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.metrics.TotalSignalsProcessed += int64(len(results))
	sp.metrics.LastProcessingTime = time.Now()
	sp.metrics.AverageProcessingTime = float64(processingTime.Nanoseconds()) / 1e6

	for _, result := range results {
		if result.Error != nil {
			sp.metrics.FailedSignals++
		} else {
			sp.metrics.SuccessfulSignals++
		}

		if result.QualityScore < sp.config.QualityThreshold {
			sp.metrics.QualityFilteredSignals++
		}

		if result.NotificationSent {
			sp.metrics.NotificationsSent++
		}
	}

	// Calculate error rate
	if sp.metrics.TotalSignalsProcessed > 0 {
		sp.metrics.ErrorRate = float64(sp.metrics.FailedSignals) / float64(sp.metrics.TotalSignalsProcessed)
	}
}

// updateThroughputMetrics calculates and updates the throughput per minute metric.
func (sp *SignalProcessor) updateThroughputMetrics() {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Calculate throughput per minute based on recent activity
	if !sp.lastRun.IsZero() {
		minutesSinceLastRun := time.Since(sp.lastRun).Minutes()
		if minutesSinceLastRun > 0 {
			sp.metrics.ThroughputPerMinute = float64(sp.metrics.TotalSignalsProcessed) / minutesSinceLastRun
		}
	}
}

// countSuccessful counts the number of successful results in a batch.
func (sp *SignalProcessor) countSuccessful(results []ProcessingResult) int {
	count := 0
	for _, result := range results {
		if result.Error == nil {
			count++
		}
	}
	return count
}

// GetDefaultSignalProcessorConfig provides a default configuration for the SignalProcessor.
func GetDefaultSignalProcessorConfig() *SignalProcessorConfig {
	return &SignalProcessorConfig{
		ProcessingInterval:  time.Minute * 5,
		BatchSize:           100,
		MaxConcurrentBatch:  4,
		SignalTTL:           time.Hour * 24,
		QualityThreshold:    0.7,
		NotificationEnabled: true,
		RetryAttempts:       3,
		RetryDelay:          time.Second * 30,
	}
}
