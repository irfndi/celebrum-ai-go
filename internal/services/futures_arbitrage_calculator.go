package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/models"
	"github.com/shopspring/decimal"
)

// FuturesArbitrageCalculator handles all calculations for futures arbitrage opportunities
type FuturesArbitrageCalculator struct {
	// Configuration
	DefaultFundingInterval  int             // Hours between funding payments (usually 8)
	RiskFreeRate            decimal.Decimal // Annual risk-free rate for calculations
	DefaultVolatilityWindow int             // Days for volatility calculation
}

// NewFuturesArbitrageCalculator creates a new calculator instance
func NewFuturesArbitrageCalculator() *FuturesArbitrageCalculator {
	return &FuturesArbitrageCalculator{
		DefaultFundingInterval:  8,
		RiskFreeRate:            decimal.NewFromFloat(0.05), // 5% annual risk-free rate
		DefaultVolatilityWindow: 30,
	}
}

// CalculateFuturesArbitrage calculates a complete futures arbitrage opportunity
func (calc *FuturesArbitrageCalculator) CalculateFuturesArbitrage(
	input models.FuturesArbitrageCalculationInput,
) (*models.FuturesArbitrageOpportunity, error) {

	// Calculate net funding rate (the key profit driver)
	netFundingRate := input.ShortFundingRate.Sub(input.LongFundingRate)

	// Calculate price difference
	priceDiff := input.ShortMarkPrice.Sub(input.LongMarkPrice)
	priceDiffPercentage := decimal.Zero
	if !input.LongMarkPrice.IsZero() {
		priceDiffPercentage = priceDiff.Div(input.LongMarkPrice).Mul(decimal.NewFromInt(100))
	}

	// Calculate time-based profit rates
	hourlyRate := calc.calculateHourlyRate(netFundingRate, input.FundingInterval)
	dailyRate := hourlyRate.Mul(decimal.NewFromInt(24))
	apy := calc.CalculateAPY(hourlyRate)

	// Calculate estimated profits for different time periods
	profit8h := calc.calculatePeriodProfit(input.BaseAmount, netFundingRate, 1) // 1 funding period
	profitDaily := calc.calculatePeriodProfit(input.BaseAmount, hourlyRate, 24)
	profitWeekly := calc.calculatePeriodProfit(input.BaseAmount, hourlyRate, 24*7)
	profitMonthly := calc.calculatePeriodProfit(input.BaseAmount, hourlyRate, 24*30)

	// Calculate risk metrics
	riskScore := calc.CalculateRiskScore(input)
	volatilityScore := calc.calculateVolatilityScore(input)
	liquidityScore := calc.calculateLiquidityScore(input)

	// Calculate position sizing recommendations
	positionSizing := calc.CalculatePositionSizing(input, riskScore)

	// Determine next funding time (simplified - would need real exchange data)
	nextFunding := calc.calculateNextFundingTime(input.FundingInterval)
	timeToFunding := int(time.Until(nextFunding).Minutes())

	opportunity := &models.FuturesArbitrageOpportunity{
		Symbol:                    input.Symbol,
		LongExchange:              input.LongExchange,
		ShortExchange:             input.ShortExchange,
		LongFundingRate:           input.LongFundingRate,
		ShortFundingRate:          input.ShortFundingRate,
		NetFundingRate:            netFundingRate,
		FundingInterval:           input.FundingInterval,
		LongMarkPrice:             input.LongMarkPrice,
		ShortMarkPrice:            input.ShortMarkPrice,
		PriceDifference:           priceDiff,
		PriceDifferencePercentage: priceDiffPercentage,
		HourlyRate:                hourlyRate,
		DailyRate:                 dailyRate,
		APY:                       apy,
		EstimatedProfit8h:         profit8h,
		EstimatedProfitDaily:      profitDaily,
		EstimatedProfitWeekly:     profitWeekly,
		EstimatedProfitMonthly:    profitMonthly,
		RiskScore:                 riskScore,
		VolatilityScore:           volatilityScore,
		LiquidityScore:            liquidityScore,
		RecommendedPositionSize:   positionSizing.KellyPositionSize,
		MaxLeverage:               positionSizing.MaxSafeLeverage,
		RecommendedLeverage:       positionSizing.OptimalLeverage,
		StopLossPercentage:        positionSizing.MaxLossPercentage,
		MinPositionSize:           positionSizing.ConservativeSize,
		MaxPositionSize:           positionSizing.AggressiveSize,
		OptimalPositionSize:       positionSizing.ModerateSize,
		DetectedAt:                time.Now(),
		ExpiresAt:                 time.Now().Add(time.Hour * time.Duration(input.FundingInterval)),
		NextFundingTime:           nextFunding,
		TimeToNextFunding:         timeToFunding,
		IsActive:                  calc.isOpportunityActive(netFundingRate, riskScore),
	}

	return opportunity, nil
}

