package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/internal/ccxt"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type ArbitrageHandler struct {
	db                  *database.PostgresDB
	ccxtService         ccxt.CCXTService
	notificationService *services.NotificationService
	redisClient         *redis.Client
}

type ArbitrageOpportunity struct {
	Symbol          string    `json:"symbol"`
	BuyExchange     string    `json:"buy_exchange"`
	SellExchange    string    `json:"sell_exchange"`
	BuyPrice        float64   `json:"buy_price"`
	SellPrice       float64   `json:"sell_price"`
	ProfitPercent   float64   `json:"profit_percent"`
	ProfitAmount    float64   `json:"profit_amount"`
	Volume          float64   `json:"volume"`
	Timestamp       time.Time `json:"timestamp"`
	OpportunityType string    `json:"opportunity_type"` // "arbitrage", "technical", "ai_generated"
}

type ArbitrageResponse struct {
	Opportunities []ArbitrageOpportunity `json:"opportunities"`
	Count         int                    `json:"count"`
	Timestamp     time.Time              `json:"timestamp"`
}

type ArbitrageHistoryItem struct {
	ID            int       `json:"id"`
	Symbol        string    `json:"symbol"`
	BuyExchange   string    `json:"buy_exchange"`
	SellExchange  string    `json:"sell_exchange"`
	BuyPrice      float64   `json:"buy_price"`
	SellPrice     float64   `json:"sell_price"`
	ProfitPercent float64   `json:"profit_percent"`
	DetectedAt    time.Time `json:"detected_at"`
}

type ArbitrageHistoryResponse struct {
	History []ArbitrageHistoryItem `json:"history"`
	Count   int                    `json:"count"`
	Page    int                    `json:"page"`
	Limit   int                    `json:"limit"`
}

func NewArbitrageHandler(db *database.PostgresDB, ccxtService ccxt.CCXTService, notificationService *services.NotificationService, redisClient *redis.Client) *ArbitrageHandler {
	return &ArbitrageHandler{
		db:                  db,
		ccxtService:         ccxtService,
		notificationService: notificationService,
		redisClient:         redisClient,
	}
}

// GetArbitrageOpportunities finds real-time arbitrage opportunities
func (h *ArbitrageHandler) GetArbitrageOpportunities(c *gin.Context) {
	// Parse query parameters
	minProfitStr := c.DefaultQuery("min_profit", "0.5") // Default 0.5% minimum profit
	maxResults := c.DefaultQuery("limit", "50")
	symbolFilter := c.Query("symbol")

	minProfit, err := strconv.ParseFloat(minProfitStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid min_profit parameter"})
		return
	}

	limit, err := strconv.Atoi(maxResults)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	log.Printf("Finding arbitrage opportunities with min_profit=%.2f, limit=%d, symbol=%s", minProfit, limit, symbolFilter)

	// Get recent market data from database
	opportunities, err := h.FindArbitrageOpportunities(c.Request.Context(), minProfit, limit, symbolFilter)
	if err != nil {
		log.Printf("Error finding arbitrage opportunities: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find arbitrage opportunities"})
		return
	}

	log.Printf("Found %d arbitrage opportunities", len(opportunities))

	// Send notifications for high-profit opportunities
	go h.sendArbitrageNotifications(opportunities)

	response := ArbitrageResponse{
		Opportunities: opportunities,
		Count:         len(opportunities),
		Timestamp:     time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// sendArbitrageNotifications sends notifications for arbitrage opportunities
func (h *ArbitrageHandler) sendArbitrageNotifications(opportunities []ArbitrageOpportunity) {
	if h.notificationService == nil {
		return
	}

	// Filter for high-profit opportunities (> 1%)
	var notifiableOpportunities []services.ArbitrageOpportunity
	for _, opp := range opportunities {
		if opp.ProfitPercent > 1.0 {
			notifiableOpportunities = append(notifiableOpportunities, services.ArbitrageOpportunity{
				Symbol:          opp.Symbol,
				BuyExchange:     opp.BuyExchange,
				SellExchange:    opp.SellExchange,
				BuyPrice:        opp.BuyPrice,
				SellPrice:       opp.SellPrice,
				ProfitPercent:   opp.ProfitPercent,
				ProfitAmount:    opp.ProfitAmount,
				Volume:          opp.Volume,
				Timestamp:       opp.Timestamp,
				OpportunityType: opp.OpportunityType,
			})
		}
	}

	if len(notifiableOpportunities) > 0 {
		ctx := context.Background()
		if err := h.notificationService.NotifyArbitrageOpportunities(ctx, notifiableOpportunities); err != nil {
			log.Printf("Failed to send arbitrage notifications: %v", err)
		}
	}
}

// GetArbitrageHistory returns historical arbitrage opportunities
func (h *ArbitrageHandler) GetArbitrageHistory(c *gin.Context) {
	// Parse pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	symbolFilter := c.Query("symbol")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter (1-100)"})
		return
	}

	offset := (page - 1) * limit

	// Query historical data from database
	history, err := h.getArbitrageHistory(c.Request.Context(), limit, offset, symbolFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get arbitrage history"})
		return
	}

	response := ArbitrageHistoryResponse{
		History: history,
		Count:   len(history),
		Page:    page,
		Limit:   limit,
	}

	c.JSON(http.StatusOK, response)
}

