package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
	"gorm.io/gorm"
)

// CollectorConfig holds configuration for the collector service
type CollectorConfig struct {
	IntervalSeconds int `mapstructure:"interval_seconds"`
	MaxErrors       int `mapstructure:"max_errors"`
}

// CollectorService handles market data collection from exchanges
type CollectorService struct {
	db              *gorm.DB
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
func NewCollectorService(db *gorm.DB, ccxtService ccxt.CCXTService, cfg *config.Config) *CollectorService {
	ctx, cancel := context.WithCancel(context.Background())
	collectorConfig := CollectorConfig{
		IntervalSeconds: 60, // Default 60 seconds
		MaxErrors:       5,  // Default 5 max errors
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

	// Create workers for each exchange
	for _, exchangeID := range exchanges {
		if err := c.createWorker(exchangeID); err != nil {
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
func (c *CollectorService) createWorker(exchangeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get trading pairs for this exchange
	var tradingPairs []models.TradingPair
	if err := c.db.Where("exchange_id = ? AND is_active = ?", exchangeID, true).Find(&tradingPairs).Error; err != nil {
		return fmt.Errorf("failed to get trading pairs for exchange %s: %w", exchangeID, err)
	}

	if len(tradingPairs) == 0 {
		log.Printf("No active trading pairs found for exchange %s, skipping worker creation", exchangeID)
		return nil
	}

	// Extract symbols
	symbols := make([]string, len(tradingPairs))
	for i, pair := range tradingPairs {
		symbols[i] = pair.Symbol
	}

	// Create worker
	worker := &Worker{
		Exchange:  exchangeID,
		Symbols:   symbols,
		Interval:  time.Duration(c.collectorConfig.IntervalSeconds) * time.Second,
		MaxErrors: c.collectorConfig.MaxErrors,
		IsRunning: true,
	}

	c.workers[exchangeID] = worker

	// Start worker goroutine
	c.wg.Add(1)
	go c.runWorker(worker)

	log.Printf("Created worker for exchange %s with %d symbols", exchangeID, len(symbols))
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
		}
	}
}

// collectMarketData collects market data for all symbols of a specific exchange
func (c *CollectorService) collectMarketData(worker *Worker) error {
	log.Printf("Collecting market data for exchange %s (%d symbols)", worker.Exchange, len(worker.Symbols))

	// Collect ticker data for all symbols
	for _, symbol := range worker.Symbols {
		if err := c.collectTickerData(worker.Exchange, symbol); err != nil {
			log.Printf("Failed to collect ticker data for %s:%s: %v", worker.Exchange, symbol, err)
			// Continue with other symbols even if one fails
			continue
		}
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

	// Find exchange
	var exchangeModel models.Exchange
	if err := c.db.Where("name = ?", exchange).First(&exchangeModel).Error; err != nil {
		return fmt.Errorf("failed to find exchange: %w", err)
	}

	// Get trading pair ID
	var tradingPair models.TradingPair
	if err := c.db.Where("exchange_id = ? AND symbol = ?", exchangeModel.ID, symbol).First(&tradingPair).Error; err != nil {
		return fmt.Errorf("failed to find trading pair: %w", err)
	}

	// Create market data record
	marketData := &models.MarketData{
		ExchangeID:    exchangeModel.ID,
		TradingPairID: int(tradingPair.ID.ID()), // Convert UUID to int
		Timestamp:     ticker.Timestamp,
		Price:         ticker.Price,
		Volume:        ticker.Volume,
		CreatedAt:     time.Now(),
	}

	// Save to database
	if err := c.db.Create(marketData).Error; err != nil {
		return fmt.Errorf("failed to save market data: %w", err)
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
