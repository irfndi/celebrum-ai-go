package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Struct(t *testing.T) {
	config := Config{
		Environment: "test",
		LogLevel:    "debug",
		Server: ServerConfig{
			Port:           8080,
			AllowedOrigins: []string{"http://localhost:3000"},
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "postgres",
			Password:        "password",
			DBName:          "test_db",
			SSLMode:         "disable",
			DatabaseURL:     "postgres://user:pass@localhost/db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: "300s",
			ConnMaxIdleTime: "60s",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "redis_pass",
			DB:       0,
		},
		CCXT: CCXTConfig{
			ServiceURL: "http://localhost:3001",
			Timeout:    30,
		},
		Telegram: TelegramConfig{
			BotToken:   "test_token",
			WebhookURL: "https://example.com/webhook",
		},
	}

	assert.Equal(t, "test", config.Environment)
	assert.Equal(t, "debug", config.LogLevel)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, []string{"http://localhost:3000"}, config.Server.AllowedOrigins)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "postgres", config.Database.User)
	assert.Equal(t, "password", config.Database.Password)
	assert.Equal(t, "test_db", config.Database.DBName)
	assert.Equal(t, "disable", config.Database.SSLMode)
	assert.Equal(t, "postgres://user:pass@localhost/db", config.Database.DatabaseURL)
	assert.Equal(t, 25, config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Database.MaxIdleConns)
	assert.Equal(t, "300s", config.Database.ConnMaxLifetime)
	assert.Equal(t, "60s", config.Database.ConnMaxIdleTime)
	assert.Equal(t, "localhost", config.Redis.Host)
	assert.Equal(t, 6379, config.Redis.Port)
	assert.Equal(t, "redis_pass", config.Redis.Password)
	assert.Equal(t, 0, config.Redis.DB)
	assert.Equal(t, "http://localhost:3000", config.CCXT.ServiceURL)
	assert.Equal(t, 30, config.CCXT.Timeout)
	assert.Equal(t, "test_token", config.Telegram.BotToken)
	assert.Equal(t, "https://example.com/webhook", config.Telegram.WebhookURL)
}

func TestServerConfig_Struct(t *testing.T) {
	config := ServerConfig{
		Port:           9000,
		AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
	}

	assert.Equal(t, 9000, config.Port)
	assert.Equal(t, []string{"http://localhost:3000", "https://example.com"}, config.AllowedOrigins)
}

func TestDatabaseConfig_Struct(t *testing.T) {
	config := DatabaseConfig{
		Host:            "db.example.com",
		Port:            5433,
		User:            "dbuser",
		Password:        "dbpass",
		DBName:          "production_db",
		SSLMode:         "require",
		DatabaseURL:     "postgres://user:pass@db.example.com/production_db",
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: "600s",
		ConnMaxIdleTime: "120s",
	}

	assert.Equal(t, "db.example.com", config.Host)
	assert.Equal(t, 5433, config.Port)
	assert.Equal(t, "dbuser", config.User)
	assert.Equal(t, "dbpass", config.Password)
	assert.Equal(t, "production_db", config.DBName)
	assert.Equal(t, "require", config.SSLMode)
	assert.Equal(t, "postgres://user:pass@db.example.com/production_db", config.DatabaseURL)
	assert.Equal(t, 50, config.MaxOpenConns)
	assert.Equal(t, 10, config.MaxIdleConns)
	assert.Equal(t, "600s", config.ConnMaxLifetime)
	assert.Equal(t, "120s", config.ConnMaxIdleTime)
}

func TestRedisConfig_Struct(t *testing.T) {
	config := RedisConfig{
		Host:     "redis.example.com",
		Port:     6380,
		Password: "redis_secret",
		DB:       1,
	}

	assert.Equal(t, "redis.example.com", config.Host)
	assert.Equal(t, 6380, config.Port)
	assert.Equal(t, "redis_secret", config.Password)
	assert.Equal(t, 1, config.DB)
}

func TestCCXTConfig_Struct(t *testing.T) {
	config := CCXTConfig{
		ServiceURL: "http://ccxt.example.com:3000",
		Timeout:    60,
	}

	assert.Equal(t, "http://ccxt.example.com:3000", config.ServiceURL)
	assert.Equal(t, 60, config.Timeout)
}

func TestTelegramConfig_Struct(t *testing.T) {
	config := TelegramConfig{
		BotToken:   "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk",
		WebhookURL: "https://api.example.com/telegram/webhook",
	}

	assert.Equal(t, "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk", config.BotToken)
	assert.Equal(t, "https://api.example.com/telegram/webhook", config.WebhookURL)
}

