package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/ccxt"
	"github.com/redis/go-redis/v9"
)

// RedisInterface defines the interface for Redis operations used by ExchangeHandler
type RedisInterface interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

// CollectorInterface defines the interface for collector service operations
type CollectorInterface interface {
	Start() error
	Stop()
	RestartWorker(exchangeID string) error
	IsReady() bool
	IsInitialized() bool
}

// ExchangeHandler handles exchange management API endpoints
type ExchangeHandler struct {
	ccxtService      ccxt.CCXTService
	collectorService CollectorInterface
	redisClient      RedisInterface
}

// NewExchangeHandler creates a new exchange handler
func NewExchangeHandler(ccxtService ccxt.CCXTService, collectorService CollectorInterface, redisClient RedisInterface) *ExchangeHandler {
	return &ExchangeHandler{
		ccxtService:      ccxtService,
		collectorService: collectorService,
		redisClient:      redisClient,
	}
}

// GetExchangeConfig retrieves the current exchange configuration
func (h *ExchangeHandler) GetExchangeConfig(c *gin.Context) {
	cacheKey := "exchange:config"
	ctx := context.Background()

	// Try to get from cache first
	cachedData, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var config interface{}
		if err := json.Unmarshal([]byte(cachedData), &config); err != nil {
			// Log the error but continue to fetch fresh data
			c.Header("X-Cache-Error", "Failed to unmarshal cached data")
		} else {
			c.JSON(http.StatusOK, config)
			return
		}
	}

	// Cache miss or error, fetch from service
	config, err := h.ccxtService.GetExchangeConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get exchange configuration",
			"message": err.Error(),
		})
		return
	}

	// Cache the result for 1 hour
	if configData, err := json.Marshal(config); err == nil {
		h.redisClient.Set(ctx, cacheKey, configData, time.Hour)
	}

	c.JSON(http.StatusOK, config)
}

// AddExchangeToBlacklist adds an exchange to the blacklist
func (h *ExchangeHandler) AddExchangeToBlacklist(c *gin.Context) {
	exchange := c.Param("exchange")
	if exchange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Exchange parameter is required",
		})
		return
	}

	response, err := h.ccxtService.AddExchangeToBlacklist(c.Request.Context(), exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add exchange to blacklist",
			"message": err.Error(),
		})
		return
	}

	// Stop the worker for this exchange if it exists
	if h.collectorService != nil {
		if err := h.collectorService.RestartWorker(exchange); err != nil {
			// Log the error but don't fail the request
			// The exchange is already blacklisted in CCXT service
			c.Header("X-Worker-Restart-Warning", "Failed to restart worker: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, response)
}

// RemoveExchangeFromBlacklist removes an exchange from the blacklist
func (h *ExchangeHandler) RemoveExchangeFromBlacklist(c *gin.Context) {
	exchange := c.Param("exchange")
	if exchange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Exchange parameter is required",
		})
		return
	}

	response, err := h.ccxtService.RemoveExchangeFromBlacklist(c.Request.Context(), exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to remove exchange from blacklist",
			"message": err.Error(),
		})
		return
	}

	// Restart the collector service to pick up the new exchange
	if h.collectorService != nil {
		if err := h.collectorService.RestartWorker(exchange); err != nil {
			// If worker doesn't exist, that's fine - it will be created on next refresh
			c.Header("X-Worker-Restart-Info", "Worker restart info: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, response)
}

// RefreshExchanges refreshes all non-blacklisted exchanges
func (h *ExchangeHandler) RefreshExchanges(c *gin.Context) {
	response, err := h.ccxtService.RefreshExchanges(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to refresh exchanges",
			"message": err.Error(),
		})
		return
	}

	// Restart the collector service to pick up any new exchanges
	if h.collectorService != nil {
		// Stop and restart the entire collector service
		h.collectorService.Stop()
		if err := h.collectorService.Start(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to restart collector service",
				"message": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, response)
}

// AddExchange dynamically adds and initializes a new exchange
func (h *ExchangeHandler) AddExchange(c *gin.Context) {
	exchange := c.Param("exchange")
	if exchange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Exchange parameter is required",
		})
		return
	}

	response, err := h.ccxtService.AddExchange(c.Request.Context(), exchange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add exchange",
			"message": err.Error(),
		})
		return
	}

	// Restart the collector service to pick up the new exchange
	if h.collectorService != nil {
		h.collectorService.Stop()
		if err := h.collectorService.Start(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to restart collector service",
				"message": err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetSupportedExchanges returns the list of currently supported exchanges
func (h *ExchangeHandler) GetSupportedExchanges(c *gin.Context) {
	cacheKey := "exchange:supported"
	ctx := context.Background()

	// Try to get from cache first
	cachedData, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var exchanges []string
		if err := json.Unmarshal([]byte(cachedData), &exchanges); err != nil {
			// Log the error but continue to fetch fresh data
			c.Header("X-Cache-Error", "Failed to unmarshal cached data")
		} else {
			c.JSON(http.StatusOK, gin.H{
				"exchanges": exchanges,
				"count":     len(exchanges),
			})
			return
		}
	}

	// Cache miss or error, fetch from service
	exchanges := h.ccxtService.GetSupportedExchanges()

	// Cache the result for 30 minutes
	if exchangesData, err := json.Marshal(exchanges); err == nil {
		h.redisClient.Set(ctx, cacheKey, exchangesData, 30*time.Minute)
	}

	c.JSON(http.StatusOK, gin.H{
		"exchanges": exchanges,
		"count":     len(exchanges),
	})
}

// GetWorkerStatus returns the status of exchange workers
func (h *ExchangeHandler) GetWorkerStatus(c *gin.Context) {
	if h.collectorService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Collector service not available",
		})
		return
	}

	// Get worker status from collector service
	// This would need to be implemented in the collector service
	c.JSON(http.StatusOK, gin.H{
		"message": "Worker status endpoint - implementation needed in collector service",
	})
}

// RestartWorker restarts a specific exchange worker
func (h *ExchangeHandler) RestartWorker(c *gin.Context) {
	exchange := c.Param("exchange")
	if exchange == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Exchange parameter is required",
		})
		return
	}

	if h.collectorService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Collector service not available",
		})
		return
	}

	if err := h.collectorService.RestartWorker(exchange); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to restart worker",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Worker restarted successfully",
		"exchange": exchange,
	})
}
