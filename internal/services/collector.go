package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
)

// CollectorConfig holds configuration for the collector service
type CollectorConfig struct {
	IntervalSeconds int `mapstructure:"interval_seconds"`
	MaxErrors       int `mapstructure:"max_errors"`
}

// CollectorService handles market data collection from exchanges
type CollectorService struct {
	db              *database.PostgresDB
	ccxtService     ccxt.CCXTService
	config          *config.Config
	collectorConfig CollectorConfig
	workers         map[string]*Worker
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// Worker represents a background worker for collecting data from a specific exchange
type Worker struct {
	Exchange   string
	Symbols    []string
	Interval   time.Duration
	LastUpdate time.Time
	IsRunning  bool
	ErrorCount int
	MaxErrors  int
}

// NewCollectorService creates a new market data collector service
func NewCollectorService(db *database.PostgresDB, ccxtService ccxt.CCXTService, cfg *config.Config) *CollectorService {
	ctx, cancel := context.WithCancel(context.Background())

	// Parse collection interval from config
	intervalSeconds := 300 // Default 5 minutes
	if cfg.MarketData.CollectionInterval != "" {
		if duration, err := time.ParseDuration(cfg.MarketData.CollectionInterval); err == nil {
			intervalSeconds = int(duration.Seconds())
		}
	}

	collectorConfig := CollectorConfig{
		IntervalSeconds: intervalSeconds,
		MaxErrors:       5, // Default 5 max errors
	}
	return &CollectorService{
		db:              db,
		ccxtService:     ccxtService,
		config:          cfg,
		collectorConfig: collectorConfig,
		workers:         make(map[string]*Worker),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start initializes and starts all collection workers
func (c *CollectorService) Start() error {
	log.Println("Starting market data collector service...")

	// Initialize CCXT service
	if err := c.ccxtService.Initialize(c.ctx); err != nil {
		return fmt.Errorf("failed to initialize CCXT service: %w", err)
	}

	// Get supported exchanges
	exchanges := c.ccxtService.GetSupportedExchanges()

	// Get symbols that appear on multiple exchanges for arbitrage
	multiExchangeSymbols, err := c.getMultiExchangeSymbols(exchanges)
	if err != nil {
		log.Printf("Warning: Failed to get multi-exchange symbols: %v", err)
		// Continue with individual exchange symbols as fallback
	}

	// Create workers for each exchange
	for _, exchangeID := range exchanges {
		if err := c.createWorker(exchangeID, multiExchangeSymbols); err != nil {
			log.Printf("Failed to create worker for exchange %s: %v", exchangeID, err)
			continue
		}
	}

	log.Printf("Started %d collection workers", len(c.workers))
	return nil
}

// Stop gracefully stops all collection workers
func (c *CollectorService) Stop() {
	log.Println("Stopping market data collector service...")
	c.cancel()
	c.wg.Wait()
	log.Println("Market data collector service stopped")
}

// createWorker creates and starts a worker for a specific exchange
func (c *CollectorService) createWorker(exchangeID string, multiExchangeSymbols map[string]int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get all available markets from the exchange first
	markets, err := c.ccxtService.FetchMarkets(c.ctx, exchangeID)
	if err != nil {
		return fmt.Errorf("failed to fetch markets for exchange %s: %w", exchangeID, err)
	}

	// Filter out invalid symbols (options, derivatives, etc.)
	validSymbols := c.filterValidSymbols(markets.Symbols)

	// Further filter to only include symbols that appear on multiple exchanges (for arbitrage)
	arbitrageSymbols := c.filterArbitrageSymbols(validSymbols, multiExchangeSymbols)
	log.Printf("Filtered %d arbitrage symbols from %d valid symbols (%d total) for %s",
		len(arbitrageSymbols), len(validSymbols), len(markets.Symbols), exchangeID)

	// Use arbitrage symbols if available, otherwise fall back to valid symbols
	finalSymbols := arbitrageSymbols
	if len(finalSymbols) == 0 {
		log.Printf("No arbitrage symbols found for %s, using all valid symbols", exchangeID)
		finalSymbols = validSymbols
	}

	// Ensure all trading pairs exist in database
	for _, symbol := range finalSymbols {
		if err := c.ensureTradingPairExists(symbol); err != nil {
			log.Printf("Warning: Failed to ensure trading pair %s exists: %v", symbol, err)
		}
	}

	if len(finalSymbols) == 0 {
		log.Printf("No valid trading pairs found for exchange %s, skipping worker creation", exchangeID)
		return nil
	}

	// Create worker with filtered symbols
	worker := &Worker{
		Exchange:  exchangeID,
		Symbols:   finalSymbols,
		Interval:  time.Duration(c.collectorConfig.IntervalSeconds) * time.Second,
		MaxErrors: c.collectorConfig.MaxErrors,
		IsRunning: true,
	}

	c.workers[exchangeID] = worker

	// Start worker goroutine
	c.wg.Add(1)
	go c.runWorker(worker)

	log.Printf("Created worker for exchange %s with %d symbols", exchangeID, len(finalSymbols))
	return nil
}

// runWorker runs the collection loop for a specific worker
func (c *CollectorService) runWorker(worker *Worker) {
	defer c.wg.Done()

	ticker := time.NewTicker(worker.Interval)
	defer ticker.Stop()

	log.Printf("Worker for exchange %s started with %d symbols", worker.Exchange, len(worker.Symbols))

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("Worker for exchange %s stopping due to context cancellation", worker.Exchange)
			return
		case <-ticker.C:
			// Collect market data for active trading pairs
			if err := c.collectMarketData(worker); err != nil {
				worker.ErrorCount++
				log.Printf("Error collecting data for exchange %s: %v (error count: %d)", worker.Exchange, err, worker.ErrorCount)

				if worker.ErrorCount >= worker.MaxErrors {
					log.Printf("Worker for exchange %s exceeded max errors (%d), stopping", worker.Exchange, worker.MaxErrors)
					worker.IsRunning = false
					return
				}
			} else {
				// Reset error count on successful collection
				worker.ErrorCount = 0
				worker.LastUpdate = time.Now()
			}

			// Collect all available market data (not limited to active pairs)
			if err := c.collectAllMarketData(worker); err != nil {
				log.Printf("Warning: Failed to collect all market data for exchange %s: %v", worker.Exchange, err)
			}

			// Collect funding rates for futures markets
			if err := c.collectFundingRates(worker); err != nil {
				log.Printf("Warning: Failed to collect funding rates for exchange %s: %v", worker.Exchange, err)
			}
		}
	}
}

// collectMarketData collects market data for all symbols of a specific exchange
func (c *CollectorService) collectMarketData(worker *Worker) error {
	log.Printf("Collecting market data for exchange %s (%d symbols)", worker.Exchange, len(worker.Symbols))

	// Collect ticker data for all symbols with rate limiting
	for i, symbol := range worker.Symbols {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		if err := c.collectTickerData(worker.Exchange, symbol); err != nil {
			log.Printf("Failed to collect ticker data for %s:%s: %v", worker.Exchange, symbol, err)
			// Continue with other symbols even if one fails
			continue
		}

		// Add rate limiting delay between requests (aggressive mode: 30ms)
		if i < len(worker.Symbols)-1 {
			time.Sleep(30 * time.Millisecond)
		}
	}

	// Collect funding rates for futures markets
	if err := c.collectFundingRates(worker); err != nil {
		log.Printf("Failed to collect funding rates for %s: %v", worker.Exchange, err)
		// Continue even if funding rate collection fails
	}

	return nil
}

// collectTickerData collects and stores ticker data for a specific symbol
func (c *CollectorService) collectTickerData(exchange, symbol string) error {
	// Fetch ticker data from CCXT service
	ticker, err := c.ccxtService.FetchSingleTicker(c.ctx, exchange, symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch ticker data: %w", err)
	}

	// Ensure exchange exists and get its ID
	exchangeID, err := c.getOrCreateExchange(exchange)
	if err != nil {
		return fmt.Errorf("failed to get or create exchange: %w", err)
	}

	// Ensure trading pair exists and get its ID
	tradingPairID, err := c.getOrCreateTradingPair(symbol)
	if err != nil {
		return fmt.Errorf("failed to get or create trading pair: %w", err)
	}

	// Save market data to database with proper column mapping
	_, err = c.db.Pool.Exec(c.ctx,
		`INSERT INTO market_data (exchange_id, trading_pair_id, last_price, volume_24h, timestamp, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		exchangeID, tradingPairID, ticker.Price, ticker.Volume, ticker.Timestamp, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to save market data: %w", err)
	}

	return nil
}

// collectAllMarketData collects ticker data for all available markets on an exchange
func (c *CollectorService) collectAllMarketData(worker *Worker) error {
	// Get all available markets from the exchange
	markets, err := c.ccxtService.FetchMarkets(c.ctx, worker.Exchange)
	if err != nil {
		return fmt.Errorf("failed to fetch markets for %s: %w", worker.Exchange, err)
	}

	// Filter out invalid symbols (options, derivatives, etc.)
	validSymbols := c.filterValidSymbols(markets.Symbols)
	log.Printf("Filtered %d valid symbols from %d total symbols for %s", len(validSymbols), len(markets.Symbols), worker.Exchange)

	// Collect ticker data for each valid market symbol with rate limiting
	for i, symbol := range validSymbols {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			if err := c.collectTickerData(worker.Exchange, symbol); err != nil {
				log.Printf("Failed to collect ticker for %s:%s: %v", worker.Exchange, symbol, err)
				// Continue with other symbols even if one fails
			}

			// Add rate limiting delay between requests (aggressive mode: 20ms)
			if i < len(validSymbols)-1 {
				time.Sleep(20 * time.Millisecond)
			}
		}
	}

	// Collect funding rates for futures markets
	if err := c.collectFundingRates(worker); err != nil {
		log.Printf("Failed to collect funding rates for %s: %v", worker.Exchange, err)
		// Continue even if funding rate collection fails
	}

	return nil
}

// collectFundingRates collects funding rates for futures markets
func (c *CollectorService) collectFundingRates(worker *Worker) error {
	// Get all funding rates for the exchange
	fundingRates, err := c.ccxtService.FetchAllFundingRates(c.ctx, worker.Exchange)
	if err != nil {
		return fmt.Errorf("failed to fetch funding rates for %s: %w", worker.Exchange, err)
	}

	// Store funding rates in database with rate limiting
	for i, rate := range fundingRates {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			if err := c.storeFundingRate(worker.Exchange, rate); err != nil {
				log.Printf("Failed to store funding rate for %s:%s: %v", worker.Exchange, rate.Symbol, err)
				// Continue with other rates even if one fails
			}

			// Add rate limiting delay between database writes (25ms)
			if i < len(fundingRates)-1 {
				time.Sleep(25 * time.Millisecond)
			}
		}
	}

	return nil
}

// storeFundingRate stores funding rate data in the database
func (c *CollectorService) storeFundingRate(exchange string, rate ccxt.FundingRate) error {
	// Ensure exchange exists and get its ID
	exchangeID, err := c.getOrCreateExchange(exchange)
	if err != nil {
		return fmt.Errorf("failed to get or create exchange: %w", err)
	}

	// Ensure trading pair exists and get its ID
	tradingPairID, err := c.getOrCreateTradingPair(rate.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get or create trading pair: %w", err)
	}

	// Save funding rate to database
	_, err = c.db.Pool.Exec(c.ctx,
		"INSERT INTO funding_rates (exchange_id, trading_pair_id, funding_rate, funding_time, next_funding_time, mark_price, index_price, timestamp, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		exchangeID, tradingPairID, rate.FundingRate, rate.FundingTimestamp.Time(), rate.NextFundingTime.Time(), rate.MarkPrice, rate.IndexPrice, rate.Timestamp.Time(), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to save funding rate: %w", err)
	}

	return nil
}

// GetWorkerStatus returns the status of all workers
func (c *CollectorService) GetWorkerStatus() map[string]*Worker {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]*Worker)
	for exchange, worker := range c.workers {
		status[exchange] = &Worker{
			Exchange:   worker.Exchange,
			Symbols:    worker.Symbols,
			Interval:   worker.Interval,
			LastUpdate: worker.LastUpdate,
			IsRunning:  worker.IsRunning,
			ErrorCount: worker.ErrorCount,
			MaxErrors:  worker.MaxErrors,
		}
	}
	return status
}

// RestartWorker restarts a specific worker
func (c *CollectorService) RestartWorker(exchangeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	worker, exists := c.workers[exchangeID]
	if !exists {
		return fmt.Errorf("worker for exchange %s not found", exchangeID)
	}

	// Reset worker state
	worker.ErrorCount = 0
	worker.IsRunning = true

	// Start new worker goroutine
	c.wg.Add(1)
	go c.runWorker(worker)

	log.Printf("Restarted worker for exchange %s", exchangeID)
	return nil
}

// IsHealthy checks if the collector service is healthy
func (c *CollectorService) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.workers) == 0 {
		return false
	}

	// Check if at least 50% of workers are running
	runningWorkers := 0
	for _, worker := range c.workers {
		if worker.IsRunning {
			runningWorkers++
		}
	}

	return float64(runningWorkers)/float64(len(c.workers)) >= 0.5
}

// ensureTradingPairExists ensures a trading pair exists in the database
func (c *CollectorService) ensureTradingPairExists(symbol string) error {
	_, err := c.getOrCreateTradingPair(symbol)
	return err
}

// getOrCreateTradingPair gets or creates a trading pair and returns its ID
func (c *CollectorService) getOrCreateTradingPair(symbol string) (int, error) {
	// First try to get existing trading pair
	var tradingPairID int
	err := c.db.Pool.QueryRow(c.ctx, "SELECT id FROM trading_pairs WHERE symbol = $1", symbol).Scan(&tradingPairID)
	if err == nil {
		return tradingPairID, nil
	}

	// If not found, create new trading pair
	baseCurrency, quoteCurrency := c.parseSymbol(symbol)
	if baseCurrency == "" || quoteCurrency == "" {
		return 0, fmt.Errorf("failed to parse symbol: %s", symbol)
	}

	// Check if it's a futures pair
	isFutures := c.isFuturesSymbol(symbol)

	// Insert new trading pair
	err = c.db.Pool.QueryRow(c.ctx,
		"INSERT INTO trading_pairs (symbol, base_currency, quote_currency, is_futures) VALUES ($1, $2, $3, $4) RETURNING id",
		symbol, baseCurrency, quoteCurrency, isFutures).Scan(&tradingPairID)
	if err != nil {
		return 0, fmt.Errorf("failed to create trading pair: %w", err)
	}

	log.Printf("Created new trading pair: %s (ID: %d)", symbol, tradingPairID)
	return tradingPairID, nil
}

// getOrCreateExchange gets or creates an exchange and returns its ID
func (c *CollectorService) getOrCreateExchange(ccxtID string) (int, error) {
	// First try to get existing exchange
	var exchangeID int
	err := c.db.Pool.QueryRow(c.ctx, "SELECT id FROM exchanges WHERE ccxt_id = $1", ccxtID).Scan(&exchangeID)
	if err == nil {
		return exchangeID, nil
	}

	// If not found, create new exchange with basic information
	caser := cases.Title(language.English)
	displayName := caser.String(ccxtID)
	name := strings.ToLower(ccxtID)

	// Insert new exchange with default values
	err = c.db.Pool.QueryRow(c.ctx,
		"INSERT INTO exchanges (name, display_name, ccxt_id, status, has_spot, has_futures) VALUES ($1, $2, $3, 'active', true, true) RETURNING id",
		name, displayName, ccxtID).Scan(&exchangeID)
	if err != nil {
		return 0, fmt.Errorf("failed to create exchange: %w", err)
	}

	log.Printf("Created new exchange: %s (ID: %d)", ccxtID, exchangeID)
	return exchangeID, nil
}

// parseSymbol parses a trading symbol into base and quote currencies
func (c *CollectorService) parseSymbol(symbol string) (string, string) {
	// Handle common separators
	if strings.Contains(symbol, "/") {
		parts := strings.Split(symbol, "/")
		if len(parts) >= 2 {
			base := parts[0]
			quote := strings.Split(parts[1], ":")[0] // Remove settlement currency if present
			return base, quote
		}
	}

	// Handle symbols without separators (like BTCUSDT)
	commonQuotes := []string{"USDT", "USDC", "BTC", "ETH", "BNB", "USD", "EUR", "GBP"}
	for _, quote := range commonQuotes {
		if strings.HasSuffix(symbol, quote) {
			base := strings.TrimSuffix(symbol, quote)
			if len(base) > 0 {
				return base, quote
			}
		}
	}

	return "", ""
}

// filterValidSymbols filters out invalid symbols that cause ticker fetch errors
func (c *CollectorService) filterValidSymbols(symbols []string) []string {
	var validSymbols []string

	for _, symbol := range symbols {
		// Skip options contracts (contain dates and strike prices)
		if c.isOptionsContract(symbol) {
			continue
		}

		// Skip symbols with unusual formats that typically cause errors
		if c.isInvalidSymbolFormat(symbol) {
			continue
		}

		validSymbols = append(validSymbols, symbol)
	}

	return validSymbols
}

// isOptionsContract checks if a symbol represents an options contract
func (c *CollectorService) isOptionsContract(symbol string) bool {
	// Options contracts typically have dates and strike prices
	// Examples: SOLUSDT:USDT-250815-180-C, BTC-25DEC20-20000-C
	return strings.Contains(symbol, "-C") || strings.Contains(symbol, "-P") ||
		(strings.Contains(symbol, "-") && (strings.Contains(symbol, "20") || strings.Contains(symbol, "25")))
}

// isInvalidSymbolFormat checks for other invalid symbol formats
func (c *CollectorService) isInvalidSymbolFormat(symbol string) bool {
	// Skip symbols that are too long (likely derivatives)
	if len(symbol) > 20 {
		return true
	}

	// Skip symbols with multiple colons (complex derivatives)
	if strings.Count(symbol, ":") > 1 {
		return true
	}

	// Skip symbols with unusual characters that indicate derivatives
	if strings.Contains(symbol, "_") && strings.Contains(symbol, "-") {
		return true
	}

	return false
}

// isFuturesSymbol determines if a symbol represents a futures contract
func (c *CollectorService) isFuturesSymbol(symbol string) bool {
	// Common futures indicators
	futuresIndicators := []string{"PERP", "-PERP", "_PERP", "SWAP", "-SWAP", "_SWAP"}
	for _, indicator := range futuresIndicators {
		if strings.Contains(strings.ToUpper(symbol), indicator) {
			return true
		}
	}

	// Check for settlement currency (e.g., BTC/USDT:USDT)
	if strings.Contains(symbol, ":") {
		return true
	}

	return false
}

// getMultiExchangeSymbols collects symbols from all exchanges and returns those that appear on multiple exchanges
func (c *CollectorService) getMultiExchangeSymbols(exchanges []string) (map[string]int, error) {
	symbolCount := make(map[string]int)
	minExchanges := 2 // Minimum number of exchanges a symbol must appear on

	log.Printf("Collecting symbols from %d exchanges for arbitrage filtering...", len(exchanges))

	// Collect symbols from each exchange
	for _, exchangeID := range exchanges {
		markets, err := c.ccxtService.FetchMarkets(c.ctx, exchangeID)
		if err != nil {
			log.Printf("Warning: Failed to fetch markets for %s: %v", exchangeID, err)
			continue
		}

		// Filter valid symbols for this exchange
		validSymbols := c.filterValidSymbols(markets.Symbols)

		// Count occurrences of each symbol
		for _, symbol := range validSymbols {
			symbolCount[symbol]++
		}

		log.Printf("Found %d valid symbols on %s", len(validSymbols), exchangeID)
	}

	// Filter to only symbols that appear on multiple exchanges
	multiExchangeSymbols := make(map[string]int)
	for symbol, count := range symbolCount {
		if count >= minExchanges {
			multiExchangeSymbols[symbol] = count
		}
	}

	log.Printf("Found %d symbols that appear on %d+ exchanges (out of %d total unique symbols)",
		len(multiExchangeSymbols), minExchanges, len(symbolCount))

	return multiExchangeSymbols, nil
}

// filterArbitrageSymbols filters symbols to only include those that appear on multiple exchanges
func (c *CollectorService) filterArbitrageSymbols(symbols []string, multiExchangeSymbols map[string]int) []string {
	if len(multiExchangeSymbols) == 0 {
		return symbols // Return all symbols if no multi-exchange data available
	}

	var arbitrageSymbols []string
	for _, symbol := range symbols {
		if _, exists := multiExchangeSymbols[symbol]; exists {
			arbitrageSymbols = append(arbitrageSymbols, symbol)
		}
	}

	return arbitrageSymbols
}