func TestLoad_WithDefaults(t *testing.T) {
	// Clear any existing environment variables that might interfere
	os.Clearenv()

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test default values
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, []string{"http://localhost:3000"}, config.Server.AllowedOrigins)
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "postgres", config.Database.User)
	assert.Equal(t, "postgres", config.Database.Password)
	assert.Equal(t, "celebrum_ai", config.Database.DBName)
	assert.Equal(t, "disable", config.Database.SSLMode)
	assert.Equal(t, "", config.Database.DatabaseURL)
	assert.Equal(t, 25, config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Database.MaxIdleConns)
	assert.Equal(t, "300s", config.Database.ConnMaxLifetime)
	assert.Equal(t, "60s", config.Database.ConnMaxIdleTime)
	assert.Equal(t, "localhost", config.Redis.Host)
	assert.Equal(t, 6379, config.Redis.Port)
	assert.Equal(t, "", config.Redis.Password)
	assert.Equal(t, 0, config.Redis.DB)
	assert.Equal(t, "http://localhost:3000", config.CCXT.ServiceURL)
	assert.Equal(t, 30, config.CCXT.Timeout)
	assert.Equal(t, "", config.Telegram.BotToken)
	assert.Equal(t, "", config.Telegram.WebhookURL)
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	_ = os.Setenv("ENVIRONMENT", "production")
	_ = os.Setenv("LOG_LEVEL", "error")
	_ = os.Setenv("SERVER_PORT", "9000")
	_ = os.Setenv("DATABASE_HOST", "prod-db.example.com")
	_ = os.Setenv("DATABASE_PORT", "5433")
	_ = os.Setenv("DATABASE_USER", "prod_user")
	_ = os.Setenv("DATABASE_PASSWORD", "prod_pass")
	_ = os.Setenv("DATABASE_DBNAME", "prod_db")
	_ = os.Setenv("DATABASE_SSLMODE", "require")
	_ = os.Setenv("REDIS_HOST", "prod-redis.example.com")
	_ = os.Setenv("REDIS_PORT", "6380")
	_ = os.Setenv("REDIS_PASSWORD", "redis_prod_pass")
	_ = os.Setenv("REDIS_DB", "1")
	_ = os.Setenv("CCXT_SERVICE_URL", "http://prod-ccxt.example.com:3000")
	_ = os.Setenv("CCXT_TIMEOUT", "60")
	_ = os.Setenv("TELEGRAM_BOT_TOKEN", "prod_bot_token")
	_ = os.Setenv("TELEGRAM_WEBHOOK_URL", "https://prod-api.example.com/webhook")

	defer func() {
		// Clean up environment variables
		_ = os.Unsetenv("ENVIRONMENT")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("SERVER_PORT")
		_ = os.Unsetenv("DATABASE_HOST")
		_ = os.Unsetenv("DATABASE_PORT")
		_ = os.Unsetenv("DATABASE_USER")
		_ = os.Unsetenv("DATABASE_PASSWORD")
		_ = os.Unsetenv("DATABASE_DBNAME")
		_ = os.Unsetenv("DATABASE_SSLMODE")
		_ = os.Unsetenv("REDIS_HOST")
		_ = os.Unsetenv("REDIS_PORT")
		_ = os.Unsetenv("REDIS_PASSWORD")
		_ = os.Unsetenv("REDIS_DB")
		_ = os.Unsetenv("CCXT_SERVICE_URL")
		_ = os.Unsetenv("CCXT_TIMEOUT")
		_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
		_ = os.Unsetenv("TELEGRAM_WEBHOOK_URL")
	}()

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Test environment variable values
	assert.Equal(t, "production", config.Environment)
	assert.Equal(t, "error", config.LogLevel)
	assert.Equal(t, 9000, config.Server.Port)
	assert.Equal(t, "prod-db.example.com", config.Database.Host)
	assert.Equal(t, 5433, config.Database.Port)
	assert.Equal(t, "prod_user", config.Database.User)
	assert.Equal(t, "prod_pass", config.Database.Password)
	assert.Equal(t, "prod_db", config.Database.DBName)
	assert.Equal(t, "require", config.Database.SSLMode)
	assert.Equal(t, "prod-redis.example.com", config.Redis.Host)
	assert.Equal(t, 6380, config.Redis.Port)
	assert.Equal(t, "redis_prod_pass", config.Redis.Password)
	assert.Equal(t, 1, config.Redis.DB)
	assert.Equal(t, "http://prod-ccxt.example.com:3000", config.CCXT.ServiceURL)
	assert.Equal(t, 60, config.CCXT.Timeout)
	assert.Equal(t, "prod_bot_token", config.Telegram.BotToken)
	assert.Equal(t, "https://prod-api.example.com/webhook", config.Telegram.WebhookURL)
}
