package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/internal/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// FuturesArbitrageHandler handles futures arbitrage related endpoints
type FuturesArbitrageHandler struct {
	db         *pgxpool.Pool
	calculator *services.FuturesArbitrageCalculator
}

// NewFuturesArbitrageHandler creates a new futures arbitrage handler
func NewFuturesArbitrageHandler(db *pgxpool.Pool) *FuturesArbitrageHandler {
	return &FuturesArbitrageHandler{
		db:         db,
		calculator: services.NewFuturesArbitrageCalculator(),
	}
}

// GetFuturesArbitrageOpportunities handles GET /api/futures-arbitrage/opportunities
func (h *FuturesArbitrageHandler) GetFuturesArbitrageOpportunities(c *gin.Context) {
	// Parse query parameters
	req := h.parseArbitrageRequest(c)

	// Get opportunities from database
	opportunities, totalCount, err := h.getFuturesOpportunitiesFromDB(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch opportunities", "details": err.Error()})
		return
	}

	// Calculate market summary
	marketSummary := h.calculateMarketSummary(opportunities)

	// Prepare response
	response := models.FuturesArbitrageResponse{
		Opportunities: opportunities,
		Count:         len(opportunities),
		TotalCount:    totalCount,
		Page:          req.Page,
		Limit:         req.Limit,
		Timestamp:     time.Now(),
		MarketSummary: marketSummary,
	}

	// Include strategies if requested
	if req.IncludePositionSizing {
		strategies, err := h.generateStrategies(opportunities, req)
		if err == nil {
			response.Strategies = strategies
		}
	}

	c.JSON(http.StatusOK, response)
}

