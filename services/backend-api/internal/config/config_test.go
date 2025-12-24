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
		Fees: FeesConfig{
			DefaultTakerFee: 0.001,
			DefaultMakerFee: 0.0008,
		},
		Analytics: AnalyticsConfig{
			EnableForecasting:       true,
			EnableCorrelation:       true,
			EnableRegimeDetection:   true,
			ForecastLookback:        120,
			ForecastHorizon:         8,
			CorrelationWindow:       200,
			CorrelationMinPoints:    30,
			RegimeShortWindow:       20,
			RegimeLongWindow:        60,
			VolatilityHighThreshold: 0.03,
			VolatilityLowThreshold:  0.005,
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
	assert.Equal(t, "http://localhost:3001", config.CCXT.ServiceURL)
	assert.Equal(t, 30, config.CCXT.Timeout)
	assert.Equal(t, "test_token", config.Telegram.BotToken)
	assert.Equal(t, "https://example.com/webhook", config.Telegram.WebhookURL)
	assert.Equal(t, 0.001, config.Fees.DefaultTakerFee)
	assert.Equal(t, 0.0008, config.Fees.DefaultMakerFee)
	assert.True(t, config.Analytics.EnableForecasting)
	assert.Equal(t, 120, config.Analytics.ForecastLookback)
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
	assert.Equal(t, "change-me-in-production", config.Database.Password)
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
	assert.Equal(t, "http://localhost:3001", config.CCXT.ServiceURL)
	assert.Equal(t, 30, config.CCXT.Timeout)
	assert.Equal(t, "", config.Telegram.BotToken)
	assert.Equal(t, "", config.Telegram.WebhookURL)
	assert.Equal(t, 0.001, config.Fees.DefaultTakerFee)
	assert.Equal(t, 0.001, config.Fees.DefaultMakerFee)
	assert.True(t, config.Analytics.EnableForecasting)
	assert.True(t, config.Analytics.EnableCorrelation)
	assert.True(t, config.Analytics.EnableRegimeDetection)
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	// Clear any existing environment variables and set new ones
	os.Clearenv()

	// Set environment variables - Viper converts nested keys to uppercase with underscores
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("SERVER_PORT", "9000")
	t.Setenv("DATABASE_HOST", "prod-db.example.com")
	t.Setenv("DATABASE_PORT", "5433")
	t.Setenv("DATABASE_USER", "prod_user")
	t.Setenv("DATABASE_PASSWORD", "prod_pass")
	t.Setenv("DATABASE_DBNAME", "prod_db")
	t.Setenv("DATABASE_SSLMODE", "require")
	t.Setenv("REDIS_HOST", "prod-redis.example.com")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "redis_prod_pass")
	t.Setenv("REDIS_DB", "1")
	t.Setenv("CCXT_SERVICE_URL", "http://prod-ccxt.example.com:3000")
	t.Setenv("CCXT_TIMEOUT", "60")
	t.Setenv("TELEGRAM_BOT_TOKEN", "prod_bot_token")
	t.Setenv("TELEGRAM_WEBHOOK_URL", "https://prod-api.example.com/webhook")
	t.Setenv("AUTH_JWT_SECRET", "ci-test-secret-key-should-be-32-chars!!")

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
	assert.Equal(t, "ci-test-secret-key-should-be-32-chars!!", config.Auth.JWTSecret)
}

func TestCCXTConfig_GetServiceURL(t *testing.T) {
	config := CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	assert.Equal(t, "http://localhost:3001", config.GetServiceURL())
}

func TestCCXTConfig_GetTimeout(t *testing.T) {
	config := CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    30,
	}

	assert.Equal(t, 30, config.GetTimeout())
}

func TestCCXTConfig_GetServiceURL_Empty(t *testing.T) {
	config := CCXTConfig{
		ServiceURL: "",
		Timeout:    30,
	}

	assert.Equal(t, "", config.GetServiceURL())
}

func TestCCXTConfig_GetTimeout_Zero(t *testing.T) {
	config := CCXTConfig{
		ServiceURL: "http://localhost:3001",
		Timeout:    0,
	}

	assert.Equal(t, 0, config.GetTimeout())
}

// TestValidateConfig_Production_EmptyJWTSecret validates that production requires JWT secret
func TestValidateConfig_Production_EmptyJWTSecret(t *testing.T) {
	config := &Config{
		Environment: "production",
		Auth: AuthConfig{
			JWTSecret: "",
		},
	}

	err := validateConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET cannot be empty")
}

