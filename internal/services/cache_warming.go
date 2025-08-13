package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
	"github.com/redis/go-redis/v9"
)

// CacheWarmingService handles cache warming on application startup
type CacheWarmingService struct {
	redisClient *redis.Client
	ccxtService ccxt.CCXTService
	db          *database.PostgresDB
}

// NewCacheWarmingService creates a new cache warming service
func NewCacheWarmingService(redisClient *redis.Client, ccxtService ccxt.CCXTService, db *database.PostgresDB) *CacheWarmingService {
	return &CacheWarmingService{
		redisClient: redisClient,
		ccxtService: ccxtService,
		db:          db,
	}
}

// WarmCache performs cache warming for frequently accessed data
func (c *CacheWarmingService) WarmCache(ctx context.Context) error {
	log.Println("Starting cache warming...")
	start := time.Now()

	// Warm exchange configuration cache
	if err := c.warmExchangeConfig(ctx); err != nil {
		log.Printf("Warning: Failed to warm exchange config cache: %v", err)
	}

	// Warm supported exchanges cache
	if err := c.warmSupportedExchanges(ctx); err != nil {
		log.Printf("Warning: Failed to warm supported exchanges cache: %v", err)
	}

	// Warm trading pairs cache
	if err := c.warmTradingPairs(ctx); err != nil {
		log.Printf("Warning: Failed to warm trading pairs cache: %v", err)
	}

	// Warm exchange data cache
	if err := c.warmExchanges(ctx); err != nil {
		log.Printf("Warning: Failed to warm exchanges cache: %v", err)
	}

	// Warm funding rates cache
	if err := c.warmFundingRates(ctx); err != nil {
		log.Printf("Warning: Failed to warm funding rates cache: %v", err)
	}

	duration := time.Since(start)
	log.Printf("Cache warming completed in %v", duration)
	return nil
}

// warmExchangeConfig warms the exchange configuration cache
func (c *CacheWarmingService) warmExchangeConfig(ctx context.Context) error {
	log.Println("Warming exchange configuration cache...")

	// Get exchange config from CCXT service
	config, err := c.ccxtService.GetExchangeConfig(ctx)
	if err != nil {
		return err
	}

	// Marshal and cache the config
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// Cache with 1 hour TTL
	err = c.redisClient.Set(ctx, "exchange:config", configJSON, time.Hour).Err()
	if err != nil {
		return err
	}

	log.Println("Exchange configuration cache warmed successfully")
	return nil
}

// warmSupportedExchanges warms the supported exchanges cache
func (c *CacheWarmingService) warmSupportedExchanges(ctx context.Context) error {
	log.Println("Warming supported exchanges cache...")

	// Get supported exchanges from CCXT service
	exchanges := c.ccxtService.GetSupportedExchanges()

	// Marshal and cache the exchanges
	exchangesJSON, err := json.Marshal(exchanges)
	if err != nil {
		return err
	}

	// Cache with 30 minutes TTL
	err = c.redisClient.Set(ctx, "exchange:supported", exchangesJSON, 30*time.Minute).Err()
	if err != nil {
		return err
	}

	log.Printf("Supported exchanges cache warmed successfully (%d exchanges)", len(exchanges))
	return nil
}

// warmTradingPairs warms the trading pairs cache
func (c *CacheWarmingService) warmTradingPairs(ctx context.Context) error {
	log.Println("Warming trading pairs cache...")

	// Get all trading pairs from database
	query := `SELECT id, symbol, base_asset, quote_asset FROM trading_pairs LIMIT 1000`
	rows, err := c.db.Pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var symbol, baseAsset, quoteAsset string
		if err := rows.Scan(&id, &symbol, &baseAsset, &quoteAsset); err != nil {
			continue
		}

		// Cache trading pair by symbol with 24 hour TTL
		cacheKey := "trading_pair:" + symbol
		pairData := map[string]interface{}{
			"id":          id,
			"symbol":      symbol,
			"base_asset":  baseAsset,
			"quote_asset": quoteAsset,
		}

		pairJSON, err := json.Marshal(pairData)
		if err != nil {
			continue
		}

		err = c.redisClient.Set(ctx, cacheKey, pairJSON, 24*time.Hour).Err()
		if err != nil {
			continue
		}

		count++
	}

	log.Printf("Trading pairs cache warmed successfully (%d pairs)", count)
	return nil
}

// warmExchanges warms the exchanges cache
func (c *CacheWarmingService) warmExchanges(ctx context.Context) error {
	log.Println("Warming exchanges cache...")

	// Get all exchanges from database
	query := `SELECT id, name FROM exchanges LIMIT 100`
	rows, err := c.db.Pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}

		// Cache exchange ID by name with 24 hour TTL
		cacheKey := "exchange:" + name
		err = c.redisClient.Set(ctx, cacheKey, id, 24*time.Hour).Err()
		if err != nil {
			continue
		}

		count++
	}

	log.Printf("Exchanges cache warmed successfully (%d exchanges)", count)
	return nil
}

// warmFundingRates warms the funding rates cache
func (c *CacheWarmingService) warmFundingRates(ctx context.Context) error {
	log.Println("Warming funding rates cache...")

	// Get latest funding rates from database for each exchange-symbol combination
	query := `
		SELECT DISTINCT ON (e.name, tp.symbol) 
			e.name as exchange_name,
			tp.symbol,
			fr.funding_rate,
			fr.next_funding_time,
			fr.timestamp
		FROM funding_rates fr
		JOIN exchanges e ON fr.exchange_id = e.id
		JOIN trading_pairs tp ON fr.trading_pair_id = tp.id
		WHERE fr.timestamp > NOW() - INTERVAL '24 hours'
		ORDER BY e.name, tp.symbol, fr.timestamp DESC
		LIMIT 1000
	`

	rows, err := c.db.Pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var exchangeName, symbol string
		var fundingRate float64
		var nextFundingTime, timestamp time.Time

		if err := rows.Scan(&exchangeName, &symbol, &fundingRate, &nextFundingTime, &timestamp); err != nil {
			continue
		}

		// Cache funding rate data with 1 minute TTL
		cacheKey := "funding_rate:" + exchangeName + ":" + symbol
		fundingData := map[string]interface{}{
			"exchange":          exchangeName,
			"symbol":            symbol,
			"funding_rate":      fundingRate,
			"next_funding_time": nextFundingTime,
			"timestamp":         timestamp,
		}

		fundingJSON, err := json.Marshal(fundingData)
		if err != nil {
			continue
		}

		err = c.redisClient.Set(ctx, cacheKey, fundingJSON, time.Minute).Err()
		if err != nil {
			continue
		}

		count++
	}

	log.Printf("Funding rates cache warmed successfully (%d rates)", count)
	return nil
}
