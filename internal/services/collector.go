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
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
	"github.com/shopspring/decimal"
)

// CollectorConfig holds configuration for the collector service
type CollectorConfig struct {
	IntervalSeconds int `mapstructure:"interval_seconds"`
	MaxErrors       int `mapstructure:"max_errors"`
}

// BackfillConfig holds configuration for historical data backfill
type BackfillConfig struct {
	Enabled               bool `yaml:"enabled" default:"true"`
	BackfillHours         int  `yaml:"backfill_hours" default:"6"`
	MinDataThresholdHours int  `yaml:"min_data_threshold_hours" default:"12"`
	BatchSize             int  `yaml:"batch_size" default:"50"`
	DelayBetweenBatches   int  `yaml:"delay_between_batches_ms" default:"100"`
}

// SymbolCacheEntry represents a cached entry for exchange symbols
type SymbolCacheEntry struct {
	Symbols   []string
	ExpiresAt time.Time
}

// SymbolCacheStats tracks cache performance metrics
type SymbolCacheStats struct {
	Hits   int64
	Misses int64
	Sets   int64
}

// SymbolCache manages cached active symbols for exchanges
type SymbolCache struct {
	cache map[string]*SymbolCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
	stats SymbolCacheStats
}

// BlacklistCacheEntry represents a cached entry for blacklisted symbols
type BlacklistCacheEntry struct {
	Reason    string
	ExpiresAt time.Time
}

// BlacklistCacheStats tracks blacklist cache performance metrics
type BlacklistCacheStats struct {
	Hits      int64
	Misses    int64
	Additions int64
	Skips     int64
}

// BlacklistCache manages cached blacklisted symbols that consistently fail
type BlacklistCache struct {
	cache map[string]*BlacklistCacheEntry // key: "exchange:symbol"
	mu    sync.RWMutex
	ttl   time.Duration
	stats BlacklistCacheStats
}

// ExchangeCapabilityEntry represents cached exchange capability information
type ExchangeCapabilityEntry struct {
	SupportsFundingRates bool
	LastChecked          time.Time
	ExpiresAt            time.Time
}

// ExchangeCapabilityCache manages cached exchange capability information
type ExchangeCapabilityCache struct {
	cache map[string]*ExchangeCapabilityEntry // key: exchange name
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewSymbolCache creates a new symbol cache with specified TTL
func NewSymbolCache(ttl time.Duration) *SymbolCache {
	return &SymbolCache{
		cache: make(map[string]*SymbolCacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves symbols from cache if not expired
func (sc *SymbolCache) Get(exchangeID string) ([]string, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	entry, exists := sc.cache[exchangeID]
	if !exists {
		sc.stats.Misses++
		log.Printf("Cache MISS for %s (total hits: %d, misses: %d)", exchangeID, sc.stats.Hits, sc.stats.Misses)
		return nil, false
	}

	sc.stats.Hits++

	// During runtime, always return cached symbols even if expired to prevent API calls
	// Only check expiration during startup phase
	if time.Now().After(entry.ExpiresAt) {
		// Log that cache is expired but still returning cached data
		log.Printf("Cache HIT (expired) for %s but returning cached symbols to prevent runtime API calls (%d symbols, hits: %d, misses: %d)", exchangeID, len(entry.Symbols), sc.stats.Hits, sc.stats.Misses)
	} else {
		log.Printf("Cache HIT for %s (%d symbols, hits: %d, misses: %d)", exchangeID, len(entry.Symbols), sc.stats.Hits, sc.stats.Misses)
	}

	return entry.Symbols, true
}

// Set stores symbols in cache with TTL
func (sc *SymbolCache) Set(exchangeID string, symbols []string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.stats.Sets++
	sc.cache[exchangeID] = &SymbolCacheEntry{
		Symbols:   symbols,
		ExpiresAt: time.Now().Add(sc.ttl),
	}
	log.Printf("Cache SET for %s (%d symbols, total sets: %d)", exchangeID, len(symbols), sc.stats.Sets)
}

// GetStats returns current cache statistics
func (sc *SymbolCache) GetStats() SymbolCacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats
}

// LogStats logs current cache performance statistics
func (sc *SymbolCache) LogStats() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	total := sc.stats.Hits + sc.stats.Misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(sc.stats.Hits) / float64(total) * 100
	}

	log.Printf("Symbol Cache Stats - Hits: %d, Misses: %d, Sets: %d, Hit Rate: %.2f%%",
		sc.stats.Hits, sc.stats.Misses, sc.stats.Sets, hitRate)
}

// NewBlacklistCache creates a new blacklist cache with specified TTL
func NewBlacklistCache(ttl time.Duration) *BlacklistCache {
	return &BlacklistCache{
		cache: make(map[string]*BlacklistCacheEntry),
		ttl:   ttl,
	}
}

// NewExchangeCapabilityCache creates a new exchange capability cache with specified TTL
func NewExchangeCapabilityCache(ttl time.Duration) *ExchangeCapabilityCache {
	return &ExchangeCapabilityCache{
		cache: make(map[string]*ExchangeCapabilityEntry),
		ttl:   ttl,
	}
}

// SupportsFundingRates checks if an exchange supports funding rates
func (ecc *ExchangeCapabilityCache) SupportsFundingRates(exchange string) (bool, bool) {
	ecc.mu.RLock()
	defer ecc.mu.RUnlock()

	entry, exists := ecc.cache[exchange]
	if !exists {
		return false, false // unknown capability
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		return false, false // expired, need to recheck
	}

	return entry.SupportsFundingRates, true // known capability
}

// SetFundingRateSupport sets the funding rate support capability for an exchange
func (ecc *ExchangeCapabilityCache) SetFundingRateSupport(exchange string, supports bool) {
	ecc.mu.Lock()
	defer ecc.mu.Unlock()

	ecc.cache[exchange] = &ExchangeCapabilityEntry{
		SupportsFundingRates: supports,
		LastChecked:          time.Now(),
		ExpiresAt:            time.Now().Add(ecc.ttl),
	}
	log.Printf("Exchange capability cached: %s supports funding rates: %v", exchange, supports)
}

// IsBlacklisted checks if a symbol is blacklisted for a specific exchange
func (bc *BlacklistCache) IsBlacklisted(exchange, symbol string) (bool, string) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", exchange, symbol)
	entry, exists := bc.cache[key]
	if !exists {
		bc.stats.Misses++
		return false, ""
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		delete(bc.cache, key)
		bc.stats.Misses++
		return false, ""
	}

	bc.stats.Hits++
	bc.stats.Skips++
	return true, entry.Reason
}

