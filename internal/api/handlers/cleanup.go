package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/services"
)

// CleanupInterface defines the interface for cleanup operations
type CleanupInterface interface {
	GetDataStats(ctx context.Context) (map[string]int64, error)
	RunCleanup(config services.CleanupConfig) error
}

// CleanupHandler handles cleanup-related API endpoints
type CleanupHandler struct {
	cleanupService CleanupInterface
}

// NewCleanupHandler creates a new cleanup handler
func NewCleanupHandler(cleanupService CleanupInterface) *CleanupHandler {
	return &CleanupHandler{
		cleanupService: cleanupService,
	}
}

// DataStatsResponse represents the response for data statistics
type DataStatsResponse struct {
	MarketDataCount                    int64 `json:"market_data_count"`
	FundingRatesCount                  int64 `json:"funding_rates_count"`
	ArbitrageOpportunitiesCount        int64 `json:"arbitrage_opportunities_count"`
	FundingArbitrageOpportunitiesCount int64 `json:"funding_arbitrage_opportunities_count"`
	TotalRecords                       int64 `json:"total_records"`
}

// GetDataStats returns statistics about current data storage
func (h *CleanupHandler) GetDataStats(c *gin.Context) {
	stats, err := h.cleanupService.GetDataStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get data statistics"})
		return
	}

	totalRecords := stats["market_data_count"] + stats["funding_rates_count"] +
		stats["arbitrage_opportunities_count"] + stats["funding_arbitrage_opportunities_count"]

	response := DataStatsResponse{
		MarketDataCount:                    stats["market_data_count"],
		FundingRatesCount:                  stats["funding_rates_count"],
		ArbitrageOpportunitiesCount:        stats["arbitrage_opportunities_count"],
		FundingArbitrageOpportunitiesCount: stats["funding_arbitrage_opportunities_count"],
		TotalRecords:                       totalRecords,
	}

	c.JSON(http.StatusOK, response)
}

// parseIntParam parses an integer parameter from string
func parseIntParam(param string) (int, error) {
	return strconv.Atoi(param)
}

// TriggerCleanup manually triggers a cleanup operation
func (h *CleanupHandler) TriggerCleanup(c *gin.Context) {
	// Parse optional retention hours from query parameters
	marketDataHours := 24
	fundingRateHours := 24
	arbitrageHours := 72

	if hours := c.Query("market_data_hours"); hours != "" {
		if parsed, err := parseIntParam(hours); err == nil && parsed > 0 {
			marketDataHours = parsed
		}
	}

	if hours := c.Query("funding_rate_hours"); hours != "" {
		if parsed, err := parseIntParam(hours); err == nil && parsed > 0 {
			fundingRateHours = parsed
		}
	}

	if hours := c.Query("arbitrage_hours"); hours != "" {
		if parsed, err := parseIntParam(hours); err == nil && parsed > 0 {
			arbitrageHours = parsed
		}
	}

	// Create cleanup config for manual trigger
	config := services.CleanupConfig{
		MarketDataRetentionHours:  marketDataHours,
		FundingRateRetentionHours: fundingRateHours,
		ArbitrageRetentionHours:   arbitrageHours,
	}

	// Trigger cleanup (this will run synchronously)
	if err := h.cleanupService.RunCleanup(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to run cleanup"})
		return
	}

	// Get updated stats after cleanup
	stats, err := h.cleanupService.GetDataStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cleanup completed but failed to get updated statistics"})
		return
	}

	totalRecords := stats["market_data_count"] + stats["funding_rates_count"] +
		stats["arbitrage_opportunities_count"] + stats["funding_arbitrage_opportunities_count"]

	response := struct {
		Message string            `json:"message"`
		Stats   DataStatsResponse `json:"stats"`
	}{
		Message: "Cleanup completed successfully",
		Stats: DataStatsResponse{
			MarketDataCount:                    stats["market_data_count"],
			FundingRatesCount:                  stats["funding_rates_count"],
			ArbitrageOpportunitiesCount:        stats["arbitrage_opportunities_count"],
			FundingArbitrageOpportunitiesCount: stats["funding_arbitrage_opportunities_count"],
			TotalRecords:                       totalRecords,
		},
	}

	c.JSON(http.StatusOK, response)
}