// calculateHourlyRate converts funding rate to hourly rate
func (calc *FuturesArbitrageCalculator) calculateHourlyRate(netFundingRate decimal.Decimal, fundingInterval int) decimal.Decimal {
	if fundingInterval <= 0 {
		fundingInterval = calc.DefaultFundingInterval
	}

	// Convert funding rate (per funding period) to hourly rate
	return netFundingRate.Div(decimal.NewFromInt(int64(fundingInterval)))
}

// CalculateAPY calculates Annual Percentage Yield from hourly rate
func (calc *FuturesArbitrageCalculator) CalculateAPY(hourlyRate decimal.Decimal) decimal.Decimal {
	// APY = (1 + hourly_rate)^(24*365) - 1
	// Using compound interest formula

	if hourlyRate.IsZero() {
		return decimal.Zero
	}

	// Convert to float for power calculation
	hourlyRateFloat, _ := hourlyRate.Float64()
	hoursPerYear := 24.0 * 365.0

	// Calculate compound growth
	apyFloat := math.Pow(1+hourlyRateFloat, hoursPerYear) - 1

	// Convert back to decimal and express as percentage
	return decimal.NewFromFloat(apyFloat * 100)
}

// calculatePeriodProfit calculates profit for a specific time period
func (calc *FuturesArbitrageCalculator) calculatePeriodProfit(
	baseAmount decimal.Decimal,
	hourlyRate decimal.Decimal,
	hours int,
) decimal.Decimal {
	// Simple compound interest: amount * (1 + rate)^periods - amount
	if hourlyRate.IsZero() || baseAmount.IsZero() {
		return decimal.Zero
	}

	rateFloat, _ := hourlyRate.Float64()
	amountFloat, _ := baseAmount.Float64()

	finalAmount := amountFloat * math.Pow(1+rateFloat, float64(hours))
	profit := finalAmount - amountFloat

	return decimal.NewFromFloat(profit)
}

// CalculateRiskScore calculates overall risk score (0-100)
func (calc *FuturesArbitrageCalculator) CalculateRiskScore(
	input models.FuturesArbitrageCalculationInput,
) decimal.Decimal {
	// Risk factors:
	// 1. Price difference risk (higher price diff = higher risk)
	// 2. Funding rate volatility
	// 3. Exchange reliability
	// 4. Leverage risk

	var riskFactors []decimal.Decimal

	// Price difference risk (0-30 points)
	priceDiffRisk := decimal.Zero
	if !input.LongMarkPrice.IsZero() {
		priceDiffPercentage := input.ShortMarkPrice.Sub(input.LongMarkPrice).Div(input.LongMarkPrice).Abs()
		// Higher price difference = higher risk
		priceDiffRisk = priceDiffPercentage.Mul(decimal.NewFromInt(3000)) // Scale to 0-30
		if priceDiffRisk.GreaterThan(decimal.NewFromInt(30)) {
			priceDiffRisk = decimal.NewFromInt(30)
		}
	}
	riskFactors = append(riskFactors, priceDiffRisk)

	// Funding rate magnitude risk (0-25 points)
	fundingRateRisk := input.LongFundingRate.Sub(input.ShortFundingRate).Abs().Mul(decimal.NewFromInt(2500))
	if fundingRateRisk.GreaterThan(decimal.NewFromInt(25)) {
		fundingRateRisk = decimal.NewFromInt(25)
	}
	riskFactors = append(riskFactors, fundingRateRisk)

	// Leverage risk (0-25 points)
	leverageRisk := decimal.Zero
	if input.MaxLeverageAllowed.GreaterThan(decimal.NewFromInt(1)) {
		// Higher leverage = higher risk
		leverageRisk = input.MaxLeverageAllowed.Sub(decimal.NewFromInt(1)).Mul(decimal.NewFromInt(5))
		if leverageRisk.GreaterThan(decimal.NewFromInt(25)) {
			leverageRisk = decimal.NewFromInt(25)
		}
	}
	riskFactors = append(riskFactors, leverageRisk)

	// Exchange risk (0-20 points) - simplified
	exchangeRisk := decimal.NewFromInt(10) // Base exchange risk
	riskFactors = append(riskFactors, exchangeRisk)

	// Sum all risk factors
	totalRisk := decimal.Zero
	for _, risk := range riskFactors {
		totalRisk = totalRisk.Add(risk)
	}

	// Ensure risk score is between 0-100
	if totalRisk.GreaterThan(decimal.NewFromInt(100)) {
		totalRisk = decimal.NewFromInt(100)
	}

	return totalRisk
}

