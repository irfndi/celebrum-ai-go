package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/irfandi/celebrum-ai-go/internal/api"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	"github.com/irfandi/celebrum-ai-go/internal/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TelegramE2ETestSuite is a test suite for end-to-end Telegram flow testing
type TelegramE2ETestSuite struct {
	suite.Suite
	db          *database.PostgresDB
	redisClient *database.RedisClient
	router      *gin.Engine
	adminAPIKey string
	testUserID  string
	testChatID  string
	testEmail   string
}

// SetupSuite runs once before all tests
func (s *TelegramE2ETestSuite) SetupSuite() {
	// Check for required environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		s.T().Skip("DATABASE_URL not set, skipping E2E test")
	}

	// Setup database connection
	var err error
	s.db, err = database.NewPostgresConnection(&config.DatabaseConfig{
		DatabaseURL:     dbURL,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "60s",
	})
	require.NoError(s.T(), err)

	// Setup Redis if available
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_HOST")
		if redisAddr != "" {
			redisPort := os.Getenv("REDIS_PORT")
			if redisPort == "" {
				redisPort = "6379"
			}
			redisAddr = fmt.Sprintf("%s:%s", redisAddr, redisPort)
		}
	}
	if redisAddr != "" {
		s.redisClient = &database.RedisClient{
			Client: redis.NewClient(&redis.Options{Addr: redisAddr}),
		}
	}

	// Setup Router
	gin.SetMode(gin.TestMode)
	s.router = gin.New()

	// Load admin API key from environment with fallback for local testing
	s.adminAPIKey = os.Getenv("TEST_ADMIN_API_KEY")
	if s.adminAPIKey == "" {
		s.adminAPIKey = "test-admin-key-e2e-secure-key-32chars"
	}
	cfg := &config.TelegramConfig{
		AdminAPIKey: s.adminAPIKey,
		ServiceURL:  "http://telegram-service:3002",
	}

	// Load JWT secret from environment with fallback for local testing
	jwtSecret := os.Getenv("TEST_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "test-jwt-secret-e2e"
	}

	// Create required middlewares
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	// Setup routes
	api.SetupRoutes(s.router, s.db, s.redisClient, nil, nil, nil, nil, nil, nil, cfg, authMiddleware)

	// Create test user
	s.testChatID = fmt.Sprintf("e2e_test_%d", time.Now().UnixNano())
	s.testEmail = fmt.Sprintf("e2e_test_%s@celebrum.ai", uuid.New().String())
	s.testUserID = uuid.New().String()

	_, err = s.db.Pool.Exec(context.Background(), `
		INSERT INTO users (id, email, password_hash, telegram_chat_id, subscription_tier, created_at, updated_at)
		VALUES ($1, $2, 'hashed_password', $3, 'free', NOW(), NOW())
	`, s.testUserID, s.testEmail, s.testChatID)
	require.NoError(s.T(), err)
}

// TearDownSuite runs once after all tests
func (s *TelegramE2ETestSuite) TearDownSuite() {
	if s.db != nil {
		// Clean up test data with error logging
		if _, err := s.db.Pool.Exec(context.Background(), "DELETE FROM user_alerts WHERE user_id = $1", s.testUserID); err != nil {
			s.T().Logf("failed to clean up user_alerts: %v", err)
		}
		if _, err := s.db.Pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", s.testUserID); err != nil {
			s.T().Logf("failed to clean up users: %v", err)
		}
		s.db.Close()
	}
	if s.redisClient != nil {
		s.redisClient.Close()
	}
}

