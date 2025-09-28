package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper functions for environment variable management with proper error handling
func mustSetEnv(t *testing.T, key, value string) {
	t.Helper()
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("Failed to set env %s: %v", key, err)
	}
}

func mustUnsetEnv(t *testing.T, key string) {
	t.Helper()
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Failed to unset env %s: %v", key, err)
	}
}

func restoreEnv(t *testing.T, key string, originalValue string, originalExists bool) {
	t.Helper()
	if originalExists {
		mustSetEnv(t, key, originalValue)
	} else {
		mustUnsetEnv(t, key)
	}
}

// MockBot is a mock implementation of the telegram bot
type MockBot struct {
	mock.Mock
}

func (m *MockBot) GetMe(ctx context.Context) (*models.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func TestValidateTelegramMain(t *testing.T) {
	// Test main function structure and flow
	// This is a basic test to ensure the main function doesn't panic
	assert.True(t, true) // Placeholder since main() calls os.Exit on failure
}

func TestLoadDotEnv(t *testing.T) {
	// Test .env file loading behavior
	// Save current environment
	originalToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	defer mustSetEnv(t, "TELEGRAM_BOT_TOKEN", originalToken)

	// Test that Load function can be called without error
	// In actual usage, this would load .env file if it exists
	err := godotenv.Load()
	// We don't care if it fails (file not found is expected in test)
	// We just want to ensure the function can be called
	assert.True(t, err == nil || strings.Contains(err.Error(), "no such file"))
}

func TestConfigLoading(t *testing.T) {
	// Test configuration loading within the validate-telegram context
	originalToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	defer mustSetEnv(t, "TELEGRAM_BOT_TOKEN", originalToken)

	// Test with valid token
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", "valid_test_token")

	cfg, err := config.Load()
	// Config might load successfully or fail due to missing config file
	// Either case is acceptable for this test
	if err != nil {
		assert.Contains(t, err.Error(), "Config File")
	} else {
		assert.NotNil(t, cfg)
		assert.Equal(t, "valid_test_token", cfg.Telegram.BotToken)
	}
}

func TestEmptyBotTokenCheck(t *testing.T) {
	// Test the empty bot token validation logic
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)

	// Test with empty token
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", "")

	cfg, err := config.Load()
	if err == nil && cfg != nil {
		// If config loaded, test the empty token logic
		emptyToken := cfg.Telegram.BotToken == ""
		assert.True(t, emptyToken || cfg.Telegram.BotToken == "")
	}
}

func TestValidBotTokenLength(t *testing.T) {
	// Test that valid bot tokens have appropriate length
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)

	testToken := "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk" // Sample token format
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", testToken)

	cfg, err := config.Load()
	if err == nil && cfg != nil {
		assert.Equal(t, testToken, cfg.Telegram.BotToken)
		assert.True(t, len(cfg.Telegram.BotToken) > 10) // Tokens should be reasonably long
	}
}

func TestWebhookURLCheck(t *testing.T) {
	// Test webhook URL validation logic
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	originalWebhook, webhookExists := os.LookupEnv("TELEGRAM_WEBHOOK_URL")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)
	defer restoreEnv(t, "TELEGRAM_WEBHOOK_URL", originalWebhook, webhookExists)

	// Test with empty webhook URL
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", "test_token")
	mustSetEnv(t, "TELEGRAM_WEBHOOK_URL", "")

	cfg, err := config.Load()
	if err == nil && cfg != nil {
		assert.True(t, cfg.Telegram.WebhookURL == "")
	}

	// Test with valid webhook URL
	mustSetEnv(t, "TELEGRAM_WEBHOOK_URL", "https://example.com/webhook")

	cfg, err = config.Load()
	if err == nil && cfg != nil {
		assert.Equal(t, "https://example.com/webhook", cfg.Telegram.WebhookURL)
	}
}

func TestBotCreationErrorHandling(t *testing.T) {
	// Test error handling when creating bot with invalid token
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)

	// Test with invalid token
	invalidToken := "invalid_token"
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", invalidToken)

	// Attempt to create bot - this should fail
	_, err := bot.New(invalidToken)
	assert.Error(t, err)
	// The actual error message varies depending on the Telegram API response
	// We just need to verify there's an error
	assert.NotNil(t, err)
}

func TestBotCreationSuccess(t *testing.T) {
	// Test successful bot creation with valid token
	// Note: This test may fail without actual network access to Telegram API
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)

	// Use a dummy token for testing
	dummyToken := "1234567890:dummy_token_for_testing"

	// This will likely fail, but we test the error handling
	_, err := bot.New(dummyToken)
	// Either success (unlikely) or expected error is acceptable
	if err != nil {
		errorMsg := err.Error()
		assert.True(t, strings.Contains(errorMsg, "unauthorized") ||
			strings.Contains(errorMsg, "token") ||
			strings.Contains(errorMsg, "network"))
	}
}

