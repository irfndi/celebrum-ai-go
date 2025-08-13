package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// FuturesArbitrageService manages futures arbitrage opportunity calculation and storage
type FuturesArbitrageService struct {
	db          *database.PostgresDB
	redisClient *redis.Client
	calculator  *FuturesArbitrageCalculator
	config      *config.Config
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex

	// Error recovery components
	errorRecoveryManager *ErrorRecoveryManager
	resourceManager      *ResourceManager
	performanceMonitor   *PerformanceMonitor
}

// NewFuturesArbitrageService creates a new futures arbitrage service
func NewFuturesArbitrageService(
	db *database.PostgresDB,
	redisClient *redis.Client,
	cfg *config.Config,
	errorRecoveryManager *ErrorRecoveryManager,
	resourceManager *ResourceManager,
	performanceMonitor *PerformanceMonitor,
) *FuturesArbitrageService {
	return &FuturesArbitrageService{
		db:                   db,
		redisClient:          redisClient,
		calculator:           NewFuturesArbitrageCalculator(),
		config:               cfg,
		errorRecoveryManager: errorRecoveryManager,
		resourceManager:      resourceManager,
		performanceMonitor:   performanceMonitor,
	}
}

// Start begins the futures arbitrage opportunity calculation service
func (s *FuturesArbitrageService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("futures arbitrage service is already running")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.running = true

	// Start the opportunity calculation worker
	s.wg.Add(1)
	go s.runOpportunityCalculator()

	log.Println("Futures arbitrage service started")
	return nil
}

// Stop gracefully stops the futures arbitrage service
func (s *FuturesArbitrageService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.wg.Wait()
	s.running = false

	log.Println("Futures arbitrage service stopped")
}

// runOpportunityCalculator runs the periodic opportunity calculation
func (s *FuturesArbitrageService) runOpportunityCalculator() {
	defer s.wg.Done()

	// Register this operation with resource manager
	operationID := fmt.Sprintf("futures_arbitrage_calculator_%d", time.Now().UnixNano())
	s.resourceManager.RegisterResource(operationID, GoroutineResource, func() error {
		return nil
	}, map[string]interface{}{"operation": "periodic_calculation", "service": "futures_arbitrage"})
	defer func() {
		if err := s.resourceManager.CleanupResource(operationID); err != nil {
			log.Printf("Failed to cleanup resource %s: %v", operationID, err)
		}
	}()

	ticker := time.NewTicker(30 * time.Second) // Calculate every 30 seconds
	defer ticker.Stop()

	log.Println("Futures arbitrage opportunity calculator started")

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Futures arbitrage opportunity calculator stopped")
			return
		case <-ticker.C:
			// Create timeout context for each calculation cycle
			calcCtx, calcCancel := context.WithTimeout(s.ctx, 25*time.Second)

			// Execute with error recovery
			err := s.errorRecoveryManager.ExecuteWithRetry(calcCtx, "calculate_opportunities", func() error {
				return s.calculateAndStoreOpportunities(calcCtx)
			})

			if err != nil {
				log.Printf("Error calculating opportunities: %v", err)
				// Record failure in performance monitor
				if s.performanceMonitor != nil {
					metrics := s.performanceMonitor.GetApplicationMetrics().CollectorMetrics
					metrics.FailedCollections++
					s.performanceMonitor.UpdateCollectorMetrics(metrics)
				}
			}

			calcCancel()
		}
	}
}

