package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/irfndi/celebrum-ai-go/internal/services"
)

// TestCacheEndpointsIntegration tests the cache endpoints with real CacheAnalyticsService
func TestCacheEndpointsIntegration(t *testing.T) {
	// Setup
	mockRedisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	cacheAnalytics := services.NewCacheAnalyticsService(mockRedisClient)
	cacheHandler := NewCacheHandler(cacheAnalytics)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register cache endpoints
	cacheGroup := router.Group("/api/v1/cache")
	{
		cacheGroup.GET("/stats", cacheHandler.GetCacheStats)
		cacheGroup.GET("/stats/:category", cacheHandler.GetCacheStatsByCategory)
		cacheGroup.GET("/metrics", cacheHandler.GetCacheMetrics)
		cacheGroup.POST("/stats/reset", cacheHandler.ResetCacheStats)
		cacheGroup.POST("/hit", cacheHandler.RecordCacheHit)
		cacheGroup.POST("/miss", cacheHandler.RecordCacheMiss)
	}

	t.Run("GET /api/v1/cache/stats - Overall Stats", func(t *testing.T) {
		// Record some cache operations first
		cacheAnalytics.RecordHit("test_category")
		cacheAnalytics.RecordMiss("test_category")

		// Test the endpoint
		req, _ := http.NewRequest("GET", "/api/v1/cache/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		t.Logf("Response status: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check response structure
		assert.Contains(t, response, "success")
		assert.Contains(t, response, "data")
		assert.Equal(t, true, response["success"])

		// Check data structure
		if data, ok := response["data"].(map[string]interface{}); ok {
			// Check that overall stats are present
			assert.Contains(t, data, "overall")
			if overall, ok := data["overall"].(map[string]interface{}); ok {
				assert.Contains(t, overall, "hits")
				assert.Contains(t, overall, "misses")
				assert.Contains(t, overall, "total_ops")
				assert.Contains(t, overall, "hit_rate")
			}
			t.Logf("Overall stats response: %+v", data)
		} else {
			t.Fatalf("Data field is not a map: %+v", response["data"])
		}
	})

	t.Run("GET /api/v1/cache/stats/:category - Category Stats", func(t *testing.T) {
		// Record operations for specific category
		cacheAnalytics.RecordHit("tickers")
		cacheAnalytics.RecordHit("tickers")
		cacheAnalytics.RecordMiss("tickers")

		// Test the endpoint
		req, _ := http.NewRequest("GET", "/api/v1/cache/stats/tickers", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		data := response["data"].(map[string]interface{})
		assert.Contains(t, data, "hits")
		assert.Contains(t, data, "misses")

		t.Logf("Tickers category stats: %+v", data)
	})

	t.Run("GET /api/v1/cache/metrics - Comprehensive Metrics", func(t *testing.T) {
		// Test the metrics endpoint
		req, _ := http.NewRequest("GET", "/api/v1/cache/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		t.Logf("Metrics response status: %d", w.Code)
		t.Logf("Metrics response body: %s", w.Body.String())

		// The metrics endpoint may fail if Redis is not available, which is expected in unit tests
		if w.Code == http.StatusInternalServerError {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, false, response["success"])
			assert.Contains(t, response, "error")
			t.Logf("Expected Redis connection error: %s", response["error"])
			return
		}

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 or 500, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		if data, ok := response["data"].(map[string]interface{}); ok {
			// Check comprehensive metrics structure
			assert.Contains(t, data, "overall")
			assert.Contains(t, data, "by_category")
			assert.Contains(t, data, "redis_info")
			assert.Contains(t, data, "memory_usage_bytes")
			assert.Contains(t, data, "connected_clients")
			assert.Contains(t, data, "key_count")
			t.Logf("Comprehensive metrics response: %+v", data)
		} else {
			t.Fatalf("Data field is not a map: %+v", response["data"])
		}
	})

	t.Run("POST /api/v1/cache/hit - Record Cache Hit", func(t *testing.T) {
		// Test recording cache hit
		req, _ := http.NewRequest("POST", "/api/v1/cache/hit?category=test_endpoint", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		assert.Equal(t, "Cache hits recorded successfully", response["message"])

		// Verify the hit was recorded
		stats := cacheAnalytics.GetStats("test_endpoint")
		assert.Greater(t, stats.Hits, int64(0))
	})

	t.Run("POST /api/v1/cache/miss - Record Cache Miss", func(t *testing.T) {
		// Test recording cache miss
		req, _ := http.NewRequest("POST", "/api/v1/cache/miss?category=test_endpoint", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		assert.Equal(t, "Cache misses recorded successfully", response["message"])

		// Verify the miss was recorded
		stats := cacheAnalytics.GetStats("test_endpoint")
		assert.Greater(t, stats.Misses, int64(0))
	})

	t.Run("POST /api/v1/cache/stats/reset - Reset Cache Stats", func(t *testing.T) {
		// Record some operations first
		cacheAnalytics.RecordHit("reset_test")
		cacheAnalytics.RecordMiss("reset_test")

		// Verify stats exist
		stats := cacheAnalytics.GetStats("reset_test")
		assert.Greater(t, stats.TotalOps, int64(0))

		// Test reset endpoint
		req, _ := http.NewRequest("POST", "/api/v1/cache/stats/reset", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, true, response["success"])
		assert.Equal(t, "Cache statistics reset successfully", response["message"])

		// Verify stats were reset
		resetStats := cacheAnalytics.GetStats("reset_test")
		assert.Equal(t, int64(0), resetStats.TotalOps)
	})
}

// TestCacheEndpointsWithRealData tests cache endpoints with actual market data operations
func TestCacheEndpointsWithRealData(t *testing.T) {
	// Setup
	mockRedisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	cacheAnalytics := services.NewCacheAnalyticsService(mockRedisClient)
	cacheHandler := NewCacheHandler(cacheAnalytics)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/cache/metrics", cacheHandler.GetCacheMetrics)

	// Simulate real market data cache operations
	categories := []string{"tickers", "bulk_tickers", "order_books", "market_data"}
	for _, category := range categories {
		// Simulate cache hits and misses for each category
		cacheAnalytics.RecordHit(category)
		cacheAnalytics.RecordHit(category)
		cacheAnalytics.RecordMiss(category)
	}

	// Test comprehensive metrics endpoint
	req, _ := http.NewRequest("GET", "/api/v1/cache/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Logf("Real data test - Metrics response status: %d", w.Code)
	t.Logf("Real data test - Metrics response body: %s", w.Body.String())

	// The metrics endpoint may fail if Redis is not available, which is expected in unit tests
	if w.Code == http.StatusInternalServerError {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, false, response["success"])
		assert.Contains(t, response, "error")
		t.Logf("Expected Redis connection error in real data test: %s", response["error"])
		return
	}

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	data := response["data"].(map[string]interface{})

	// Verify overall stats show aggregated data
	overall := data["overall"].(map[string]interface{})
	assert.Greater(t, overall["hits"], float64(0))
	assert.Greater(t, overall["misses"], float64(0))
	assert.Greater(t, overall["total_ops"], float64(0))

	// Verify category-specific data
	byCategory := data["by_category"].(map[string]interface{})
	for _, category := range categories {
		assert.Contains(t, byCategory, category)
		categoryData := byCategory[category].(map[string]interface{})
		assert.Greater(t, categoryData["hits"], float64(0))
		assert.Greater(t, categoryData["misses"], float64(0))
	}

	// Verify Redis info is included
	redisInfo := data["redis_info"].(map[string]interface{})
	assert.Contains(t, redisInfo, "connected_clients")
	assert.Contains(t, redisInfo, "redis_version")
	assert.Contains(t, redisInfo, "used_memory_human")

	t.Logf("Real data metrics - Overall: %+v", overall)
	t.Logf("Real data metrics - Categories: %+v", byCategory)
	t.Logf("Real data metrics - Redis Info: %+v", redisInfo)
}