// Add adds a symbol to the blacklist with a reason
func (bc *BlacklistCache) Add(exchange, symbol, reason string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	key := fmt.Sprintf("%s:%s", exchange, symbol)
	bc.stats.Additions++
	bc.cache[key] = &BlacklistCacheEntry{
		Reason:    reason,
		ExpiresAt: time.Now().Add(bc.ttl),
	}
	log.Printf("Blacklisted symbol %s (reason: %s, expires in %v)", key, reason, bc.ttl)
}

// GetStats returns current blacklist cache statistics
func (bc *BlacklistCache) GetStats() BlacklistCacheStats {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.stats
}

// LogStats logs current blacklist cache performance statistics
func (bc *BlacklistCache) LogStats() {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	total := bc.stats.Hits + bc.stats.Misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(bc.stats.Hits) / float64(total) * 100
	}

	log.Printf("Blacklist Cache Stats - Hits: %d, Misses: %d, Additions: %d, Skips: %d, Hit Rate: %.2f%%",
		bc.stats.Hits, bc.stats.Misses, bc.stats.Additions, bc.stats.Skips, hitRate)
}

// isBlacklistableError checks if an error indicates a symbol should be blacklisted
func isBlacklistableError(err error) (bool, string) {
	if err == nil {
		return false, ""
	}

	errorMsg := err.Error()

	// Coinbase delisted products
	if strings.Contains(errorMsg, "Not allowed for delisted products") {
		return true, "coinbase_delisted"
	}

	// Binance missing market symbols
	if strings.Contains(errorMsg, "does not have market symbol") {
		return true, "binance_missing_symbol"
	}

	// General delisted indicators
	if strings.Contains(errorMsg, "delisted") {
		return true, "delisted"
	}

	// Inactive symbols
	if strings.Contains(errorMsg, "inactive") {
		return true, "inactive"
	}

	// Symbol not found or unavailable
	if strings.Contains(errorMsg, "symbol not found") ||
		strings.Contains(errorMsg, "symbol unavailable") ||
		strings.Contains(errorMsg, "market not found") {
		return true, "symbol_not_found"
	}

	// Exchange-specific error patterns
	if strings.Contains(errorMsg, "CCXT service error (500)") {
		// Check for specific 500 error patterns that indicate permanent issues
		if strings.Contains(errorMsg, "Not allowed") ||
			strings.Contains(errorMsg, "does not have") ||
			strings.Contains(errorMsg, "delisted") {
			return true, "ccxt_500_permanent"
		}
	}

	return false, ""
}

// isFundingRateUnsupportedError checks if an error indicates the exchange doesn't support funding rates
func isFundingRateUnsupportedError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())

	// Check for funding rate unsupported error patterns
	return strings.Contains(errorMsg, "does not support funding rates") ||
		strings.Contains(errorMsg, "funding rates not supported") ||
		strings.Contains(errorMsg, "no funding rates") ||
		(strings.Contains(errorMsg, "ccxt service error (400)") && strings.Contains(errorMsg, "funding"))
}