// calculateVolatilityScore calculates volatility-based risk score
func (calc *FuturesArbitrageCalculator) calculateVolatilityScore(
	input models.FuturesArbitrageCalculationInput,
) decimal.Decimal {
	// Simplified volatility calculation based on price difference
	// In a real implementation, this would use historical price data

	if input.LongMarkPrice.IsZero() {
		return decimal.NewFromInt(50) // Medium volatility as default
	}

	priceDiffPercentage := input.ShortMarkPrice.Sub(input.LongMarkPrice).Div(input.LongMarkPrice).Abs()
	volatilityScore := priceDiffPercentage.Mul(decimal.NewFromInt(1000)) // Scale to 0-100

	if volatilityScore.GreaterThan(decimal.NewFromInt(100)) {
		volatilityScore = decimal.NewFromInt(100)
	}

	return volatilityScore
}

// calculateLiquidityScore calculates liquidity-based score
func (calc *FuturesArbitrageCalculator) calculateLiquidityScore(
	input models.FuturesArbitrageCalculationInput,
) decimal.Decimal {
	// Calculate liquidity score based on available market data
	// Higher score indicates better liquidity

	// Base score from price convergence - smaller spread indicates better liquidity
	priceSpread := input.LongMarkPrice.Sub(input.ShortMarkPrice).Abs()
	priceSpreadPercent := priceSpread.Div(input.LongMarkPrice).Mul(decimal.NewFromInt(100))

	// Score decreases as price spread increases
	spreadScore := decimal.NewFromInt(100).Sub(priceSpreadPercent.Mul(decimal.NewFromFloat(2.0)))
	if spreadScore.LessThan(decimal.NewFromInt(0)) {
		spreadScore = decimal.NewFromInt(0)
	}

	// Base amount consideration - larger amounts may have liquidity impact
	amountScore := decimal.NewFromInt(100)
	if input.BaseAmount.GreaterThan(decimal.NewFromInt(100000)) { // > $100k
		amountScore = decimal.NewFromInt(80) // Reduced score for large amounts
	} else if input.BaseAmount.GreaterThan(decimal.NewFromInt(50000)) { // > $50k
		amountScore = decimal.NewFromInt(90)
	}

	// Exchange-specific liquidity factors (simplified)
	exchangeScore := decimal.NewFromInt(100)
	if input.LongExchange != input.ShortExchange {
		// Different exchanges may have different liquidity profiles
		exchangeScore = decimal.NewFromInt(90)
	}

	// Weighted average of different factors
	liquidityScore := spreadScore.Mul(decimal.NewFromFloat(0.5)).
		Add(amountScore.Mul(decimal.NewFromFloat(0.3))).
		Add(exchangeScore.Mul(decimal.NewFromFloat(0.2)))

	// Ensure score is within 0-100 range
	if liquidityScore.GreaterThan(decimal.NewFromInt(100)) {
		liquidityScore = decimal.NewFromInt(100)
	} else if liquidityScore.LessThan(decimal.NewFromInt(0)) {
		liquidityScore = decimal.NewFromInt(0)
	}

	return liquidityScore
}

