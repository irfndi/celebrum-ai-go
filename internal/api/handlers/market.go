package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
	"github.com/shopspring/decimal"
)

type MarketHandler struct {
	db               *database.PostgresDB
	ccxtService      ccxt.CCXTService
	collectorService *services.CollectorService
}

func NewMarketHandler(db *database.PostgresDB, ccxtService ccxt.CCXTService, collectorService *services.CollectorService) *MarketHandler {
	return &MarketHandler{
		db:               db,
		ccxtService:      ccxtService,
		collectorService: collectorService,
	}
}

// MarketPricesResponse represents the response for market prices
type MarketPricesResponse struct {
	Data      []MarketPriceData `json:"data"`
	Total     int               `json:"total"`
	Page      int               `json:"page"`
	Limit     int               `json:"limit"`
	Timestamp time.Time         `json:"timestamp"`
}

type MarketPriceData struct {
	Exchange    string          `json:"exchange"`
	Symbol      string          `json:"symbol"`
	Price       decimal.Decimal `json:"price"`
	Volume      decimal.Decimal `json:"volume"`
	Timestamp   time.Time       `json:"timestamp"`
	LastUpdated time.Time       `json:"last_updated"`
}

// TickerResponse represents the response for a single ticker
type TickerResponse struct {
	Exchange  string          `json:"exchange"`
	Symbol    string          `json:"symbol"`
	Price     decimal.Decimal `json:"price"`
	Volume    decimal.Decimal `json:"volume"`
	Timestamp time.Time       `json:"timestamp"`
}

// GetMarketPrices retrieves market prices with pagination and filtering
func (h *MarketHandler) GetMarketPrices(c *gin.Context) {
	// Parse query parameters
	exchange := c.Query("exchange")
	symbol := c.Query("symbol")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	offset := (page - 1) * limit

	// Build SQL query
	sqlQuery := `
		SELECT e.name as exchange, tp.symbol, md.price, md.volume, md.timestamp, md.created_at
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
	`
	args := []interface{}{}
	whereClauses := []string{}

	// Apply filters
	if exchange != "" {
		whereClauses = append(whereClauses, "e.name = $"+strconv.Itoa(len(args)+1))
		args = append(args, exchange)
	}
	if symbol != "" {
		whereClauses = append(whereClauses, "tp.symbol = $"+strconv.Itoa(len(args)+1))
		args = append(args, symbol)
	}

	if len(whereClauses) > 0 {
		sqlQuery += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			sqlQuery += " AND " + whereClauses[i]
		}
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + sqlQuery + ") as count_query"
	var total int64
	if err := h.db.Pool.QueryRow(context.Background(), countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count market data"})
		return
	}

	// Add ordering and pagination
	sqlQuery += " ORDER BY md.created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	// Execute query
	rows, err := h.db.Pool.Query(context.Background(), sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve market data"})
		return
	}
	defer rows.Close()

	var results []struct {
		Exchange  string          `json:"exchange"`
		Symbol    string          `json:"symbol"`
		Price     decimal.Decimal `json:"price"`
		Volume    decimal.Decimal `json:"volume"`
		Timestamp time.Time       `json:"timestamp"`
		CreatedAt time.Time       `json:"created_at"`
	}

	for rows.Next() {
		var result struct {
			Exchange  string          `json:"exchange"`
			Symbol    string          `json:"symbol"`
			Price     decimal.Decimal `json:"price"`
			Volume    decimal.Decimal `json:"volume"`
			Timestamp time.Time       `json:"timestamp"`
			CreatedAt time.Time       `json:"created_at"`
		}
		if err := rows.Scan(&result.Exchange, &result.Symbol, &result.Price, &result.Volume, &result.Timestamp, &result.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan market data"})
			return
		}
		results = append(results, result)
	}

	// Convert to response format
	data := make([]MarketPriceData, len(results))
	for i, result := range results {
		data[i] = MarketPriceData{
			Exchange:    result.Exchange,
			Symbol:      result.Symbol,
			Price:       result.Price,
			Volume:      result.Volume,
			Timestamp:   result.Timestamp,
			LastUpdated: result.CreatedAt,
		}
	}

	response := MarketPricesResponse{
		Data:      data,
		Total:     int(total),
		Page:      page,
		Limit:     limit,
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetTicker retrieves the latest ticker data for a specific exchange and symbol
func (h *MarketHandler) GetTicker(c *gin.Context) {
	exchange := c.Param("exchange")
	symbol := c.Param("symbol")

	if exchange == "" || symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange and symbol are required"})
		return
	}

	// Try to get live data from CCXT service first
	if h.ccxtService.IsHealthy(c.Request.Context()) {
		ticker, err := h.ccxtService.FetchSingleTicker(c.Request.Context(), exchange, symbol)
		if err == nil && ticker != nil {
			response := TickerResponse{
				Exchange:  exchange,
				Symbol:    symbol,
				Price:     ticker.Price,
				Volume:    ticker.Volume,
				Timestamp: ticker.Timestamp,
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Fallback to database data
	var result struct {
		Exchange  string          `json:"exchange"`
		Symbol    string          `json:"symbol"`
		Price     decimal.Decimal `json:"price"`
		Volume    decimal.Decimal `json:"volume"`
		Timestamp time.Time       `json:"timestamp"`
	}

	sqlQuery := `
		SELECT e.name as exchange, tp.symbol, md.price, md.volume, md.timestamp
		FROM market_data md
		JOIN exchanges e ON md.exchange_id = e.id
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		WHERE e.name = $1 AND tp.symbol = $2
		ORDER BY md.created_at DESC
		LIMIT 1
	`

	err := h.db.Pool.QueryRow(context.Background(), sqlQuery, exchange, symbol).Scan(
		&result.Exchange, &result.Symbol, &result.Price, &result.Volume, &result.Timestamp)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticker data not found"})
		return
	}

	response := TickerResponse{
		Exchange:  result.Exchange,
		Symbol:    result.Symbol,
		Price:     result.Price,
		Volume:    result.Volume,
		Timestamp: result.Timestamp,
	}

	c.JSON(http.StatusOK, response)
}

// GetOrderBook retrieves order book data for a specific exchange and symbol
func (h *MarketHandler) GetOrderBook(c *gin.Context) {
	exchange := c.Param("exchange")
	symbol := c.Param("symbol")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if exchange == "" || symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange and symbol are required"})
		return
	}

	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Check if CCXT service is available
	if !h.ccxtService.IsHealthy(c.Request.Context()) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Market data service is currently unavailable"})
		return
	}

	// Fetch order book from CCXT service
	orderBook, err := h.ccxtService.FetchOrderBook(c.Request.Context(), exchange, symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order book data"})
		return
	}

	c.JSON(http.StatusOK, orderBook)
}

// GetWorkerStatus returns the status of all collection workers
func (h *MarketHandler) GetWorkerStatus(c *gin.Context) {
	if h.collectorService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "Collector service is not available",
			"timestamp": time.Now(),
		})
		return
	}

	status := h.collectorService.GetWorkerStatus()
	c.JSON(http.StatusOK, gin.H{
		"workers":   status,
		"timestamp": time.Now(),
	})
}