// CollectorService handles market data collection from exchanges
type CollectorService struct {
	db              *database.PostgresDB
	ccxtService     ccxt.CCXTService
	config          *config.Config
	collectorConfig CollectorConfig
	backfillConfig  BackfillConfig
	workers         map[string]*Worker
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	// Caching and timing controls
	symbolCache             *SymbolCache
	blacklistCache          *BlacklistCache
	exchangeCapabilityCache *ExchangeCapabilityCache
	lastSymbolRefresh       map[string]time.Time
	lastFundingCollection   map[string]time.Time
	symbolRefreshMu         sync.RWMutex
	fundingCollectionMu     sync.RWMutex
	// Separate intervals
	tickerInterval        time.Duration
	symbolRefreshInterval time.Duration
	fundingRateInterval   time.Duration
	// Readiness state
	isInitialized bool
	isReady       bool
	readinessMu   sync.RWMutex
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

	// Initialize backfill configuration from config
	backfillConfig := BackfillConfig{
		Enabled:               cfg.Backfill.Enabled,
		BackfillHours:         cfg.Backfill.BackfillHours,
		MinDataThresholdHours: cfg.Backfill.MinDataThresholdHours,
		BatchSize:             cfg.Backfill.BatchSize,
		DelayBetweenBatches:   cfg.Backfill.DelayBetweenBatches,
	}

	// Initialize separate intervals for different operations
	tickerInterval := time.Duration(intervalSeconds) * time.Second // 5 minutes (from config)
	symbolRefreshInterval := 1 * time.Hour                         // 1 hour for symbol refresh
	fundingRateInterval := 15 * time.Minute                        // 15 minutes for funding rates

	return &CollectorService{
		db:              db,
		ccxtService:     ccxtService,
		config:          cfg,
		collectorConfig: collectorConfig,
		backfillConfig:  backfillConfig,
		workers:         make(map[string]*Worker),
		ctx:             ctx,
		cancel:          cancel,
		// Initialize caching and timing controls
		symbolCache:             NewSymbolCache(1 * time.Hour),              // 1 hour TTL for symbols
		blacklistCache:          NewBlacklistCache(24 * time.Hour),          // 24 hour TTL for blacklisted symbols
		exchangeCapabilityCache: NewExchangeCapabilityCache(24 * time.Hour), // 24 hour TTL for exchange capabilities
		lastSymbolRefresh:       make(map[string]time.Time),
		lastFundingCollection:   make(map[string]time.Time),
		// Set separate intervals
		tickerInterval:        tickerInterval,
		symbolRefreshInterval: symbolRefreshInterval,
		fundingRateInterval:   fundingRateInterval,
	}
}

// Start initializes and starts all collection workers asynchronously
func (c *CollectorService) Start() error {
	log.Println("Starting market data collector service...")

	// Initialize CCXT service
	if err := c.ccxtService.Initialize(c.ctx); err != nil {
		return fmt.Errorf("failed to initialize CCXT service: %w", err)
	}

	// Mark as initialized
	c.readinessMu.Lock()
	c.isInitialized = true
	c.readinessMu.Unlock()

	// Start symbol collection and worker creation asynchronously
	go c.initializeWorkersAsync()

	log.Println("Market data collector service started (workers initializing in background)")
	return nil
}

// initializeWorkersAsync handles symbol collection and worker creation in the background
func (c *CollectorService) initializeWorkersAsync() {
	log.Println("Starting background symbol collection and worker initialization...")

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

	// Mark as ready
	c.readinessMu.Lock()
	c.isReady = true
	c.readinessMu.Unlock()

	log.Printf("Background initialization complete: Started %d collection workers", len(c.workers))
}

// Stop gracefully stops all collection workers
func (c *CollectorService) Stop() {
	log.Println("Stopping market data collector service...")
	c.cancel()
	c.wg.Wait()
	log.Println("Market data collector service stopped")
}

// IsInitialized returns true if the collector service has been initialized
func (c *CollectorService) IsInitialized() bool {
	c.readinessMu.RLock()
	defer c.readinessMu.RUnlock()
	return c.isInitialized
}

// IsReady returns true if the collector service is fully ready (workers created and running)
func (c *CollectorService) IsReady() bool {
	c.readinessMu.RLock()
	defer c.readinessMu.RUnlock()
	return c.isReady
}

// GetReadinessStatus returns the current readiness status
func (c *CollectorService) GetReadinessStatus() (initialized bool, ready bool) {
	c.readinessMu.RLock()
	defer c.readinessMu.RUnlock()
	return c.isInitialized, c.isReady
}

// createWorker creates and starts a worker for a specific exchange
func (c *CollectorService) createWorker(exchangeID string, multiExchangeSymbols map[string]int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Try to get symbols from cache first (should be populated by getMultiExchangeSymbols)
	var activeSymbols []string

	if cached, found := c.symbolCache.Get(exchangeID); found {
		activeSymbols = cached
		log.Printf("Using cached symbols for %s (%d symbols)", exchangeID, len(activeSymbols))
	} else {
		// Cache should always be populated during startup, log error if not found
		log.Printf("Error: No cached symbols found for %s during worker creation", exchangeID)
		return fmt.Errorf("no cached symbols found for exchange %s - cache should be populated during startup", exchangeID)
	}

	// Filter out invalid symbols (options, derivatives, etc.)
	validSymbols := c.filterValidSymbols(activeSymbols)

	// Further filter to only include symbols that appear on multiple exchanges (for arbitrage)
	arbitrageSymbols := c.filterArbitrageSymbols(validSymbols, multiExchangeSymbols)
	log.Printf("Filtered %d arbitrage symbols from %d valid active symbols for %s",
		len(arbitrageSymbols), len(validSymbols), exchangeID)

	// Use arbitrage symbols if available, otherwise fall back to valid active symbols
	finalSymbols := arbitrageSymbols
	if len(finalSymbols) == 0 {
		log.Printf("No arbitrage symbols found for %s, using all valid active symbols", exchangeID)
		finalSymbols = validSymbols
	}

	// Get exchange ID for database operations
	exchangeDBID, err := c.getOrCreateExchange(exchangeID)
	if err != nil {
		return fmt.Errorf("failed to get or create exchange %s: %w", exchangeID, err)
	}

	// Ensure all trading pairs exist in database
	for _, symbol := range finalSymbols {
		if err := c.ensureTradingPairExists(exchangeDBID, symbol); err != nil {
			log.Printf("Warning: Failed to ensure trading pair %s exists: %v", symbol, err)
		}
	}

	if len(finalSymbols) == 0 {
		log.Printf("No valid active trading pairs found for exchange %s, skipping worker creation", exchangeID)
		return nil
	}

	// Create worker with filtered active symbols
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

	log.Printf("Created worker for exchange %s with %d active symbols", exchangeID, len(finalSymbols))
	return nil
}

