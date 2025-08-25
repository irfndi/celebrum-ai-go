package services

import (
	"context"
	"fmt"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/telemetry"
	"log/slog"
)

// CleanupService handles automatic cleanup of old data
type CleanupService struct {
	db                   *database.PostgresDB
	ctx                  context.Context
	cancel               context.CancelFunc
	errorRecoveryManager *ErrorRecoveryManager
	resourceManager      *ResourceManager
	performanceMonitor   *PerformanceMonitor
	logger               *slog.Logger
}

// CleanupConfig defines cleanup configuration
type CleanupConfig struct {
	MarketData struct {
		RetentionHours int `yaml:"retention_hours" default:"36"`
		DeletionHours  int `yaml:"deletion_hours" default:"12"`
	} `yaml:"market_data"`
	FundingRates struct {
		RetentionHours int `yaml:"retention_hours" default:"36"`
		DeletionHours  int `yaml:"deletion_hours" default:"12"`
	} `yaml:"funding_rates"`
	ArbitrageOpportunities struct {
		RetentionHours int `yaml:"retention_hours" default:"72"`
	} `yaml:"arbitrage_opportunities"`
	IntervalMinutes    int  `yaml:"interval_minutes" default:"60"`
	EnableSmartCleanup bool `yaml:"enable_smart_cleanup" default:"true"`
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(db *database.PostgresDB, errorRecoveryManager *ErrorRecoveryManager, resourceManager *ResourceManager, performanceMonitor *PerformanceMonitor) *CleanupService {
	ctx, cancel := context.WithCancel(context.Background())
	return &CleanupService{
		db:                   db,
		ctx:                  ctx,
		cancel:               cancel,
		errorRecoveryManager: errorRecoveryManager,
		resourceManager:      resourceManager,
		performanceMonitor:   performanceMonitor,
		logger:               telemetry.GetLogger().Logger(),
	}
}

// Start begins the cleanup service with periodic cleanup
func (c *CleanupService) Start(config CleanupConfig) {
	if config.EnableSmartCleanup {
		c.logger.Info("Starting cleanup service with smart cleanup",
			"market_data_retention_hours", config.MarketData.RetentionHours,
			"market_data_deletion_hours", config.MarketData.DeletionHours)
	} else {
		c.logger.Info("Starting cleanup service with traditional cleanup",
			"market_data_retention_hours", config.MarketData.RetentionHours,
			"funding_rates_retention_hours", config.FundingRates.RetentionHours)
	}

	// Note: Resource manager registration not needed for cleanup service

	// Run initial cleanup
	go func() {
		// Create timeout context for initial cleanup
		ctx, cancel := context.WithTimeout(c.ctx, 30*time.Minute)
		defer cancel()

		err := c.errorRecoveryManager.ExecuteWithRetry(ctx, "initial_cleanup", func() error {
			return c.runCleanup(ctx, config)
		})
		if err != nil {
			c.logger.Error("Initial cleanup failed", "error", err)
			// Note: Performance monitor doesn't have RecordFailure method
		}
	}()

	// Start periodic cleanup
	ticker := time.NewTicker(time.Duration(config.IntervalMinutes) * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				// Create timeout context for periodic cleanup
				ctx, cancel := context.WithTimeout(c.ctx, 30*time.Minute)

				err := c.errorRecoveryManager.ExecuteWithRetry(ctx, "periodic_cleanup", func() error {
					return c.runCleanup(ctx, config)
				})
				if err != nil {
					c.logger.Error("Cleanup failed", "error", err)
					// Note: Performance monitor doesn't have RecordFailure method
				}
				cancel()
			}
		}
	}()
}

// Stop stops the cleanup service
func (c *CleanupService) Stop() {
	c.logger.Info("Stopping cleanup service")
	c.cancel()
}

// RunCleanup performs a manual cleanup operation
func (c *CleanupService) RunCleanup(config CleanupConfig) error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Minute)
	defer cancel()
	return c.runCleanup(ctx, config)
}