// CalculatePositionSizing calculates recommended position sizes and leverage
func (calc *FuturesArbitrageCalculator) CalculatePositionSizing(
	input models.FuturesArbitrageCalculationInput,
	riskScore decimal.Decimal,
) models.FuturesPositionSizing {

	// Kelly Criterion calculation (simplified)
	// Kelly % = (bp - q) / b
	// where b = odds, p = probability of win, q = probability of loss

	// Estimate win probability based on funding rate differential
	netFundingRate := input.ShortFundingRate.Sub(input.LongFundingRate)
	winProbability := decimal.NewFromFloat(0.6) // Base 60% win rate
	if netFundingRate.GreaterThan(decimal.NewFromFloat(0.001)) {
		winProbability = decimal.NewFromFloat(0.75) // Higher win rate for strong signals
	}

	// Calculate Kelly percentage (conservative approach)
	kellyPercentage := winProbability.Sub(decimal.NewFromFloat(0.5)).Mul(decimal.NewFromInt(2))
	if kellyPercentage.LessThan(decimal.Zero) {
		kellyPercentage = decimal.Zero
	}
	if kellyPercentage.GreaterThan(decimal.NewFromFloat(0.25)) {
		kellyPercentage = decimal.NewFromFloat(0.25) // Cap at 25%
	}

	// Calculate position sizes based on available capital
	kellyPositionSize := input.AvailableCapital.Mul(kellyPercentage)

	// Risk-adjusted position sizes
	riskAdjustment := decimal.NewFromInt(100).Sub(riskScore).Div(decimal.NewFromInt(100))

	conservativeSize := input.AvailableCapital.Mul(decimal.NewFromFloat(0.05)).Mul(riskAdjustment)
	moderateSize := input.AvailableCapital.Mul(decimal.NewFromFloat(0.15)).Mul(riskAdjustment)
	aggressiveSize := input.AvailableCapital.Mul(decimal.NewFromFloat(0.30)).Mul(riskAdjustment)

	// Leverage calculations
	maxSafeLeverage := calc.calculateMaxSafeLeverage(riskScore)
	optimalLeverage := maxSafeLeverage.Mul(decimal.NewFromFloat(0.7)) // 70% of max safe
	minLeverage := decimal.NewFromInt(1)

	// Risk management
	maxLossPercentage := calc.calculateMaxLossPercentage(input.UserRiskTolerance)
	stopLossPrice := input.LongMarkPrice.Mul(decimal.NewFromInt(1).Sub(maxLossPercentage.Div(decimal.NewFromInt(100))))
	takeProfitPrice := input.LongMarkPrice.Mul(decimal.NewFromInt(1).Add(netFundingRate.Mul(decimal.NewFromInt(3))))

	return models.FuturesPositionSizing{
		KellyPercentage:   kellyPercentage.Mul(decimal.NewFromInt(100)),
		KellyPositionSize: kellyPositionSize,
		ConservativeSize:  conservativeSize,
		ModerateSize:      moderateSize,
		AggressiveSize:    aggressiveSize,
		MinLeverage:       minLeverage,
		OptimalLeverage:   optimalLeverage,
		MaxSafeLeverage:   maxSafeLeverage,
		StopLossPrice:     stopLossPrice,
		TakeProfitPrice:   takeProfitPrice,
		MaxLossPercentage: maxLossPercentage,
	}
}

// calculateMaxSafeLeverage determines maximum safe leverage based on risk
func (calc *FuturesArbitrageCalculator) calculateMaxSafeLeverage(riskScore decimal.Decimal) decimal.Decimal {
	// Lower risk = higher safe leverage
	// Risk score 0-20: up to 10x leverage
	// Risk score 20-40: up to 5x leverage
	// Risk score 40-60: up to 3x leverage
	// Risk score 60-80: up to 2x leverage
	// Risk score 80+: 1x leverage only

	if riskScore.LessThan(decimal.NewFromInt(20)) {
		return decimal.NewFromInt(10)
	} else if riskScore.LessThan(decimal.NewFromInt(40)) {
		return decimal.NewFromInt(5)
	} else if riskScore.LessThan(decimal.NewFromInt(60)) {
		return decimal.NewFromInt(3)
	} else if riskScore.LessThan(decimal.NewFromInt(80)) {
		return decimal.NewFromInt(2)
	}

	return decimal.NewFromInt(1)
}

// calculateMaxLossPercentage determines maximum acceptable loss based on risk tolerance
func (calc *FuturesArbitrageCalculator) calculateMaxLossPercentage(riskTolerance string) decimal.Decimal {
	switch riskTolerance {
	case "low":
		return decimal.NewFromFloat(2.0) // 2% max loss
	case "medium":
		return decimal.NewFromFloat(5.0) // 5% max loss
	case "high":
		return decimal.NewFromFloat(10.0) // 10% max loss
	default:
		return decimal.NewFromFloat(3.0) // 3% default
	}
}

