package services

import (
	"context"
	"fmt"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/models"

	"github.com/cinar/indicator/v2/asset"
	"github.com/cinar/indicator/v2/helper"
	"github.com/cinar/indicator/v2/momentum"
	"github.com/cinar/indicator/v2/trend"
	"github.com/cinar/indicator/v2/volatility"
	"github.com/cinar/indicator/v2/volume"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// TechnicalAnalysisService provides technical analysis capabilities
type TechnicalAnalysisService struct {
	config               *config.Config
	db                   *database.PostgresDB
	logger               *logrus.Logger
	errorRecoveryManager *ErrorRecoveryManager
	resourceManager      *ResourceManager
	performanceMonitor   *PerformanceMonitor
}

// IndicatorResult represents the result of a technical indicator calculation
type IndicatorResult struct {
	Name      string            `json:"name"`
	Values    []decimal.Decimal `json:"values"`
	Signal    string            `json:"signal"` // "buy", "sell", "hold"
	Strength  decimal.Decimal   `json:"strength"`
	Timestamp time.Time         `json:"timestamp"`
}

// TechnicalAnalysisResult contains all calculated indicators for a symbol
type TechnicalAnalysisResult struct {
	Symbol        string             `json:"symbol"`
	Exchange      string             `json:"exchange"`
	Timeframe     string             `json:"timeframe"`
	Indicators    []*IndicatorResult `json:"indicators"`
	OverallSignal string             `json:"overall_signal"`
	Confidence    decimal.Decimal    `json:"confidence"`
	CalculatedAt  time.Time          `json:"calculated_at"`
}

// PriceData represents historical price data for analysis
type PriceData struct {
	Symbol     string            `json:"symbol"`
	Exchange   string            `json:"exchange"`
	Open       []decimal.Decimal `json:"open"`
	High       []decimal.Decimal `json:"high"`
	Low        []decimal.Decimal `json:"low"`
	Close      []decimal.Decimal `json:"close"`
	Volume     []decimal.Decimal `json:"volume"`
	Timestamps []time.Time       `json:"timestamps"`
}

// IndicatorConfig holds configuration for technical indicators
type IndicatorConfig struct {
	// Moving Averages
	SMAPeriods []int `json:"sma_periods"`
	EMAPeriods []int `json:"ema_periods"`

	// Momentum Indicators
	RSIPeriod    int `json:"rsi_period"`
	StochKPeriod int `json:"stoch_k_period"`
	StochDPeriod int `json:"stoch_d_period"`

	// Trend Indicators
	MACDFast   int `json:"macd_fast"`
	MACDSlow   int `json:"macd_slow"`
	MACDSignal int `json:"macd_signal"`

	// Volatility Indicators
	BBPeriod  int     `json:"bb_period"`
	BBStdDev  float64 `json:"bb_std_dev"`
	ATRPeriod int     `json:"atr_period"`

	// Volume Indicators
	OBVEnabled  bool `json:"obv_enabled"`
	VWAPEnabled bool `json:"vwap_enabled"`
}

// NewTechnicalAnalysisService creates a new technical analysis service
func NewTechnicalAnalysisService(
	cfg *config.Config,
	db *database.PostgresDB,
	logger *logrus.Logger,
	errorRecoveryManager *ErrorRecoveryManager,
	resourceManager *ResourceManager,
	performanceMonitor *PerformanceMonitor,
) *TechnicalAnalysisService {
	return &TechnicalAnalysisService{
		config:               cfg,
		db:                   db,
		logger:               logger,
		errorRecoveryManager: errorRecoveryManager,
		resourceManager:      resourceManager,
		performanceMonitor:   performanceMonitor,
	}
}

// GetDefaultIndicatorConfig returns default configuration for indicators
func (tas *TechnicalAnalysisService) GetDefaultIndicatorConfig() *IndicatorConfig {
	return &IndicatorConfig{
		SMAPeriods:   []int{10, 20, 50},
		EMAPeriods:   []int{12, 26},
		RSIPeriod:    14,
		StochKPeriod: 14,
		StochDPeriod: 3,
		MACDFast:     12,
		MACDSlow:     26,
		MACDSignal:   9,
		BBPeriod:     20,
		BBStdDev:     2.0,
		ATRPeriod:    14,
		OBVEnabled:   true,
		VWAPEnabled:  true,
	}
}

// AnalyzeSymbol performs comprehensive technical analysis on a symbol
func (tas *TechnicalAnalysisService) AnalyzeSymbol(ctx context.Context, symbol, exchange string, config *IndicatorConfig) (*TechnicalAnalysisResult, error) {
	// Register operation with resource manager
	operationID := fmt.Sprintf("technical_analysis_%s_%s_%d", symbol, exchange, time.Now().UnixNano())
	tas.resourceManager.RegisterResource(operationID, GoroutineResource, func() error {
		return nil
	}, map[string]interface{}{"symbol": symbol, "exchange": exchange, "operation": "technical_analysis"})
	defer func() {
		if err := tas.resourceManager.CleanupResource(operationID); err != nil {
			tas.logger.WithFields(logrus.Fields{"operation_id": operationID}).Warnf("Failed to cleanup resource: %v", err)
		}
	}()

	// Create timeout context for analysis
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	tas.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"exchange": exchange,
	}).Info("Starting technical analysis")

	// Fetch price data from database with error recovery
	var priceData *PriceData
	err := tas.errorRecoveryManager.ExecuteWithRetry(ctx, "fetch_price_data", func() error {
		var err error
		priceData, err = tas.fetchPriceData(ctx, symbol, exchange)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price data: %w", err)
	}

	if len(priceData.Close) < 50 {
		return nil, fmt.Errorf("insufficient price data: need at least 50 points, got %d", len(priceData.Close))
	}

	// Convert to asset snapshots for cinar/indicator
	snapshots := tas.convertToSnapshots(priceData)

	// Calculate all indicators with error recovery
	var indicators []*IndicatorResult
	err = tas.errorRecoveryManager.ExecuteWithRetry(ctx, "calculate_indicators", func() error {
		indicators = tas.calculateAllIndicators(snapshots, config)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to calculate indicators: %w", err)
	}

	// Determine overall signal and confidence
	overallSignal, confidence := tas.determineOverallSignal(indicators)

	return &TechnicalAnalysisResult{
		Symbol:        symbol,
		Exchange:      exchange,
		Timeframe:     "1h", // Default timeframe
		Indicators:    indicators,
		OverallSignal: overallSignal,
		Confidence:    confidence,
		CalculatedAt:  time.Now(),
	}, nil
}

// calculateAllIndicators computes all configured technical indicators
func (tas *TechnicalAnalysisService) calculateAllIndicators(snapshots []*asset.Snapshot, config *IndicatorConfig) []*IndicatorResult {
	var indicators []*IndicatorResult

	// Extract price arrays
	closePrices := make([]float64, len(snapshots))
	highPrices := make([]float64, len(snapshots))
	lowPrices := make([]float64, len(snapshots))
	volumes := make([]float64, len(snapshots))

	for i, snapshot := range snapshots {
		closePrices[i] = snapshot.Close
		highPrices[i] = snapshot.High
		lowPrices[i] = snapshot.Low
		volumes[i] = snapshot.Volume
	}

	// Moving Averages
	for _, period := range config.SMAPeriods {
		if result := tas.calculateSMA(closePrices, period); result != nil {
			indicators = append(indicators, result)
		}
	}

	for _, period := range config.EMAPeriods {
		if result := tas.calculateEMA(closePrices, period); result != nil {
			indicators = append(indicators, result)
		}
	}

	// Momentum Indicators
	if result := tas.calculateRSI(closePrices, config.RSIPeriod); result != nil {
		indicators = append(indicators, result)
	}

	if result := tas.calculateStochastic(highPrices, lowPrices, closePrices, config.StochKPeriod, config.StochDPeriod); result != nil {
		indicators = append(indicators, result)
	}

	// Trend Indicators
	if result := tas.calculateMACD(closePrices, config.MACDFast, config.MACDSlow, config.MACDSignal); result != nil {
		indicators = append(indicators, result)
	}

	// Volatility Indicators
	if result := tas.calculateBollingerBands(closePrices, config.BBPeriod, config.BBStdDev); result != nil {
		indicators = append(indicators, result)
	}

	if result := tas.calculateATR(highPrices, lowPrices, closePrices, config.ATRPeriod); result != nil {
		indicators = append(indicators, result)
	}

	// Volume Indicators
	if config.OBVEnabled {
		if result := tas.calculateOBV(closePrices, volumes); result != nil {
			indicators = append(indicators, result)
		}
	}

	return indicators
}

// calculateSMA calculates Simple Moving Average
func (tas *TechnicalAnalysisService) calculateSMA(prices []float64, period int) *IndicatorResult {
	if len(prices) < period {
		return nil
	}

	// Calculate SMA
	smaIndicator := trend.NewSmaWithPeriod[float64](period)
	result := helper.ChanToSlice(smaIndicator.Compute(helper.SliceToChan(prices)))

	// Convert to decimal and determine signal
	values := make([]decimal.Decimal, len(result))
	for i, val := range result {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeSMASignal(prices, result, period)

	return &IndicatorResult{
		Name:      fmt.Sprintf("SMA_%d", period),
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// calculateEMA calculates Exponential Moving Average
func (tas *TechnicalAnalysisService) calculateEMA(prices []float64, period int) *IndicatorResult {
	if len(prices) < period {
		return nil
	}

	// Calculate EMA
	emaIndicator := trend.NewEmaWithPeriod[float64](period)
	result := helper.ChanToSlice(emaIndicator.Compute(helper.SliceToChan(prices)))

	values := make([]decimal.Decimal, len(result))
	for i, val := range result {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeEMASignal(prices, result, period)

	return &IndicatorResult{
		Name:      fmt.Sprintf("EMA_%d", period),
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// calculateRSI calculates Relative Strength Index
func (tas *TechnicalAnalysisService) calculateRSI(prices []float64, period int) *IndicatorResult {
	if len(prices) < period+1 {
		return nil
	}

	// Calculate RSI
	rsiIndicator := momentum.NewRsiWithPeriod[float64](period)
	result := helper.ChanToSlice(rsiIndicator.Compute(helper.SliceToChan(prices)))

	values := make([]decimal.Decimal, len(result))
	for i, val := range result {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeRSISignal(result)

	return &IndicatorResult{
		Name:      fmt.Sprintf("RSI_%d", period),
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// calculateMACD calculates Moving Average Convergence Divergence
func (tas *TechnicalAnalysisService) calculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) *IndicatorResult {
	if len(prices) < slowPeriod+signalPeriod {
		return nil
	}

	// Calculate MACD
	macdIndicator := trend.NewMacdWithPeriod[float64](12, 26, 9)
	macdLine, macdSignal := macdIndicator.Compute(helper.SliceToChan(prices))
	macdLineSlice := helper.ChanToSlice(macdLine)
	macdSignalSlice := helper.ChanToSlice(macdSignal)

	// Calculate histogram
	histogram := make([]float64, len(macdLineSlice))
	for i := range macdLineSlice {
		histogram[i] = macdLineSlice[i] - macdSignalSlice[i]
	}

	result := macdLineSlice

	values := make([]decimal.Decimal, len(result))
	for i, val := range result {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeMACDSignal(result)

	return &IndicatorResult{
		Name:      "MACD",
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// calculateBollingerBands calculates Bollinger Bands
func (tas *TechnicalAnalysisService) calculateBollingerBands(prices []float64, period int, stdDev float64) *IndicatorResult {
	if len(prices) < period {
		return nil
	}

	// Bollinger Bands calculation - using SMA as approximation
	smaIndicator := trend.NewSmaWithPeriod[float64](period)
	result := helper.ChanToSlice(smaIndicator.Compute(helper.SliceToChan(prices)))

	values := make([]decimal.Decimal, len(result))
	for i, val := range result {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeBollingerBandsSignal(prices, result, period)

	return &IndicatorResult{
		Name:      "BB",
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// calculateATR calculates Average True Range
func (tas *TechnicalAnalysisService) calculateATR(high, low, close []float64, period int) *IndicatorResult {
	if len(high) < period || len(low) < period || len(close) < period {
		return nil
	}

	// Use ATR with default period, then slice to get the desired period
	atrIndicator := volatility.NewAtr[float64]()

	// Convert to channels for ATR calculation
	highChan := helper.SliceToChan(high)
	lowChan := helper.SliceToChan(low)
	closeChan := helper.SliceToChan(close)

	result := helper.ChanToSlice(atrIndicator.Compute(highChan, lowChan, closeChan))

	// Take only the last values based on period
	startIdx := 0
	if len(result) > period {
		startIdx = len(result) - period
	}
	filteredResult := result[startIdx:]

	values := make([]decimal.Decimal, len(filteredResult))
	for i, val := range filteredResult {
		values[i] = decimal.NewFromFloat(val)
	}

	return &IndicatorResult{
		Name:      fmt.Sprintf("ATR_%d", period),
		Values:    values,
		Signal:    "hold", // ATR is primarily for volatility measurement
		Strength:  decimal.NewFromFloat(0.5),
		Timestamp: time.Now(),
	}
}

// calculateStochastic calculates Stochastic Oscillator (simplified implementation)
func (tas *TechnicalAnalysisService) calculateStochastic(high, low, close []float64, kPeriod, dPeriod int) *IndicatorResult {
	if len(high) < kPeriod || len(low) < kPeriod || len(close) < kPeriod {
		return nil
	}

	// Simple Stochastic %K calculation
	result := make([]float64, len(close))
	for i := kPeriod - 1; i < len(close); i++ {
		// Find highest high and lowest low in the period
		highestHigh := high[i-kPeriod+1]
		lowestLow := low[i-kPeriod+1]
		for j := i - kPeriod + 2; j <= i; j++ {
			if high[j] > highestHigh {
				highestHigh = high[j]
			}
			if low[j] < lowestLow {
				lowestLow = low[j]
			}
		}

		// Calculate %K
		if highestHigh != lowestLow {
			result[i] = ((close[i] - lowestLow) / (highestHigh - lowestLow)) * 100
		} else {
			result[i] = 50 // Default to middle when no range
		}
	}

	// Take only valid results
	validResults := result[kPeriod-1:]

	values := make([]decimal.Decimal, len(validResults))
	for i, val := range validResults {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeStochasticSignal(validResults)

	return &IndicatorResult{
		Name:      "STOCH",
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// calculateOBV calculates On-Balance Volume
func (tas *TechnicalAnalysisService) calculateOBV(prices, volumes []float64) *IndicatorResult {
	if len(prices) != len(volumes) || len(prices) < 2 {
		return nil
	}

	obvIndicator := volume.NewObv[float64]()

	// Convert to channels for OBV calculation
	priceChan := helper.SliceToChan(prices)
	volumeChan := helper.SliceToChan(volumes)

	result := helper.ChanToSlice(obvIndicator.Compute(priceChan, volumeChan))

	values := make([]decimal.Decimal, len(result))
	for i, val := range result {
		values[i] = decimal.NewFromFloat(val)
	}

	signal, strength := tas.analyzeOBVSignal(result, prices)

	return &IndicatorResult{
		Name:      "OBV",
		Values:    values,
		Signal:    signal,
		Strength:  strength,
		Timestamp: time.Now(),
	}
}

// Signal analysis functions

func (tas *TechnicalAnalysisService) analyzeSMASignal(prices, sma []float64, period int) (string, decimal.Decimal) {
	if len(prices) < 2 || len(sma) < 2 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	currentPrice := prices[len(prices)-1]
	currentSMA := sma[len(sma)-1]
	prevPrice := prices[len(prices)-2]
	prevSMA := sma[len(sma)-2]

	// Price crossing above SMA
	if currentPrice > currentSMA && prevPrice <= prevSMA {
		return "buy", decimal.NewFromFloat(0.7)
	}
	// Price crossing below SMA
	if currentPrice < currentSMA && prevPrice >= prevSMA {
		return "sell", decimal.NewFromFloat(0.7)
	}
	// Price above SMA (bullish)
	if currentPrice > currentSMA {
		return "buy", decimal.NewFromFloat(0.6)
	}
	// Price below SMA (bearish)
	if currentPrice < currentSMA {
		return "sell", decimal.NewFromFloat(0.6)
	}

	return "hold", decimal.NewFromFloat(0.5)
}

func (tas *TechnicalAnalysisService) analyzeEMASignal(prices, ema []float64, period int) (string, decimal.Decimal) {
	// Similar logic to SMA but with higher sensitivity
	if len(prices) < 2 || len(ema) < 2 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	currentPrice := prices[len(prices)-1]
	currentEMA := ema[len(ema)-1]
	prevPrice := prices[len(prices)-2]
	prevEMA := ema[len(ema)-2]

	if currentPrice > currentEMA && prevPrice <= prevEMA {
		return "buy", decimal.NewFromFloat(0.75)
	}
	if currentPrice < currentEMA && prevPrice >= prevEMA {
		return "sell", decimal.NewFromFloat(0.75)
	}
	if currentPrice > currentEMA {
		return "buy", decimal.NewFromFloat(0.65)
	}
	if currentPrice < currentEMA {
		return "sell", decimal.NewFromFloat(0.65)
	}

	return "hold", decimal.NewFromFloat(0.5)
}

func (tas *TechnicalAnalysisService) analyzeRSISignal(rsi []float64) (string, decimal.Decimal) {
	if len(rsi) == 0 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	currentRSI := rsi[len(rsi)-1]

	if currentRSI < 30 {
		return "buy", decimal.NewFromFloat(0.8) // Oversold
	}
	if currentRSI > 70 {
		return "sell", decimal.NewFromFloat(0.8) // Overbought
	}
	if currentRSI < 40 {
		return "buy", decimal.NewFromFloat(0.6)
	}
	if currentRSI > 60 {
		return "sell", decimal.NewFromFloat(0.6)
	}

	return "hold", decimal.NewFromFloat(0.5)
}

func (tas *TechnicalAnalysisService) analyzeMACDSignal(macd []float64) (string, decimal.Decimal) {
	if len(macd) < 2 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	current := macd[len(macd)-1]
	previous := macd[len(macd)-2]

	// MACD crossing above zero
	if current > 0 && previous <= 0 {
		return "buy", decimal.NewFromFloat(0.8)
	}
	// MACD crossing below zero
	if current < 0 && previous >= 0 {
		return "sell", decimal.NewFromFloat(0.8)
	}
	// MACD above zero (bullish)
	if current > 0 {
		return "buy", decimal.NewFromFloat(0.6)
	}
	// MACD below zero (bearish)
	if current < 0 {
		return "sell", decimal.NewFromFloat(0.6)
	}

	return "hold", decimal.NewFromFloat(0.5)
}

func (tas *TechnicalAnalysisService) analyzeBollingerBandsSignal(prices, bb []float64, period int) (string, decimal.Decimal) {
	// This is a simplified analysis - in practice, you'd need upper, middle, and lower bands
	if len(prices) == 0 || len(bb) == 0 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	currentPrice := prices[len(prices)-1]
	currentBB := bb[len(bb)-1]

	// Price near lower band (potential buy)
	if currentPrice < currentBB*0.98 {
		return "buy", decimal.NewFromFloat(0.7)
	}
	// Price near upper band (potential sell)
	if currentPrice > currentBB*1.02 {
		return "sell", decimal.NewFromFloat(0.7)
	}

	return "hold", decimal.NewFromFloat(0.5)
}

func (tas *TechnicalAnalysisService) analyzeStochasticSignal(stoch []float64) (string, decimal.Decimal) {
	if len(stoch) == 0 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	current := stoch[len(stoch)-1]

	if current < 20 {
		return "buy", decimal.NewFromFloat(0.75) // Oversold
	}
	if current > 80 {
		return "sell", decimal.NewFromFloat(0.75) // Overbought
	}

	return "hold", decimal.NewFromFloat(0.5)
}

func (tas *TechnicalAnalysisService) analyzeOBVSignal(obv, prices []float64) (string, decimal.Decimal) {
	if len(obv) < 2 || len(prices) < 2 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	currentOBV := obv[len(obv)-1]
	prevOBV := obv[len(obv)-2]
	currentPrice := prices[len(prices)-1]
	prevPrice := prices[len(prices)-2]

	// Volume confirming price trend
	if currentPrice > prevPrice && currentOBV > prevOBV {
		return "buy", decimal.NewFromFloat(0.7)
	}
	if currentPrice < prevPrice && currentOBV < prevOBV {
		return "sell", decimal.NewFromFloat(0.7)
	}

	return "hold", decimal.NewFromFloat(0.5)
}

// determineOverallSignal analyzes all indicators to determine overall signal
func (tas *TechnicalAnalysisService) determineOverallSignal(indicators []*IndicatorResult) (string, decimal.Decimal) {
	if len(indicators) == 0 {
		return "hold", decimal.NewFromFloat(0.5)
	}

	buyScore := decimal.Zero
	sellScore := decimal.Zero
	totalWeight := decimal.Zero

	for _, indicator := range indicators {
		weight := indicator.Strength
		totalWeight = totalWeight.Add(weight)

		switch indicator.Signal {
		case "buy":
			buyScore = buyScore.Add(weight)
		case "sell":
			sellScore = sellScore.Add(weight)
		}
	}

	if totalWeight.IsZero() {
		return "hold", decimal.NewFromFloat(0.5)
	}

	buyRatio := buyScore.Div(totalWeight)
	sellRatio := sellScore.Div(totalWeight)

	// Determine overall signal
	if buyRatio.GreaterThan(decimal.NewFromFloat(0.6)) {
		return "buy", buyRatio
	}
	if sellRatio.GreaterThan(decimal.NewFromFloat(0.6)) {
		return "sell", sellRatio
	}

	return "hold", decimal.NewFromFloat(0.5)
}

// Helper functions

func (tas *TechnicalAnalysisService) fetchPriceData(ctx context.Context, symbol, exchange string) (*PriceData, error) {
	// Fetch market data from database
	var marketData []models.MarketData
	query := `SELECT last_price, volume_24h, timestamp FROM market_data 
			 WHERE trading_pair_id IN (SELECT id FROM trading_pairs WHERE symbol = $1) 
			 AND exchange_id IN (SELECT id FROM exchanges WHERE name = $2) 
			 ORDER BY timestamp DESC LIMIT 200`
	rows, err := tas.db.Pool.Query(ctx, query, symbol, exchange)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var md models.MarketData
		err := rows.Scan(&md.LastPrice, &md.Volume24h, &md.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market data: %w", err)
		}
		marketData = append(marketData, md)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating market data rows: %w", err)
	}

	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data found for %s on %s", symbol, exchange)
	}

	// Reverse to get chronological order
	for i, j := 0, len(marketData)-1; i < j; i, j = i+1, j-1 {
		marketData[i], marketData[j] = marketData[j], marketData[i]
	}

	// Convert to PriceData
	priceData := &PriceData{
		Symbol:     symbol,
		Exchange:   exchange,
		Open:       make([]decimal.Decimal, len(marketData)),
		High:       make([]decimal.Decimal, len(marketData)),
		Low:        make([]decimal.Decimal, len(marketData)),
		Close:      make([]decimal.Decimal, len(marketData)),
		Volume:     make([]decimal.Decimal, len(marketData)),
		Timestamps: make([]time.Time, len(marketData)),
	}

	for i, data := range marketData {
		priceData.Open[i] = data.LastPrice // Using last price as OHLC for now
		priceData.High[i] = data.LastPrice
		priceData.Low[i] = data.LastPrice
		priceData.Close[i] = data.LastPrice
		priceData.Volume[i] = data.Volume24h
		priceData.Timestamps[i] = data.Timestamp
	}

	return priceData, nil
}

func (tas *TechnicalAnalysisService) convertToSnapshots(priceData *PriceData) []*asset.Snapshot {
	snapshots := make([]*asset.Snapshot, len(priceData.Close))

	for i := range priceData.Close {
		open, _ := priceData.Open[i].Float64()
		high, _ := priceData.High[i].Float64()
		low, _ := priceData.Low[i].Float64()
		close, _ := priceData.Close[i].Float64()
		volume, _ := priceData.Volume[i].Float64()

		snapshots[i] = &asset.Snapshot{
			Date:   priceData.Timestamps[i],
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}
	}

	return snapshots
}