// GetFundingRateArbitrage retrieves funding rate arbitrage opportunities
func (h *ArbitrageHandler) GetFundingRateArbitrage(c *gin.Context) {
	minProfitStr := c.DefaultQuery("min_profit", "0.01") // Default 1% daily
	maxRiskStr := c.DefaultQuery("max_risk", "3.0")      // Default max risk 3.0
	limitStr := c.DefaultQuery("limit", "20")

	minProfit, err := strconv.ParseFloat(minProfitStr, 64)
	if err != nil || minProfit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid min_profit parameter"})
		return
	}

	maxRisk, err := strconv.ParseFloat(maxRiskStr, 64)
	if err != nil || maxRisk < 1.0 || maxRisk > 5.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid max_risk parameter (must be between 1.0 and 5.0)"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter (must be between 1 and 100)"})
		return
	}

	// Get symbols and exchanges from query parameters
	symbols := c.QueryArray("symbols")
	exchanges := c.QueryArray("exchanges")

	// If no symbols specified, use default futures symbols
	if len(symbols) == 0 {
		symbols = []string{
			"BTC/USDT:USDT", "ETH/USDT:USDT", "BNB/USDT:USDT",
			"ADA/USDT:USDT", "SOL/USDT:USDT", "DOT/USDT:USDT",
			"MATIC/USDT:USDT", "AVAX/USDT:USDT", "LINK/USDT:USDT",
			"UNI/USDT:USDT", "XRP/USDT:USDT", "DOGE/USDT:USDT",
		}
	}

	// If no exchanges specified, use default futures exchanges
	if len(exchanges) == 0 {
		exchanges = []string{"binance", "bybit", "okx", "bitget"}
	}

	// Calculate funding rate arbitrage opportunities
	opportunities, err := h.ccxtService.CalculateFundingRateArbitrage(c.Request.Context(), symbols, exchanges, minProfit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate funding rate arbitrage"})
		return
	}

	// Filter by risk score and limit results
	var filteredOpportunities []ccxt.FundingArbitrageOpportunity
	for _, opp := range opportunities {
		if opp.RiskScore <= maxRisk {
			filteredOpportunities = append(filteredOpportunities, opp)
			if len(filteredOpportunities) >= limit {
				break
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"opportunities": filteredOpportunities,
		"count":         len(filteredOpportunities),
		"filters": gin.H{
			"min_profit": minProfit,
			"max_risk":   maxRisk,
			"symbols":    symbols,
			"exchanges":  exchanges,
		},
	})
}

