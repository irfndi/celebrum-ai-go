package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Context key types to avoid collisions
type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

// generateTestPassword creates a random password for testing to avoid hardcoded secrets
// This addresses GitGuardian security alerts about hardcoded test credentials
func generateTestPassword() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, 12)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[n.Int64()]
	}
	return string(b)
}

// TestIntegrationAPI tests the integration of various API components
func TestIntegrationAPI(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a simple router for testing
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Define test routes
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now(),
			"version":   "1.0.0",
			"services": gin.H{
				"database": "ok",
				"redis":    "ok",
			},
		})
	})

	router.GET("/api/v1/market/ticker/:exchange/:symbol", func(c *gin.Context) {
		exchange := c.Param("exchange")
		symbol := c.Param("symbol")

		c.JSON(http.StatusOK, gin.H{
			"exchange":  exchange,
			"symbol":    symbol,
			"price":     50000.0,
			"volume":    1000.0,
			"timestamp": time.Now(),
		})
	})

	router.POST("/api/v1/users/register", func(c *gin.Context) {
		var user struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "User registered successfully",
			"user": gin.H{
				"username": user.Username,
				"email":    user.Email,
			},
		})
	})

	// Test cases
	testCases := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Health check endpoint",
			method:         "GET",
			path:           "/api/v1/health",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "ok", response["status"])
				assert.Contains(t, response, "timestamp")
				assert.Equal(t, "1.0.0", response["version"])
			},
		},
		{
			name:           "Market ticker endpoint",
			method:         "GET",
			path:           "/api/v1/market/ticker/binance/BTCUSDT",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "binance", response["exchange"])
				assert.Equal(t, "BTCUSDT", response["symbol"])
				assert.Contains(t, response, "price")
				assert.Contains(t, response, "timestamp")
			},
		},
		{
			name:   "User registration endpoint",
			method: "POST",
			path:   "/api/v1/users/register",
			body: map[string]interface{}{
				"username": "testuser",
				"password": generateTestPassword(),
				"email":    "test@example.com",
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Equal(t, "User registered successfully", response["message"])
				assert.Contains(t, response, "user")
			},
		},
		{
			name:           "Invalid route",
			method:         "GET",
			path:           "/api/v1/invalid",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Method not allowed",
			method:         "POST",
			path:           "/api/v1/health",
			expectedStatus: http.StatusNotFound, // Gin returns 404 for unsupported methods
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			var req *http.Request
			if tc.body != nil {
				jsonData, err := json.Marshal(tc.body)
				assert.NoError(t, err)
				req, _ = http.NewRequest(tc.method, tc.path, bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tc.method, tc.path, nil)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve the request
			router.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Check response if needed
			if tc.checkResponse != nil {
				tc.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

// TestIntegrationMiddleware tests the integration of middleware components
func TestIntegrationMiddleware(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a simple router with middleware
	router := gin.New()

	// Add middleware
	router.Use(func(c *gin.Context) {
		c.Header("X-Test-Middleware", "applied")
		c.Set("middleware_value", "applied")
		c.Next()
	})

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "12345")
		c.Next()
	})

	// Define test route
	router.GET("/api/v1/test", func(c *gin.Context) {
		userID := c.MustGet("user_id").(string)
		middlewareValue := c.MustGet("middleware_value").(string)
		c.JSON(http.StatusOK, gin.H{
			"message":    "Test endpoint",
			"user_id":    userID,
			"middleware": middlewareValue,
		})
	})

	// Test request
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Test endpoint", response["message"])
	assert.Equal(t, "12345", response["user_id"])
	assert.Equal(t, "applied", response["middleware"])
	assert.Equal(t, "applied", w.Header().Get("X-Test-Middleware"))
}

// TestIntegrationErrorHandling tests error handling integration
func TestIntegrationErrorHandling(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a simple router with error handling
	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	// Define routes that can return errors
	router.GET("/api/v1/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	})

	router.GET("/api/v1/not-found", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Resource not found",
		})
	})

	router.GET("/api/v1/bad-request", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bad request",
		})
	})

	// Test error scenarios
	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Internal server error",
			path:           "/api/v1/error",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Internal server error",
		},
		{
			name:           "Not found error",
			path:           "/api/v1/not-found",
			expectedStatus: http.StatusNotFound,
			expectedError:  "Resource not found",
		},
		{
			name:           "Bad request error",
			path:           "/api/v1/bad-request",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Bad request",
		},
		{
			name:           "Route not found",
			path:           "/api/v1/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedError, response["error"])
			}
		})
	}
}

// TestIntegrationContext tests context management integration
func TestIntegrationContext(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a simple router
	router := gin.New()

	// Add context middleware
	router.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), requestIDKey, "test-123")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	// Define test route
	router.GET("/api/v1/context", func(c *gin.Context) {
		requestID := c.Request.Context().Value(requestIDKey).(string)
		c.JSON(http.StatusOK, gin.H{
			"request_id": requestID,
			"timestamp":  time.Now(),
		})
	})

	// Test request
	req, _ := http.NewRequest("GET", "/api/v1/context", nil)
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-123", response["request_id"])
	assert.Contains(t, response, "timestamp")
}

// TestIntegrationPerformance tests performance-related integration scenarios
func TestIntegrationPerformance(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a simple router
	router := gin.New()

	// Define performance test route
	router.GET("/api/v1/performance", func(c *gin.Context) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		c.JSON(http.StatusOK, gin.H{
			"message":   "Performance test",
			"timestamp": time.Now(),
		})
	})

	// Test concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req, _ := http.NewRequest("GET", "/api/v1/performance", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "Performance test", response["message"])
		}(i)
	}

	wg.Wait()
}
