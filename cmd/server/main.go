package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/api"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redis, err := database.NewRedisConnection(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Initialize CCXT service
	ccxtService := ccxt.NewService(&cfg.CCXT)

	// Initialize collector service
	collectorService := services.NewCollectorService(db, ccxtService, cfg)
	if err := collectorService.Start(); err != nil {
		log.Fatalf("Failed to start collector service: %v", err)
	}
	defer collectorService.Stop()

	// Perform historical data backfill if needed
	log.Println("Checking for historical data backfill requirements...")
	if err := collectorService.PerformBackfillIfNeeded(); err != nil {
		log.Printf("Warning: Backfill failed: %v", err)
		// Don't fail startup if backfill fails, just log the warning
	} else {
		log.Println("Historical data backfill check completed successfully")
	}

	// Initialize futures arbitrage service
	futuresArbitrageService := services.NewFuturesArbitrageService(db, cfg)
	if err := futuresArbitrageService.Start(); err != nil {
		log.Fatalf("Failed to start futures arbitrage service: %v", err)
	}
	defer futuresArbitrageService.Stop()

	// Initialize cleanup service
	cleanupService := services.NewCleanupService(db)

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
	api.SetupRoutes(router, db, redis, ccxtService, collectorService, cleanupService, &cfg.Telegram)

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
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
