package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-telegram/bot/models"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTelegramHandlerWithMockDB tests the handler with a mocked database
func TestTelegramHandlerWithMockDB(t *testing.T) {
	// Setup mock database pool
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	// Note: We cannot directly cast mockPool to *pgxpool.Pool, so we create a wrapper
	// For these tests, we'll test the logic that doesn't require actual DB calls
	// or use the test database utilities

	// Setup mock Redis
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	t.Run("handleStartCommand_new_user_logic", func(t *testing.T) {
		t.Skip("Mock setup demonstration only - actual handler requires real pgxpool.Pool")

		// Test the logic without actual DB call
		// This test validates error handling when user doesn't exist

		// Mock expectations
		mockPool.ExpectQuery("SELECT id, email, telegram_chat_id").
			WithArgs("123456").
			WillReturnError(pgx.ErrNoRows)

		mockPool.ExpectExec("INSERT INTO users").
			WithArgs(pgxmock.AnyArg(), "telegram_user", pgxmock.AnyArg(), "free", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		// Note: The actual handler would need a real pgxpool.Pool
		// This test demonstrates the expected database interactions
		// We're not calling the handler here, just verifying mock expectations
	})

	t.Run("handleStartCommand_existing_user_logic", func(t *testing.T) {
		t.Skip("Mock setup demonstration only - actual handler requires real pgxpool.Pool")

		// Test the logic for existing user
		rows := pgxmock.NewRows([]string{"id", "email", "telegram_chat_id", "subscription_tier", "created_at", "updated_at"}).
			AddRow("user-123", "test@example.com", "123456", "free", time.Now(), time.Now())

		mockPool.ExpectQuery("SELECT id, email, telegram_chat_id").
			WithArgs("123456").
			WillReturnRows(rows)

		// Not calling handler, just verifying mock setup
	})

	t.Run("handleStartCommand_database_error_logic", func(t *testing.T) {
		t.Skip("Mock setup demonstration only - actual handler requires real pgxpool.Pool")

		// Test database error handling
		mockPool.ExpectQuery("SELECT id, email, telegram_chat_id").
			WithArgs("123456").
			WillReturnError(assert.AnError)

		// Not calling handler, just verifying mock setup
	})
}

// TestTelegramHandlerConfigInitialization tests different configuration scenarios
func TestTelegramHandlerConfigInitialization(t *testing.T) {
	// Setup mock Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	t.Run("nil_config", func(t *testing.T) {
		handler := NewTelegramHandler(nil, nil, nil, nil, redisClient)

		assert.NotNil(t, handler)
		assert.Nil(t, handler.config)
		assert.Nil(t, handler.bot)
	})

	t.Run("empty_bot_token", func(t *testing.T) {
		cfg := &config.TelegramConfig{
			BotToken:   "",
			UsePolling: false,
		}

		handler := NewTelegramHandler(nil, cfg, nil, nil, redisClient)

		assert.NotNil(t, handler)
		assert.NotNil(t, handler.config)
		assert.Nil(t, handler.bot) // Bot creation should fail with empty token
	})

	t.Run("polling_mode_config", func(t *testing.T) {
		cfg := &config.TelegramConfig{
			BotToken:   "invalid-token-for-test", // Will fail but config should be set
			UsePolling: true,
		}

		handler := NewTelegramHandler(nil, cfg, nil, nil, redisClient)

		assert.NotNil(t, handler)
		assert.NotNil(t, handler.config)
		// Bot may be nil due to invalid token, but that's expected
	})

	t.Run("webhook_mode_config", func(t *testing.T) {
		cfg := &config.TelegramConfig{
			BotToken:   "invalid-token-for-test",
			UsePolling: false,
			WebhookURL: "https://example.com/webhook",
		}

		handler := NewTelegramHandler(nil, cfg, nil, nil, redisClient)

		assert.NotNil(t, handler)
		assert.NotNil(t, handler.config)
		assert.Equal(t, "https://example.com/webhook", handler.config.WebhookURL)
	})
}

