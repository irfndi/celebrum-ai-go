package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/api/handlers"
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

func SetupRoutes(router *gin.Engine, db *database.PostgresDB, redis *database.RedisClient, ccxtService ccxt.CCXTService, collectorService *services.CollectorService) {
	// Health check endpoint
	router.GET("/health", healthCheck(db, redis))

	// Initialize handlers
	marketHandler := handlers.NewMarketHandler(db, ccxtService, collectorService)

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
			arbitrage.GET("/opportunities", getArbitrageOpportunities)
			arbitrage.GET("/history", getArbitrageHistory)
		}

		// Technical analysis routes
		analysis := v1.Group("/analysis")
		{
			analysis.GET("/indicators", getTechnicalIndicators)
			analysis.GET("/signals", getTradingSignals)
		}

		// Telegram webhook
		telegram := v1.Group("/telegram")
		{
			telegram.POST("/webhook", handleTelegramWebhook)
		}

		// User management (future)
		users := v1.Group("/users")
		{
			users.POST("/register", registerUser)
			users.POST("/login", loginUser)
			users.GET("/profile", getUserProfile)
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

		statusCode := http.StatusOK
		if response.Status == "degraded" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, response)
	}
}

// Placeholder handlers - to be implemented

func getArbitrageOpportunities(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Arbitrage opportunities endpoint - to be implemented"})
}

func getArbitrageHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Arbitrage history endpoint - to be implemented"})
}

func getTechnicalIndicators(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Technical indicators endpoint - to be implemented"})
}

func getTradingSignals(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Trading signals endpoint - to be implemented"})
}

func handleTelegramWebhook(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Telegram webhook endpoint - to be implemented"})
}

func registerUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User registration endpoint - to be implemented"})
}

func loginUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User login endpoint - to be implemented"})
}

func getUserProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User profile endpoint - to be implemented"})
}

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
