package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/services"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// DatabaseHealthChecker interface for database health checks
type DatabaseHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// RedisHealthChecker interface for redis health checks
type RedisHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

type HealthHandler struct {
	db             DatabaseHealthChecker
	redis          RedisHealthChecker
	ccxtURL        string
	cacheAnalytics CacheAnalyticsInterface
}

type HealthResponse struct {
	Status       string                         `json:"status"`
	Timestamp    time.Time                      `json:"timestamp"`
	Services     map[string]string              `json:"services"`
	Version      string                         `json:"version"`
	Uptime       string                         `json:"uptime"`
	CacheMetrics *services.CacheMetrics         `json:"cache_metrics,omitempty"`
	CacheStats   map[string]services.CacheStats `json:"cache_stats,omitempty"`
}

type ServiceStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func NewHealthHandler(db DatabaseHealthChecker, redis RedisHealthChecker, ccxtURL string, cacheAnalytics CacheAnalyticsInterface) *HealthHandler {
	return &HealthHandler{
		db:             db,
		redis:          redis,
		ccxtURL:        ccxtURL,
		cacheAnalytics: cacheAnalytics,
	}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("celebrum-ai-app")
	
	// Create context with timeout for health checks
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	
	ctx, span := tracer.Start(ctx, "health_check")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("handler.name", "HealthCheck"),
	)

	services := make(map[string]string)

	// Check database
	span.AddEvent("checking database health")
	if h.db != nil {
		if err := h.db.HealthCheck(ctx); err != nil {
			services["database"] = "unhealthy: " + err.Error()
			span.SetAttributes(attribute.String("database.status", "unhealthy"))
			span.RecordError(err)
		} else {
			services["database"] = "healthy"
			span.SetAttributes(attribute.String("database.status", "healthy"))
		}
	} else {
		services["database"] = "unhealthy: not configured"
		span.SetAttributes(attribute.String("database.status", "not_configured"))
	}

	// Check Redis
	span.AddEvent("checking redis health")
	if h.redis != nil {
		if err := h.redis.HealthCheck(ctx); err != nil {
			services["redis"] = "unhealthy: " + err.Error()
			span.SetAttributes(attribute.String("redis.status", "unhealthy"))
			span.RecordError(err)
		} else {
			services["redis"] = "healthy"
			span.SetAttributes(attribute.String("redis.status", "healthy"))
		}
	} else {
		services["redis"] = "unhealthy: not configured"
		span.SetAttributes(attribute.String("redis.status", "not_configured"))
	}

	// Check CCXT service
	span.AddEvent("checking ccxt service health")
	if err := h.checkCCXTService(); err != nil {
		services["ccxt"] = "unhealthy: " + err.Error()
		span.SetAttributes(attribute.String("ccxt.status", "unhealthy"))
		span.RecordError(err)
	} else {
		services["ccxt"] = "healthy"
		span.SetAttributes(attribute.String("ccxt.status", "healthy"))
	}

	// Check Telegram bot configuration
	span.AddEvent("checking telegram configuration")
	if os.Getenv("TELEGRAM_BOT_TOKEN") == "" {
		services["telegram"] = "unhealthy: TELEGRAM_BOT_TOKEN not set"
		span.SetAttributes(attribute.String("telegram.status", "not_configured"))
	} else {
		services["telegram"] = "healthy"
		span.SetAttributes(attribute.String("telegram.status", "healthy"))
	}

	// Determine overall status
	span.AddEvent("determining overall status")
	overallStatus := "healthy"
	for _, status := range services {
		if status != "healthy" {
			overallStatus = "unhealthy"
			break
		}
	}
	span.SetAttributes(attribute.String("overall.status", overallStatus))

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Services:  services,
		Version:   os.Getenv("APP_VERSION"),
		Uptime:    time.Since(startTime).String(),
	}

	// Add cache metrics if cache analytics service is available
	if h.cacheAnalytics != nil {
		if cacheMetrics, err := h.cacheAnalytics.GetMetrics(r.Context()); err == nil {
			response.CacheMetrics = cacheMetrics
		}
		response.CacheStats = h.cacheAnalytics.GetAllStats()
	}

	w.Header().Set("Content-Type", "application/json")
	if overallStatus == "healthy" {
		w.WriteHeader(http.StatusOK)
		span.SetStatus(codes.Ok, "Health check completed successfully")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		span.SetStatus(codes.Error, "Some services are unhealthy")
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *HealthHandler) checkCCXTService() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(h.ccxtURL + "/health")
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CCXT service returned status: %d", resp.StatusCode)
	}

	return nil
}

// Global start time for uptime calculation
var startTime = time.Now()

// Readiness check for Kubernetes-style deployments
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("celebrum-ai-app")
	ctx, span := tracer.Start(r.Context(), "readiness_check")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("handler.name", "ReadinessCheck"),
	)

	// Similar to HealthCheck but more strict
	services := make(map[string]string)

	// All services must be healthy for readiness
	span.AddEvent("checking database readiness")
	if h.db != nil {
		if err := h.db.HealthCheck(ctx); err == nil {
			services["database"] = "ready"
			span.SetAttributes(attribute.String("database.readiness", "ready"))
		} else {
			services["database"] = "not ready"
			span.SetAttributes(attribute.String("database.readiness", "not_ready"))
			span.RecordError(err)
			span.SetStatus(codes.Error, "Database not ready")
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"ready":    false,
				"services": services,
			}); err != nil {
				span.RecordError(err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
	}

	// All checks passed
	span.AddEvent("all readiness checks passed")
	span.SetStatus(codes.Ok, "Service is ready")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"ready":    true,
		"services": services,
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Liveness check for container restarts
func (h *HealthHandler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("celebrum-ai-app")
	_, span := tracer.Start(r.Context(), "liveness_check")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.String()),
		attribute.String("handler.name", "LivenessCheck"),
	)

	// Simple liveness check - just ensure the app is responsive
	span.AddEvent("performing liveness check")
	span.SetStatus(codes.Ok, "Service is alive")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to encode response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