// calculateNextFundingTime estimates next funding time
func (calc *FuturesArbitrageCalculator) calculateNextFundingTime(fundingInterval int) time.Time {
	// Calculate next funding time based on interval
	now := time.Now().UTC()

	// Common funding intervals: 8 hours (00:00, 08:00, 16:00) or 4 hours
	var fundingHours []int
	if fundingInterval == 4 {
		fundingHours = []int{0, 4, 8, 12, 16, 20}
	} else {
		// Default to 8-hour intervals
		fundingHours = []int{0, 8, 16}
	}

	currentHour := now.Hour()
	var nextFundingHour int

	for _, hour := range fundingHours {
		if hour > currentHour {
			nextFundingHour = hour
			break
		}
	}

	// If no funding hour found today, use first hour of next day
	if nextFundingHour == 0 && currentHour >= fundingHours[len(fundingHours)-1] {
		nextFundingHour = fundingHours[0]
		return time.Date(now.Year(), now.Month(), now.Day()+1, nextFundingHour, 0, 0, 0, time.UTC)
	}

	return time.Date(now.Year(), now.Month(), now.Day(), nextFundingHour, 0, 0, 0, time.UTC)
}

// isOpportunityActive determines if an opportunity is worth pursuing
func (calc *FuturesArbitrageCalculator) isOpportunityActive(
	netFundingRate decimal.Decimal,
	riskScore decimal.Decimal,
) bool {
	// Minimum thresholds for active opportunities
	minFundingRate := decimal.NewFromFloat(0.00001) // 0.001% minimum (realistic for crypto funding rates)
	maxRiskScore := decimal.NewFromInt(80)          // Maximum 80% risk

	return netFundingRate.GreaterThan(minFundingRate) && riskScore.LessThan(maxRiskScore)
}

// CalculateRiskMetrics calculates comprehensive risk assessment
func (calc *FuturesArbitrageCalculator) CalculateRiskMetrics(
	input models.FuturesArbitrageCalculationInput,
	historicalData []models.FundingRateHistoryPoint,
) models.FuturesArbitrageRiskMetrics {

	// Calculate various risk metrics
	priceCorrelation := calc.calculatePriceCorrelation(historicalData)
	priceVolatility := calc.calculatePriceVolatility(historicalData)
	fundingRateVolatility := calc.calculateFundingRateVolatility(historicalData)
	fundingRateStability := calc.calculateFundingRateStability(historicalData)

	// Market risk metrics (simplified)
	bidAskSpread := decimal.NewFromFloat(0.001) // 0.1% default spread
	marketDepth := decimal.NewFromInt(75)       // 75% depth score
	slippageRisk := decimal.NewFromInt(25)      // 25% slippage risk

	// Exchange risk (would be enhanced with real data)
	exchangeReliability := decimal.NewFromInt(85) // 85% reliability
	counterpartyRisk := decimal.NewFromInt(15)    // 15% counterparty risk

	// Calculate overall risk score
	overallRisk := calc.calculateOverallRiskScore(
		priceVolatility, fundingRateVolatility, slippageRisk, counterpartyRisk,
	)

	// Determine risk category and recommendation
	riskCategory, recommendation := calc.categorizeRisk(overallRisk)

	return models.FuturesArbitrageRiskMetrics{
		PriceCorrelation:      priceCorrelation,
		PriceVolatility:       priceVolatility,
		FundingRateVolatility: fundingRateVolatility,
		FundingRateStability:  fundingRateStability,
		BidAskSpread:          bidAskSpread,
		MarketDepth:           marketDepth,
		SlippageRisk:          slippageRisk,
		ExchangeReliability:   exchangeReliability,
		CounterpartyRisk:      counterpartyRisk,
		OverallRiskScore:      overallRisk,
		RiskCategory:          riskCategory,
		Recommendation:        recommendation,
	}
}

// Helper methods for risk calculations
func (calc *FuturesArbitrageCalculator) calculatePriceCorrelation(data []models.FundingRateHistoryPoint) decimal.Decimal {
	if len(data) < 2 {
		return decimal.NewFromFloat(0.95) // Default high correlation
	}

	// Extract mark prices for correlation calculation
	prices := make([]float64, len(data))
	for i, point := range data {
		prices[i] = point.MarkPrice.InexactFloat64()
	}

	// Calculate Pearson correlation coefficient
	return calc.calculatePearsonCorrelation(prices)
}

