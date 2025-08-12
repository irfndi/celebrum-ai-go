package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/api/handlers"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	futuresHandlers "github.com/irfndi/celebrum-ai-go/internal/handlers"
	"github.com/irfndi/celebrum-ai-go/internal/services"
	"github.com/irfndi/celebrum-ai-go/pkg/ccxt"
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

func SetupRoutes(router *gin.Engine, db *database.PostgresDB, redis *database.RedisClient, ccxtService ccxt.CCXTService, collectorService *services.CollectorService, cleanupService *services.CleanupService, telegramConfig *config.TelegramConfig) {
	// Initialize health handler
	healthHandler := handlers.NewHealthHandler(db, redis, ccxtService.GetServiceURL())

	// Health check endpoints
	router.GET("/health", gin.WrapF(healthHandler.HealthCheck))
	router.HEAD("/health", gin.WrapF(healthHandler.HealthCheck))
	router.GET("/ready", gin.WrapF(healthHandler.ReadinessCheck))
	router.GET("/live", gin.WrapF(healthHandler.LivenessCheck))

	// Initialize notification service
	notificationService := services.NewNotificationService(db, telegramConfig.BotToken)

	// Initialize handlers
	marketHandler := handlers.NewMarketHandler(db, ccxtService, collectorService)
	arbitrageHandler := handlers.NewArbitrageHandler(db, ccxtService, notificationService)
	telegramHandler := handlers.NewTelegramHandler(db, telegramConfig, arbitrageHandler)
	analysisHandler := handlers.NewAnalysisHandler(db, ccxtService)
	userHandler := handlers.NewUserHandler(db)
	cleanupHandler := handlers.NewCleanupHandler(cleanupService)
	exchangeHandler := handlers.NewExchangeHandler(ccxtService, collectorService)

	// Initialize futures arbitrage handler with error handling
	var futuresArbitrageHandler *futuresHandlers.FuturesArbitrageHandler
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during futures arbitrage handler initialization: %v\n", r)
				futuresArbitrageHandler = nil
			}
		}()
		futuresArbitrageHandler = futuresHandlers.NewFuturesArbitrageHandler(db.Pool)
		fmt.Println("Futures arbitrage handler initialized successfully")
	}()

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Market data routes
		market := v1.Group("/market")
		{
			market.GET("/prices", marketHandler.GetMarketPrices)
			market.GET("/ticker/:exchange/:symbol", marketHandler.GetTicker)
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
			users.GET("/profile", userHandler.GetUserProfile)
		}

		// Alerts management
		alerts := v1.Group("/alerts")
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
			exchanges.GET("/config", exchangeHandler.GetExchangeConfig)
			exchanges.GET("/supported", exchangeHandler.GetSupportedExchanges)
			exchanges.GET("/workers/status", exchangeHandler.GetWorkerStatus)
			exchanges.POST("/refresh", exchangeHandler.RefreshExchanges)
			exchanges.POST("/add/:exchange", exchangeHandler.AddExchange)
			exchanges.POST("/blacklist/:exchange", exchangeHandler.AddExchangeToBlacklist)
			exchanges.DELETE("/blacklist/:exchange", exchangeHandler.RemoveExchangeFromBlacklist)
			exchanges.POST("/workers/:exchange/restart", exchangeHandler.RestartWorker)
		}
	}
}

// Placeholder handlers - to be implemented

// Arbitrage handlers are now implemented in handlers/arbitrage.go
// Technical analysis handlers are now implemented in handlers/analysis.go

func getUserAlerts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get user alerts endpoint - to be implemented"})
}

func createAlert(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Create alert endpoint - to be implemented"})
}

func updateAlert(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update alert endpoint - to be implemented"})
}

func deleteAlert(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete alert endpoint - to be implemented"})
}