// calculateAndStoreOpportunities fetches funding rates and calculates arbitrage opportunities
func (s *FuturesArbitrageService) calculateAndStoreOpportunities(ctx context.Context) error {
	log.Println("Starting futures arbitrage opportunity calculation...")

	// Clean up expired opportunities first with error recovery
	err := s.errorRecoveryManager.ExecuteWithRetry(ctx, "cleanup_opportunities", func() error {
		return s.cleanupExpiredOpportunities(ctx)
	})
	if err != nil {
		log.Printf("Warning: Failed to cleanup expired opportunities: %v", err)
	}

	// Get latest funding rates grouped by symbol with error recovery
	var fundingRateMap map[string][]FundingRateData
	err = s.errorRecoveryManager.ExecuteWithRetry(ctx, "get_funding_rates", func() error {
		var err error
		fundingRateMap, err = s.getLatestFundingRates(ctx)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to get funding rates: %w", err)
	}

	if len(fundingRateMap) == 0 {
		log.Println("No funding rates available for opportunity calculation")
		return nil
	}

	opportunitiesCalculated := 0
	opportunitiesStored := 0

	// Calculate opportunities for each symbol
	for symbol, exchangeRates := range fundingRateMap {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if len(exchangeRates) < 2 {
			continue // Need at least 2 exchanges for arbitrage
		}

		// Find all possible arbitrage pairs
		for i, longRate := range exchangeRates {
			for j, shortRate := range exchangeRates {
				if i == j {
					continue // Same exchange
				}

				// Calculate net funding rate (short - long)
				netFundingRate := shortRate.Rate.Sub(longRate.Rate)

				// Only consider profitable opportunities (positive net funding rate)
				if netFundingRate.LessThanOrEqual(decimal.NewFromFloat(0.0001)) {
					continue // Minimum 0.01% threshold
				}

				opportunitiesCalculated++

				// Create calculation input
				input := models.FuturesArbitrageCalculationInput{
					Symbol:           symbol,
					LongExchange:     longRate.Exchange,
					ShortExchange:    shortRate.Exchange,
					LongFundingRate:  longRate.Rate,
					ShortFundingRate: shortRate.Rate,
					LongMarkPrice:    longRate.MarkPrice,
					ShortMarkPrice:   shortRate.MarkPrice,
					FundingInterval:  8, // Default 8 hours
				}

				// Calculate opportunity
				opportunity, err := s.calculator.CalculateFuturesArbitrage(input)
				if err != nil {
					log.Printf("Failed to calculate opportunity for %s %s/%s: %v",
						symbol, longRate.Exchange, shortRate.Exchange, err)
					continue
				}

				// Only store active opportunities with error recovery
				if opportunity.IsActive {
					err := s.errorRecoveryManager.ExecuteWithRetry(ctx, "store_opportunity", func() error {
						return s.storeOpportunity(ctx, opportunity)
					})
					if err != nil {
						log.Printf("Failed to store opportunity for %s %s/%s: %v",
							symbol, longRate.Exchange, shortRate.Exchange, err)
						continue
					}
					opportunitiesStored++
				}
			}
		}
	}

	log.Printf("Opportunity calculation completed: %d calculated, %d stored",
		opportunitiesCalculated, opportunitiesStored)
	return nil
}

// FundingRateData represents funding rate data for opportunity calculation
type FundingRateData struct {
	Exchange  string
	Symbol    string
	Rate      decimal.Decimal
	MarkPrice decimal.Decimal
	Timestamp time.Time
}

// getLatestFundingRates retrieves the latest funding rates grouped by symbol
func (s *FuturesArbitrageService) getLatestFundingRates(ctx context.Context) (map[string][]FundingRateData, error) {
	cacheKey := "funding_rates:latest:all"

	// Check Redis cache first
	if s.redisClient != nil {
		if cachedData, err := s.redisClient.Get(ctx, cacheKey).Result(); err == nil {
			var fundingRateMap map[string][]FundingRateData
			if json.Unmarshal([]byte(cachedData), &fundingRateMap) == nil {
				return fundingRateMap, nil
			}
		}
	}

	query := `
		SELECT DISTINCT ON (e.name, tp.symbol) 
			e.name as exchange,
			tp.symbol,
			fr.funding_rate,
			fr.mark_price,
			fr.timestamp
		FROM funding_rates fr
		JOIN exchanges e ON fr.exchange_id = e.id
		JOIN trading_pairs tp ON fr.trading_pair_id = tp.id
		WHERE fr.timestamp > NOW() - INTERVAL '1 hour'
		  AND fr.funding_rate IS NOT NULL
		  AND fr.mark_price IS NOT NULL
		  AND fr.mark_price > 0
		ORDER BY e.name, tp.symbol, fr.timestamp DESC
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query funding rates: %w", err)
	}
	defer rows.Close()

	fundingRateMap := make(map[string][]FundingRateData)

	for rows.Next() {
		var data FundingRateData
		if err := rows.Scan(
			&data.Exchange,
			&data.Symbol,
			&data.Rate,
			&data.MarkPrice,
			&data.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("failed to scan funding rate: %w", err)
		}

		fundingRateMap[data.Symbol] = append(fundingRateMap[data.Symbol], data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating funding rates: %w", err)
	}

	// Cache the result with 1-minute TTL using error recovery
	if s.redisClient != nil {
		if jsonData, err := json.Marshal(fundingRateMap); err == nil {
			err := s.errorRecoveryManager.ExecuteWithRetry(ctx, "redis_set_funding_rates", func() error {
				return s.redisClient.Set(ctx, cacheKey, jsonData, time.Minute).Err()
			})
			if err != nil {
				log.Printf("Failed to cache funding rates: %v", err)
			} else {
				log.Printf("Cached %d symbols funding rates for 1 minute", len(fundingRateMap))
			}
		}
	}

	return fundingRateMap, nil
}

// storeOpportunity stores a calculated opportunity in the database
func (s *FuturesArbitrageService) storeOpportunity(ctx context.Context, opportunity *models.FuturesArbitrageOpportunity) error {
	query := `
		INSERT INTO futures_arbitrage_opportunities (
			symbol, base_currency, quote_currency,
			long_exchange, short_exchange,
			long_funding_rate, short_funding_rate, net_funding_rate, funding_interval,
			long_mark_price, short_mark_price, price_difference, price_difference_percentage,
			hourly_rate, daily_rate, apy,
			estimated_profit_8h, estimated_profit_daily, estimated_profit_weekly, estimated_profit_monthly,
			risk_score, volatility_score, liquidity_score,
			recommended_position_size, max_leverage, recommended_leverage, stop_loss_percentage,
			min_position_size, max_position_size, optimal_position_size,
			detected_at, expires_at, next_funding_time, time_to_next_funding, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33, $34, $35
		)
		ON CONFLICT (symbol, long_exchange, short_exchange) 
		DO UPDATE SET
			long_funding_rate = EXCLUDED.long_funding_rate,
			short_funding_rate = EXCLUDED.short_funding_rate,
			net_funding_rate = EXCLUDED.net_funding_rate,
			long_mark_price = EXCLUDED.long_mark_price,
			short_mark_price = EXCLUDED.short_mark_price,
			price_difference = EXCLUDED.price_difference,
			price_difference_percentage = EXCLUDED.price_difference_percentage,
			hourly_rate = EXCLUDED.hourly_rate,
			daily_rate = EXCLUDED.daily_rate,
			apy = EXCLUDED.apy,
			estimated_profit_8h = EXCLUDED.estimated_profit_8h,
			estimated_profit_daily = EXCLUDED.estimated_profit_daily,
			estimated_profit_weekly = EXCLUDED.estimated_profit_weekly,
			estimated_profit_monthly = EXCLUDED.estimated_profit_monthly,
			risk_score = EXCLUDED.risk_score,
			volatility_score = EXCLUDED.volatility_score,
			liquidity_score = EXCLUDED.liquidity_score,
			recommended_position_size = EXCLUDED.recommended_position_size,
			max_leverage = EXCLUDED.max_leverage,
			recommended_leverage = EXCLUDED.recommended_leverage,
			stop_loss_percentage = EXCLUDED.stop_loss_percentage,
			min_position_size = EXCLUDED.min_position_size,
			max_position_size = EXCLUDED.max_position_size,
			optimal_position_size = EXCLUDED.optimal_position_size,
			detected_at = EXCLUDED.detected_at,
			expires_at = EXCLUDED.expires_at,
			next_funding_time = EXCLUDED.next_funding_time,
			time_to_next_funding = EXCLUDED.time_to_next_funding,
			is_active = EXCLUDED.is_active
	`

	_, err := s.db.Pool.Exec(ctx, query,
		opportunity.Symbol, opportunity.BaseCurrency, opportunity.QuoteCurrency,
		opportunity.LongExchange, opportunity.ShortExchange,
		opportunity.LongFundingRate, opportunity.ShortFundingRate, opportunity.NetFundingRate, opportunity.FundingInterval,
		opportunity.LongMarkPrice, opportunity.ShortMarkPrice, opportunity.PriceDifference, opportunity.PriceDifferencePercentage,
		opportunity.HourlyRate, opportunity.DailyRate, opportunity.APY,
		opportunity.EstimatedProfit8h, opportunity.EstimatedProfitDaily, opportunity.EstimatedProfitWeekly, opportunity.EstimatedProfitMonthly,
		opportunity.RiskScore, opportunity.VolatilityScore, opportunity.LiquidityScore,
		opportunity.RecommendedPositionSize, opportunity.MaxLeverage, opportunity.RecommendedLeverage, opportunity.StopLossPercentage,
		opportunity.MinPositionSize, opportunity.MaxPositionSize, opportunity.OptimalPositionSize,
		opportunity.DetectedAt, opportunity.ExpiresAt, opportunity.NextFundingTime, opportunity.TimeToNextFunding, opportunity.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to store opportunity: %w", err)
	}

	return nil
}

// cleanupExpiredOpportunities removes expired opportunities from the database
func (s *FuturesArbitrageService) cleanupExpiredOpportunities(ctx context.Context) error {
	query := `
		DELETE FROM futures_arbitrage_opportunities 
		WHERE expires_at < NOW() OR detected_at < NOW() - INTERVAL '1 hour'
	`

	result, err := s.db.Pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired opportunities: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d expired futures arbitrage opportunities", rowsAffected)
	}

	return nil
}

// IsRunning returns whether the service is currently running
func (s *FuturesArbitrageService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
