package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/services"
)

// CacheAnalyticsInterface defines the interface for cache analytics operations
type CacheAnalyticsInterface interface {
	GetStats(category string) services.CacheStats
	GetAllStats() map[string]services.CacheStats
	GetMetrics(ctx context.Context) (*services.CacheMetrics, error)
	ResetStats()
	RecordHit(category string)
	RecordMiss(category string)
}

// CacheHandler handles cache monitoring and analytics endpoints
type CacheHandler struct {
	cacheAnalytics CacheAnalyticsInterface
}

// NewCacheHandler creates a new cache handler
func NewCacheHandler(cacheAnalytics CacheAnalyticsInterface) *CacheHandler {
	return &CacheHandler{
		cacheAnalytics: cacheAnalytics,
	}
}

// GetCacheStats returns cache statistics for all categories
// @Summary Get cache statistics
// @Description Get comprehensive cache hit/miss statistics for all categories
// @Tags cache
// @Produce json
// @Success 200 {object} map[string]services.CacheStats
// @Router /api/cache/stats [get]
func (h *CacheHandler) GetCacheStats(c *gin.Context) {
	stats := h.cacheAnalytics.GetAllStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetCacheStatsByCategory returns cache statistics for a specific category
// @Summary Get cache statistics by category
// @Description Get cache hit/miss statistics for a specific category
// @Tags cache
// @Param category path string true "Cache category (e.g., market_data, funding_rates, exchanges)"
// @Produce json
// @Success 200 {object} services.CacheStats
// @Router /api/cache/stats/{category} [get]
func (h *CacheHandler) GetCacheStatsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Category parameter is required",
		})
		return
	}

	stats := h.cacheAnalytics.GetStats(category)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetCacheMetrics returns comprehensive cache metrics including Redis info
// @Summary Get comprehensive cache metrics
// @Description Get detailed cache metrics including Redis information and memory usage
// @Tags cache
// @Produce json
// @Success 200 {object} services.CacheMetrics
// @Router /api/cache/metrics [get]
func (h *CacheHandler) GetCacheMetrics(c *gin.Context) {
	metrics, err := h.cacheAnalytics.GetMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get cache metrics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// ResetCacheStats resets all cache statistics
// @Summary Reset cache statistics
// @Description Reset all cache hit/miss statistics to zero
// @Tags cache
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/cache/stats/reset [post]
func (h *CacheHandler) ResetCacheStats(c *gin.Context) {
	h.cacheAnalytics.ResetStats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache statistics reset successfully",
	})
}

// RecordCacheHit manually records a cache hit (for testing purposes)
// @Summary Record cache hit
// @Description Manually record a cache hit for testing purposes
// @Tags cache
// @Param category query string true "Cache category"
// @Param count query int false "Number of hits to record (default: 1)"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/cache/hit [post]
func (h *CacheHandler) RecordCacheHit(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Category parameter is required",
		})
		return
	}

	count := 1
	if countStr := c.Query("count"); countStr != "" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	for i := 0; i < count; i++ {
		h.cacheAnalytics.RecordHit(category)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache hits recorded successfully",
		"count":   count,
	})
}

// RecordCacheMiss manually records a cache miss (for testing purposes)
// @Summary Record cache miss
// @Description Manually record a cache miss for testing purposes
// @Tags cache
// @Param category query string true "Cache category"
// @Param count query int false "Number of misses to record (default: 1)"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/cache/miss [post]
func (h *CacheHandler) RecordCacheMiss(c *gin.Context) {
	category := c.Query("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Category parameter is required",
		})
		return
	}

	count := 1
	if countStr := c.Query("count"); countStr != "" {
		if parsedCount, err := strconv.Atoi(countStr); err == nil && parsedCount > 0 {
			count = parsedCount
		}
	}

	for i := 0; i < count; i++ {
		h.cacheAnalytics.RecordMiss(category)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cache misses recorded successfully",
		"count":   count,
	})
}