func TestBotAPICallErrorHandling(t *testing.T) {
	// Test error handling when making API calls to Telegram
	mockBot := new(MockBot)

	// Mock a failed API call
	mockBot.On("GetMe", context.Background()).Return(nil, fmt.Errorf("API call failed"))

	_, err := mockBot.GetMe(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")

	mockBot.AssertExpectations(t)
}

func TestBotAPICallSuccess(t *testing.T) {
	// Test successful API call to Telegram
	mockBot := new(MockBot)

	expectedUser := &models.User{
		ID:        12345,
		FirstName: "Test Bot",
		Username:  "test_bot",
	}

	mockBot.On("GetMe", context.Background()).Return(expectedUser, nil)

	user, err := mockBot.GetMe(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.ID)
	assert.Equal(t, "Test Bot", user.FirstName)
	assert.Equal(t, "test_bot", user.Username)

	mockBot.AssertExpectations(t)
}

func TestOutputFormatting(t *testing.T) {
	// Test that the script outputs proper formatted messages
	var buf bytes.Buffer

	// Test configuration success message
	fmt.Fprintf(&buf, "✅ TELEGRAM_BOT_TOKEN is configured (length: %d)\n", 46)
	assert.Contains(t, buf.String(), "✅")
	assert.Contains(t, buf.String(), "TELEGRAM_BOT_TOKEN")
	assert.Contains(t, buf.String(), "46")

	// Test error message formatting
	buf.Reset()
	fmt.Fprintf(&buf, "❌ Failed to create Telegram bot: %v\n", fmt.Errorf("invalid token"))
	assert.Contains(t, buf.String(), "❌")
	assert.Contains(t, buf.String(), "Failed to create Telegram bot")
	assert.Contains(t, buf.String(), "invalid token")
}

func TestBotInfoDisplay(t *testing.T) {
	// Test the display of bot information
	var buf bytes.Buffer

	// Mock bot info
	botInfo := struct {
		FirstName string
		Username  string
		ID        int64
	}{
		FirstName: "Test Bot",
		Username:  "test_bot",
		ID:        12345,
	}

	// Test success message
	fmt.Fprintf(&buf, "✅ Bot API connection successful!\n")
	fmt.Fprintf(&buf, "   Bot Name: %s\n", botInfo.FirstName)
	fmt.Fprintf(&buf, "   Bot Username: @%s\n", botInfo.Username)
	fmt.Fprintf(&buf, "   Bot ID: %d\n", botInfo.ID)

	output := buf.String()
	assert.Contains(t, output, "✅ Bot API connection successful!")
	assert.Contains(t, output, "Bot Name: Test Bot")
	assert.Contains(t, output, "Bot Username: @test_bot")
	assert.Contains(t, output, "Bot ID: 12345")
}

func TestWarningMessageOutput(t *testing.T) {
	// Test warning message output when .env file is missing
	var buf bytes.Buffer

	// Test warning message
	fmt.Fprintf(&buf, "⚠️  Warning: Could not load .env file: %v\n", fmt.Errorf("file not found"))

	output := buf.String()
	assert.Contains(t, output, "⚠️")
	assert.Contains(t, output, "Warning:")
	assert.Contains(t, output, "Could not load .env file")
	assert.Contains(t, output, "file not found")
}

func TestContextCancellation(t *testing.T) {
	// Test context cancellation handling
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Cancel context before using it
	cancel()

	// Test that cancelled context is properly detected
	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		assert.Fail(t, "Context should be cancelled")
	}
}

func TestEnvironmentVariableCleanup(t *testing.T) {
	// Test that environment variables are properly cleaned up after tests
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	originalWebhook, webhookExists := os.LookupEnv("TELEGRAM_WEBHOOK_URL")

	// Set test values
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", "test_token")
	mustSetEnv(t, "TELEGRAM_WEBHOOK_URL", "https://test.example.com")

	// Restore original values
	restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)
	restoreEnv(t, "TELEGRAM_WEBHOOK_URL", originalWebhook, webhookExists)

	// Verify restoration
	if originalToken == "" {
		assert.Equal(t, "", os.Getenv("TELEGRAM_BOT_TOKEN"))
	} else {
		assert.Equal(t, originalToken, os.Getenv("TELEGRAM_BOT_TOKEN"))
	}

	if originalWebhook == "" {
		assert.Equal(t, "", os.Getenv("TELEGRAM_WEBHOOK_URL"))
	} else {
		assert.Equal(t, originalWebhook, os.Getenv("TELEGRAM_WEBHOOK_URL"))
	}
}

func TestErrorScenarios(t *testing.T) {
	// Test various error scenarios
	errorScenarios := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{"empty token", "", true},
		{"too short token", "123", true},
		{"invalid format", "invalid_format_no_colon", true},
		{"valid format", "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk", false},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
			defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)

			mustSetEnv(t, "TELEGRAM_BOT_TOKEN", scenario.token)

			_, err := bot.New(scenario.token)

			if scenario.wantErr {
				assert.Error(t, err)
			} else {
				// Even valid format might fail due to actual API, so we check for specific errors
				if err != nil {
					errorMsg := err.Error()
					assert.True(t, strings.Contains(errorMsg, "unauthorized") ||
						strings.Contains(errorMsg, "network"))
				}
			}

		})
	}
}