// GetFundingRates retrieves current funding rates for specified exchanges and symbols
func (h *ArbitrageHandler) GetFundingRates(c *gin.Context) {
	exchange := c.Param("exchange")
	if exchange == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange parameter is required"})
		return
	}

	// Get symbols from query parameters
	symbols := c.QueryArray("symbols")

	// Create cache key based on exchange and symbols
	var cacheKey string
	if len(symbols) == 0 {
		cacheKey = fmt.Sprintf("funding_rates:all:%s", exchange)
	} else {
		// Sort symbols for consistent cache key
		sortedSymbols := make([]string, len(symbols))
		copy(sortedSymbols, symbols)
		sort.Strings(sortedSymbols)
		cacheKey = fmt.Sprintf("funding_rates:%s:%s", exchange, strings.Join(sortedSymbols, ","))
	}

	// Check Redis cache first
	if h.redisClient != nil {
		if cachedData, err := h.redisClient.Get(c.Request.Context(), cacheKey).Result(); err == nil {
			var fundingRates []ccxt.FundingRate
			if json.Unmarshal([]byte(cachedData), &fundingRates) == nil {
				c.JSON(http.StatusOK, gin.H{
					"exchange":      exchange,
					"funding_rates": fundingRates,
					"count":         len(fundingRates),
					"cached":        true,
				})
				return
			}
		}
	}

	var fundingRates []ccxt.FundingRate
	var err error

	if len(symbols) == 0 {
		// Get all funding rates for the exchange
		fundingRates, err = h.ccxtService.FetchAllFundingRates(c.Request.Context(), exchange)
	} else {
		// Get funding rates for specific symbols
		fundingRates, err = h.ccxtService.FetchFundingRates(c.Request.Context(), exchange, symbols)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch funding rates"})
		return
	}

	// Cache the result with 1-minute TTL
	if h.redisClient != nil {
		if jsonData, err := json.Marshal(fundingRates); err == nil {
			h.redisClient.Set(c.Request.Context(), cacheKey, jsonData, time.Minute)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"exchange":      exchange,
		"funding_rates": fundingRates,
		"count":         len(fundingRates),
		"cached":        false,
	})
}

// FindArbitrageOpportunities analyzes market data to find arbitrage opportunities
// Uses multiple strategies: price differences, technical analysis, and volatility patterns
func (h *ArbitrageHandler) FindArbitrageOpportunities(ctx context.Context, minProfit float64, limit int, symbolFilter string) ([]ArbitrageOpportunity, error) {
	// Return empty slice if database is not available (for testing)
	if h.db == nil {
		return []ArbitrageOpportunity{}, nil
	}

	var opportunities []ArbitrageOpportunity

	// Strategy 1: Cross-exchange price differences
	crossExchangeOpps, err := h.findCrossExchangeOpportunities(ctx, minProfit, symbolFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find cross-exchange opportunities: %w", err)
	}
	opportunities = append(opportunities, crossExchangeOpps...)

	// Strategy 2: Technical analysis based opportunities
	technicalOpps, err := h.findTechnicalAnalysisOpportunities(ctx, minProfit, symbolFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find technical opportunities: %w", err)
	}
	opportunities = append(opportunities, technicalOpps...)

	// Strategy 3: Volatility and momentum based opportunities
	volatilityOpps, err := h.findVolatilityOpportunities(ctx, minProfit, symbolFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find volatility opportunities: %w", err)
	}
	opportunities = append(opportunities, volatilityOpps...)

	// Strategy 4: Bid-Ask spread analysis
	spreadOpps, err := h.findSpreadOpportunities(ctx, minProfit, symbolFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find spread opportunities: %w", err)
	}
	opportunities = append(opportunities, spreadOpps...)

	// Sort by profit percentage (descending)
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].ProfitPercent > opportunities[j].ProfitPercent
	})

	// Limit results
	if len(opportunities) > limit {
		opportunities = opportunities[:limit]
	}

	return opportunities, nil
}

