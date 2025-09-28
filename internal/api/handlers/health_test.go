package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/irfandi/celebrum-ai-go/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// DatabaseInterface for mocking database operations
type DatabaseInterface interface {
	HealthCheck(ctx context.Context) error
}

// RedisHealthInterface for mocking Redis health operations
type RedisHealthInterface interface {
	HealthCheck(ctx context.Context) error
}

// MockDatabase mocks the database interface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockRedisHealthClient is a mock implementation of RedisHealthInterface
type MockRedisHealthClient struct {
	mock.Mock
}

func (m *MockRedisHealthClient) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewHealthHandler(t *testing.T) {
	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:8080", mockCacheAnalytics)

	assert.NotNil(t, handler)
	assert.Equal(t, mockDB, handler.db)
	assert.Equal(t, mockRedis, handler.redis)
	assert.Equal(t, "http://localhost:8080", handler.ccxtURL)
	assert.Equal(t, mockCacheAnalytics, handler.cacheAnalytics)
}

func TestHealthHandler_HealthCheck(t *testing.T) {
	// Set up a mock HTTP server for CCXT service
	mockCCXTServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer mockCCXTServer.Close()

	// Set environment variable for Telegram bot token
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	tests := []struct {
		name           string
		dbError        error
		redisError     error
		expectedStatus int
	}{
		{
			name:           "all services healthy",
			dbError:        nil,
			redisError:     nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "database error",
			dbError:        assert.AnError,
			redisError:     nil,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "redis error",
			dbError:        nil,
			redisError:     assert.AnError,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockRedis := &MockRedisHealthClient{}
			mockCacheAnalytics := NewMockCacheAnalyticsService()

			mockDB.On("HealthCheck", mock.Anything).Return(tt.dbError)
			mockRedis.On("HealthCheck", mock.Anything).Return(tt.redisError)
			mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
			mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

			handler := NewHealthHandler(mockDB, mockRedis, mockCCXTServer.URL, mockCacheAnalytics)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)

			handler.HealthCheck(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response, "status")
			assert.Contains(t, response, "services")
			assert.Contains(t, response, "timestamp")

			mockDB.AssertExpectations(t)
			mockRedis.AssertExpectations(t)
			mockCacheAnalytics.AssertExpectations(t)
		})
	}
}

func TestHealthHandler_ReadinessCheck(t *testing.T) {
	tests := []struct {
		name           string
		dbError        error
		expectedStatus int
	}{
		{
			name:           "all services ready",
			dbError:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "database not ready",
			dbError:        assert.AnError,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockRedis := &MockRedisHealthClient{}
			mockCacheAnalytics := NewMockCacheAnalyticsService()

			mockDB.On("HealthCheck", mock.Anything).Return(tt.dbError)

			handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:8080", mockCacheAnalytics)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/ready", nil)

			handler.ReadinessCheck(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response, "ready")
			assert.Contains(t, response, "services")

			mockDB.AssertExpectations(t)
		})
	}
}

func TestHealthHandler_LivenessCheck(t *testing.T) {
	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:8080", mockCacheAnalytics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/live", nil)

	handler.LivenessCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")
}
