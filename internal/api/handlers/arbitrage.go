package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
)

type ArbitrageHandler struct {
	db          *database.PostgresDB
	ccxtService ccxt.CCXTService
}

type ArbitrageOpportunity struct {
	Symbol        string    `json:"symbol"`
	BuyExchange   string    `json:"buy_exchange"`
	SellExchange  string    `json:"sell_exchange"`
	BuyPrice      float64   `json:"buy_price"`
	SellPrice     float64   `json:"sell_price"`
	ProfitPercent float64   `json:"profit_percent"`
	ProfitAmount  float64   `json:"profit_amount"`
	Volume        float64   `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
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

func NewArbitrageHandler(db *database.PostgresDB, ccxtService ccxt.CCXTService) *ArbitrageHandler {
	return &ArbitrageHandler{
		db:          db,
		ccxtService: ccxtService,
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

	// Get recent market data from database
	opportunities, err := h.findArbitrageOpportunities(c.Request.Context(), minProfit, limit, symbolFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find arbitrage opportunities"})
		return
	}

	response := ArbitrageResponse{
		Opportunities: opportunities,
		Count:         len(opportunities),
		Timestamp:     time.Now(),
	}

	c.JSON(http.StatusOK, response)
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

// findArbitrageOpportunities analyzes market data to find arbitrage opportunities
func (h *ArbitrageHandler) findArbitrageOpportunities(ctx context.Context, minProfit float64, limit int, symbolFilter string) ([]ArbitrageOpportunity, error) {
	// Get recent ticker data from the last 5 minutes
	query := `
		SELECT symbol, exchange, bid, ask, volume, timestamp
		FROM market_data 
		WHERE timestamp > NOW() - INTERVAL '5 minutes'
		  AND bid > 0 AND ask > 0
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " ORDER BY timestamp DESC"

	rows, err := h.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	// Group data by symbol and exchange
	marketData := make(map[string]map[string]struct {
		bid       float64
		ask       float64
		volume    float64
		timestamp time.Time
	})

	for rows.Next() {
		var symbol, exchange string
		var bid, ask, volume float64
		var timestamp time.Time

		if err := rows.Scan(&symbol, &exchange, &bid, &ask, &volume, &timestamp); err != nil {
			continue
		}

		if marketData[symbol] == nil {
			marketData[symbol] = make(map[string]struct {
				bid       float64
				ask       float64
				volume    float64
				timestamp time.Time
			})
		}

		// Keep only the most recent data for each exchange
		if existing, exists := marketData[symbol][exchange]; !exists || timestamp.After(existing.timestamp) {
			marketData[symbol][exchange] = struct {
				bid       float64
				ask       float64
				volume    float64
				timestamp time.Time
			}{bid, ask, volume, timestamp}
		}
	}

	// Find arbitrage opportunities
	var opportunities []ArbitrageOpportunity

	for symbol, exchanges := range marketData {
		if len(exchanges) < 2 {
			continue // Need at least 2 exchanges for arbitrage
		}

		// Find best buy and sell prices
		var bestBuy, bestSell struct {
			exchange string
			price    float64
			volume   float64
			timestamp time.Time
		}

		for exchange, data := range exchanges {
			// Best buy price (lowest ask)
			if bestBuy.price == 0 || data.ask < bestBuy.price {
				bestBuy.exchange = exchange
				bestBuy.price = data.ask
				bestBuy.volume = data.volume
				bestBuy.timestamp = data.timestamp
			}

			// Best sell price (highest bid)
			if bestSell.price == 0 || data.bid > bestSell.price {
				bestSell.exchange = exchange
				bestSell.price = data.bid
				bestSell.volume = data.volume
				bestSell.timestamp = data.timestamp
			}
		}

		// Calculate profit
		if bestBuy.price > 0 && bestSell.price > bestBuy.price && bestBuy.exchange != bestSell.exchange {
			profitPercent := ((bestSell.price - bestBuy.price) / bestBuy.price) * 100

			if profitPercent >= minProfit {
				// Use minimum volume between exchanges
				volume := math.Min(bestBuy.volume, bestSell.volume)
				profitAmount := (bestSell.price - bestBuy.price) * volume

				opportunity := ArbitrageOpportunity{
					Symbol:        symbol,
					BuyExchange:   bestBuy.exchange,
					SellExchange:  bestSell.exchange,
					BuyPrice:      bestBuy.price,
					SellPrice:     bestSell.price,
					ProfitPercent: profitPercent,
					ProfitAmount:  profitAmount,
					Volume:        volume,
					Timestamp:     time.Now(),
				}

				opportunities = append(opportunities, opportunity)
			}
		}
	}

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

// getArbitrageHistory retrieves historical arbitrage data
func (h *ArbitrageHandler) getArbitrageHistory(ctx context.Context, limit, offset int, symbolFilter string) ([]ArbitrageHistoryItem, error) {
	// For now, we'll simulate historical data since we don't have a dedicated arbitrage_history table
	// In a real implementation, you would store detected opportunities in a separate table
	query := `
		SELECT 
			ROW_NUMBER() OVER (ORDER BY timestamp DESC) as id,
			symbol,
			exchange as buy_exchange,
			'simulated' as sell_exchange,
			ask as buy_price,
			bid as sell_price,
			((bid - ask) / ask * 100) as profit_percent,
			timestamp as detected_at
		FROM market_data 
		WHERE timestamp > NOW() - INTERVAL '24 hours'
		  AND bid > ask
		  AND ((bid - ask) / ask * 100) > 0.1
	`
	args := []interface{}{}

	if symbolFilter != "" {
		query += " AND symbol = $1"
		args = append(args, symbolFilter)
	}

	query += " ORDER BY timestamp DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
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