// runCleanup performs the actual cleanup operations
func (c *CleanupService) runCleanup(ctx context.Context, config CleanupConfig) error {
	c.logger.Info("Starting cleanup process", "smart_cleanup", config.EnableSmartCleanup)

	// Get data statistics before cleanup with error recovery
	var statsBefore map[string]int64
	err := c.errorRecoveryManager.ExecuteWithRetry(ctx, "get_stats_before", func() error {
		var err error
		statsBefore, err = c.GetDataStats(ctx)
		return err
	})
	if err != nil {
		c.logger.Warn("Failed to get data statistics before cleanup", "error", err)
	} else {
		c.logger.Info("Data statistics before cleanup",
			"market_data_count", statsBefore["market_data_count"],
			"funding_rates_count", statsBefore["funding_rates_count"],
			"arbitrage_opportunities_count", statsBefore["arbitrage_opportunities_count"],
			"funding_arbitrage_opportunities_count", statsBefore["funding_arbitrage_opportunities_count"])
	}

	// Clean up market data using smart cleanup if enabled
	if config.EnableSmartCleanup {
		c.logger.Info("Using smart cleanup strategy",
			"market_data_retention_hours", config.MarketData.RetentionHours,
			"market_data_deletion_hours", config.MarketData.DeletionHours,
			"funding_rates_retention_hours", config.FundingRates.RetentionHours,
			"funding_rates_deletion_hours", config.FundingRates.DeletionHours)
		cleanupErr := c.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_market_data_smart", func() error {
			return c.cleanupMarketDataSmart(ctx, config.MarketData.RetentionHours, config.MarketData.DeletionHours)
		})
		if cleanupErr != nil {
			return fmt.Errorf("failed to cleanup market data: %w", cleanupErr)
		}
		fundingCleanupErr := c.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_funding_rates_smart", func() error {
			return c.cleanupFundingRatesSmart(ctx, config.FundingRates.RetentionHours, config.FundingRates.DeletionHours)
		})
		if fundingCleanupErr != nil {
			return fmt.Errorf("failed to cleanup funding rates: %w", fundingCleanupErr)
		}
	} else {
		// Fallback to traditional cleanup
		c.logger.Info("Using traditional cleanup strategy",
			"market_data_retention_hours", config.MarketData.RetentionHours,
			"funding_rates_retention_hours", config.FundingRates.RetentionHours)
		cleanupErr := c.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_market_data", func() error {
			return c.cleanupMarketData(ctx, config.MarketData.RetentionHours)
		})
		if cleanupErr != nil {
			return fmt.Errorf("failed to cleanup market data: %w", cleanupErr)
		}
		err = c.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_funding_rates", func() error {
			return c.cleanupFundingRates(ctx, config.FundingRates.RetentionHours)
		})
		if err != nil {
			return fmt.Errorf("failed to cleanup funding rates: %w", err)
		}
	}

	// Clean up old arbitrage opportunities with error recovery
	c.logger.Info("Cleaning up arbitrage opportunities", "retention_hours", config.ArbitrageOpportunities.RetentionHours)
	err = c.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_arbitrage_opportunities", func() error {
		return c.cleanupArbitrageOpportunities(ctx, config.ArbitrageOpportunities.RetentionHours)
	})
	if err != nil {
		return fmt.Errorf("failed to cleanup arbitrage opportunities: %w", err)
	}

	// Clean up old funding arbitrage opportunities with error recovery
	err = c.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_funding_arbitrage_opportunities", func() error {
		return c.cleanupFundingArbitrageOpportunities(ctx, config.ArbitrageOpportunities.RetentionHours)
	})
	if err != nil {
		return fmt.Errorf("failed to cleanup funding arbitrage opportunities: %w", err)
	}

	// Get data statistics after cleanup with error recovery
	var statsAfter map[string]int64
	err = c.errorRecoveryManager.ExecuteWithRetry(ctx, "get_stats_after", func() error {
		var getStatsErr error
		statsAfter, getStatsErr = c.GetDataStats(ctx)
		return getStatsErr
	})
	if err != nil {
		c.logger.Warn("Failed to get data statistics after cleanup", "error", err)
	} else {
		c.logger.Info("Data statistics after cleanup",
			"market_data_count", statsAfter["market_data_count"],
			"funding_rates_count", statsAfter["funding_rates_count"],
			"arbitrage_opportunities_count", statsAfter["arbitrage_opportunities_count"],
			"funding_arbitrage_opportunities_count", statsAfter["funding_arbitrage_opportunities_count"])

		// Log cleanup summary
		if statsBefore != nil {
			marketDataDeleted := statsBefore["market_data_count"] - statsAfter["market_data_count"]
			fundingRatesDeleted := statsBefore["funding_rates_count"] - statsAfter["funding_rates_count"]
			arbitrageDeleted := statsBefore["arbitrage_opportunities_count"] - statsAfter["arbitrage_opportunities_count"]
			fundingArbitrageDeleted := statsBefore["funding_arbitrage_opportunities_count"] - statsAfter["funding_arbitrage_opportunities_count"]

			c.logger.Info("Cleanup summary",
				"market_data_deleted", marketDataDeleted,
				"funding_rates_deleted", fundingRatesDeleted,
				"arbitrage_deleted", arbitrageDeleted,
				"funding_arbitrage_deleted", fundingArbitrageDeleted)
		}
	}

	c.logger.Info("Cleanup process completed successfully")
	return nil
}

