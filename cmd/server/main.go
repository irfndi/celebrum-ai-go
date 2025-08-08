package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/api"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	setupLogger(cfg.LogLevel)

	// Initialize database
	db, err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := database.NewRedisConnection(cfg.Redis)
	if err != nil {
		logrus.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Setup Gin router
	router := setupRouter(cfg)

	// Initialize API routes
	api.SetupRoutes(router, db, redisClient)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Starting server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}

func setupRouter(cfg *config.Config) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.Server.AllowedOrigins
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(corsConfig))

	return router
}

func setupLogger(level string) {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	switch level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}