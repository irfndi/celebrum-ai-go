package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/database"
)

var startTime = time.Now()

type HealthHandler struct {
	db      *database.PostgresDB
	redis   *database.RedisClient
	ccxtURL string
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
}

func NewHealthHandler(db *database.PostgresDB, redis *database.RedisClient, ccxtURL string) *HealthHandler {
	return &HealthHandler{
		db:      db,
		redis:   redis,
		ccxtURL: ccxtURL,
	}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]string)

	// Check database
	if h.db != nil {
		if err := h.db.HealthCheck(r.Context()); err != nil {
			services["database"] = "unhealthy: " + err.Error()
		} else {
			services["database"] = "healthy"
		}
	} else {
		services["database"] = "unhealthy: not configured"
	}

	// Check Redis
	if h.redis != nil {
		ctx := r.Context()
		if err := h.redis.HealthCheck(ctx); err != nil {
			services["redis"] = "unhealthy: " + err.Error()
		} else {
			services["redis"] = "healthy"
		}
	} else {
		services["redis"] = "unhealthy: not configured"
	}

	// Check CCXT service
	if err := h.checkCCXTService(); err != nil {
		services["ccxt"] = "unhealthy: " + err.Error()
	} else {
		services["ccxt"] = "healthy"
	}

	// Check Telegram bot configuration
	if os.Getenv("TELEGRAM_BOT_TOKEN") == "" {
		services["telegram"] = "unhealthy: TELEGRAM_BOT_TOKEN not set"
	} else {
		services["telegram"] = "healthy"
	}

	// Determine overall status
	overallStatus := "healthy"
	for _, status := range services {
		if status != "healthy" {
			overallStatus = "unhealthy"
			break
		}
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Services:  services,
		Version:   os.Getenv("APP_VERSION"),
		Uptime:    time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	if overallStatus == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
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

// Readiness check for Kubernetes-style deployments
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	// Similar to HealthCheck but more strict
	services := make(map[string]string)

	// All services must be healthy for readiness
	if h.db != nil {
		if err := h.db.HealthCheck(r.Context()); err == nil {
			services["database"] = "ready"
		} else {
			services["database"] = "not ready"
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"ready":    false,
				"services": services,
			}); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
			return
		}
	}

	// All checks passed
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"ready":    true,
		"services": services,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Liveness check for container restarts
func (h *HealthHandler) LivenessCheck(w http.ResponseWriter, r *http.Request) {
	// Simple liveness check - just ensure the app is responsive
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
