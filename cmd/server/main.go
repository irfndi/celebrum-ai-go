package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/api"
	"github.com/irfndi/celebrum-ai-go/internal/cache"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/logging"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize standardized logger
	logger := logging.NewStandardLogger(cfg.LogLevel, cfg.Environment)
	appLogger := logger.WithService("celebrum-ai-go")

	// Initialize database
	db, err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Initialize Redis
	redis, err := database.NewRedisConnection(cfg.Redis)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to Redis")
	}
	defer redis.Close()

	// Initialize blacklist cache with database persistence
	blacklistRepo := database.NewBlacklistRepository(db.Pool)
	blacklistCache := cache.NewRedisBlacklistCache(redis.Client, blacklistRepo)

	// Initialize CCXT service with blacklist cache
	ccxtService := ccxt.NewService(&cfg.CCXT, logger.Logger, blacklistCache)

	// Initialize cache analytics service
	cacheAnalyticsService := services.NewCacheAnalyticsService(redis.Client)

	// Start periodic reporting of cache stats
	ctx := context.Background()
	cacheAnalyticsService.StartPeriodicReporting(ctx, 5*time.Minute)

	// Initialize and perform cache warming
	cacheWarmingService := services.NewCacheWarmingService(redis.Client, ccxtService, db)
	if err := cacheWarmingService.WarmCache(ctx); err != nil {
		appLogger.WithError(err).Warn("Cache warming failed")
		// Don't fail startup if cache warming fails, just log the warning
	}

	// Initialize collector service
	collectorService := services.NewCollectorService(db, ccxtService, cfg, redis.Client, blacklistCache)
	if err := collectorService.Start(); err != nil {
		appLogger.WithError(err).Fatal("Failed to start collector service")
	}
	defer collectorService.Stop()

	// Perform historical data backfill if needed
	appLogger.Info("Checking for historical data backfill requirements")
	if err := collectorService.PerformBackfillIfNeeded(); err != nil {
		appLogger.WithError(err).Warn("Backfill failed")
		// Don't fail startup if backfill fails, just log the warning
	} else {
		appLogger.Info("Historical data backfill check completed successfully")
	}

	// Initialize support services for futures arbitrage
	resourceManager := services.NewResourceManager(logger.Logger)
	errorRecoveryManager := services.NewErrorRecoveryManager(logger.Logger)
	performanceMonitor := services.NewPerformanceMonitor(logger.Logger, redis.Client, ctx)

	// Initialize futures arbitrage service
	futuresArbitrageService := services.NewFuturesArbitrageService(db, redis.Client, cfg, errorRecoveryManager, resourceManager, performanceMonitor)
	if err := futuresArbitrageService.Start(); err != nil {
		appLogger.WithError(err).Fatal("Failed to start futures arbitrage service")
	}
	defer futuresArbitrageService.Stop()

	// Initialize cleanup service
	cleanupService := services.NewCleanupService(db, errorRecoveryManager, resourceManager, performanceMonitor)

	// Start cleanup service with configuration
	cleanupConfig := services.CleanupConfig{
		MarketDataRetentionHours:  cfg.Cleanup.MarketData.RetentionHours,
		MarketDataDeletionHours:   cfg.Cleanup.MarketData.DeletionHours,
		FundingRateRetentionHours: cfg.Cleanup.FundingRates.RetentionHours,
		FundingRateDeletionHours:  cfg.Cleanup.FundingRates.DeletionHours,
		ArbitrageRetentionHours:   cfg.Cleanup.ArbitrageOpportunities.RetentionHours,
		CleanupIntervalMinutes:    cfg.Cleanup.IntervalMinutes,
		EnableSmartCleanup:        cfg.Cleanup.EnableSmartCleanup,
	}
	go cleanupService.Start(cleanupConfig)
	defer cleanupService.Stop()

	// Setup Gin router
	router := gin.Default()

	// Setup routes
	api.SetupRoutes(router, db, redis, ccxtService, collectorService, cleanupService, cacheAnalyticsService, &cfg.Telegram)

	// Create HTTP server with security timeouts
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.LogStartup("celebrum-ai-go", "1.0.0", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.LogShutdown("celebrum-ai-go", "signal received")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.WithError(err).Fatal("Server forced to shutdown")
	}

	appLogger.Info("Server exited gracefully")
}