// findCrossExchangeOpportunities finds arbitrage opportunities across different exchanges
func (h *ArbitrageHandler) findCrossExchangeOpportunities(ctx context.Context, minProfit float64, symbolFilter string) ([]ArbitrageOpportunity, error) {
	// Return empty slice if database is not available
	if h.db == nil {
		return []ArbitrageOpportunity{}, nil
	}

	var opportunities []ArbitrageOpportunity

	// Get recent market data from multiple exchanges
	query := `
		SELECT tp.symbol, e.name as exchange, md.last_price, md.volume_24h, md.timestamp
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		WHERE md.timestamp > NOW() - INTERVAL '5 minutes'
		  AND md.last_price > 0
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND tp.symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " ORDER BY md.timestamp DESC"

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	// Group data by symbol and exchange
	marketData := make(map[string]map[string]struct {
		price     float64
		volume    float64
		timestamp time.Time
	})

	for rows.Next() {
		var symbol, exchange string
		var price, volume float64
		var timestamp time.Time

		if err := rows.Scan(&symbol, &exchange, &price, &volume, &timestamp); err != nil {
			continue
		}

		if marketData[symbol] == nil {
			marketData[symbol] = make(map[string]struct {
				price     float64
				volume    float64
				timestamp time.Time
			})
		}

		// Keep only the most recent data for each exchange
		if existing, exists := marketData[symbol][exchange]; !exists || timestamp.After(existing.timestamp) {
			marketData[symbol][exchange] = struct {
				price     float64
				volume    float64
				timestamp time.Time
			}{price, volume, timestamp}
		}
	}

	// Find cross-exchange arbitrage opportunities
	for symbol, exchanges := range marketData {
		if len(exchanges) >= 2 {
			opportunities = append(opportunities, h.findCrossExchangeArbitrage(symbol, exchanges, minProfit)...)
		}
	}

	return opportunities, nil
}

// findCrossExchangeArbitrage finds arbitrage opportunities between different exchanges
func (h *ArbitrageHandler) findCrossExchangeArbitrage(symbol string, exchanges map[string]struct {
	price     float64
	volume    float64
	timestamp time.Time
}, minProfit float64) []ArbitrageOpportunity {
	var opportunities []ArbitrageOpportunity

	// Find lowest and highest prices across exchanges
	var lowestPrice, highestPrice struct {
		exchange  string
		price     float64
		volume    float64
		timestamp time.Time
	}

	for exchange, data := range exchanges {
		// Find lowest price (best buy opportunity)
		if lowestPrice.price == 0 || data.price < lowestPrice.price {
			lowestPrice.exchange = exchange
			lowestPrice.price = data.price
			lowestPrice.volume = data.volume
			lowestPrice.timestamp = data.timestamp
		}

		// Find highest price (best sell opportunity)
		if highestPrice.price == 0 || data.price > highestPrice.price {
			highestPrice.exchange = exchange
			highestPrice.price = data.price
			highestPrice.volume = data.volume
			highestPrice.timestamp = data.timestamp
		}
	}

	// Calculate profit opportunity using high-precision decimals
	if lowestPrice.price > 0 && highestPrice.price > lowestPrice.price && lowestPrice.exchange != highestPrice.exchange {
		buy := decimal.NewFromFloat(lowestPrice.price)
		sell := decimal.NewFromFloat(highestPrice.price)
		if buy.GreaterThan(decimal.Zero) && sell.GreaterThan(buy) {
			profitPercentDec := sell.Sub(buy).Div(buy).Mul(decimal.NewFromInt(100))
			minProfitDec := decimal.NewFromFloat(minProfit)
			if profitPercentDec.GreaterThanOrEqual(minProfitDec) {
				// Use minimum volume between exchanges
				minVol := math.Min(lowestPrice.volume, highestPrice.volume)
				volDec := decimal.NewFromFloat(minVol)
				profitAmountDec := sell.Sub(buy).Mul(volDec)

				buyF, _ := buy.Float64()
				sellF, _ := sell.Float64()
				profitPercentF, _ := profitPercentDec.Float64()
				profitAmountF, _ := profitAmountDec.Float64()

				opportunity := ArbitrageOpportunity{
					Symbol:          symbol,
					BuyExchange:     lowestPrice.exchange,
					SellExchange:    highestPrice.exchange,
					BuyPrice:        buyF,
					SellPrice:       sellF,
					ProfitPercent:   profitPercentF,
					ProfitAmount:    profitAmountF,
					Volume:          minVol,
					Timestamp:       time.Now(),
					OpportunityType: "arbitrage", // True cross-exchange arbitrage
				}

				opportunities = append(opportunities, opportunity)
			}
		}
	}

	return opportunities
}

// findTechnicalAnalysisOpportunities finds opportunities based on technical indicators
func (h *ArbitrageHandler) findTechnicalAnalysisOpportunities(ctx context.Context, minProfit float64, symbolFilter string) ([]ArbitrageOpportunity, error) {
	// Return empty slice if database is not available
	if h.db == nil {
		return []ArbitrageOpportunity{}, nil
	}

	var opportunities []ArbitrageOpportunity

	// Get recent market data for technical analysis
	query := `
		SELECT tp.symbol, e.name as exchange, md.last_price, md.volume_24h, md.timestamp
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		WHERE md.timestamp > NOW() - INTERVAL '30 minutes'
		  AND md.last_price > 0
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND tp.symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " ORDER BY tp.symbol, e.name, md.timestamp DESC"

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data for technical analysis: %w", err)
	}
	defer rows.Close()

	// Group data by symbol and exchange
	marketData := make(map[string]map[string][]struct {
		price     float64
		volume    float64
		timestamp time.Time
	})

	for rows.Next() {
		var symbol, exchange string
		var price, volume float64
		var timestamp time.Time

		if err := rows.Scan(&symbol, &exchange, &price, &volume, &timestamp); err != nil {
			continue
		}

		if marketData[symbol] == nil {
			marketData[symbol] = make(map[string][]struct {
				price     float64
				volume    float64
				timestamp time.Time
			})
		}

		marketData[symbol][exchange] = append(marketData[symbol][exchange], struct {
			price     float64
			volume    float64
			timestamp time.Time
		}{price, volume, timestamp})
	}

	// Analyze each symbol for technical opportunities
	for symbol, exchanges := range marketData {
		for exchange, priceHistory := range exchanges {
			if len(priceHistory) < 5 {
				continue // Need at least 5 data points
			}

			// Calculate moving averages and volatility
			var prices []float64
			for _, data := range priceHistory {
				prices = append(prices, data.price)
			}

			// Simple moving average
			sma := h.calculateSMA(prices, 5)
			currentPrice := prices[0] // Most recent price

			// Check for oversold/overbought conditions using decimals
			smaDec := decimal.NewFromFloat(sma)
			curDec := decimal.NewFromFloat(currentPrice)
			if smaDec.IsZero() {
				continue
			}
			deviationPercentDec := curDec.Sub(smaDec).Div(smaDec).Mul(decimal.NewFromInt(100)).Abs()
			minProfitDec := decimal.NewFromFloat(minProfit)

			if deviationPercentDec.GreaterThanOrEqual(minProfitDec) {
				// Create technical opportunity
				var buyPriceDec, sellPriceDec decimal.Decimal
				var opportunityType string

				if curDec.LessThan(smaDec) {
					// Oversold - buy opportunity
					buyPriceDec = curDec
					sellPriceDec = smaDec
					opportunityType = "technical_oversold"
				} else {
					// Overbought - sell opportunity
					buyPriceDec = smaDec
					sellPriceDec = curDec
					opportunityType = "technical_overbought"
				}

				// Use a conservative fraction of volume
				volDec := decimal.NewFromFloat(priceHistory[0].volume).Mul(decimal.NewFromFloat(0.1))
				profitAmountDec := sellPriceDec.Sub(buyPriceDec).Abs().Mul(volDec)

				buyF, _ := buyPriceDec.Float64()
				sellF, _ := sellPriceDec.Float64()
				profitPercentF, _ := deviationPercentDec.Float64()
				profitAmountF, _ := profitAmountDec.Float64()
				volF, _ := volDec.Float64()

				opportunity := ArbitrageOpportunity{
					Symbol:          symbol,
					BuyExchange:     exchange,
					SellExchange:    exchange + " (technical)",
					BuyPrice:        buyF,
					SellPrice:       sellF,
					ProfitPercent:   profitPercentF,
					ProfitAmount:    profitAmountF,
					Volume:          volF,
					Timestamp:       time.Now(),
					OpportunityType: opportunityType,
				}
				opportunities = append(opportunities, opportunity)
			}
		}
	}

	return opportunities, nil
}