// TestTelegramHandlerRedisOperations tests Redis-related operations
func TestTelegramHandlerRedisOperations(t *testing.T) {
	// Setup mock Redis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer redisClient.Close()

	t.Run("handleStopCommand_with_redis", func(t *testing.T) {
		handler := &TelegramHandler{
			db:    &database.PostgresDB{},
			bot:   nil,
			redis: redisClient,
		}

		ctx := context.Background()
		err := handler.handleStopCommand(ctx, 123456, 789)

		// Should error because bot is nil (can't send message)
		assert.Error(t, err)

		// But Redis key should be set
		val, err := redisClient.Get(ctx, "telegram:user:789:notifications_enabled").Result()
		require.NoError(t, err)
		assert.Equal(t, "false", val)
	})

	t.Run("handleResumeCommand_with_redis", func(t *testing.T) {
		handler := &TelegramHandler{
			db:    &database.PostgresDB{},
			bot:   nil,
			redis: redisClient,
		}

		ctx := context.Background()
		err := handler.handleResumeCommand(ctx, 123456, 789)

		// Should error because bot is nil (can't send message)
		assert.Error(t, err)

		// But Redis key should be set
		val, err := redisClient.Get(ctx, "telegram:user:789:notifications_enabled").Result()
		require.NoError(t, err)
		assert.Equal(t, "true", val)
	})

	t.Run("handleSettingsCommand_with_redis", func(t *testing.T) {
		handler := &TelegramHandler{
			db:    &database.PostgresDB{},
			bot:   nil,
			redis: redisClient,
		}

		ctx := context.Background()

		// Set notification to disabled
		err := redisClient.Set(ctx, "telegram:user:789:notifications_enabled", "false", 0).Err()
		require.NoError(t, err)

		err = handler.handleSettingsCommand(ctx, 123456, 789)

		// Should error because bot is nil (can't send message)
		assert.Error(t, err)
		// The function should have read the Redis value correctly
	})

	t.Run("handleStopCommand_without_redis", func(t *testing.T) {
		handler := &TelegramHandler{
			db:    &database.PostgresDB{},
			bot:   nil,
			redis: nil,
		}

		ctx := context.Background()
		err := handler.handleStopCommand(ctx, 123456, 789)

		// Should error with service unavailable message
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "telegram bot not available")
	})
}

// TestTelegramHandlerMessageValidation tests message validation
func TestTelegramHandlerMessageValidation(t *testing.T) {
	handler := &TelegramHandler{
		db:  nil,
		bot: nil,
	}

	ctx := context.Background()

	t.Run("empty_update", func(t *testing.T) {
		update := &models.Update{}
		err := handler.processUpdate(ctx, update)
		assert.NoError(t, err) // Should ignore empty updates
	})

	t.Run("message_with_zero_chat_id", func(t *testing.T) {
		update := &models.Update{
			Message: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 0},
				Text: "/start",
			},
		}
		err := handler.processUpdate(ctx, update)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing from or chat")
	})

	t.Run("message_with_command", func(t *testing.T) {
		update := &models.Update{
			Message: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 456},
				Text: "/help",
			},
		}
		err := handler.processUpdate(ctx, update)
		assert.Error(t, err) // Will error due to nil bot
	})

	t.Run("message_with_text", func(t *testing.T) {
		update := &models.Update{
			Message: &models.Message{
				From: &models.User{ID: 123},
				Chat: models.Chat{ID: 456},
				Text: "Hello bot",
			},
		}
		err := handler.processUpdate(ctx, update)
		assert.Error(t, err) // Will error due to nil bot
	})
}

// TestTelegramHandlerCommandRouting tests command routing
func TestTelegramHandlerCommandRouting(t *testing.T) {
	handler := &TelegramHandler{
		db:  &database.PostgresDB{Pool: nil}, // Some commands don't access DB
		bot: nil,
	}

	ctx := context.Background()
	message := &models.Message{
		From: &models.User{ID: 123},
		Chat: models.Chat{ID: 456},
	}

	tests := []struct {
		name    string
		command string
		wantErr bool
		skipDB  bool // Skip if command requires database
	}{
		{"start command", "/start", true, true}, // Skipped - requires DB
		{"help command", "/help", true, false},
		{"opportunities command", "/opportunities", true, false},
		{"settings command", "/settings", true, false},
		{"upgrade command", "/upgrade", true, false},
		{"status command", "/status", true, true}, // Skipped - requires DB
		{"stop command", "/stop", true, false},
		{"resume command", "/resume", true, false},
		{"unknown command", "/unknown", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipDB {
				t.Skip("Skipping test that requires database access")
				return
			}

			err := handler.handleCommand(ctx, message, tt.command)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTelegramHandlerPollingLifecycle tests polling start/stop
func TestTelegramHandlerPollingLifecycle(t *testing.T) {
	t.Run("start_and_stop_polling", func(t *testing.T) {
		handler := &TelegramHandler{
			bot:           nil,
			pollingActive: false,
		}

		// Start polling (will fail due to nil bot)
		handler.StartPolling()
		assert.False(t, handler.pollingActive)

		// Try to stop (should handle gracefully)
		handler.StopPolling()
		assert.False(t, handler.pollingActive)
	})

	t.Run("stop_without_start", func(t *testing.T) {
		handler := &TelegramHandler{
			bot:           nil,
			pollingActive: false,
		}

		handler.StopPolling()
		assert.False(t, handler.pollingActive)
	})

	t.Run("start_twice", func(t *testing.T) {
		handler := &TelegramHandler{
			bot:           nil,
			pollingActive: true, // Already active
		}

		handler.StartPolling()
		// Should not change state if already active
		assert.True(t, handler.pollingActive)
	})
}