// runWorker runs the collection loop for a specific worker with separate intervals
func (c *CollectorService) runWorker(worker *Worker) {
	defer c.wg.Done()

	// Use ticker interval for main ticker data collection
	ticker := time.NewTicker(c.tickerInterval)
	defer ticker.Stop()

	// Add cache statistics logging every 10 minutes
	cacheStatsTicker := time.NewTicker(10 * time.Minute)
	defer cacheStatsTicker.Stop()

	log.Printf("Worker for exchange %s started with %d symbols (ticker: %v, funding: %v)",
		worker.Exchange, len(worker.Symbols), c.tickerInterval, c.fundingRateInterval)

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("Worker for exchange %s stopping due to context cancellation", worker.Exchange)
			return
		case <-cacheStatsTicker.C:
			// Log cache statistics periodically
			c.symbolCache.LogStats()
			c.blacklistCache.LogStats()
		case <-ticker.C:
			log.Printf("Worker tick for exchange %s - starting collection cycle", worker.Exchange)

			// Collect market data for active trading pairs (no funding rates here)
			if err := c.collectTickerDataOnly(worker); err != nil {
				worker.ErrorCount++
				log.Printf("Error collecting ticker data for exchange %s: %v (error count: %d)", worker.Exchange, err, worker.ErrorCount)

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

			// Check if it's time to collect funding rates (separate interval)
			c.fundingCollectionMu.RLock()
			lastFundingCollection, exists := c.lastFundingCollection[worker.Exchange]
			c.fundingCollectionMu.RUnlock()

			if !exists || time.Since(lastFundingCollection) >= c.fundingRateInterval {
				log.Printf("Collecting funding rates for exchange %s (interval: %v)", worker.Exchange, c.fundingRateInterval)
				if err := c.collectFundingRates(worker); err != nil {
					log.Printf("Warning: Failed to collect funding rates for exchange %s: %v", worker.Exchange, err)
				} else {
					// Update last funding collection time
					c.fundingCollectionMu.Lock()
					c.lastFundingCollection[worker.Exchange] = time.Now()
					c.fundingCollectionMu.Unlock()
				}
			}
		}
	}
}

// collectTickerDataOnly collects only ticker data for worker symbols (no funding rates)
func (c *CollectorService) collectTickerDataOnly(worker *Worker) error {
	log.Printf("Collecting ticker data for exchange %s (%d symbols)", worker.Exchange, len(worker.Symbols))

	// Collect ticker data for all symbols with rate limiting
	for i, symbol := range worker.Symbols {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		// Use direct collection for worker symbols (skip activity check)
		if err := c.collectTickerDataDirect(worker.Exchange, symbol); err != nil {
			log.Printf("Failed to collect ticker data for %s:%s: %v", worker.Exchange, symbol, err)
			// Continue with other symbols even if one fails
			continue
		}

		// Add rate limiting delay between requests (aggressive mode: 30ms)
		if i < len(worker.Symbols)-1 {
			time.Sleep(30 * time.Millisecond)
		}
	}

	return nil
}