// cleanupMarketData removes market data using smart cleanup strategy
func (c *CleanupService) cleanupMarketData(ctx context.Context, retentionHours int) error {
	return c.cleanupMarketDataSmart(ctx, retentionHours, 12) // Default 12h deletion
}

// cleanupMarketDataSmart removes oldest data while keeping a buffer
func (c *CleanupService) cleanupMarketDataSmart(ctx context.Context, retentionHours, deletionHours int) error {
	// Delete data older than retention hours (e.g., older than 36h)
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(ctx,
		"DELETE FROM market_data WHERE created_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old market data: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		c.logger.Info("Smart cleanup: Removed market data records",
			"records_deleted", rowsAffected,
			"retention_hours", retentionHours,
			"buffer_hours", retentionHours-deletionHours)
	}

	return nil
}

// cleanupFundingRates removes funding rates using smart cleanup strategy
func (c *CleanupService) cleanupFundingRates(ctx context.Context, retentionHours int) error {
	return c.cleanupFundingRatesSmart(ctx, retentionHours, 12) // Default 12h deletion
}

// cleanupFundingRatesSmart removes oldest funding rates while keeping a buffer
func (c *CleanupService) cleanupFundingRatesSmart(ctx context.Context, retentionHours, deletionHours int) error {
	// Delete data older than retention hours (e.g., older than 36h)
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(ctx,
		"DELETE FROM funding_rates WHERE created_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old funding rates: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		c.logger.Info("Smart cleanup: Removed funding rate records",
			"records_deleted", rowsAffected,
			"retention_hours", retentionHours,
			"buffer_hours", retentionHours-deletionHours)
	}

	return nil
}

// cleanupArbitrageOpportunities removes old arbitrage opportunities
func (c *CleanupService) cleanupArbitrageOpportunities(ctx context.Context, retentionHours int) error {
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(ctx,
		"DELETE FROM arbitrage_opportunities WHERE detected_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old arbitrage opportunities: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		c.logger.Info("Cleaned up old arbitrage opportunity records",
			"records_deleted", rowsAffected,
			"retention_hours", retentionHours)
	}

	return nil
}

// cleanupFundingArbitrageOpportunities removes old funding arbitrage opportunities
func (c *CleanupService) cleanupFundingArbitrageOpportunities(ctx context.Context, retentionHours int) error {
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(ctx,
		"DELETE FROM funding_arbitrage_opportunities WHERE created_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old funding arbitrage opportunities: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		c.logger.Info("Cleaned up old funding arbitrage opportunity records",
			"records_deleted", rowsAffected,
			"retention_hours", retentionHours)
	}

	return nil
}

// GetDataStats returns statistics about current data storage
func (c *CleanupService) GetDataStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count market data records
	var marketDataCount int64
	err := c.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM market_data").Scan(&marketDataCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count market data: %w", err)
	}
	stats["market_data_count"] = marketDataCount

	// Count funding rates
	var fundingRatesCount int64
	err = c.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM funding_rates").Scan(&fundingRatesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count funding rates: %w", err)
	}
	stats["funding_rates_count"] = fundingRatesCount

	// Count arbitrage opportunities
	var arbitrageOpportunitiesCount int64
	err = c.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM arbitrage_opportunities").Scan(&arbitrageOpportunitiesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count arbitrage opportunities: %w", err)
	}
	stats["arbitrage_opportunities_count"] = arbitrageOpportunitiesCount

	// Count funding arbitrage opportunities
	var fundingArbitrageOpportunitiesCount int64
	err = c.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM funding_arbitrage_opportunities").Scan(&fundingArbitrageOpportunitiesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count funding arbitrage opportunities: %w", err)
	}
	stats["funding_arbitrage_opportunities_count"] = fundingArbitrageOpportunitiesCount

	return stats, nil
}