// CalculateFuturesArbitrage handles POST /api/futures-arbitrage/calculate
func (h *FuturesArbitrageHandler) CalculateFuturesArbitrage(c *gin.Context) {
	var input models.FuturesArbitrageCalculationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate input
	if err := h.validateCalculationInput(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input parameters", "details": err.Error()})
		return
	}

	// Calculate arbitrage opportunity
	opportunity, err := h.calculator.CalculateFuturesArbitrage(input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate arbitrage", "details": err.Error()})
		return
	}

	// Get historical data for risk metrics if available
	historicalData, _ := h.getFundingRateHistory(input.Symbol, input.LongExchange, input.ShortExchange)

	// Calculate comprehensive risk metrics
	riskMetrics := h.calculator.CalculateRiskMetrics(input, historicalData)

	// Store opportunity in database
	if err := h.storeFuturesOpportunity(opportunity, &riskMetrics); err != nil {
		// Log error but don't fail the request
		// In production, you'd want proper logging here
	}

	response := struct {
		Opportunity *models.FuturesArbitrageOpportunity `json:"opportunity"`
		RiskMetrics *models.FuturesArbitrageRiskMetrics `json:"risk_metrics"`
		Timestamp   time.Time                           `json:"timestamp"`
	}{
		Opportunity: opportunity,
		RiskMetrics: &riskMetrics,
		Timestamp:   time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetFuturesArbitrageStrategy handles GET /api/futures-arbitrage/strategy/{id}
func (h *FuturesArbitrageHandler) GetFuturesArbitrageStrategy(c *gin.Context) {
	strategyID := c.Param("id")

	strategy, err := h.getFuturesStrategyFromDB(strategyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch strategy", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// GetFuturesMarketSummary handles GET /api/futures-arbitrage/market-summary
func (h *FuturesArbitrageHandler) GetFuturesMarketSummary(c *gin.Context) {
	// Get all active opportunities
	opportunities, _, err := h.getFuturesOpportunitiesFromDB(models.FuturesArbitrageRequest{
		Limit: 1000, // Get all for summary
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch market data", "details": err.Error()})
		return
	}

	marketSummary := h.calculateMarketSummary(opportunities)

	response := struct {
		MarketSummary models.FuturesMarketSummary `json:"market_summary"`
		Timestamp     time.Time                   `json:"timestamp"`
	}{
		MarketSummary: marketSummary,
		Timestamp:     time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetPositionSizingRecommendation handles POST /api/futures-arbitrage/position-sizing
func (h *FuturesArbitrageHandler) GetPositionSizingRecommendation(c *gin.Context) {
	var input models.FuturesArbitrageCalculationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Calculate risk score first
	riskScore := h.calculator.CalculateRiskScore(input)

	// Calculate position sizing
	positionSizing := h.calculator.CalculatePositionSizing(input, riskScore)

	response := struct {
		PositionSizing models.FuturesPositionSizing `json:"position_sizing"`
		RiskScore      decimal.Decimal              `json:"risk_score"`
		Timestamp      time.Time                    `json:"timestamp"`
	}{
		PositionSizing: positionSizing,
		RiskScore:      riskScore,
		Timestamp:      time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *FuturesArbitrageHandler) parseArbitrageRequest(c *gin.Context) models.FuturesArbitrageRequest {
	req := models.FuturesArbitrageRequest{
		Limit: 50, // Default limit
		Page:  1,  // Default page
	}

	// Parse query parameters
	if symbols := c.Query("symbols"); symbols != "" {
		req.Symbols = []string{symbols} // Simplified - could parse comma-separated
	}

	if exchanges := c.Query("exchanges"); exchanges != "" {
		req.Exchanges = []string{exchanges} // Simplified - could parse comma-separated
	}

	if minAPY := c.Query("min_apy"); minAPY != "" {
		if val, err := decimal.NewFromString(minAPY); err == nil {
			req.MinAPY = val
		}
	}

	if maxRisk := c.Query("max_risk_score"); maxRisk != "" {
		if val, err := decimal.NewFromString(maxRisk); err == nil {
			req.MaxRiskScore = val
		}
	}

	if riskTolerance := c.Query("risk_tolerance"); riskTolerance != "" {
		req.RiskTolerance = riskTolerance
	}

	if capital := c.Query("available_capital"); capital != "" {
		if val, err := decimal.NewFromString(capital); err == nil {
			req.AvailableCapital = val
		}
	}

	if maxLev := c.Query("max_leverage"); maxLev != "" {
		if val, err := decimal.NewFromString(maxLev); err == nil {
			req.MaxLeverage = val
		}
	}

	if timeHorizon := c.Query("time_horizon"); timeHorizon != "" {
		req.TimeHorizon = timeHorizon
	}

	if includeRisk := c.Query("include_risk_metrics"); includeRisk == "true" {
		req.IncludeRiskMetrics = true
	}

	if includePositioning := c.Query("include_position_sizing"); includePositioning == "true" {
		req.IncludePositionSizing = true
	}

	if limit := c.Query("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil && val > 0 && val <= 1000 {
			req.Limit = val
		}
	}

	if page := c.Query("page"); page != "" {
		if val, err := strconv.Atoi(page); err == nil && val > 0 {
			req.Page = val
		}
	}

	return req
}

func (h *FuturesArbitrageHandler) validateCalculationInput(input models.FuturesArbitrageCalculationInput) error {
	if input.Symbol == "" {
		return utils.NewValidationError("symbol is required")
	}
	if input.LongExchange == "" {
		return utils.NewValidationError("long_exchange is required")
	}
	if input.ShortExchange == "" {
		return utils.NewValidationError("short_exchange is required")
	}
	if input.LongMarkPrice.IsZero() || input.ShortMarkPrice.IsZero() {
		return utils.NewValidationError("mark prices must be greater than zero")
	}
	if input.BaseAmount.IsZero() {
		return utils.NewValidationError("base_amount must be greater than zero")
	}
	if input.FundingInterval <= 0 {
		input.FundingInterval = 8 // Default to 8 hours
	}

	return nil
}

func (h *FuturesArbitrageHandler) getFuturesOpportunitiesFromDB(req models.FuturesArbitrageRequest) ([]models.FuturesArbitrageOpportunity, int, error) {
	// Build query with filters
	query := `
		SELECT 
			id, symbol, base_currency, quote_currency,
			long_exchange, short_exchange, long_exchange_id, short_exchange_id,
			long_funding_rate, short_funding_rate, net_funding_rate, funding_interval,
			long_mark_price, short_mark_price, price_difference, price_difference_percentage,
			hourly_rate, daily_rate, apy,
			estimated_profit_8h, estimated_profit_daily, estimated_profit_weekly, estimated_profit_monthly,
			risk_score, volatility_score, liquidity_score,
			recommended_position_size, max_leverage, recommended_leverage, stop_loss_percentage,
			min_position_size, max_position_size, optimal_position_size,
			detected_at, expires_at, next_funding_time, time_to_next_funding, is_active,
			market_trend, volume_24h, open_interest
		FROM futures_arbitrage_opportunities
		WHERE is_active = true AND expires_at > NOW()
	`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	if len(req.Symbols) > 0 {
		query += " AND symbol = ANY($" + strconv.Itoa(argIndex) + ")"
		args = append(args, req.Symbols)
		argIndex++
	}

	if !req.MinAPY.IsZero() {
		query += " AND apy >= $" + strconv.Itoa(argIndex)
		args = append(args, req.MinAPY)
		argIndex++
	}

	if !req.MaxRiskScore.IsZero() {
		query += " AND risk_score <= $" + strconv.Itoa(argIndex)
		args = append(args, req.MaxRiskScore)
		argIndex++
	}

	// Order by APY descending, risk score ascending
	query += " ORDER BY apy DESC, risk_score ASC"

	// Add pagination
	offset := (req.Page - 1) * req.Limit
	query += " LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, req.Limit, offset)

	// Execute query
	rows, err := h.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var opportunities []models.FuturesArbitrageOpportunity
	for rows.Next() {
		var opp models.FuturesArbitrageOpportunity
		err := rows.Scan(
			&opp.ID, &opp.Symbol, &opp.BaseCurrency, &opp.QuoteCurrency,
			&opp.LongExchange, &opp.ShortExchange, &opp.LongExchangeID, &opp.ShortExchangeID,
			&opp.LongFundingRate, &opp.ShortFundingRate, &opp.NetFundingRate, &opp.FundingInterval,
			&opp.LongMarkPrice, &opp.ShortMarkPrice, &opp.PriceDifference, &opp.PriceDifferencePercentage,
			&opp.HourlyRate, &opp.DailyRate, &opp.APY,
			&opp.EstimatedProfit8h, &opp.EstimatedProfitDaily, &opp.EstimatedProfitWeekly, &opp.EstimatedProfitMonthly,
			&opp.RiskScore, &opp.VolatilityScore, &opp.LiquidityScore,
			&opp.RecommendedPositionSize, &opp.MaxLeverage, &opp.RecommendedLeverage, &opp.StopLossPercentage,
			&opp.MinPositionSize, &opp.MaxPositionSize, &opp.OptimalPositionSize,
			&opp.DetectedAt, &opp.ExpiresAt, &opp.NextFundingTime, &opp.TimeToNextFunding, &opp.IsActive,
			&opp.MarketTrend, &opp.Volume24h, &opp.OpenInterest,
		)
		if err != nil {
			return nil, 0, err
		}
		opportunities = append(opportunities, opp)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW()"
	var totalCount int
	err = h.db.QueryRow(context.Background(), countQuery).Scan(&totalCount)
	if err != nil {
		totalCount = len(opportunities) // Fallback
	}

	return opportunities, totalCount, nil
}

func (h *FuturesArbitrageHandler) calculateMarketSummary(opportunities []models.FuturesArbitrageOpportunity) models.FuturesMarketSummary {
	if len(opportunities) == 0 {
		return models.FuturesMarketSummary{
			TotalOpportunities: 0,
			MarketCondition:    "unfavorable",
			FundingRateTrend:   "stable",
		}
	}

	// Calculate averages and metrics
	totalAPY := decimal.Zero
	totalRisk := decimal.Zero
	highestAPY := decimal.Zero
	volatilitySum := decimal.Zero

	for _, opp := range opportunities {
		totalAPY = totalAPY.Add(opp.APY)
		totalRisk = totalRisk.Add(opp.RiskScore)
		volatilitySum = volatilitySum.Add(opp.VolatilityScore)

		if opp.APY.GreaterThan(highestAPY) {
			highestAPY = opp.APY
		}
	}

	count := decimal.NewFromInt(int64(len(opportunities)))
	averageAPY := totalAPY.Div(count)
	averageRisk := totalRisk.Div(count)
	averageVolatility := volatilitySum.Div(count)

	// Determine market condition
	marketCondition := "neutral"
	if averageAPY.GreaterThan(decimal.NewFromInt(15)) && averageRisk.LessThan(decimal.NewFromInt(50)) {
		marketCondition = "favorable"
	} else if averageAPY.LessThan(decimal.NewFromInt(5)) || averageRisk.GreaterThan(decimal.NewFromInt(70)) {
		marketCondition = "unfavorable"
	}

	// Determine funding rate trend (simplified)
	fundingRateTrend := "stable"
	if averageAPY.GreaterThan(decimal.NewFromInt(20)) {
		fundingRateTrend = "increasing"
	} else if averageAPY.LessThan(decimal.NewFromInt(5)) {
		fundingRateTrend = "decreasing"
	}

	// Recommended strategy
	recommendedStrategy := "conservative"
	if marketCondition == "favorable" && averageRisk.LessThan(decimal.NewFromInt(40)) {
		recommendedStrategy = "aggressive"
	} else if marketCondition == "neutral" {
		recommendedStrategy = "moderate"
	}

	return models.FuturesMarketSummary{
		TotalOpportunities:  len(opportunities),
		AverageAPY:          averageAPY,
		HighestAPY:          highestAPY,
		AverageRiskScore:    averageRisk,
		MarketVolatility:    averageVolatility,
		FundingRateTrend:    fundingRateTrend,
		RecommendedStrategy: recommendedStrategy,
		MarketCondition:     marketCondition,
	}
}

func (h *FuturesArbitrageHandler) getFundingRateHistory(symbol, longExchange, shortExchange string) ([]models.FundingRateHistoryPoint, error) {
	// This would query the funding_rate_history table
	// For now, return empty slice
	return []models.FundingRateHistoryPoint{}, nil
}

func (h *FuturesArbitrageHandler) storeFuturesOpportunity(opportunity *models.FuturesArbitrageOpportunity, riskMetrics *models.FuturesArbitrageRiskMetrics) error {
	// Store opportunity in database
	// This would be a full implementation with proper error handling
	return nil
}

func (h *FuturesArbitrageHandler) getFuturesStrategyFromDB(strategyID string) (*models.FuturesArbitrageStrategy, error) {
	// This would query the futures_arbitrage_strategies table
	return nil, pgx.ErrNoRows
}

func (h *FuturesArbitrageHandler) generateStrategies(opportunities []models.FuturesArbitrageOpportunity, req models.FuturesArbitrageRequest) ([]models.FuturesArbitrageStrategy, error) {
	// This would generate complete trading strategies
	return []models.FuturesArbitrageStrategy{}, nil
}
