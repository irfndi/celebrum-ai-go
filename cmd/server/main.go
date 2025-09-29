package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfandi/celebrum-ai-go/internal/api"
	"github.com/irfandi/celebrum-ai-go/internal/cache"
	"github.com/irfandi/celebrum-ai-go/internal/ccxt"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	"github.com/irfandi/celebrum-ai-go/internal/logging"
	"github.com/irfandi/celebrum-ai-go/internal/middleware"
	"github.com/irfandi/celebrum-ai-go/internal/services"
	"github.com/irfandi/celebrum-ai-go/internal/telemetry"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize telemetry first
	if err := telemetry.InitTelemetry(telemetry.TelemetryConfig{
		Enabled:        cfg.Telemetry.Enabled,
		OTLPEndpoint:   cfg.Telemetry.OTLPEndpoint,
		ServiceName:    cfg.Telemetry.ServiceName,
		ServiceVersion: cfg.Telemetry.ServiceVersion,
		LogLevel:       cfg.Telemetry.LogLevel,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize telemetry: %v\n", err)
		// Don't exit on telemetry failure, continue with reduced observability
	}
	defer func() {
		if err := telemetry.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shutdown telemetry: %v\n", err)
		}
	}()

	// Initialize services with OTLP logger
	var logger *logging.OTLPLogger
	if cfg.Telemetry.Enabled {
		logger = telemetry.GetLogger()
		if logger == nil {
			// Fallback to standard logger if OTLP logger is not available
			otlpLogger, _ := logging.NewOTLPLogger(logging.OTLPConfig{
				Enabled: false, // Use disabled OTLP logger that falls back to stdout
			})
			logger = otlpLogger
		}
	} else {
		// Use standard logger when telemetry is disabled
		otlpLogger, _ := logging.NewOTLPLogger(logging.OTLPConfig{
			Enabled: false, // Use disabled OTLP logger that falls back to stdout
		})
		logger = otlpLogger
	}

	// Create logrus logger for services that require it (backward compatibility)
	logrusLogger := logrus.New()
	logrusLogger.SetLevel(logging.ParseLogrusLevel(cfg.LogLevel))
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})

	// Initialize database
	db, err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize error recovery manager for Redis connection
	errorRecoveryManager := services.NewErrorRecoveryManager(logrusLogger)

	// Register retry policies for Redis operations
	retryPolicies := services.DefaultRetryPolicies()
	for name, policy := range retryPolicies {
		errorRecoveryManager.RegisterRetryPolicy(name, policy)
	}

	// Initialize Redis with retry mechanism
	redisClient, err := database.NewRedisConnectionWithRetry(cfg.Redis, errorRecoveryManager)
	if err != nil {
		logrusLogger.WithError(err).Error("Failed to connect to Redis - continuing without cache")
		// Don't fail startup on Redis connection issues, continue without cache
		redisClient = nil
	} else {
		defer redisClient.Close()
	}

	// Helper function to safely get Redis client
	getRedisClient := func() *redis.Client {
		if redisClient != nil {
			return redisClient.Client
		}
		return nil
	}

	// Initialize blacklist cache with database persistence
	blacklistRepo := database.NewBlacklistRepository(db.Pool)
	var blacklistCache cache.BlacklistCache
	if redisClient != nil {
		blacklistCache = cache.NewRedisBlacklistCache(redisClient.Client, blacklistRepo)
	} else {
		// Fallback to in-memory cache if Redis is not available
		blacklistCache = cache.NewInMemoryBlacklistCache()
	}

	// Initialize CCXT service with blacklist cache
	ccxtService := ccxt.NewService(&cfg.CCXT, logrusLogger, blacklistCache)

	// Initialize JWT authentication middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Auth.JWTSecret)

	// Initialize cache analytics service
	cacheAnalyticsService := services.NewCacheAnalyticsService(getRedisClient())

	// Start periodic reporting of cache stats
	ctx := context.Background()
	cacheAnalyticsService.StartPeriodicReporting(ctx, 5*time.Minute)

	// Initialize and perform cache warming
	cacheWarmingService := services.NewCacheWarmingService(getRedisClient(), ccxtService, db)
	if err := cacheWarmingService.WarmCache(ctx); err != nil {
		logrusLogger.WithError(err).Warn("Cache warming failed")
		// Don't fail startup if cache warming fails, just log the warning
	}

	// Initialize collector service
	collectorService := services.NewCollectorService(db, ccxtService, cfg, getRedisClient(), blacklistCache)
	if err := collectorService.Start(); err != nil {
		logrusLogger.WithError(err).Fatal("Failed to start collector service")
	}
	defer collectorService.Stop()

	// Initialize support services for futures arbitrage and cleanup
	resourceManager := services.NewResourceManager(logrusLogger)
	defer resourceManager.Shutdown()
	performanceMonitor := services.NewPerformanceMonitor(logrusLogger, getRedisClient(), ctx)
	defer performanceMonitor.Stop()

	// Start historical data backfill in background if needed
	go func() {
		logrusLogger.Info("Checking for historical data backfill requirements")
		if err := collectorService.PerformBackfillIfNeeded(); err != nil {
			logrusLogger.WithError(err).Warn("Backfill failed")
		} else {
			logrusLogger.Info("Historical data backfill check completed successfully")
		}
	}()

	// Initialize futures arbitrage calculator
	arbitrageCalculator := services.NewFuturesArbitrageCalculator()

	// Initialize regular arbitrage service
	arbitrageService := services.NewArbitrageService(db, cfg, arbitrageCalculator)
	if err := arbitrageService.Start(); err != nil {
		logrusLogger.WithError(err).Fatal("Failed to start arbitrage service")
	}
	defer arbitrageService.Stop()

	// Initialize futures arbitrage service
	futuresArbitrageService := services.NewFuturesArbitrageService(db, getRedisClient(), cfg, errorRecoveryManager, resourceManager, performanceMonitor)
	if err := futuresArbitrageService.Start(); err != nil {
		logrusLogger.WithError(err).Fatal("Failed to start futures arbitrage service")
	}
	defer futuresArbitrageService.Stop()

	// Initialize signal aggregator service
	signalAggregator := services.NewSignalAggregator(cfg, db, logrusLogger)

	// Initialize technical analysis service
	technicalAnalysisService := services.NewTechnicalAnalysisService(
		cfg,
		db,
		logrusLogger,
		errorRecoveryManager,
		resourceManager,
		performanceMonitor,
	)

	// Initialize signal quality scorer
	signalQualityScorer := services.NewSignalQualityScorer(cfg, db, logrusLogger)

	// Initialize notification service
	notificationService := services.NewNotificationService(db, redisClient, cfg.Telegram.BotToken)

	// Initialize circuit breaker for signal processing
	signalProcessorCircuitBreaker := services.NewCircuitBreaker(
		"signal_processor",
		services.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			ResetTimeout:     60 * time.Second,
			MaxRequests:      10,
		},
		logrusLogger,
	)

	// Initialize signal processor
	signalProcessor := services.NewSignalProcessor(
		db.Pool,
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		signalAggregator,
		signalQualityScorer,
		technicalAnalysisService,
		notificationService,
		collectorService,
		signalProcessorCircuitBreaker,
	)

	// Start signal processor
	if err := signalProcessor.Start(); err != nil {
		logrusLogger.WithError(err).Fatal("Failed to start signal processor")
	}
	defer func() {
		if err := signalProcessor.Stop(); err != nil {
			logrusLogger.WithError(err).Error("Failed to stop signal processor")
		}
	}()

	// Initialize cleanup service
	cleanupService := services.NewCleanupService(db, errorRecoveryManager, resourceManager, performanceMonitor)

	// Start cleanup service with configuration
	cleanupConfig := cfg.Cleanup
	go cleanupService.Start(cleanupConfig)
	defer cleanupService.Stop()

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("github.com/irfandi/celebrum-ai-go"))

	// Setup routes
	api.SetupRoutes(router, db, redisClient, ccxtService, collectorService, cleanupService, cacheAnalyticsService, signalAggregator, &cfg.Telegram, authMiddleware)

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
		logger.Logger().Info("Application startup",
			"service", "github.com/irfandi/celebrum-ai-go",
			"version", "1.0.0",
			"port", cfg.Server.Port,
			"event", "startup",
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrusLogger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Logger().Info("Application shutdown",
		"service", "github.com/irfandi/celebrum-ai-go",
		"event", "shutdown",
		"reason", "signal received",
	)

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrusLogger.WithError(err).Fatal("Server forced to shutdown")
	}

	logrusLogger.Info("Server exited gracefully")
	return nil
}