// findVolatilityOpportunities finds opportunities based on volatility patterns
func (h *ArbitrageHandler) findVolatilityOpportunities(ctx context.Context, minProfit float64, symbolFilter string) ([]ArbitrageOpportunity, error) {
	// Return empty slice if database is not available
	if h.db == nil {
		return []ArbitrageOpportunity{}, nil
	}

	var opportunities []ArbitrageOpportunity

	// Get market data for volatility analysis
	query := `
		SELECT tp.symbol, e.name as exchange, 
		       MIN(md.last_price) as min_price,
		       MAX(md.last_price) as max_price,
		       AVG(md.last_price) as avg_price,
		       AVG(md.volume_24h) as avg_volume
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		WHERE md.timestamp > NOW() - INTERVAL '1 hour'
		  AND md.last_price > 0
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND tp.symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " GROUP BY tp.symbol, e.name HAVING COUNT(*) >= 10"

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query volatility data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var symbol, exchange string
		var minPrice, maxPrice, avgPrice, avgVolume float64

		if err := rows.Scan(&symbol, &exchange, &minPrice, &maxPrice, &avgPrice, &avgVolume); err != nil {
			continue
		}

		// Calculate volatility percentage using decimals
		minDec := decimal.NewFromFloat(minPrice)
		maxDec := decimal.NewFromFloat(maxPrice)
		avgDec := decimal.NewFromFloat(avgPrice)
		if avgDec.IsZero() || maxDec.LessThan(minDec) {
			continue
		}
		volatilityDec := maxDec.Sub(minDec).Div(avgDec).Mul(decimal.NewFromInt(100))
		minProfitDec := decimal.NewFromFloat(minProfit)

		if volatilityDec.GreaterThanOrEqual(minProfitDec) {
			volPortion := decimal.NewFromFloat(avgVolume).Mul(decimal.NewFromFloat(0.05))
			profitAmountDec := maxDec.Sub(minDec).Mul(volPortion)

			buyF, _ := minDec.Float64()
			sellF, _ := maxDec.Float64()
			profitPercentF, _ := volatilityDec.Float64()
			profitAmountF, _ := profitAmountDec.Float64()
			volF, _ := volPortion.Float64()

			opportunity := ArbitrageOpportunity{
				Symbol:          symbol,
				BuyExchange:     exchange,
				SellExchange:    exchange + " (volatility)",
				BuyPrice:        buyF,
				SellPrice:       sellF,
				ProfitPercent:   profitPercentF,
				ProfitAmount:    profitAmountF,
				Volume:          volF,
				Timestamp:       time.Now(),
				OpportunityType: "volatility",
			}
			opportunities = append(opportunities, opportunity)
		}
	}

	return opportunities, nil
}

// findSpreadOpportunities finds opportunities based on bid-ask spreads
func (h *ArbitrageHandler) findSpreadOpportunities(ctx context.Context, minProfit float64, symbolFilter string) ([]ArbitrageOpportunity, error) {
	// Return empty slice if database is not available
	if h.db == nil {
		return []ArbitrageOpportunity{}, nil
	}

	var opportunities []ArbitrageOpportunity

	// Get market data for spread analysis using last_price variations
	query := `
		SELECT tp.symbol, e.name as exchange, md.last_price, md.volume_24h, md.timestamp
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		WHERE md.timestamp > NOW() - INTERVAL '5 minutes'
		  AND md.last_price > 0 AND md.volume_24h > 0
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND tp.symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " ORDER BY md.timestamp DESC"

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query spread data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var symbol, exchange string
		var price, volume float64
		var timestamp time.Time

		if err := rows.Scan(&symbol, &exchange, &price, &volume, &timestamp); err != nil {
			continue
		}

		// Skip this strategy as we don't have bid/ask data
		// This function will return empty opportunities for now
		continue
	}

	return opportunities, nil
}