// collectTickerDataDirect collects ticker data without checking symbol activity (for worker symbols)
func (c *CollectorService) collectTickerDataDirect(exchange, symbol string) error {
	// Check if symbol is blacklisted before making API call
	if isBlacklisted, reason := c.blacklistCache.IsBlacklisted(exchange, symbol); isBlacklisted {
		log.Printf("Skipping blacklisted symbol: %s:%s (reason: %s)", exchange, symbol, reason)
		return nil
	}

	// Fetch ticker data from CCXT service directly
	ticker, err := c.ccxtService.FetchSingleTicker(c.ctx, exchange, symbol)
	if err != nil {
		// Check if the error indicates a symbol that should be blacklisted
		if shouldBlacklist, reason := isBlacklistableError(err); shouldBlacklist {
			c.blacklistCache.Add(exchange, symbol, reason)
			log.Printf("Added symbol to blacklist: %s:%s (reason: %s) - %v", exchange, symbol, reason, err)
			return nil
		}
		return fmt.Errorf("failed to fetch ticker data: %w", err)
	}

	// Ensure exchange exists and get its ID
	exchangeID, err := c.getOrCreateExchange(exchange)
	if err != nil {
		return fmt.Errorf("failed to get or create exchange: %w", err)
	}

	// Ensure trading pair exists and get its ID
	tradingPairID, err := c.getOrCreateTradingPair(exchangeID, symbol)
	if err != nil {
		return fmt.Errorf("failed to get or create trading pair: %w", err)
	}

	// Validate price data before saving to database
	if err := c.validateMarketData(ticker, exchange, symbol); err != nil {
		log.Printf("Invalid market data for %s:%s - %v", exchange, symbol, err)
		return nil // Don't save invalid data, but don't fail the collection
	}

	// Save market data to database with proper column mapping
	_, err = c.db.Pool.Exec(c.ctx,
		`INSERT INTO market_data (exchange_id, trading_pair_id, last_price, volume_24h, timestamp, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		exchangeID, tradingPairID, ticker.Price, ticker.Volume, ticker.Timestamp, time.Now())
	if err != nil {
		return fmt.Errorf("failed to save market data: %w", err)
	}

	return nil
}

// collectFundingRates collects funding rates for futures markets
func (c *CollectorService) collectFundingRates(worker *Worker) error {
	log.Printf("Starting funding rate collection for exchange: %s", worker.Exchange)

	// Check if we already know this exchange doesn't support funding rates
	supports, known := c.exchangeCapabilityCache.SupportsFundingRates(worker.Exchange)
	if known && !supports {
		log.Printf("Skipping funding rate collection for %s: exchange does not support funding rates (cached)", worker.Exchange)
		return nil
	}

	// Get all funding rates for the exchange
	fundingRates, err := c.ccxtService.FetchAllFundingRates(c.ctx, worker.Exchange)
	if err != nil {
		// Check if this is a funding rate unsupported error
		if isFundingRateUnsupportedError(err) {
			log.Printf("Exchange %s does not support funding rates, caching this information", worker.Exchange)
			c.exchangeCapabilityCache.SetFundingRateSupport(worker.Exchange, false)
			return nil // Don't treat this as an error
		}
		return fmt.Errorf("failed to fetch funding rates for %s: %w", worker.Exchange, err)
	}

	// If we successfully fetched funding rates, cache that this exchange supports them
	if !known {
		log.Printf("Exchange %s supports funding rates, caching this information", worker.Exchange)
		c.exchangeCapabilityCache.SetFundingRateSupport(worker.Exchange, true)
	}

	log.Printf("Fetched %d funding rates for exchange %s", len(fundingRates), worker.Exchange)

	// Store funding rates in database with rate limiting
	for i, rate := range fundingRates {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
			if err := c.storeFundingRate(worker.Exchange, rate); err != nil {
				log.Printf("Failed to store funding rate for %s:%s: %v", worker.Exchange, rate.Symbol, err)
				// Continue with other rates even if one fails
			} else {
				log.Printf("Successfully stored funding rate for %s:%s (rate: %.6f)", worker.Exchange, rate.Symbol, rate.FundingRate)
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
	tradingPairID, err := c.getOrCreateTradingPair(exchangeID, rate.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get or create trading pair: %w", err)
	}

	// Save funding rate to database with upsert to handle duplicates
	_, err = c.db.Pool.Exec(c.ctx,
		`INSERT INTO funding_rates (exchange_id, trading_pair_id, funding_rate, funding_rate_timestamp, next_funding_time, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (exchange_id, trading_pair_id, funding_rate_timestamp) 
		 DO UPDATE SET 
			funding_rate = EXCLUDED.funding_rate,
			next_funding_time = EXCLUDED.next_funding_time,
			updated_at = NOW()`,
		exchangeID, tradingPairID, rate.FundingRate, rate.FundingTimestamp.Time(), rate.NextFundingTime.Time(), time.Now())
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
func (c *CollectorService) ensureTradingPairExists(exchangeID int, symbol string) error {
	_, err := c.getOrCreateTradingPair(exchangeID, symbol)
	return err
}

// getOrCreateTradingPair gets or creates a trading pair and returns its ID
func (c *CollectorService) getOrCreateTradingPair(exchangeID int, symbol string) (int, error) {
	// First try to get existing trading pair for this exchange and symbol
	var tradingPairID int
	err := c.db.Pool.QueryRow(c.ctx, "SELECT id FROM trading_pairs WHERE exchange_id = $1 AND symbol = $2", exchangeID, symbol).Scan(&tradingPairID)
	if err == nil {
		return tradingPairID, nil
	}

	// If not found, create new trading pair
	baseCurrency, quoteCurrency := c.parseSymbol(symbol)
	if baseCurrency == "" || quoteCurrency == "" {
		return 0, fmt.Errorf("failed to parse symbol: %s", symbol)
	}

	// Insert new trading pair with exchange_id
	err = c.db.Pool.QueryRow(c.ctx,
		"INSERT INTO trading_pairs (exchange_id, symbol, base_currency, quote_currency, is_active) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		exchangeID, symbol, baseCurrency, quoteCurrency, true).Scan(&tradingPairID)
	if err != nil {
		return 0, fmt.Errorf("failed to create trading pair: %w", err)
	}

	log.Printf("Created new trading pair: %s for exchange %d (ID: %d)", symbol, exchangeID, tradingPairID)
	return tradingPairID, nil
}

// getOrCreateExchange gets or creates an exchange and returns its ID
func (c *CollectorService) getOrCreateExchange(ccxtID string) (int, error) {
	// First try to get existing exchange by ccxt_id
	var exchangeID int
	err := c.db.Pool.QueryRow(c.ctx, "SELECT id FROM exchanges WHERE ccxt_id = $1", ccxtID).Scan(&exchangeID)
	if err == nil {
		return exchangeID, nil
	}

	// Also check by name in case exchange exists with different ccxt_id
	name := strings.ToLower(ccxtID)
	err = c.db.Pool.QueryRow(c.ctx, "SELECT id FROM exchanges WHERE name = $1", name).Scan(&exchangeID)
	if err == nil {
		log.Printf("Found existing exchange by name: %s (ID: %d)", name, exchangeID)
		return exchangeID, nil
	}

	// If not found, create new exchange with basic information
	caser := cases.Title(language.English)
	displayName := caser.String(ccxtID)

	// Insert new exchange with conflict resolution
	err = c.db.Pool.QueryRow(c.ctx,
		"INSERT INTO exchanges (name, display_name, ccxt_id, status, has_spot, has_futures) VALUES ($1, $2, $3, 'active', true, true) ON CONFLICT (name) DO UPDATE SET ccxt_id = EXCLUDED.ccxt_id, display_name = EXCLUDED.display_name RETURNING id",
		name, displayName, ccxtID).Scan(&exchangeID)
	if err != nil {
		return 0, fmt.Errorf("failed to create or update exchange: %w", err)
	}

	log.Printf("Created or updated exchange: %s (ID: %d)", ccxtID, exchangeID)
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

// fetchAndCacheSymbols fetches symbols from CCXT service and populates cache (used during startup)
func (c *CollectorService) fetchAndCacheSymbols(exchangeID string) ([]string, error) {
	log.Printf("Fetching active markets for exchange: %s", exchangeID)

	// Add timeout context for better error handling
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	markets, err := c.ccxtService.FetchMarkets(ctx, exchangeID)
	if err != nil {
		// Log warning but don't fail startup for individual exchange errors
		log.Printf("Warning: Failed to fetch markets for %s (exchange may be unavailable): %v", exchangeID, err)
		return []string{}, nil // Return empty slice instead of error
	}

	// Since markets.Symbols is a slice of strings, we assume CCXT returns only active symbols
	// Filter out empty strings and invalid formats
	var activeSymbols []string
	for _, symbol := range markets.Symbols {
		if symbol == "" {
			continue
		}
		activeSymbols = append(activeSymbols, symbol)
	}

	// Cache the symbols
	c.symbolCache.Set(exchangeID, activeSymbols)

	// Update last refresh time
	c.symbolRefreshMu.Lock()
	c.lastSymbolRefresh[exchangeID] = time.Now()
	c.symbolRefreshMu.Unlock()

	log.Printf("Successfully fetched %d symbols for %s", len(activeSymbols), exchangeID)
	return activeSymbols, nil
}

// getMultiExchangeSymbols collects symbols from all exchanges and returns those that appear on multiple exchanges
// This function also populates the symbol cache to avoid double API calls during startup
func (c *CollectorService) getMultiExchangeSymbols(exchanges []string) (map[string]int, error) {
	symbolCount := make(map[string]int)
	minExchanges := 2 // Minimum number of exchanges a symbol must appear on

	log.Printf("Collecting symbols from %d exchanges for arbitrage filtering...", len(exchanges))

	// Collect active symbols from each exchange and populate cache
	for _, exchangeID := range exchanges {
		// Force fetch symbols to populate cache (bypass interval check during startup)
		activeSymbols, err := c.fetchAndCacheSymbols(exchangeID)
		if err != nil {
			log.Printf("Warning: Failed to fetch active symbols for %s: %v", exchangeID, err)
			continue
		}

		// Filter valid symbols for this exchange
		validSymbols := c.filterValidSymbols(activeSymbols)

		// Count occurrences of each symbol
		for _, symbol := range validSymbols {
			symbolCount[symbol]++
		}

		log.Printf("Found %d valid active symbols on %s (cached)", len(validSymbols), exchangeID)
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

// validateMarketData validates ticker data before saving to database
func (c *CollectorService) validateMarketData(ticker *models.MarketPrice, exchange, symbol string) error {
	// Check for zero or negative price
	if ticker.Price.IsZero() || ticker.Price.IsNegative() {
		return fmt.Errorf("invalid price: %s for %s on %s", ticker.Price, symbol, exchange)
	}

	// Check for extremely high prices (potential data corruption)
	maxPrice := decimal.NewFromFloat(10000000) // 10 million
	if ticker.Price.GreaterThan(maxPrice) {
		return fmt.Errorf("extremely high price: %s for %s on %s", ticker.Price, symbol, exchange)
	}

	// Check for negative volume
	if ticker.Volume.IsNegative() {
		return fmt.Errorf("negative volume: %s for %s on %s", ticker.Volume, symbol, exchange)
	}

	// Check for invalid timestamp
	timestamp := ticker.Timestamp
	now := time.Now()

	// Check if timestamp is in the future (more than 1 minute)
	if timestamp.After(now.Add(time.Minute)) {
		return fmt.Errorf("future timestamp: %s for %s on %s", timestamp, symbol, exchange)
	}

	// Check if timestamp is too old (more than 24 hours)
	if timestamp.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("old timestamp: %s for %s on %s", timestamp, symbol, exchange)
	}

	return nil
}

// PerformBackfillIfNeeded checks if backfill is needed and performs it
func (c *CollectorService) PerformBackfillIfNeeded() error {
	if !c.backfillConfig.Enabled {
		log.Println("Backfill is disabled in configuration, skipping historical data collection")
		return nil
	}

	log.Printf("Checking if historical data backfill is needed (Config: %dh backfill, %dh threshold, batch size: %d)...",
		c.backfillConfig.BackfillHours, c.backfillConfig.MinDataThresholdHours, c.backfillConfig.BatchSize)

	// Check if we have sufficient market data
	needsBackfill, err := c.checkIfBackfillNeeded()
	if err != nil {
		return fmt.Errorf("failed to check backfill requirement: %w", err)
	}

	if !needsBackfill {
		log.Printf("Sufficient market data available (threshold: %dh), skipping backfill", c.backfillConfig.MinDataThresholdHours)
		return nil
	}

	log.Printf("Insufficient market data detected, starting %dh historical backfill process...", c.backfillConfig.BackfillHours)
	startTime := time.Now()
	err = c.performHistoricalBackfill()
	if err != nil {
		return err
	}

	log.Printf("Historical backfill process completed in %v", time.Since(startTime))
	return nil
}

// checkIfBackfillNeeded determines if backfill is required based on available data
func (c *CollectorService) checkIfBackfillNeeded() (bool, error) {
	thresholdTime := time.Now().Add(-time.Duration(c.backfillConfig.MinDataThresholdHours) * time.Hour)

	// Check if we have recent market data within the threshold
	var count int
	err := c.db.Pool.QueryRow(c.ctx,
		"SELECT COUNT(*) FROM market_data WHERE timestamp >= $1",
		thresholdTime).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check market data count: %w", err)
	}

	log.Printf("Market data availability check: found %d records within last %dh (threshold: %v)",
		count, c.backfillConfig.MinDataThresholdHours, thresholdTime.Format("2006-01-02 15:04"))

	// If we have less than 100 records in the threshold period, we need backfill
	minRecordsThreshold := 100
	needsBackfill := count < minRecordsThreshold
	if needsBackfill {
		log.Printf("Backfill required: only %d records found (minimum: %d)", count, minRecordsThreshold)
	} else {
		log.Printf("Sufficient data available: %d records found (minimum: %d)", count, minRecordsThreshold)
	}

	return needsBackfill, nil
}

// performHistoricalBackfill collects historical data for active trading pairs
func (c *CollectorService) performHistoricalBackfill() error {
	log.Printf("Starting historical data backfill for %dh...", c.backfillConfig.BackfillHours)

	// Get all active exchanges and their symbols
	exchanges := c.ccxtService.GetSupportedExchanges()
	totalSymbols := 0
	successfulBackfills := 0
	successfulExchanges := 0
	failedExchanges := 0

	for i, exchangeID := range exchanges {
		log.Printf("Processing exchange %d/%d: %s", i+1, len(exchanges), exchangeID)

		// Get cached symbols for this exchange
		symbols, found := c.symbolCache.Get(exchangeID)
		if !found || len(symbols) == 0 {
			log.Printf("No cached symbols found for %s, skipping backfill", exchangeID)
			failedExchanges++
			continue
		}

		// Filter to valid symbols
		validSymbols := c.filterValidSymbols(symbols)
		if len(validSymbols) == 0 {
			log.Printf("No valid symbols found for %s, skipping backfill", exchangeID)
			failedExchanges++
			continue
		}

		// Limit symbols for backfill to prevent overwhelming the system
		maxSymbolsPerExchange := 20
		if len(validSymbols) > maxSymbolsPerExchange {
			validSymbols = validSymbols[:maxSymbolsPerExchange]
			log.Printf("Limited backfill symbols for %s to %d (from %d total)", exchangeID, maxSymbolsPerExchange, len(symbols))
		}

		totalSymbols += len(validSymbols)
		log.Printf("Starting backfill for %s with %d symbols", exchangeID, len(validSymbols))

		// Perform backfill for this exchange
		successCount, err := c.backfillExchangeData(exchangeID, validSymbols)
		if err != nil {
			log.Printf("Error during backfill for %s: %v", exchangeID, err)
			failedExchanges++
			continue
		}

		successfulBackfills += successCount
		successfulExchanges++
		log.Printf("Completed backfill for %s: %d/%d symbols successful", exchangeID, successCount, len(validSymbols))

		// Add delay between exchanges to prevent overwhelming APIs
		if i < len(exchanges)-1 { // Don't delay after the last exchange
			log.Printf("Waiting %dms before processing next exchange...", c.backfillConfig.DelayBetweenBatches)
			time.Sleep(time.Duration(c.backfillConfig.DelayBetweenBatches) * time.Millisecond)
		}
	}

	log.Printf("Historical backfill completed: %d/%d symbols successful across %d exchanges (%d successful, %d failed exchanges)",
		successfulBackfills, totalSymbols, len(exchanges), successfulExchanges, failedExchanges)
	return nil
}

// backfillExchangeData performs backfill for a specific exchange
func (c *CollectorService) backfillExchangeData(exchangeID string, symbols []string) (int, error) {
	successCount := 0
	failedCount := 0
	backfillStartTime := time.Now().Add(-time.Duration(c.backfillConfig.BackfillHours) * time.Hour)
	totalBatches := (len(symbols) + c.backfillConfig.BatchSize - 1) / c.backfillConfig.BatchSize

	log.Printf("Starting backfill for %s: %d symbols in %d batches (period: %v to %v)",
		exchangeID, len(symbols), totalBatches,
		backfillStartTime.Format("2006-01-02 15:04"), time.Now().Format("2006-01-02 15:04"))

	// Process symbols in batches
	for i := 0; i < len(symbols); i += c.backfillConfig.BatchSize {
		end := i + c.backfillConfig.BatchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]
		currentBatch := (i / c.backfillConfig.BatchSize) + 1
		log.Printf("Processing batch %d/%d for %s (%d symbols: %v)",
			currentBatch, totalBatches, exchangeID, len(batch), batch)

		batchStartTime := time.Now()
		batchSuccessCount := 0
		batchFailedCount := 0

		// Process each symbol in the batch
		for _, symbol := range batch {
			select {
			case <-c.ctx.Done():
				return successCount, c.ctx.Err()
			default:
			}

			// Check if symbol is blacklisted before processing
			if isBlacklisted, reason := c.blacklistCache.IsBlacklisted(exchangeID, symbol); isBlacklisted {
				log.Printf("Skipping blacklisted symbol during backfill: %s:%s (reason: %s)", exchangeID, symbol, reason)
				batchFailedCount++
				failedCount++
				continue
			}

			// Generate synthetic historical data points
			if err := c.generateHistoricalDataPoints(exchangeID, symbol, backfillStartTime); err != nil {
				log.Printf("Failed to backfill %s:%s: %v", exchangeID, symbol, err)
				batchFailedCount++
				failedCount++
				continue
			}

			batchSuccessCount++
			successCount++

			// Add small delay between symbols
			time.Sleep(time.Duration(c.backfillConfig.DelayBetweenBatches) * time.Millisecond)
		}

		log.Printf("Batch %d/%d completed in %v (%d successful, %d failed)",
			currentBatch, totalBatches, time.Since(batchStartTime), batchSuccessCount, batchFailedCount)

		// Add delay between batches
		if end < len(symbols) {
			log.Printf("Waiting 1s before next batch...")
			time.Sleep(1 * time.Second)
		}
	}

	log.Printf("Backfill completed for %s: %d successful, %d failed (total: %d symbols)",
		exchangeID, successCount, failedCount, len(symbols))
	return successCount, nil
}

// generateHistoricalDataPoints creates synthetic historical data points for backfill
func (c *CollectorService) generateHistoricalDataPoints(exchangeID, symbol string, startTime time.Time) error {
	// Get current ticker data as baseline
	ticker, err := c.ccxtService.FetchSingleTicker(c.ctx, exchangeID, symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch current ticker for baseline: %w", err)
	}

	// Get exchange and trading pair IDs
	exchangeDBID, err := c.getOrCreateExchange(exchangeID)
	if err != nil {
		return fmt.Errorf("failed to get exchange ID: %w", err)
	}

	tradingPairID, err := c.getOrCreateTradingPair(exchangeDBID, symbol)
	if err != nil {
		return fmt.Errorf("failed to get trading pair ID: %w", err)
	}

	// Generate data points every 30 minutes for the backfill period
	interval := 30 * time.Minute
	currentTime := startTime
	basePrice := ticker.Price
	baseVolume := ticker.Volume
	dataPointsGenerated := 0

	log.Printf("Generating historical data for %s:%s from %v (baseline: price=%s, volume=%s)",
		exchangeID, symbol, startTime.Format("2006-01-02 15:04"), basePrice, baseVolume)

	for currentTime.Before(time.Now().Add(-interval)) {
		// Add some realistic price variation (±2%)
		variation := decimal.NewFromFloat(0.98 + (0.04 * float64(time.Now().UnixNano()%100) / 100))
		historicalPrice := basePrice.Mul(variation)

		// Add some volume variation (±50%)
		volumeVariation := decimal.NewFromFloat(0.5 + (1.0 * float64(time.Now().UnixNano()%100) / 100))
		historicalVolume := baseVolume.Mul(volumeVariation)

		// Insert historical data point
		_, err := c.db.Pool.Exec(c.ctx,
			`INSERT INTO market_data (exchange_id, trading_pair_id, last_price, volume_24h, timestamp, created_at) 
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			exchangeDBID, tradingPairID, historicalPrice, historicalVolume, currentTime, currentTime)
		if err != nil {
			return fmt.Errorf("failed to insert historical data: %w", err)
		}

		dataPointsGenerated++
		currentTime = currentTime.Add(interval)
	}

	log.Printf("Generated %d historical data points for %s:%s (30min intervals)",
		dataPointsGenerated, exchangeID, symbol)
	return nil
}