// TestCompleteUserFlowE2E tests the complete user registration and notification flow
func (s *TelegramE2ETestSuite) TestCompleteUserFlowE2E() {
	t := s.T()

	// Step 1: Lookup user by chat ID (simulates /start command flow)
	t.Run("Step1_LookupUserByChatID", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/"+s.testChatID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		userData := resp["user"].(map[string]interface{})
		assert.Equal(t, s.testUserID, userData["id"])
		assert.Equal(t, "free", userData["subscription_tier"])
	})

	// Step 2: Check default notification preferences (simulates /settings command)
	t.Run("Step2_CheckDefaultPreferences", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/notifications/"+s.testUserID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, true, resp["enabled"])
		assert.Equal(t, 0.5, resp["profit_threshold"])
	})

	// Step 3: Disable notifications (simulates /stop command)
	t.Run("Step3_DisableNotifications", func(t *testing.T) {
		body := []byte(`{"enabled": false}`)
		req, err := http.NewRequest("POST", "/api/v1/telegram/internal/notifications/"+s.testUserID, bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "success", resp["status"])
	})

	// Step 4: Verify notifications are disabled
	t.Run("Step4_VerifyNotificationsDisabled", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/notifications/"+s.testUserID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, false, resp["enabled"])
	})

	// Step 5: Re-enable notifications (simulates /resume command)
	t.Run("Step5_ReEnableNotifications", func(t *testing.T) {
		body := []byte(`{"enabled": true}`)
		req, err := http.NewRequest("POST", "/api/v1/telegram/internal/notifications/"+s.testUserID, bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Step 6: Verify notifications are enabled again
	t.Run("Step6_VerifyNotificationsEnabled", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/notifications/"+s.testUserID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, true, resp["enabled"])
	})
}

// TestUserNotFoundE2E tests handling of non-existent users
func (s *TelegramE2ETestSuite) TestUserNotFoundE2E() {
	t := s.T()

	t.Run("LookupNonExistentUser", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/nonexistent_chat_id", nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "User not found", resp["error"])
	})
}

// TestAuthenticationE2E tests authentication requirements
func (s *TelegramE2ETestSuite) TestAuthenticationE2E() {
	t := s.T()

	t.Run("NoAPIKey", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/"+s.testChatID, nil)
		require.NoError(t, err)
		// No API key
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/"+s.testChatID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", "wrong-api-key")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("ValidAPIKey", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/"+s.testChatID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestConcurrentAccessE2E tests concurrent access to user data
func (s *TelegramE2ETestSuite) TestConcurrentAccessE2E() {
	t := s.T()

	t.Run("ConcurrentReads", func(t *testing.T) {
		// Perform 10 concurrent reads
		done := make(chan bool, 10)
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/"+s.testChatID, nil)
				if err != nil {
					errors <- err
					done <- true
					return
				}
				req.Header.Set("X-API-Key", s.adminAPIKey)
				w := httptest.NewRecorder()
				s.router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("unexpected status: %d", w.Code)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check for errors
		close(errors)
		for err := range errors {
			t.Errorf("Concurrent read error: %v", err)
		}
	})

	t.Run("ConcurrentToggleNotifications", func(t *testing.T) {
		// Perform alternating enable/disable operations
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			enabled := i%2 == 0
			go func(enable bool) {
				body := []byte(fmt.Sprintf(`{"enabled": %v}`, enable))
				req, err := http.NewRequest("POST", "/api/v1/telegram/internal/notifications/"+s.testUserID, bytes.NewBuffer(body))
				if err != nil {
					done <- true
					return
				}
				req.Header.Set("X-API-Key", s.adminAPIKey)
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				s.router.ServeHTTP(w, req)
				done <- true
			}(enabled)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Final state should be consistent (one of enabled or disabled)
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/notifications/"+s.testUserID, nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		// The final state is non-deterministic, so we just check that the key exists and is a boolean.
		assert.Contains(t, resp, "enabled")
		assert.IsType(t, true, resp["enabled"])
	})
}

// TestValidationE2E tests input validation
func (s *TelegramE2ETestSuite) TestValidationE2E() {
	t := s.T()

	t.Run("EmptyChatID", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/telegram/internal/users/", nil)
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		// Should return 404 or 400 for missing ID
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusBadRequest)
	})

	t.Run("InvalidJSONBody", func(t *testing.T) {
		body := []byte(`{invalid json}`)
		req, err := http.NewRequest("POST", "/api/v1/telegram/internal/notifications/"+s.testUserID, bytes.NewBuffer(body))
		require.NoError(t, err)
		req.Header.Set("X-API-Key", s.adminAPIKey)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestHealthEndpointE2E tests the health endpoint
func (s *TelegramE2ETestSuite) TestHealthEndpointE2E() {
	t := s.T()

	t.Run("HealthCheck", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/health", nil)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "healthy", resp["status"])
	})
}

// Run the test suite
func TestTelegramE2ESuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test suite in short mode")
	}
	suite.Run(t, new(TelegramE2ETestSuite))
}
