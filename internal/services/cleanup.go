package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/database"
)

// CleanupService handles automatic cleanup of old data
type CleanupService struct {
	db     *database.PostgresDB
	ctx    context.Context
	cancel context.CancelFunc
}

// CleanupConfig defines cleanup configuration
type CleanupConfig struct {
	MarketDataRetentionHours  int `yaml:"market_data_retention_hours" default:"24"`
	FundingRateRetentionHours int `yaml:"funding_rate_retention_hours" default:"24"`
	ArbitrageRetentionHours   int `yaml:"arbitrage_retention_hours" default:"72"`
	CleanupIntervalMinutes    int `yaml:"cleanup_interval_minutes" default:"60"`
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(db *database.PostgresDB) *CleanupService {
	ctx, cancel := context.WithCancel(context.Background())
	return &CleanupService{
		db:     db,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins the cleanup service with periodic cleanup
func (c *CleanupService) Start(config CleanupConfig) {
	log.Printf("Starting cleanup service with %dh retention for market data, %dh for funding rates",
		config.MarketDataRetentionHours, config.FundingRateRetentionHours)

	// Run initial cleanup
	go func() {
		if err := c.runCleanup(config); err != nil {
			log.Printf("Initial cleanup failed: %v", err)
		}
	}()

	// Start periodic cleanup
	ticker := time.NewTicker(time.Duration(config.CleanupIntervalMinutes) * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				if err := c.runCleanup(config); err != nil {
					log.Printf("Cleanup failed: %v", err)
				}
			}
		}
	}()
}

// Stop stops the cleanup service
func (c *CleanupService) Stop() {
	log.Println("Stopping cleanup service")
	c.cancel()
}

// RunCleanup performs a manual cleanup operation
func (c *CleanupService) RunCleanup(config CleanupConfig) error {
	return c.runCleanup(config)
}

// runCleanup performs the actual cleanup operations
func (c *CleanupService) runCleanup(config CleanupConfig) error {
	log.Println("Running data cleanup...")

	// Clean up old market data
	if err := c.cleanupMarketData(config.MarketDataRetentionHours); err != nil {
		return fmt.Errorf("failed to cleanup market data: %w", err)
	}

	// Clean up old funding rates
	if err := c.cleanupFundingRates(config.FundingRateRetentionHours); err != nil {
		return fmt.Errorf("failed to cleanup funding rates: %w", err)
	}

	// Clean up old arbitrage opportunities
	if err := c.cleanupArbitrageOpportunities(config.ArbitrageRetentionHours); err != nil {
		return fmt.Errorf("failed to cleanup arbitrage opportunities: %w", err)
	}

	// Clean up old funding arbitrage opportunities
	if err := c.cleanupFundingArbitrageOpportunities(config.ArbitrageRetentionHours); err != nil {
		return fmt.Errorf("failed to cleanup funding arbitrage opportunities: %w", err)
	}

	log.Println("Data cleanup completed successfully")
	return nil
}

// cleanupMarketData removes market data older than specified hours
func (c *CleanupService) cleanupMarketData(retentionHours int) error {
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(c.ctx,
		"DELETE FROM market_data WHERE created_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old market data: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old market data records (older than %dh)", rowsAffected, retentionHours)
	}

	return nil
}

// cleanupFundingRates removes funding rates older than specified hours
func (c *CleanupService) cleanupFundingRates(retentionHours int) error {
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(c.ctx,
		"DELETE FROM funding_rates WHERE created_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old funding rates: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old funding rate records (older than %dh)", rowsAffected, retentionHours)
	}

	return nil
}

// cleanupArbitrageOpportunities removes old arbitrage opportunities
func (c *CleanupService) cleanupArbitrageOpportunities(retentionHours int) error {
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(c.ctx,
		"DELETE FROM arbitrage_opportunities WHERE detected_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old arbitrage opportunities: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old arbitrage opportunity records (older than %dh)", rowsAffected, retentionHours)
	}

	return nil
}

// cleanupFundingArbitrageOpportunities removes old funding arbitrage opportunities
func (c *CleanupService) cleanupFundingArbitrageOpportunities(retentionHours int) error {
	cutoffTime := time.Now().Add(-time.Duration(retentionHours) * time.Hour)

	result, err := c.db.Pool.Exec(c.ctx,
		"DELETE FROM funding_arbitrage_opportunities WHERE created_at < $1",
		cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old funding arbitrage opportunities: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old funding arbitrage opportunity records (older than %dh)", rowsAffected, retentionHours)
	}

	return nil
}

// GetDataStats returns statistics about current data storage
func (c *CleanupService) GetDataStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	// Count market data records
	var marketDataCount int64
	err := c.db.Pool.QueryRow(c.ctx, "SELECT COUNT(*) FROM market_data").Scan(&marketDataCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count market data: %w", err)
	}
	stats["market_data_count"] = marketDataCount

	// Count funding rates
	var fundingRatesCount int64
	err = c.db.Pool.QueryRow(c.ctx, "SELECT COUNT(*) FROM funding_rates").Scan(&fundingRatesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count funding rates: %w", err)
	}
	stats["funding_rates_count"] = fundingRatesCount

	// Count arbitrage opportunities
	var arbitrageOpportunitiesCount int64
	err = c.db.Pool.QueryRow(c.ctx, "SELECT COUNT(*) FROM arbitrage_opportunities").Scan(&arbitrageOpportunitiesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count arbitrage opportunities: %w", err)
	}
	stats["arbitrage_opportunities_count"] = arbitrageOpportunitiesCount

	// Count funding arbitrage opportunities
	var fundingArbitrageOpportunitiesCount int64
	err = c.db.Pool.QueryRow(c.ctx, "SELECT COUNT(*) FROM funding_arbitrage_opportunities").Scan(&fundingArbitrageOpportunitiesCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count funding arbitrage opportunities: %w", err)
	}
	stats["funding_arbitrage_opportunities_count"] = fundingArbitrageOpportunitiesCount

	return stats, nil
}
