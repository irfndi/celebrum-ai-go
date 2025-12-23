package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/irfandi/celebrum-ai-go/internal/services"
)

func TestHealthHandler_CurlHealthcheckCompatibility(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	mockDB.On("HealthCheck", mock.Anything).Return(nil)
	mockRedis.On("HealthCheck", mock.Anything).Return(nil)
	mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
	mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:3001", mockCacheAnalytics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	handler.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestHealthHandler_CCXTServiceConnectivity(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	tests := []struct {
		name          string
		ccxtURL       string
		expectUnhealthy bool
	}{
		{
			name:          "ccxt_reachable",
			ccxtURL:       "http://localhost:3001",
			expectUnhealthy: false,
		},
		{
			name:          "ccxt_unreachable",
			ccxtURL:       "http://localhost:9999",
			expectUnhealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockRedis := &MockRedisHealthClient{}
			mockCacheAnalytics := NewMockCacheAnalyticsService()

			mockDB.On("HealthCheck", mock.Anything).Return(nil)
			mockRedis.On("HealthCheck", mock.Anything).Return(nil)
			mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
			mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

			handler := NewHealthHandler(mockDB, mockRedis, tt.ccxtURL, mockCacheAnalytics)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)

			handler.HealthCheck(w, req)

			if tt.expectUnhealthy {
				assert.Contains(t, w.Body.String(), "unhealthy")
			} else {
				assert.NotContains(t, w.Body.String(), "unhealthy")
			}

			mockDB.AssertExpectations(t)
			mockRedis.AssertExpectations(t)
			mockCacheAnalytics.AssertExpectations(t)
		})
	}
}

func TestHealthHandler_MissingTelegramToken(t *testing.T) {
	tests := []struct {
		name           string
		telegramToken  string
		expectedStatus int
	}{
		{
			name:           "telegram_token_missing",
			telegramToken:  "",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "telegram_token_present",
			telegramToken:  "test-token",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			mockRedis := &MockRedisHealthClient{}
			mockCacheAnalytics := NewMockCacheAnalyticsService()

			mockDB.On("HealthCheck", mock.Anything).Return(nil)
			mockRedis.On("HealthCheck", mock.Anything).Return(nil)
			mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
			mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

			t.Setenv("TELEGRAM_BOT_TOKEN", tt.telegramToken)

			handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:3001", mockCacheAnalytics)

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)

			handler.HealthCheck(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockDB.AssertExpectations(t)
			mockRedis.AssertExpectations(t)
			mockCacheAnalytics.AssertExpectations(t)
		})
	}
}

func TestHealthHandler_ServiceUnhealthyStatus(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	mockDB.On("HealthCheck", mock.Anything).Return(assert.AnError)
	mockRedis.On("HealthCheck", mock.Anything).Return(assert.AnError)
	mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
	mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:3001", mockCacheAnalytics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	handler.HealthCheck(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "database")
	assert.Contains(t, body, "redis")

	mockDB.AssertExpectations(t)
	mockRedis.AssertExpectations(t)
	mockCacheAnalytics.AssertExpectations(t)
}

func TestHealthHandler_ReadinessWithDBError(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	mockDB.On("HealthCheck", mock.Anything).Return(assert.AnError)

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:3001", mockCacheAnalytics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)

	handler.ReadinessCheck(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "\"ready\":false")

	mockDB.AssertExpectations(t)
}
}

func TestHealthHandler_ContextTimeout(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	mockDB.On("HealthCheck", mock.Anything).Return(context.DeadlineExceeded)
	mockRedis.On("HealthCheck", mock.Anything).Return(nil)
	mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
	mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:3001", mockCacheAnalytics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	handler.HealthCheck(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHealthHandler_LivenessAlwaysHealthy(t *testing.T) {
	handler := &HealthHandler{}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/live", nil)

	handler.LivenessCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "alive")
}

func TestHealthHandler_AllServicesHealthy(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")

	mockDB := &MockDatabase{}
	mockRedis := &MockRedisHealthClient{}
	mockCacheAnalytics := NewMockCacheAnalyticsService()

	mockDB.On("HealthCheck", mock.Anything).Return(nil)
	mockRedis.On("HealthCheck", mock.Anything).Return(nil)
	mockCacheAnalytics.On("GetMetrics", mock.Anything).Return(&services.CacheMetrics{}, nil)
	mockCacheAnalytics.On("GetAllStats").Return(map[string]services.CacheStats{})

	handler := NewHealthHandler(mockDB, mockRedis, "http://localhost:3001", mockCacheAnalytics)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)

	handler.HealthCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"status\":\"healthy\"")
	assert.Contains(t, w.Body.String(), "\"database\":\"healthy\"")
	assert.Contains(t, w.Body.String(), "\"redis\":\"healthy\"")
}
