package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	// "sync"
	"time"

	"github.com/cinar/indicator/v2/asset"
	"github.com/cinar/indicator/v2/helper"
	"github.com/cinar/indicator/v2/momentum"
	"github.com/cinar/indicator/v2/trend"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/models"
)

// SignalType represents the type of trading signal
type SignalType string

const (
	SignalTypeArbitrage SignalType = "arbitrage"
	SignalTypeTechnical SignalType = "technical"
)

// SignalStrength represents the strength of a trading signal
type SignalStrength string

const (
	SignalStrengthWeak   SignalStrength = "weak"
	SignalStrengthMedium SignalStrength = "medium"
	SignalStrengthStrong SignalStrength = "strong"
)

// AggregatedSignal represents a consolidated trading signal
type AggregatedSignal struct {
	ID              string                 `json:"id" gorm:"primaryKey"`
	SignalType      SignalType             `json:"signal_type"`
	Symbol          string                 `json:"symbol"`
	Action          string                 `json:"action"` // "buy", "sell", "hold"
	Strength        SignalStrength         `json:"strength"`
	Confidence      decimal.Decimal        `json:"confidence"`
	ProfitPotential decimal.Decimal        `json:"profit_potential"`
	RiskLevel       decimal.Decimal        `json:"risk_level"`
	Exchanges       []string               `json:"exchanges"`
	Indicators      []string               `json:"indicators"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
}

// SignalFingerprint represents a unique identifier for deduplication
type SignalFingerprint struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Hash      string    `json:"hash" gorm:"uniqueIndex"`
	SignalID  string    `json:"signal_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SignalComponent represents an individual technical indicator signal
type SignalComponent struct {
	Indicator   string          `json:"indicator"`
	Description string          `json:"description"`
	Confidence  decimal.Decimal `json:"confidence"`
	Strength    float64         `json:"strength"`
}

// TechnicalSignalInput represents input data for technical analysis
type TechnicalSignalInput struct {
	Symbol     string
	Exchange   string
	Prices     []decimal.Decimal
	Volumes    []decimal.Decimal
	Timestamps []time.Time
}

// ArbitrageSignalInput represents input data for arbitrage analysis
type ArbitrageSignalInput struct {
	Opportunities []models.ArbitrageOpportunity
	MinVolume     decimal.Decimal `json:"min_volume"`
	BaseAmount    decimal.Decimal `json:"base_amount"` // For profit calculation (e.g., $20,000)
}

// SignalAggregatorConfig holds configuration for the signal aggregator
type SignalAggregatorConfig struct {
	MinConfidence       decimal.Decimal `json:"min_confidence"`
	MinProfitThreshold  decimal.Decimal `json:"min_profit_threshold"`
	MaxRiskLevel        decimal.Decimal `json:"max_risk_level"`
	SignalTTL           time.Duration   `json:"signal_ttl"`
	DeduplicationWindow time.Duration   `json:"deduplication_window"`
	MaxSignalsPerSymbol int             `json:"max_signals_per_symbol"`
}

// SignalAggregator handles the aggregation and processing of trading signals
type SignalAggregator struct {
	config        *config.Config
	db            *database.PostgresDB
	logger        *logrus.Logger
	sigConfig     SignalAggregatorConfig
	qualityScorer *SignalQualityScorer
	cache         map[string]*AggregatedSignal
}

// NewSignalAggregator creates a new signal aggregator instance
func NewSignalAggregator(cfg *config.Config, db *database.PostgresDB, logger *logrus.Logger) *SignalAggregator {
	return &SignalAggregator{
		config: cfg,
		db:     db,
		logger: logger,
		sigConfig: SignalAggregatorConfig{
			MinConfidence:       decimal.NewFromFloat(0.6),
			MinProfitThreshold:  decimal.NewFromFloat(0.5), // 0.5%
			MaxRiskLevel:        decimal.NewFromFloat(0.3), // 30%
			SignalTTL:           15 * time.Minute,
			DeduplicationWindow: 5 * time.Minute,
			MaxSignalsPerSymbol: 3,
		},
		qualityScorer: NewSignalQualityScorer(cfg, db, logger),
		cache:         make(map[string]*AggregatedSignal),
	}
}

// AggregateArbitrageSignals processes arbitrage opportunities into aggregated signals with price ranges
func (sa *SignalAggregator) AggregateArbitrageSignals(ctx context.Context, input ArbitrageSignalInput) ([]*AggregatedSignal, error) {
	sa.logger.Info("Aggregating arbitrage signals with enhanced price ranges")

	var signals []*AggregatedSignal
	symbolGroups := make(map[string][]models.ArbitrageOpportunity)

	// Set default values if not provided
	minVolume := input.MinVolume
	if minVolume.IsZero() {
		minVolume = decimal.NewFromFloat(10000) // Default $10,000 minimum volume
	}
	baseAmount := input.BaseAmount
	if baseAmount.IsZero() {
		baseAmount = decimal.NewFromFloat(20000) // Default $20,000 base amount
	}

	// Group opportunities by symbol and filter by volume
	for _, opp := range input.Opportunities {
		var symbol string
		if opp.TradingPair != nil {
			symbol = opp.TradingPair.Symbol
		} else {
			// Skip opportunities without trading pair info
			continue
		}

		// Apply volume filtering (simulate volume check - in real implementation, this would come from exchange data)
		estimatedVolume := opp.BuyPrice.Mul(decimal.NewFromFloat(1000)) // Simulate volume calculation
		if estimatedVolume.GreaterThanOrEqual(minVolume) {
			symbolGroups[symbol] = append(symbolGroups[symbol], opp)
		} else {
			sa.logger.WithFields(logrus.Fields{
				"symbol":           symbol,
				"estimated_volume": estimatedVolume,
				"min_volume":       minVolume,
			}).Debug("Filtered out low volume arbitrage opportunity")
		}
	}

	// Process each symbol group
	for symbol, opportunities := range symbolGroups {
		// Filter by profit threshold
		var validOpps []models.ArbitrageOpportunity
		for _, opp := range opportunities {
			if opp.ProfitPercentage.GreaterThanOrEqual(sa.sigConfig.MinProfitThreshold) {
				validOpps = append(validOpps, opp)
			}
		}

		if len(validOpps) == 0 {
			continue
		}

		// Sort by profit percentage (descending)
		sort.Slice(validOpps, func(i, j int) bool {
			return validOpps[i].ProfitPercentage.GreaterThan(validOpps[j].ProfitPercentage)
		})

		// Create aggregated signal with price ranges from multiple opportunities
		if len(validOpps) > 0 {
			signal := sa.createEnhancedArbitrageSignal(validOpps, symbol, minVolume, baseAmount)

			// Assess signal quality
			qualityInput := SignalQualityInput{
				SignalType:       string(signal.SignalType),
				Symbol:           signal.Symbol,
				Exchanges:        signal.Exchanges,
				Volume:           minVolume, // Use minimum volume requirement
				ProfitPotential:  signal.ProfitPotential,
				Confidence:       signal.Confidence,
				Timestamp:        signal.CreatedAt,
				Indicators:       map[string]interface{}{"arbitrage": true, "opportunity_count": len(validOpps)},
				SignalCount:      len(validOpps),
				SignalComponents: []string{"arbitrage"},
			}

			qualityMetrics, err := sa.qualityScorer.AssessSignalQuality(ctx, &qualityInput)
			if err != nil {
				sa.logger.WithError(err).Warn("Failed to assess signal quality")
				signals = append(signals, signal) // Include signal if assessment fails
			} else if sa.qualityScorer.IsSignalQualityAcceptable(qualityMetrics, sa.qualityScorer.GetDefaultQualityThresholds()) {
				signals = append(signals, signal)
			} else {
				sa.logger.WithField("signal_id", signal.ID).Debug("Signal rejected due to low quality")
			}
		}
	}

	return signals, nil
}

// AggregateTechnicalSignals processes technical analysis data into aggregated signals
func (sa *SignalAggregator) AggregateTechnicalSignals(ctx context.Context, input TechnicalSignalInput) ([]*AggregatedSignal, error) {
	sa.logger.WithField("symbol", input.Symbol).Info("Aggregating technical signals")

	if len(input.Prices) < 20 {
		return nil, fmt.Errorf("insufficient price data for technical analysis: need at least 20 points, got %d", len(input.Prices))
	}

	// Convert decimal prices to float64 for cinar/indicator
	prices := make([]float64, len(input.Prices))
	volumes := make([]float64, len(input.Volumes))
	for i, price := range input.Prices {
		prices[i], _ = price.Float64()
		if i < len(input.Volumes) {
			volumes[i], _ = input.Volumes[i].Float64()
		}
	}

	// Create asset snapshots for cinar/indicator
	snapshots := make([]*asset.Snapshot, len(prices))
	for i := range prices {
		snapshots[i] = &asset.Snapshot{
			Date:   input.Timestamps[i],
			Open:   prices[i],
			High:   prices[i],
			Low:    prices[i],
			Close:  prices[i],
			Volume: volumes[i],
		}
	}

	// Extract prices from snapshots
	prices = make([]float64, len(snapshots))
	for i, snapshot := range snapshots {
		prices[i] = snapshot.Close
	}

	// Calculate technical indicators
	indicators := sa.calculateTechnicalIndicators(prices)

	// Generate signals based on indicators
	signals := sa.generateTechnicalSignals(input.Symbol, input.Exchange, indicators)

	// Assess quality for each technical signal
	var qualitySignals []*AggregatedSignal
	for _, signal := range signals {
		// Extract signal count and components from metadata
		signalCount := 1
		signalComponents := []string{}
		if metadata, ok := signal.Metadata["signal_count"].(int); ok {
			signalCount = metadata
		}
		if components, ok := signal.Metadata["signal_components"].([]string); ok {
			signalComponents = components
		}

		// Calculate average volume for quality assessment
		avgVolume := decimal.NewFromFloat(0)
		if len(input.Volumes) > 0 {
			totalVolume := decimal.NewFromFloat(0)
			for _, vol := range input.Volumes {
				totalVolume = totalVolume.Add(vol)
			}
			avgVolume = totalVolume.Div(decimal.NewFromInt(int64(len(input.Volumes))))
		}

		qualityInput := SignalQualityInput{
			SignalType:       string(signal.SignalType),
			Symbol:           signal.Symbol,
			Exchanges:        signal.Exchanges,
			Volume:           avgVolume,
			ProfitPotential:  signal.ProfitPotential,
			Confidence:       signal.Confidence,
			Timestamp:        signal.CreatedAt,
			Indicators:       signal.Metadata,
			SignalCount:      signalCount,
			SignalComponents: signalComponents,
		}

		qualityMetrics, err := sa.qualityScorer.AssessSignalQuality(ctx, &qualityInput)
		if err != nil {
			sa.logger.WithError(err).Warn("Failed to assess technical signal quality")
			qualitySignals = append(qualitySignals, signal) // Include signal if assessment fails
		} else if sa.qualityScorer.IsSignalQualityAcceptable(qualityMetrics, sa.qualityScorer.GetDefaultQualityThresholds()) {
			qualitySignals = append(qualitySignals, signal)
		} else {
			sa.logger.WithField("signal_id", signal.ID).Debug("Technical signal rejected due to low quality")
		}
	}

	return qualitySignals, nil
}

// DeduplicateSignals removes duplicate signals based on fingerprinting
func (sa *SignalAggregator) DeduplicateSignals(ctx context.Context, signals []*AggregatedSignal) ([]*AggregatedSignal, error) {
	sa.logger.Info("Deduplicating signals")

	var uniqueSignals []*AggregatedSignal
	seenHashes := make(map[string]bool)

	for _, signal := range signals {
		hash := sa.generateSignalHash(signal)

		// Check if we've seen this hash recently
		if !sa.isHashRecent(ctx, hash) {
			uniqueSignals = append(uniqueSignals, signal)
			seenHashes[hash] = true

			// Store fingerprint
			fingerprint := &SignalFingerprint{
				ID:        uuid.New().String(),
				Hash:      hash,
				SignalID:  signal.ID,
				CreatedAt: time.Now(),
			}
			sa.storeFingerprint(ctx, fingerprint)
		}
	}

	sa.logger.WithFields(logrus.Fields{
		"original_count": len(signals),
		"unique_count":   len(uniqueSignals),
	}).Info("Signal deduplication completed")

	return uniqueSignals, nil
}

// createEnhancedArbitrageSignal creates an aggregated signal with price ranges from multiple opportunities
func (sa *SignalAggregator) createEnhancedArbitrageSignal(opportunities []models.ArbitrageOpportunity, symbol string, minVolume, baseAmount decimal.Decimal) *AggregatedSignal {
	if len(opportunities) == 0 {
		return nil
	}

	// Calculate price ranges
	buyPrices := make([]decimal.Decimal, 0)
	sellPrices := make([]decimal.Decimal, 0)
	buyExchanges := make([]string, 0)
	sellExchanges := make([]string, 0)
	allExchanges := make(map[string]bool)

	for _, opp := range opportunities {
		buyPrices = append(buyPrices, opp.BuyPrice)
		sellPrices = append(sellPrices, opp.SellPrice)

		buyExchangeName := fmt.Sprintf("%d", opp.BuyExchangeID)
		sellExchangeName := fmt.Sprintf("%d", opp.SellExchangeID)

		if opp.BuyExchange != nil {
			buyExchangeName = opp.BuyExchange.Name
		}
		if opp.SellExchange != nil {
			sellExchangeName = opp.SellExchange.Name
		}

		buyExchanges = append(buyExchanges, buyExchangeName)
		sellExchanges = append(sellExchanges, sellExchangeName)
		allExchanges[buyExchangeName] = true
		allExchanges[sellExchangeName] = true
	}

	// Calculate buy price range
	buyMin := buyPrices[0]
	buyMax := buyPrices[0]
	buySum := decimal.NewFromFloat(0)
	for _, price := range buyPrices {
		if price.LessThan(buyMin) {
			buyMin = price
		}
		if price.GreaterThan(buyMax) {
			buyMax = price
		}
		buySum = buySum.Add(price)
	}
	buyAvg := buySum.Div(decimal.NewFromInt(int64(len(buyPrices))))

	// Calculate sell price range
	sellMin := sellPrices[0]
	sellMax := sellPrices[0]
	sellSum := decimal.NewFromFloat(0)
	for _, price := range sellPrices {
		if price.LessThan(sellMin) {
			sellMin = price
		}
		if price.GreaterThan(sellMax) {
			sellMax = price
		}
		sellSum = sellSum.Add(price)
	}
	sellAvg := sellSum.Div(decimal.NewFromInt(int64(len(sellPrices))))

	// Calculate profit ranges
	minProfitPercent := sellMin.Sub(buyMax).Div(buyMax).Mul(decimal.NewFromFloat(100))
	maxProfitPercent := sellMax.Sub(buyMin).Div(buyMin).Mul(decimal.NewFromFloat(100))
	avgProfitPercent := sellAvg.Sub(buyAvg).Div(buyAvg).Mul(decimal.NewFromFloat(100))

	// Calculate dollar amounts based on base amount
	minProfitDollar := baseAmount.Mul(minProfitPercent).Div(decimal.NewFromFloat(100))
	maxProfitDollar := baseAmount.Mul(maxProfitPercent).Div(decimal.NewFromFloat(100))
	avgProfitDollar := baseAmount.Mul(avgProfitPercent).Div(decimal.NewFromFloat(100))

	// Calculate overall confidence (higher for multiple opportunities)
	baseConfidence := sa.calculateArbitrageConfidence(opportunities[0])
	multiOpportunityBonus := decimal.NewFromFloat(float64(len(opportunities)-1) * 0.05) // 5% bonus per additional opportunity
	if multiOpportunityBonus.GreaterThan(decimal.NewFromFloat(0.2)) {
		multiOpportunityBonus = decimal.NewFromFloat(0.2) // Cap at 20% bonus
	}
	finalConfidence := baseConfidence.Add(multiOpportunityBonus)
	if finalConfidence.GreaterThan(decimal.NewFromFloat(1.0)) {
		finalConfidence = decimal.NewFromFloat(1.0)
	}

	strength := sa.determineSignalStrengthWithProfit(finalConfidence, avgProfitPercent)

	// Convert exchange map to slice
	exchangeList := make([]string, 0, len(allExchanges))
	for exchange := range allExchanges {
		exchangeList = append(exchangeList, exchange)
	}

	// Remove duplicates from buy and sell exchanges
	uniqueBuyExchanges := sa.removeDuplicateStrings(buyExchanges)
	uniqueSellExchanges := sa.removeDuplicateStrings(sellExchanges)

	return &AggregatedSignal{
		ID:              uuid.New().String(),
		SignalType:      SignalTypeArbitrage,
		Symbol:          symbol,
		Action:          "buy",
		Strength:        strength,
		Confidence:      finalConfidence,
		ProfitPotential: avgProfitPercent,
		RiskLevel:       decimal.NewFromFloat(0.08), // Lower risk for multiple opportunities
		Exchanges:       exchangeList,
		Indicators:      []string{"arbitrage"},
		Metadata: map[string]interface{}{
			"buy_price_range": map[string]interface{}{
				"min": buyMin,
				"max": buyMax,
				"avg": buyAvg,
			},
			"sell_price_range": map[string]interface{}{
				"min": sellMin,
				"max": sellMax,
				"avg": sellAvg,
			},
			"profit_range": map[string]interface{}{
				"min_percent": minProfitPercent,
				"max_percent": maxProfitPercent,
				"avg_percent": avgProfitPercent,
				"min_dollar":  minProfitDollar,
				"max_dollar":  maxProfitDollar,
				"avg_dollar":  avgProfitDollar,
				"base_amount": baseAmount,
			},
			"buy_exchanges":     uniqueBuyExchanges,
			"sell_exchanges":    uniqueSellExchanges,
			"opportunity_count": len(opportunities),
			"min_volume":        minVolume,
			"validity_minutes":  int(sa.sigConfig.SignalTTL.Minutes()),
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sa.sigConfig.SignalTTL),
	}
}

// removeDuplicateStrings removes duplicate strings from a slice
func (sa *SignalAggregator) removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

// calculateTechnicalIndicators computes various technical indicators
func (sa *SignalAggregator) calculateTechnicalIndicators(prices []float64) map[string][]float64 {
	indicators := make(map[string][]float64)

	if len(prices) < 50 {
		return indicators
	}

	// Calculate SMA 20 and 50
	sma20Indicator := trend.NewSmaWithPeriod[float64](20)
	sma50Indicator := trend.NewSmaWithPeriod[float64](50)
	sma20 := helper.ChanToSlice(sma20Indicator.Compute(helper.SliceToChan(prices)))
	sma50 := helper.ChanToSlice(sma50Indicator.Compute(helper.SliceToChan(prices)))

	// Calculate EMA 12
	ema12Indicator := trend.NewEmaWithPeriod[float64](12)
	ema12 := helper.ChanToSlice(ema12Indicator.Compute(helper.SliceToChan(prices)))

	// Calculate RSI 14
	rsiIndicator := momentum.NewRsiWithPeriod[float64](14)
	rsi := helper.ChanToSlice(rsiIndicator.Compute(helper.SliceToChan(prices)))

	// Calculate MACD
	macdIndicator := trend.NewMacdWithPeriod[float64](12, 26, 9)
	macdResult, _ := macdIndicator.Compute(helper.SliceToChan(prices))
	macd := helper.ChanToSlice(macdResult)

	indicators["sma_20"] = sma20
	indicators["sma_50"] = sma50
	indicators["ema_12"] = ema12
	indicators["rsi_14"] = rsi
	indicators["macd_line"] = macd

	return indicators
}

// generateTechnicalSignals creates signals based on technical indicators
func (sa *SignalAggregator) generateTechnicalSignals(symbol, exchange string, indicators map[string][]float64) []*AggregatedSignal {
	// Collect individual signal components
	buySignals := make([]SignalComponent, 0)
	sellSignals := make([]SignalComponent, 0)

	// RSI-based signals
	if rsi, exists := indicators["rsi_14"]; exists && len(rsi) > 0 {
		currentRSI := rsi[len(rsi)-1]
		if currentRSI < 30 {
			// Oversold - Buy signal
			buySignals = append(buySignals, SignalComponent{
				Indicator:   "rsi_oversold",
				Description: "RSI oversold recovery",
				Confidence:  decimal.NewFromFloat(0.7),
				Strength:    0.7,
			})
		} else if currentRSI > 70 {
			// Overbought - Sell signal
			sellSignals = append(sellSignals, SignalComponent{
				Indicator:   "rsi_overbought",
				Description: "RSI overbought condition",
				Confidence:  decimal.NewFromFloat(0.7),
				Strength:    0.7,
			})
		}
	}

	// Moving Average Crossover
	if sma, exists := indicators["sma_20"]; exists && len(sma) > 1 {
		if ema, emaExists := indicators["ema_12"]; emaExists && len(ema) > 1 {
			currentSMA := sma[len(sma)-1]
			currentEMA := ema[len(ema)-1]
			prevSMA := sma[len(sma)-2]
			prevEMA := ema[len(ema)-2]

			// Golden Cross (EMA crosses above SMA)
			if currentEMA > currentSMA && prevEMA <= prevSMA {
				buySignals = append(buySignals, SignalComponent{
					Indicator:   "golden_cross",
					Description: "MA20 crossed above MA50 (Golden Cross)",
					Confidence:  decimal.NewFromFloat(0.8),
					Strength:    0.8,
				})
			}
			// Death Cross (EMA crosses below SMA)
			if currentEMA < currentSMA && prevEMA >= prevSMA {
				sellSignals = append(sellSignals, SignalComponent{
					Indicator:   "death_cross",
					Description: "MA20 crossed below MA50 (Death Cross)",
					Confidence:  decimal.NewFromFloat(0.8),
					Strength:    0.8,
				})
			}
		}
	}

	// MACD signals
	if macdLine, exists := indicators["macd_line"]; exists && len(macdLine) > 1 {
		if macdSignal, signalExists := indicators["macd_signal"]; signalExists && len(macdSignal) > 1 {
			currentMACD := macdLine[len(macdLine)-1]
			currentSignal := macdSignal[len(macdSignal)-1]
			prevMACD := macdLine[len(macdLine)-2]
			prevSignal := macdSignal[len(macdSignal)-2]

			// MACD bullish crossover (MACD line crosses above signal line)
			if currentMACD > currentSignal && prevMACD <= prevSignal {
				buySignals = append(buySignals, SignalComponent{
					Indicator:   "macd_bullish",
					Description: "MACD bullish crossover",
					Confidence:  decimal.NewFromFloat(0.75),
					Strength:    0.75,
				})
			}
			// MACD bearish crossover
			if currentMACD < currentSignal && prevMACD >= prevSignal {
				sellSignals = append(sellSignals, SignalComponent{
					Indicator:   "macd_bearish",
					Description: "MACD bearish crossover",
					Confidence:  decimal.NewFromFloat(0.75),
					Strength:    0.75,
				})
			}
		}
	}

	// Aggregate signals
	var aggregatedSignals []*AggregatedSignal

	// Create aggregated buy signal if we have buy components
	if len(buySignals) > 0 {
		aggregatedSignal := sa.createAggregatedTechnicalSignal(symbol, exchange, "buy", buySignals)
		aggregatedSignals = append(aggregatedSignals, aggregatedSignal)
	}

	// Create aggregated sell signal if we have sell components
	if len(sellSignals) > 0 {
		aggregatedSignal := sa.createAggregatedTechnicalSignal(symbol, exchange, "sell", sellSignals)
		aggregatedSignals = append(aggregatedSignals, aggregatedSignal)
	}

	return aggregatedSignals
}

// createAggregatedTechnicalSignal creates an aggregated signal from multiple signal components
func (sa *SignalAggregator) createAggregatedTechnicalSignal(symbol, exchange, action string, components []SignalComponent) *AggregatedSignal {
	// Combine descriptions
	descriptions := make([]string, len(components))
	indicators := make([]string, len(components))
	totalStrength := 0.0
	totalConfidence := decimal.NewFromFloat(0.0)

	for i, component := range components {
		descriptions[i] = component.Description
		indicators[i] = component.Indicator
		totalStrength += component.Strength
		totalConfidence = totalConfidence.Add(component.Confidence)
	}

	// Calculate aggregated confidence (average with bonus for multiple signals)
	avgConfidence := totalConfidence.Div(decimal.NewFromInt(int64(len(components))))
	// Bonus for multiple confirming signals (up to 20% boost)
	multiSignalBonus := decimal.NewFromFloat(float64(len(components)-1) * 0.1)
	if multiSignalBonus.GreaterThan(decimal.NewFromFloat(0.2)) {
		multiSignalBonus = decimal.NewFromFloat(0.2)
	}
	finalConfidence := avgConfidence.Add(multiSignalBonus)
	if finalConfidence.GreaterThan(decimal.NewFromFloat(1.0)) {
		finalConfidence = decimal.NewFromFloat(1.0)
	}

	// Calculate enhanced profit potential based on signal strength
	baseProfitPotential := decimal.NewFromFloat(2.0)
	if len(components) > 1 {
		// Increase profit potential for multiple confirming signals
		enhancementFactor := decimal.NewFromFloat(1.0 + float64(len(components)-1)*0.3)
		baseProfitPotential = baseProfitPotential.Mul(enhancementFactor)
	}

	// Calculate risk level (lower risk for multiple confirming signals)
	baseRiskLevel := decimal.NewFromFloat(0.2)
	if len(components) > 1 {
		// Reduce risk for multiple confirming signals
		riskReduction := decimal.NewFromFloat(float64(len(components)-1) * 0.03)
		baseRiskLevel = baseRiskLevel.Sub(riskReduction)
		if baseRiskLevel.LessThan(decimal.NewFromFloat(0.1)) {
			baseRiskLevel = decimal.NewFromFloat(0.1)
		}
	}

	strength := sa.determineSignalStrength(finalConfidence)

	// Create combined signal description
	combinedDescription := strings.Join(descriptions, " + ")

	return &AggregatedSignal{
		ID:              uuid.New().String(),
		SignalType:      SignalTypeTechnical,
		Symbol:          symbol,
		Action:          action,
		Strength:        strength,
		Confidence:      finalConfidence,
		ProfitPotential: baseProfitPotential,
		RiskLevel:       baseRiskLevel,
		Exchanges:       []string{exchange},
		Indicators:      indicators,
		Metadata: map[string]interface{}{
			"combined_description": combinedDescription,
			"signal_count":         len(components),
			"individual_signals":   components,
			"exchange":             exchange,
			"avg_strength":         totalStrength / float64(len(components)),
			"aggregated_from":      len(components),
			"indicators":           indicators,
			"signal_text":          combinedDescription,
			"timeframe":            "4H",
			"signal_components":    indicators,
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sa.sigConfig.SignalTTL),
	}
}

// Helper functions

func (sa *SignalAggregator) calculateArbitrageConfidence(opp models.ArbitrageOpportunity) decimal.Decimal {
	// Higher profit percentage = higher confidence
	baseConfidence := decimal.NewFromFloat(0.5)
	profitBonus := opp.ProfitPercentage.Div(decimal.NewFromFloat(10)) // 1% profit = 0.1 confidence boost
	confidence := baseConfidence.Add(profitBonus)

	// Cap at 1.0
	if confidence.GreaterThan(decimal.NewFromFloat(1.0)) {
		confidence = decimal.NewFromFloat(1.0)
	}

	return confidence
}

func (sa *SignalAggregator) determineSignalStrength(confidence decimal.Decimal) SignalStrength {
	if confidence.GreaterThanOrEqual(decimal.NewFromFloat(0.8)) {
		return SignalStrengthStrong
	} else if confidence.GreaterThanOrEqual(decimal.NewFromFloat(0.6)) {
		return SignalStrengthMedium
	}
	return SignalStrengthWeak
}

// determineSignalStrengthWithProfit determines signal strength based on both confidence and profit potential
func (sa *SignalAggregator) determineSignalStrengthWithProfit(confidence, profitPotential decimal.Decimal) SignalStrength {
	// For arbitrage signals, consider both confidence and profit potential
	// Both factors must be considered together, not individually
	// Note: profitPotential is expected as decimal (e.g., 0.015 = 1.5%)

	// Strong: Very high profit (>2%) with good confidence (>0.7) OR very high confidence (>0.9) with decent profit (>1%)
	if (profitPotential.GreaterThanOrEqual(decimal.NewFromFloat(0.02)) && confidence.GreaterThanOrEqual(decimal.NewFromFloat(0.7))) ||
		(confidence.GreaterThanOrEqual(decimal.NewFromFloat(0.9)) && profitPotential.GreaterThanOrEqual(decimal.NewFromFloat(0.01))) {
		return SignalStrengthStrong
	}

	// Medium: Medium profit (>1%) with decent confidence (>0.6) OR high confidence (>0.8) with medium profit (>0.8%)
	if (profitPotential.GreaterThanOrEqual(decimal.NewFromFloat(0.01)) && confidence.GreaterThanOrEqual(decimal.NewFromFloat(0.6))) ||
		(confidence.GreaterThanOrEqual(decimal.NewFromFloat(0.8)) && profitPotential.GreaterThanOrEqual(decimal.NewFromFloat(0.008))) {
		return SignalStrengthMedium
	}

	// Weak: Everything else (including high confidence with very low profit)
	return SignalStrengthWeak
}

func (sa *SignalAggregator) generateSignalHash(signal *AggregatedSignal) string {
	// Create a hash based on signal characteristics
	// Sort exchanges to ensure consistent hashing regardless of order
	sortedExchanges := make([]string, len(signal.Exchanges))
	copy(sortedExchanges, signal.Exchanges)
	sort.Strings(sortedExchanges)

	// Sort indicators as well for consistency
	sortedIndicators := make([]string, len(signal.Indicators))
	copy(sortedIndicators, signal.Indicators)
	sort.Strings(sortedIndicators)

	hashInput := fmt.Sprintf("%s_%s_%s_%s_%s",
		signal.SignalType,
		signal.Symbol,
		signal.Action,
		strings.Join(sortedIndicators, ","),
		strings.Join(sortedExchanges, ","),
	)
	return fmt.Sprintf("%x", []byte(hashInput))
}

func (sa *SignalAggregator) isHashRecent(ctx context.Context, hash string) bool {
	// Check if hash exists in recent fingerprints
	cutoff := time.Now().Add(-sa.sigConfig.DeduplicationWindow)
	var count int64
	query := `SELECT COUNT(*) FROM signal_fingerprints WHERE hash = $1 AND created_at > $2`
	err := sa.db.Pool.QueryRow(ctx, query, hash, cutoff).Scan(&count)

	if err != nil {
		sa.logger.WithError(err).Error("Failed to check hash recency")
		return false
	}

	return count > 0
}

func (sa *SignalAggregator) storeFingerprint(ctx context.Context, fingerprint *SignalFingerprint) {
	query := `INSERT INTO signal_fingerprints (hash, signal_id, created_at) VALUES ($1, $2, $3)`
	_, err := sa.db.Pool.Exec(ctx, query, fingerprint.Hash, fingerprint.SignalID, fingerprint.CreatedAt)
	if err != nil {
		sa.logger.WithError(err).Error("Failed to store signal fingerprint")
	}
}
