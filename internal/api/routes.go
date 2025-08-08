package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/api/handlers"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
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

func SetupRoutes(router *gin.Engine, db *database.PostgresDB, redis *database.RedisClient, ccxtService ccxt.CCXTService, collectorService *services.CollectorService, telegramConfig *config.TelegramConfig) {
	// Health check endpoint - support both GET and HEAD for Docker health checks
	router.GET("/health", healthCheck(db, redis))
	router.HEAD("/health", healthCheck(db, redis))

	// Initialize handlers
	marketHandler := handlers.NewMarketHandler(db, ccxtService, collectorService)
	telegramHandler := handlers.NewTelegramHandler(db, telegramConfig)
	arbitrageHandler := handlers.NewArbitrageHandler(db, ccxtService)
	analysisHandler := handlers.NewAnalysisHandler(db, ccxtService)
	userHandler := handlers.NewUserHandler(db)

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
	}
}

func healthCheck(db *database.PostgresDB, redis *database.RedisClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := HealthResponse{
			Status:    "ok",
			Timestamp: time.Now(),
			Version:   "1.0.0",
			Services: Services{
				Database: "ok",
				Redis:    "ok",
			},
		}

		// Check database health
		if err := db.HealthCheck(c.Request.Context()); err != nil {
			response.Services.Database = "error"
			response.Status = "degraded"
		}

		// Check Redis health
		if err := redis.HealthCheck(c.Request.Context()); err != nil {
			response.Services.Redis = "error"
			response.Status = "degraded"
		}

		// Always return 200 OK for Docker health checks
		// The application is running even if dependencies are temporarily unavailable
		c.JSON(http.StatusOK, response)
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