// TestValidateConfig_Production_ShortJWTSecret validates that JWT secret must be at least 32 chars
func TestValidateConfig_Production_ShortJWTSecret(t *testing.T) {
	config := &Config{
		Environment: "production",
		Auth: AuthConfig{
			JWTSecret: "short-secret",
		},
	}

	err := validateConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 32 characters")
}

// TestValidateConfig_Production_InsecureJWTSecret validates that insecure secrets are rejected
func TestValidateConfig_Production_InsecureJWTSecret(t *testing.T) {
	insecureSecrets := []string{
		"test-jwt-secret",
		"changeme",
		"password",
	}

	for _, secret := range insecureSecrets {
		// Pad to 32 chars to pass length check
		paddedSecret := secret
		for len(paddedSecret) < 32 {
			paddedSecret += "x"
		}
		// But use original insecure secret
		config := &Config{
			Environment: "production",
			Auth: AuthConfig{
				JWTSecret: secret,
			},
		}

		err := validateConfig(config)
		// Should fail either due to length or insecure check
		require.Error(t, err, "insecure secret '%s' should be rejected", secret)
	}
}

// TestValidateConfig_Production_ValidJWTSecret validates that valid secrets pass
func TestValidateConfig_Production_ValidJWTSecret(t *testing.T) {
	config := &Config{
		Environment: "production",
		Auth: AuthConfig{
			JWTSecret: "this-is-a-secure-production-jwt-secret-key-123",
		},
	}

	err := validateConfig(config)
	require.NoError(t, err)
}

// TestValidateConfig_Staging_RequiresJWTSecret validates staging also requires JWT
func TestValidateConfig_Staging_RequiresJWTSecret(t *testing.T) {
	config := &Config{
		Environment: "staging",
		Auth: AuthConfig{
			JWTSecret: "",
		},
	}

	err := validateConfig(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET cannot be empty")
}

// TestValidateConfig_Development_AllowsEmptyJWTSecret validates dev allows empty JWT
func TestValidateConfig_Development_AllowsEmptyJWTSecret(t *testing.T) {
	config := &Config{
		Environment: "development",
		Auth: AuthConfig{
			JWTSecret: "",
		},
	}

	err := validateConfig(config)
	require.NoError(t, err)
}

// TestValidateConfig_Development_AllowsInsecureJWTSecret validates dev allows insecure secrets
func TestValidateConfig_Development_AllowsInsecureJWTSecret(t *testing.T) {
	config := &Config{
		Environment: "development",
		Auth: AuthConfig{
			JWTSecret: "test",
		},
	}

	err := validateConfig(config)
	require.NoError(t, err)
}

// TestLoad_WithDatabaseURL_Override validates DATABASE_URL can be set via env
func TestLoad_WithDatabaseURL_Override(t *testing.T) {
	os.Clearenv()

	t.Setenv("DATABASE_URL", "postgres://testuser:testpass@testhost:5432/testdb?sslmode=require")

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "postgres://testuser:testpass@testhost:5432/testdb?sslmode=require", config.Database.DatabaseURL)
}

// TestLoad_DatabaseDefaults validates database has sensible defaults
func TestLoad_DatabaseDefaults(t *testing.T) {
	os.Clearenv()

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify sensible defaults for database
	assert.Equal(t, "localhost", config.Database.Host)
	assert.Equal(t, 5432, config.Database.Port)
	assert.Equal(t, "postgres", config.Database.User)
	assert.NotEmpty(t, config.Database.Password, "Database password should have a default")
	assert.Equal(t, "celebrum_ai", config.Database.DBName)
	assert.Equal(t, "disable", config.Database.SSLMode)
	assert.Equal(t, 25, config.Database.MaxOpenConns)
	assert.Equal(t, 5, config.Database.MaxIdleConns)
}

// TestLoad_RedisDefaults validates Redis has sensible defaults
func TestLoad_RedisDefaults(t *testing.T) {
	os.Clearenv()

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "localhost", config.Redis.Host)
	assert.Equal(t, 6379, config.Redis.Port)
	assert.Equal(t, "", config.Redis.Password) // Redis password can be empty
	assert.Equal(t, 0, config.Redis.DB)
}