// calculatePearsonCorrelation calculates Pearson correlation coefficient
func (calc *FuturesArbitrageCalculator) calculatePearsonCorrelation(prices []float64) decimal.Decimal {
	n := float64(len(prices))
	if n < 2 {
		return decimal.NewFromFloat(0.0)
	}

	// Calculate mean
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	mean := sum / n

	// Calculate covariance and standard deviations
	sumCovariance := 0.0
	sumVarianceX := 0.0
	sumVarianceY := 0.0

	// Use price series with lag 1 for correlation calculation
	for i := 0; i < len(prices)-1; i++ {
		x := prices[i] - mean
		y := prices[i+1] - mean

		sumCovariance += x * y
		sumVarianceX += x * x
		sumVarianceY += y * y
	}

	// Calculate correlation coefficient
	if sumVarianceX == 0 || sumVarianceY == 0 {
		return decimal.NewFromFloat(0.0)
	}

	correlation := sumCovariance / (math.Sqrt(sumVarianceX) * math.Sqrt(sumVarianceY))

	// Clamp correlation between -1 and 1
	if correlation > 1.0 {
		correlation = 1.0
	} else if correlation < -1.0 {
		correlation = -1.0
	}

	return decimal.NewFromFloat(correlation)
}

func (calc *FuturesArbitrageCalculator) calculatePriceVolatility(data []models.FundingRateHistoryPoint) decimal.Decimal {
	if len(data) < 2 {
		return decimal.NewFromInt(50) // Default volatility
	}

	// Calculate price volatility from historical data
	var prices []float64
	for _, point := range data {
		price, _ := point.MarkPrice.Float64()
		prices = append(prices, price)
	}

	// Calculate standard deviation
	mean := calc.calculateMean(prices)
	variance := calc.calculateVariance(prices, mean)
	stdDev := math.Sqrt(variance)

	// Convert to percentage and scale to 0-100
	volatility := (stdDev / mean) * 100
	if volatility > 100 {
		volatility = 100
	}

	return decimal.NewFromFloat(volatility)
}

func (calc *FuturesArbitrageCalculator) calculateFundingRateVolatility(data []models.FundingRateHistoryPoint) decimal.Decimal {
	if len(data) < 2 {
		return decimal.NewFromInt(30) // Default funding rate volatility
	}

	var rates []float64
	for _, point := range data {
		rate, _ := point.FundingRate.Float64()
		rates = append(rates, rate)
	}

	mean := calc.calculateMean(rates)
	variance := calc.calculateVariance(rates, mean)
	stdDev := math.Sqrt(variance)

	// Scale to 0-100
	volatility := stdDev * 10000 // Scale up for funding rates
	if volatility > 100 {
		volatility = 100
	}

	return decimal.NewFromFloat(volatility)
}

func (calc *FuturesArbitrageCalculator) calculateFundingRateStability(data []models.FundingRateHistoryPoint) decimal.Decimal {
	// Stability is inverse of volatility
	volatility := calc.calculateFundingRateVolatility(data)
	return decimal.NewFromInt(100).Sub(volatility)
}

func (calc *FuturesArbitrageCalculator) calculateOverallRiskScore(
	priceVol, fundingVol, slippage, counterparty decimal.Decimal,
) decimal.Decimal {
	// Weighted average of risk factors
	weights := map[string]decimal.Decimal{
		"price":        decimal.NewFromFloat(0.3),
		"funding":      decimal.NewFromFloat(0.25),
		"slippage":     decimal.NewFromFloat(0.25),
		"counterparty": decimal.NewFromFloat(0.2),
	}

	overallRisk := priceVol.Mul(weights["price"]).
		Add(fundingVol.Mul(weights["funding"])).
		Add(slippage.Mul(weights["slippage"])).
		Add(counterparty.Mul(weights["counterparty"]))

	return overallRisk
}

func (calc *FuturesArbitrageCalculator) categorizeRisk(riskScore decimal.Decimal) (string, string) {
	if riskScore.LessThan(decimal.NewFromInt(25)) {
		return "low", "Excellent opportunity with minimal risk. Recommended for all risk profiles."
	} else if riskScore.LessThan(decimal.NewFromInt(50)) {
		return "medium", "Good opportunity with moderate risk. Suitable for balanced portfolios."
	} else if riskScore.LessThan(decimal.NewFromInt(75)) {
		return "high", "Higher risk opportunity. Only suitable for aggressive risk tolerance."
	}
	return "extreme", "Very high risk. Not recommended for most traders."
}

