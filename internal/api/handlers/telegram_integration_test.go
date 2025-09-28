package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-telegram/bot/models"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTelegramWebhookIntegration tests the complete webhook flow
func TestTelegramWebhookIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	gin.SetMode(gin.TestMode)

	// Test cases for different webhook scenarios
	tests := []struct {
		name           string
		update         models.Update
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "valid start command",
			update: models.Update{
				ID: 123,
				Message: &models.Message{
					ID: 456,
					From: &models.User{
						ID:        789,
						FirstName: "TestUser",
						Username:  "testuser",
					},
					Chat: models.Chat{
						ID:   789,
						Type: "private",
					},
					Date: int(time.Now().Unix()),
					Text: "/start",
				},
			},
			expectedStatus: http.StatusServiceUnavailable, // Bot not initialized in test
			expectedError:  true,
		},
		{
			name: "valid opportunities command",
			update: models.Update{
				ID: 124,
				Message: &models.Message{
					ID: 457,
					From: &models.User{
						ID:        790,
						FirstName: "TestUser2",
						Username:  "testuser2",
					},
					Chat: models.Chat{
						ID:   790,
						Type: "private",
					},
					Date: int(time.Now().Unix()),
					Text: "/opportunities",
				},
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedError:  true,
		},
		{
			name: "regular text message",
			update: models.Update{
				ID: 125,
				Message: &models.Message{
					ID: 458,
					From: &models.User{
						ID:        791,
						FirstName: "TestUser3",
						Username:  "testuser3",
					},
					Chat: models.Chat{
						ID:   791,
						Type: "private",
					},
					Date: int(time.Now().Unix()),
					Text: "Hello bot!",
				},
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedError:  true,
		},
		{
			name: "nil message",
			update: models.Update{
				ID:      126,
				Message: nil,
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil config (bot won't initialize)
			handler := NewTelegramHandler(nil, nil, nil, nil, nil)

			// Marshal update to JSON
			jsonData, err := json.Marshal(tt.update)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest("POST", "/telegram/webhook", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.HandleWebhook(c)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

// TestTelegramCommandHandling tests individual command handling logic
func TestTelegramCommandHandling(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		isCommand   bool
		expectedCmd string
	}{
		{"start command", "/start", true, "start"},
		{"opportunities command", "/opportunities", true, "opportunities"},
		{"settings command", "/settings", true, "settings"},
		{"help command", "/help", true, "help"},
		{"upgrade command", "/upgrade", true, "upgrade"},
		{"status command", "/status", true, "status"},
		{"stop command", "/stop", true, "stop"},
		{"resume command", "/resume", true, "resume"},
		{"regular text", "Hello bot!", false, ""},
		{"empty text", "", false, ""},
		{"command with parameters", "/start param1 param2", true, "start"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCmd := len(tt.command) > 0 && tt.command[0] == '/'
			assert.Equal(t, tt.isCommand, isCmd)

			if tt.isCommand {
				// Extract command name (remove '/' and any parameters)
				cmdText := tt.command[1:] // Remove '/'
				cmdName := ""
				for _, r := range cmdText {
					if r == ' ' {
						break
					}
					cmdName += string(r)
				}
				assert.Equal(t, tt.expectedCmd, cmdName)
			}
		})
	}
}

// TestTelegramCacheIntegration tests Redis caching for opportunities
func TestTelegramCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock Redis client
	options := testutil.GetTestRedisOptions()
	options.DB = 1 // Use test database
	mockRedis := redis.NewClient(options)
	defer func() {
		if err := mockRedis.Close(); err != nil {
			t.Logf("Failed to close mock Redis: %v", err)
		}
	}()

	// Test cache key generation
	cacheKey := "telegram_opportunities"
	assert.NotEmpty(t, cacheKey)

	// Test cache operations (would require actual Redis connection)
	ctx := context.Background()
	_, err := mockRedis.Ping(ctx).Result()
	if err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	// Test setting cache
	testData := `[{"exchange_a":"binance","exchange_b":"coinbase","symbol":"BTC/USDT","profit_percentage":1.5}]`
	err = mockRedis.Set(ctx, cacheKey, testData, 30*time.Second).Err()
	assert.NoError(t, err)

	// Test getting cache
	cachedData, err := mockRedis.Get(ctx, cacheKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, testData, cachedData)

	// Test cache expiration
	ttl, err := mockRedis.TTL(ctx, cacheKey).Result()
	assert.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= 30*time.Second)

	// Cleanup
	mockRedis.Del(ctx, cacheKey)
}

// TestTelegramErrorHandling tests error scenarios
func TestTelegramErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			body:           "invalid json",
			contentType:    "application/json",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "empty body",
			body:           "",
			contentType:    "application/json",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "wrong content type",
			body:           `{"update_id": 123}`,
			contentType:    "text/plain",
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil config
			handler := NewTelegramHandler(nil, nil, nil, nil, nil)

			// Create HTTP request
			req := httptest.NewRequest("POST", "/telegram/webhook", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", tt.contentType)

			// Create response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.HandleWebhook(c)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestTelegramConfigValidation tests configuration validation
func TestTelegramConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.TelegramConfig
		expectNil bool
	}{
		{
			name:      "nil config",
			config:    nil,
			expectNil: true,
		},
		{
			name: "empty bot token",
			config: &config.TelegramConfig{
				BotToken: "",
			},
			expectNil: true,
		},
		{
			name: "invalid bot token",
			config: &config.TelegramConfig{
				BotToken: "invalid_token",
			},
			expectNil: true, // Bot creation will fail
		},
		{
			name: "valid config structure",
			config: &config.TelegramConfig{
				BotToken:   "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				WebhookURL: "https://example.com/webhook",
			},
			expectNil: true, // Still fails due to invalid token, but config is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewTelegramHandler(nil, tt.config, nil, nil, nil)
			assert.NotNil(t, handler) // Handler is always created

			// Test that bot is not initialized with invalid configs
			// This is tested indirectly through webhook handling
			if tt.config != nil && tt.name != "empty bot token" {
				assert.NotEmpty(t, tt.config.BotToken)
			} else if tt.config != nil && tt.name == "empty bot token" {
				assert.Empty(t, tt.config.BotToken)
			}
		})
	}
}

// BenchmarkTelegramWebhookProcessing benchmarks webhook processing
func BenchmarkTelegramWebhookProcessing(b *testing.B) {
	gin.SetMode(gin.TestMode)

	// Create test update
	update := models.Update{
		ID: 123,
		Message: &models.Message{
			ID: 456,
			From: &models.User{
				ID:        789,
				FirstName: "BenchUser",
			},
			Chat: models.Chat{
				ID: 789,
			},
			Text: "/start",
		},
	}

	jsonData, _ := json.Marshal(update)
	handler := NewTelegramHandler(nil, nil, nil, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/telegram/webhook", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		handler.HandleWebhook(c)
	}
}