// TestConfig_EnvironmentVariableMappings validates env var naming conventions
func TestConfig_EnvironmentVariableMappings(t *testing.T) {
	os.Clearenv()

	// Test that both DATABASE_* and nested format work
	t.Setenv("DATABASE_HOST", "custom-host")
	t.Setenv("DATABASE_PORT", "5433")
	t.Setenv("DATABASE_USER", "custom-user")
	t.Setenv("DATABASE_PASSWORD", "custom-password")
	t.Setenv("DATABASE_DBNAME", "custom-db")

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "custom-host", config.Database.Host)
	assert.Equal(t, 5433, config.Database.Port)
	assert.Equal(t, "custom-user", config.Database.User)
	assert.Equal(t, "custom-password", config.Database.Password)
	assert.Equal(t, "custom-db", config.Database.DBName)
}

// TestConfig_JWTSecretFromEnvironment validates JWT_SECRET env var
func TestConfig_JWTSecretFromEnvironment(t *testing.T) {
	os.Clearenv()

	t.Setenv("JWT_SECRET", "env-jwt-secret-that-is-long-enough-for-production")

	config, err := Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "env-jwt-secret-that-is-long-enough-for-production", config.Auth.JWTSecret)
}

// TestConfig_DockerEnvironmentDefaults validates Docker-aware service defaults
func TestConfig_DockerEnvironmentDefaults(t *testing.T) {
	t.Run("Docker environment uses service names", func(t *testing.T) {
		os.Clearenv()

		t.Setenv("DOCKER_ENVIRONMENT", "true")
		t.Setenv("JWT_SECRET", "test-jwt-secret-that-is-long-enough-32chars")

		config, err := Load()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Should use Docker service names
		assert.Equal(t, "http://ccxt-service:3001", config.CCXT.ServiceURL)
		assert.Equal(t, "ccxt-service:50051", config.CCXT.GrpcAddress)
		assert.Equal(t, "http://telegram-service:3002", config.Telegram.ServiceURL)
		assert.Equal(t, "telegram-service:50052", config.Telegram.GrpcAddress)
	})

	t.Run("Coolify environment uses service names", func(t *testing.T) {
		os.Clearenv()

		t.Setenv("COOLIFY", "true")
		t.Setenv("JWT_SECRET", "test-jwt-secret-that-is-long-enough-32chars")

		config, err := Load()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Should use Docker service names
		assert.Equal(t, "http://ccxt-service:3001", config.CCXT.ServiceURL)
		assert.Equal(t, "ccxt-service:50051", config.CCXT.GrpcAddress)
	})

	t.Run("Non-Docker environment uses localhost", func(t *testing.T) {
		os.Clearenv()

		t.Setenv("DOCKER_ENVIRONMENT", "false")
		t.Setenv("COOLIFY", "false")
		t.Setenv("JWT_SECRET", "test-jwt-secret-that-is-long-enough-32chars")

		config, err := Load()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Should use localhost for HTTP, 127.0.0.1 for gRPC (to avoid IPv6 issues)
		assert.Equal(t, "http://localhost:3001", config.CCXT.ServiceURL)
		assert.Equal(t, "127.0.0.1:50051", config.CCXT.GrpcAddress)
		assert.Equal(t, "http://localhost:3002", config.Telegram.ServiceURL)
		assert.Equal(t, "127.0.0.1:50052", config.Telegram.GrpcAddress)
	})

	t.Run("Empty environment uses localhost", func(t *testing.T) {
		os.Clearenv()

		t.Setenv("JWT_SECRET", "test-jwt-secret-that-is-long-enough-32chars")

		config, err := Load()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Should use localhost for HTTP, 127.0.0.1 for gRPC (to avoid IPv6 issues)
		assert.Equal(t, "http://localhost:3001", config.CCXT.ServiceURL)
		assert.Equal(t, "127.0.0.1:50051", config.CCXT.GrpcAddress)
	})

	t.Run("Explicit env vars override Docker defaults", func(t *testing.T) {
		os.Clearenv()

		t.Setenv("DOCKER_ENVIRONMENT", "true")
		t.Setenv("JWT_SECRET", "test-jwt-secret-that-is-long-enough-32chars")
		t.Setenv("CCXT_SERVICE_URL", "http://custom-ccxt:9001")
		t.Setenv("CCXT_GRPC_ADDRESS", "custom-ccxt:59051")

		config, err := Load()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Explicit env vars should override defaults
		assert.Equal(t, "http://custom-ccxt:9001", config.CCXT.ServiceURL)
		assert.Equal(t, "custom-ccxt:59051", config.CCXT.GrpcAddress)
	})
}