// calculateSMA calculates Simple Moving Average
func (h *ArbitrageHandler) calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// getArbitrageHistory retrieves historical arbitrage data
func (h *ArbitrageHandler) getArbitrageHistory(ctx context.Context, limit, offset int, symbolFilter string) ([]ArbitrageHistoryItem, error) {
	// Return empty slice if database is not available
	if h.db == nil {
		return []ArbitrageHistoryItem{}, nil
	}

	// For now, we'll simulate historical data since we don't have a dedicated arbitrage_history table
	// In a real implementation, you would store detected opportunities in a separate table
	query := `
		SELECT 
			ROW_NUMBER() OVER (ORDER BY md.timestamp DESC) as id,
			tp.symbol,
			e.name as buy_exchange,
			'simulated' as sell_exchange,
			md.last_price as buy_price,
			md.last_price as sell_price,
			0.0 as profit_percent,
			md.timestamp as detected_at
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		WHERE md.timestamp > NOW() - INTERVAL '24 hours'
		  AND md.last_price > 0
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND tp.symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " ORDER BY md.timestamp DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2) // SAFE: using parameterized query
	args = append(args, limit, offset)

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query arbitrage history: %w", err)
	}
	defer rows.Close()

	var history []ArbitrageHistoryItem
	for rows.Next() {
		var item ArbitrageHistoryItem
		if err := rows.Scan(
			&item.ID,
			&item.Symbol,
			&item.BuyExchange,
			&item.SellExchange,
			&item.BuyPrice,
			&item.SellPrice,
			&item.ProfitPercent,
			&item.DetectedAt,
		); err != nil {
			continue
		}
		history = append(history, item)
	}

	return history, nil
}
