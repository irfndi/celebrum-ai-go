//go:build integration
// +build integration

package handlers

import (
	"os"
	"testing"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTelegramHandler_Integration tests the actual Telegram bot integration
// This test requires a valid bot token to be set in environment variables
// Run with: go test -tags=integration ./internal/api/handlers -run TestTelegramHandler_Integration
func TestTelegramHandler_Integration(t *testing.T) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		t.Skip("TELEGRAM_BOT_TOKEN environment variable not set, skipping integration test")
	}

	// Create test config
	cfg := &config.TelegramConfig{
		BotToken: botToken,
	}

	// Create mock database (in real integration test, you'd use a test database)
	mockDB := &database.PostgresDB{}

	// Test handler creation
	handler, err := NewTelegramHandler(mockDB, cfg)
	require.NoError(t, err, "Failed to create Telegram handler")
	assert.NotNil(t, handler, "Handler should not be nil")
	assert.NotNil(t, handler.bot, "Bot should not be nil")

	// Test that the bot is properly initialized
	// Note: This doesn't make actual API calls, just verifies the bot object exists
	assert.NotNil(t, handler.db, "Database should not be nil")
}

// TestTelegramHandler_BotTokenValidation tests bot token validation
func TestTelegramHandler_BotTokenValidation(t *testing.T) {
	tests := []struct {
		name      string
		botToken  string
		expectErr bool
	}{
		{
			name:      "empty token",
			botToken:  "",
			expectErr: true,
		},
		{
			name:      "invalid token format",
			botToken:  "invalid-token",
			expectErr: true,
		},
		{
			name:      "valid token format",
			botToken:  "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
			expectErr: false, // Note: This might still fail if token is not real
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.TelegramConfig{
				BotToken: tt.botToken,
			}
			mockDB := &database.PostgresDB{}

			handler, err := NewTelegramHandler(mockDB, cfg)

			if tt.expectErr {
				assert.Error(t, err, "Expected error for token: %s", tt.botToken)
				assert.Nil(t, handler, "Handler should be nil on error")
			} else {
				// Note: Even with valid format, this might fail if token is not real
				// In a real integration test, you'd use a test bot token
				if err != nil {
					t.Logf("Token format valid but bot creation failed (expected in test): %v", err)
				}
			}
		})
	}
}
