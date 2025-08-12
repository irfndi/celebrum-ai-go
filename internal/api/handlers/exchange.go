package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
)

// ExchangeHandler handles exchange management API endpoints
type ExchangeHandler struct {
	ccxtService      ccxt.CCXTService
	collectorService *services.CollectorService
}

// NewExchangeHandler creates a new exchange handler
func NewExchangeHandler(ccxtService ccxt.CCXTService, collectorService *services.CollectorService) *ExchangeHandler {
	return &ExchangeHandler{
		ccxtService:      ccxtService,
		collectorService: collectorService,
	}
}

// GetExchangeConfig retrieves the current exchange configuration
func (h *ExchangeHandler) GetExchangeConfig(c *gin.Context) {
	config, err := h.ccxtService.GetExchangeConfig(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get exchange configuration",
			"message": err.Error(),
		})
		return
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
	exchanges := h.ccxtService.GetSupportedExchanges()
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