// Mathematical helper functions
func (calc *FuturesArbitrageCalculator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (calc *FuturesArbitrageCalculator) calculateVariance(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values)-1)
}

// CalculateArbitrageOpportunities calculates regular price arbitrage opportunities across exchanges
func (calc *FuturesArbitrageCalculator) CalculateArbitrageOpportunities(
	ctx context.Context,
	marketData map[string][]models.MarketData,
) ([]models.ArbitrageOpportunity, error) {

	var opportunities []models.ArbitrageOpportunity

	// Group market data by symbol across all exchanges
	symbolPrices := make(map[string]map[string]models.MarketData)

	for exchange, data := range marketData {
		for _, marketData := range data {
			symbol := ""
			if marketData.TradingPair != nil && marketData.TradingPair.Symbol != "" {
				symbol = marketData.TradingPair.Symbol
			}

			if symbol == "" {
				continue
			}

			if symbolPrices[symbol] == nil {
				symbolPrices[symbol] = make(map[string]models.MarketData)
			}
			symbolPrices[symbol][exchange] = marketData
		}
	}

	// Calculate arbitrage opportunities for each symbol
	for symbol, exchangeData := range symbolPrices {
		if len(exchangeData) < 2 {
			continue // Need at least 2 exchanges for arbitrage
		}

		symbolOpportunities := calc.calculateSymbolArbitrage(symbol, exchangeData)
		opportunities = append(opportunities, symbolOpportunities...)
	}

	return opportunities, nil
}

// calculateSymbolArbitrage calculates arbitrage opportunities for a specific symbol
func (calc *FuturesArbitrageCalculator) calculateSymbolArbitrage(
	symbol string,
	exchangeData map[string]models.MarketData,
) []models.ArbitrageOpportunity {

	var opportunities []models.ArbitrageOpportunity

	// Find the lowest and highest prices
	var lowestPrice decimal.Decimal
	var highestPrice decimal.Decimal
	var lowestExchange string
	var highestExchange string
	var lowestData, highestData models.MarketData

	first := true
	for exchange, data := range exchangeData {
		if first {
			lowestPrice = data.LastPrice
			highestPrice = data.LastPrice
			lowestExchange = exchange
			highestExchange = exchange
			lowestData = data
			highestData = data
			first = false
			continue
		}

		if data.LastPrice.LessThan(lowestPrice) {
			lowestPrice = data.LastPrice
			lowestExchange = exchange
			lowestData = data
		}

		if data.LastPrice.GreaterThan(highestPrice) {
			highestPrice = data.LastPrice
			highestExchange = exchange
			highestData = data
		}
	}

	// Calculate profit percentage
	if lowestPrice.IsZero() {
		return opportunities
	}

	priceDifference := highestPrice.Sub(lowestPrice)
	profitPercentage := priceDifference.Div(lowestPrice).Mul(decimal.NewFromInt(100))

	// Only create opportunity if profit is meaningful (above 0.1%)
	if profitPercentage.GreaterThan(decimal.NewFromFloat(0.1)) {
		// Calculate estimated profit for a $10,000 position
		baseAmount := decimal.NewFromInt(10000)
		estimatedAmount := baseAmount.Div(lowestPrice).Mul(highestPrice)
		_ = estimatedAmount.Sub(baseAmount) // profitAmount for logging/debugging

		opportunity := models.ArbitrageOpportunity{
			ID:               fmt.Sprintf("%s_%s_%s_%d", symbol, lowestExchange, highestExchange, time.Now().Unix()),
			TradingPairID:    lowestData.TradingPairID,
			BuyExchangeID:    lowestData.ExchangeID,
			SellExchangeID:   highestData.ExchangeID,
			BuyPrice:         lowestPrice,
			SellPrice:        highestPrice,
			ProfitPercentage: profitPercentage,
			DetectedAt:       time.Now(),
			ExpiresAt:        time.Now().Add(time.Minute * 30), // Expire after 30 minutes
			BuyExchange:      &models.Exchange{ID: lowestData.ExchangeID, Name: lowestExchange},
			SellExchange:     &models.Exchange{ID: highestData.ExchangeID, Name: highestExchange},
			TradingPair:      lowestData.TradingPair,
		}

		opportunities = append(opportunities, opportunity)
	}

	return opportunities
}