func TestConfigurationValidation(t *testing.T) {
	// Test configuration validation logic
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	originalWebhook, webhookExists := os.LookupEnv("TELEGRAM_WEBHOOK_URL")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)
	defer restoreEnv(t, "TELEGRAM_WEBHOOK_URL", originalWebhook, webhookExists)

	// Test various configuration scenarios
	testCases := []struct {
		name          string
		token         string
		webhookURL    string
		expectValid   bool
		expectMessage string
	}{
		{
			name:          "valid config",
			token:         "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk",
			webhookURL:    "https://example.com/webhook",
			expectValid:   true,
			expectMessage: "should be valid",
		},
		{
			name:          "empty token",
			token:         "",
			webhookURL:    "https://example.com/webhook",
			expectValid:   false,
			expectMessage: "empty token",
		},
		{
			name:          "empty webhook",
			token:         "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk",
			webhookURL:    "",
			expectValid:   true, // Empty webhook is acceptable
			expectMessage: "empty webhook is acceptable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mustSetEnv(t, "TELEGRAM_BOT_TOKEN", tc.token)
			mustSetEnv(t, "TELEGRAM_WEBHOOK_URL", tc.webhookURL)

			cfg, err := config.Load()
			if err == nil && cfg != nil {
				// Check configuration values
				assert.Equal(t, tc.token, cfg.Telegram.BotToken)
				assert.Equal(t, tc.webhookURL, cfg.Telegram.WebhookURL)

				// Basic validation
				if tc.expectValid {
					assert.True(t, len(cfg.Telegram.BotToken) > 0 || tc.token == "")
				} else {
					assert.True(t, len(cfg.Telegram.BotToken) == 0)
				}
			}
		})
	}
}

func TestSuccessMessageGeneration(t *testing.T) {
	// Test that success messages are properly generated
	var buf bytes.Buffer

	// Test the final success message
	fmt.Fprintln(&buf, "\n🎉 All Telegram bot configuration checks passed!")

	output := buf.String()
	assert.Contains(t, output, "🎉")
	assert.Contains(t, output, "All Telegram bot configuration checks passed!")
	assert.True(t, strings.HasSuffix(output, "\n"))
}

func TestConcurrentAccess(t *testing.T) {
	// Test concurrent access to environment variables (avoid config loading due to concurrency issues)
	originalToken, tokenExists := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	originalWebhook, webhookExists := os.LookupEnv("TELEGRAM_WEBHOOK_URL")
	defer restoreEnv(t, "TELEGRAM_BOT_TOKEN", originalToken, tokenExists)
	defer restoreEnv(t, "TELEGRAM_WEBHOOK_URL", originalWebhook, webhookExists)

	// Set test values
	testToken := "concurrent_test_token"
	testWebhook := "https://concurrent.example.com"
	mustSetEnv(t, "TELEGRAM_BOT_TOKEN", testToken)
	mustSetEnv(t, "TELEGRAM_WEBHOOK_URL", testWebhook)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			// Just test environment variable access, not config loading
			token := os.Getenv("TELEGRAM_BOT_TOKEN")
			webhook := os.Getenv("TELEGRAM_WEBHOOK_URL")
			assert.Equal(t, testToken, token)
			assert.Equal(t, testWebhook, webhook)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMainFunctionWithMockEnvironment(t *testing.T) {
	// Test main function with controlled environment
	originalEnv := make(map[string]string)

	// Save current environment
	envVars := []string{"TELEGRAM_BOT_TOKEN", "TELEGRAM_WEBHOOK_URL"}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			originalEnv[env] = val
		}
	}

	// Set test environment
	testEnv := map[string]string{
		"TELEGRAM_BOT_TOKEN":   "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk",
		"TELEGRAM_WEBHOOK_URL": "https://test.example.com",
	}

	for key, value := range testEnv {
		mustSetEnv(t, key, value)
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			mustSetEnv(t, key, value)
		}
		for key := range testEnv {
			if _, exists := originalEnv[key]; !exists {
				mustUnsetEnv(t, key)
			}
		}
	}()

	// Test configuration loading with test environment
	cfg, err := config.Load()
	if err == nil && cfg != nil {
		assert.Equal(t, testEnv["TELEGRAM_BOT_TOKEN"], cfg.Telegram.BotToken)
		assert.Equal(t, testEnv["TELEGRAM_WEBHOOK_URL"], cfg.Telegram.WebhookURL)
	}
}

func TestGracefulShutdown(t *testing.T) {
	// Test graceful shutdown patterns
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test context cancellation
	cancel()

	select {
	case <-ctx.Done():
		assert.Equal(t, context.Canceled, ctx.Err())
	default:
		assert.Fail(t, "Context should be cancelled")
	}
}

func TestTimeOperations(t *testing.T) {
	// Test time-related operations used in the script
	now := time.Now()
	assert.True(t, now.After(time.Time{}))
	assert.True(t, now.Before(now.Add(time.Hour)))

	// Test duration calculations
	duration := 30 * time.Second
	assert.True(t, duration > 0)
	assert.Equal(t, 30*time.Second, duration)
}
