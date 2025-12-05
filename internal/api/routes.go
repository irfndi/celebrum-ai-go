package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfandi/celebrum-ai-go/internal/api/handlers"
	"github.com/irfandi/celebrum-ai-go/internal/ccxt"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	futuresHandlers "github.com/irfandi/celebrum-ai-go/internal/handlers"
	"github.com/irfandi/celebrum-ai-go/internal/middleware"
	"github.com/irfandi/celebrum-ai-go/internal/services"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Services  Services  `json:"services"`
}

type Services struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
}

func SetupRoutes(router *gin.Engine, db *database.PostgresDB, redis *database.RedisClient, ccxtService ccxt.CCXTService, collectorService *services.CollectorService, cleanupService *services.CleanupService, cacheAnalyticsService *services.CacheAnalyticsService, signalAggregator *services.SignalAggregator, telegramConfig *config.TelegramConfig, authMiddleware *middleware.AuthMiddleware) {
	// Initialize admin middleware
	adminMiddleware := middleware.NewAdminMiddleware()

	// Initialize health handler
	healthHandler := handlers.NewHealthHandler(db, redis, ccxtService.GetServiceURL(), cacheAnalyticsService)

	// Health check endpoints with telemetry
	healthGroup := router.Group("/")
	healthGroup.Use(middleware.HealthCheckTelemetryMiddleware())
	{
		healthGroup.GET("/health", gin.WrapF(healthHandler.HealthCheck))
		healthGroup.HEAD("/health", gin.WrapF(healthHandler.HealthCheck))
		healthGroup.GET("/ready", gin.WrapF(healthHandler.ReadinessCheck))
		healthGroup.GET("/live", gin.WrapF(healthHandler.LivenessCheck))
	}

	// Initialize notification service with Redis caching
	notificationService := services.NewNotificationService(db, redis, telegramConfig.BotToken)

	// Initialize handlers
	marketHandler := handlers.NewMarketHandler(db, ccxtService, collectorService, redis, cacheAnalyticsService)
	arbitrageHandler := handlers.NewArbitrageHandler(db, ccxtService, notificationService, redis.Client)

	// Initialize telegram handler with debug logging
	fmt.Printf("DEBUG: About to initialize Telegram handler with config: %+v\n", telegramConfig)
	telegramHandler := handlers.NewTelegramHandler(db, telegramConfig, arbitrageHandler, signalAggregator, redis.Client)
	fmt.Printf("DEBUG: Telegram handler initialized: %+v\n", telegramHandler != nil)
	analysisHandler := handlers.NewAnalysisHandler(db, ccxtService)
	userHandler := handlers.NewUserHandler(db, redis.Client)
	cleanupHandler := handlers.NewCleanupHandler(cleanupService)
	exchangeHandler := handlers.NewExchangeHandler(ccxtService, collectorService, redis.Client)
	cacheHandler := handlers.NewCacheHandler(cacheAnalyticsService)

	// Initialize futures arbitrage handler with error handling
	var futuresArbitrageHandler *futuresHandlers.FuturesArbitrageHandler
	if db != nil && db.Pool != nil {
		// Safely initialize the handler
		// Note: NewFuturesArbitrageHandler doesn't return an error in its current signature,
		// but we check for db.Pool to prevent panics inside it.
		futuresArbitrageHandler = futuresHandlers.NewFuturesArbitrageHandler(db.Pool)
		fmt.Println("Futures arbitrage handler initialized successfully")
	} else {
		fmt.Println("Database not available for futures arbitrage handler initialization")
	}

	// API v1 routes with telemetry
	v1 := router.Group("/api/v1")
	v1.Use(middleware.TelemetryMiddleware())
	{
		// Market data routes
		market := v1.Group("/market")
		{
			market.GET("/prices", marketHandler.GetMarketPrices)
			market.GET("/ticker/:exchange/:symbol", marketHandler.GetTicker)
			market.GET("/tickers/:exchange", marketHandler.GetBulkTickers)
			market.GET("/orderbook/:exchange/:symbol", marketHandler.GetOrderBook)
			market.GET("/workers/status", marketHandler.GetWorkerStatus)
		}

		// Arbitrage routes
		arbitrage := v1.Group("/arbitrage")
		{
			arbitrage.GET("/opportunities", arbitrageHandler.GetArbitrageOpportunities)
			arbitrage.GET("/history", arbitrageHandler.GetArbitrageHistory)
			// Funding rate arbitrage
			arbitrage.GET("/funding", arbitrageHandler.GetFundingRateArbitrage)
			arbitrage.GET("/funding-rates/:exchange", arbitrageHandler.GetFundingRates)
		}

		// Futures arbitrage routes (only if handler initialized successfully)
		if futuresArbitrageHandler != nil {
			futuresArbitrage := v1.Group("/futures-arbitrage")
			{
				futuresArbitrage.GET("/opportunities", futuresArbitrageHandler.GetFuturesArbitrageOpportunities)
				futuresArbitrage.POST("/calculate", futuresArbitrageHandler.CalculateFuturesArbitrage)
				futuresArbitrage.GET("/strategy/:id", futuresArbitrageHandler.GetFuturesArbitrageStrategy)
				futuresArbitrage.GET("/market-summary", futuresArbitrageHandler.GetFuturesMarketSummary)
				futuresArbitrage.POST("/position-sizing", futuresArbitrageHandler.GetPositionSizingRecommendation)
			}
			fmt.Println("Futures arbitrage routes registered successfully")
		} else {
			fmt.Println("Skipping futures arbitrage routes due to handler initialization failure")
		}

		// Technical analysis routes
		analysis := v1.Group("/analysis")
		{
			analysis.GET("/indicators", analysisHandler.GetTechnicalIndicators)
			analysis.GET("/signals", analysisHandler.GetTradingSignals)
		}

		// Telegram webhook
		telegram := v1.Group("/telegram")
		{
			telegram.POST("/webhook", telegramHandler.HandleWebhook)
		}

		// User management
		users := v1.Group("/users")
		{
			users.POST("/register", userHandler.RegisterUser)
			users.POST("/login", userHandler.LoginUser)
			users.GET("/profile", authMiddleware.RequireAuth(), userHandler.GetUserProfile)
		}

		// Alerts management
		alerts := v1.Group("/alerts")
		alerts.Use(authMiddleware.RequireAuth())
		{
			alerts.GET("/", getUserAlerts)
			alerts.POST("/", createAlert)
			alerts.PUT("/:id", updateAlert)
			alerts.DELETE("/:id", deleteAlert)
		}

		// Data management
		data := v1.Group("/data")
		{
			data.GET("/stats", cleanupHandler.GetDataStats)
			data.POST("/cleanup", cleanupHandler.TriggerCleanup)
		}

		// Exchange management
		exchanges := v1.Group("/exchanges")
		{
			// Public endpoints (no admin auth required)
			exchanges.GET("/config", exchangeHandler.GetExchangeConfig)
			exchanges.GET("/supported", exchangeHandler.GetSupportedExchanges)
			exchanges.GET("/workers/status", exchangeHandler.GetWorkerStatus)

			// Admin-only endpoints (require admin authentication)
			adminExchanges := exchanges.Group("/")
			adminExchanges.Use(adminMiddleware.RequireAdminAuth())
			{
				adminExchanges.POST("/refresh", exchangeHandler.RefreshExchanges)
				adminExchanges.POST("/add/:exchange", exchangeHandler.AddExchange)
				adminExchanges.POST("/blacklist/:exchange", exchangeHandler.AddExchangeToBlacklist)
				adminExchanges.DELETE("/blacklist/:exchange", exchangeHandler.RemoveExchangeFromBlacklist)
				adminExchanges.POST("/workers/:exchange/restart", exchangeHandler.RestartWorker)
			}
		}

		// Cache monitoring and analytics
		cache := v1.Group("/cache")
		{
			cache.GET("/stats", cacheHandler.GetCacheStats)
			cache.GET("/stats/:category", cacheHandler.GetCacheStatsByCategory)
			cache.GET("/metrics", cacheHandler.GetCacheMetrics)
			cache.POST("/stats/reset", cacheHandler.ResetCacheStats)
			cache.POST("/hit", cacheHandler.RecordCacheHit)
			cache.POST("/miss", cacheHandler.RecordCacheMiss)
		}
	}
}

// Placeholder handlers - to be implemented

// Arbitrage handlers are now implemented in handlers/arbitrage.go
// Technical analysis handlers are now implemented in handlers/analysis.go

func getUserAlerts(c *gin.Context) {
	// TODO: Implement actual database retrieval
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    []string{}, // Return empty list for now
		"message": "Alerts retrieval implemented (mock)",
	})
}

func createAlert(c *gin.Context) {
	// TODO: Implement actual database creation
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Alert created successfully (mock)",
	})
}

func updateAlert(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement actual database update
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Alert %s updated successfully (mock)", id),
	})
}

func deleteAlert(c *gin.Context) {
	id := c.Param("id")
	// TODO: Implement actual database deletion
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Alert %s deleted successfully (mock)", id),
	})
}
