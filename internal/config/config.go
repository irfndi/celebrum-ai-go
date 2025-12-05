package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string           `mapstructure:"environment"`
	LogLevel    string           `mapstructure:"log_level"`
	Server      ServerConfig     `mapstructure:"server"`
	Database    DatabaseConfig   `mapstructure:"database"`
	Redis       RedisConfig      `mapstructure:"redis"`
	CCXT        CCXTConfig       `mapstructure:"ccxt"`
	Telegram    TelegramConfig   `mapstructure:"telegram"`
	Telemetry   TelemetryConfig  `mapstructure:"telemetry"`
	Sentry      SentryConfig     `mapstructure:"sentry"`
	Cleanup     CleanupConfig    `mapstructure:"cleanup"`
	Backfill    BackfillConfig   `mapstructure:"backfill"`
	MarketData  MarketDataConfig `mapstructure:"market_data"`
	Arbitrage   ArbitrageConfig  `mapstructure:"arbitrage"`
	Blacklist   BlacklistConfig  `mapstructure:"blacklist"`
	Auth        AuthConfig       `mapstructure:"auth"`
}

type ServerConfig struct {
	Port           int      `mapstructure:"port"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"dbname"`
	SSLMode         string `mapstructure:"sslmode"`
	DatabaseURL     string `mapstructure:"database_url"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime string `mapstructure:"conn_max_idle_time"`
	// PostgreSQL 18 specific optimizations
	ApplicationName       string `mapstructure:"application_name"`
	ConnectTimeout        int    `mapstructure:"connect_timeout"`
	StatementTimeout      int    `mapstructure:"statement_timeout"`
	QueryTimeout          int    `mapstructure:"query_timeout"`
	PoolTimeout           int    `mapstructure:"pool_timeout"`
	PoolHealthCheckPeriod int    `mapstructure:"pool_health_check_period"`
	PoolMaxLifetime       int    `mapstructure:"pool_max_lifetime"`
	PoolIdleTimeout       int    `mapstructure:"pool_idle_timeout"`
	EnableAsync           bool   `mapstructure:"enable_async"`
	AsyncBatchSize        int    `mapstructure:"async_batch_size"`
	AsyncConcurrency      int    `mapstructure:"async_concurrency"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type CCXTConfig struct {
	ServiceURL string `mapstructure:"service_url"`
	Timeout    int    `mapstructure:"timeout"`
}

type TelegramConfig struct {
	BotToken       string `mapstructure:"bot_token"`
	WebhookURL     string `mapstructure:"webhook_url"`
	UsePolling     bool   `mapstructure:"use_polling"`
	PollingOffset  int    `mapstructure:"polling_offset"`
	PollingLimit   int    `mapstructure:"polling_limit"`
	PollingTimeout int    `mapstructure:"polling_timeout"`
}

type TelemetryConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`
	LogLevel       string `mapstructure:"log_level"`
}

type SentryConfig struct {
	Enabled            bool    `mapstructure:"enabled"`
	DSN                string  `mapstructure:"dsn"`
	Environment        string  `mapstructure:"environment"`
	Release            string  `mapstructure:"release"`
	TracesSampleRate   float64 `mapstructure:"traces_sample_rate"`
	ProfilesSampleRate float64 `mapstructure:"profiles_sample_rate"`
}

type CleanupConfig struct {
	MarketData             CleanupDataConfig      `mapstructure:"market_data"`
	FundingRates           CleanupDataConfig      `mapstructure:"funding_rates"`
	ArbitrageOpportunities CleanupArbitrageConfig `mapstructure:"arbitrage_opportunities"`
	IntervalMinutes        int                    `mapstructure:"interval"`
	EnableSmartCleanup     bool                   `mapstructure:"enable_smart_cleanup"`
}

type CleanupDataConfig struct {
	RetentionHours int `mapstructure:"retention_hours"`
	DeletionHours  int `mapstructure:"deletion_hours"`
}

type CleanupArbitrageConfig struct {
	RetentionHours int `mapstructure:"retention_hours"`
}

type BackfillConfig struct {
	Enabled               bool `mapstructure:"enabled"`
	BackfillHours         int  `mapstructure:"backfill_hours"`
	MinDataThresholdHours int  `mapstructure:"min_data_threshold_hours"`
	BatchSize             int  `mapstructure:"batch_size"`
	DelayBetweenBatches   int  `mapstructure:"delay_between_batches"`
}

type MarketDataConfig struct {
	CollectionInterval string   `mapstructure:"collection_interval"`
	BatchSize          int      `mapstructure:"batch_size"`
	MaxRetries         int      `mapstructure:"max_retries"`
	Timeout            string   `mapstructure:"timeout"`
	Exchanges          []string `mapstructure:"exchanges"`
}

type ArbitrageConfig struct {
	MinProfitThreshold float64  `mapstructure:"min_profit_threshold"`
	MaxTradeAmount     float64  `mapstructure:"max_trade_amount"`
	CheckInterval      string   `mapstructure:"check_interval"`
	EnabledPairs       []string `mapstructure:"enabled_pairs"`
	Enabled            bool     `mapstructure:"enabled"`
	IntervalSeconds    int      `mapstructure:"interval_seconds"`
	MaxAgeMinutes      int      `mapstructure:"max_age_minutes"`
	BatchSize          int      `mapstructure:"batch_size"`
}

type BlacklistConfig struct {
	TTL             string `mapstructure:"ttl"`               // Default TTL for blacklisted symbols (e.g., "24h")
	ShortTTL        string `mapstructure:"short_ttl"`         // Short TTL for temporary issues (e.g., "1h")
	LongTTL         string `mapstructure:"long_ttl"`          // Long TTL for persistent issues (e.g., "72h")
	UseRedis        bool   `mapstructure:"use_redis"`         // Whether to use Redis for blacklist persistence
	RetryAfterClear bool   `mapstructure:"retry_after_clear"` // Whether to retry symbols after blacklist expires
}

type AuthConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Set default values
	setDefaults()

	// Enable environment variable support
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Map legacy environment variables for JWT secret
	_ = viper.BindEnv("auth.jwt_secret", "JWT_SECRET")
	_ = viper.BindEnv("security.jwt_secret", "JWT_SECRET")

	// Bind standard DATABASE_URL
	_ = viper.BindEnv("database.database_url", "DATABASE_URL")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults and environment variables
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Backfill JWT secret from legacy security configuration if needed
	if config.Auth.JWTSecret == "" {
		config.Auth.JWTSecret = strings.TrimSpace(viper.GetString("auth.jwt_secret"))
	}
	if config.Auth.JWTSecret == "" {
		config.Auth.JWTSecret = strings.TrimSpace(viper.GetString("security.jwt_secret"))
	}

	// Validate critical security settings
	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func setDefaults() {
	// Environment
	viper.SetDefault("environment", "development")
	viper.SetDefault("log_level", "info")

	// Server
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.allowed_origins", []string{"http://localhost:3000"})

	// Set database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	// Set default password for testing (must be overridden in production)
	viper.SetDefault("database.password", "change-me-in-production")
	viper.SetDefault("database.dbname", "celebrum_ai")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.database_url", "")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "300s")
	viper.SetDefault("database.conn_max_idle_time", "60s")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// CCXT
	viper.SetDefault("ccxt.service_url", "http://localhost:3000")
	viper.SetDefault("ccxt.timeout", 30)

	// Telegram
	viper.SetDefault("telegram.bot_token", "")
	viper.SetDefault("telegram.webhook_url", "")
	viper.SetDefault("telegram.use_polling", false)
	viper.SetDefault("telegram.polling_offset", 0)
	viper.SetDefault("telegram.polling_limit", 100)
	viper.SetDefault("telegram.polling_timeout", 60)

	// Telemetry
	viper.SetDefault("telemetry.enabled", true)
	viper.SetDefault("telemetry.service_name", "github.com/irfandi/celebrum-ai-go")
	viper.SetDefault("telemetry.service_version", "1.0.0")
	viper.SetDefault("telemetry.log_level", "info")

	// Sentry
	viper.SetDefault("sentry.enabled", false)
	viper.SetDefault("sentry.dsn", "")
	viper.SetDefault("sentry.environment", viper.GetString("environment"))
	viper.SetDefault("sentry.release", "")
	viper.SetDefault("sentry.traces_sample_rate", 0.2)
	viper.SetDefault("sentry.profiles_sample_rate", 0.0)

	// Cleanup - Enhanced configuration with smart cleanup
	viper.SetDefault("cleanup.market_data.retention_hours", 36)
	viper.SetDefault("cleanup.market_data.deletion_hours", 12)
	viper.SetDefault("cleanup.funding_rates.retention_hours", 36)
	viper.SetDefault("cleanup.funding_rates.deletion_hours", 12)
	viper.SetDefault("cleanup.arbitrage_opportunities.retention_hours", 72)
	viper.SetDefault("cleanup.interval", 60)
	viper.SetDefault("cleanup.enable_smart_cleanup", true)

	// Backfill
	viper.SetDefault("backfill.enabled", true)
	viper.SetDefault("backfill.backfill_hours", 6)
	viper.SetDefault("backfill.min_data_threshold_hours", 4)
	viper.SetDefault("backfill.batch_size", 5)
	viper.SetDefault("backfill.delay_between_batches", 500)

	// Market Data
	viper.SetDefault("market_data.collection_interval", "5m")
	viper.SetDefault("market_data.batch_size", 100)
	viper.SetDefault("market_data.max_retries", 3)
	viper.SetDefault("market_data.timeout", "15s")
	viper.SetDefault("market_data.exchanges", []string{"binance", "coinbase", "kraken", "bitfinex", "huobi"})

	// Arbitrage
	viper.SetDefault("arbitrage.enabled", true)
	viper.SetDefault("arbitrage.interval_seconds", 60)
	viper.SetDefault("arbitrage.min_profit_threshold", 0.5)
	viper.SetDefault("arbitrage.max_age_minutes", 30)
	viper.SetDefault("arbitrage.batch_size", 100)
	viper.SetDefault("arbitrage.max_trade_amount", 1000.0)
	viper.SetDefault("arbitrage.check_interval", "2m")
	viper.SetDefault("arbitrage.enabled_pairs", []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT"})

	// Blacklist
	viper.SetDefault("blacklist.ttl", "24h")
	viper.SetDefault("blacklist.short_ttl", "1h")
	viper.SetDefault("blacklist.long_ttl", "72h")
	viper.SetDefault("blacklist.use_redis", true)
	viper.SetDefault("blacklist.retry_after_clear", true)

	// Auth
	viper.SetDefault("auth.jwt_secret", "")
}

// GetServiceURL returns the CCXT service URL
func (c *CCXTConfig) GetServiceURL() string {
	return c.ServiceURL
}

// GetTimeout returns the CCXT service timeout in seconds
func (c *CCXTConfig) GetTimeout() int {
	return c.Timeout
}

// validateConfig validates critical security and operational settings
func validateConfig(config *Config) error {
	// Validate JWT secret for production environments
	if config.Environment == "production" || config.Environment == "staging" {
		if config.Auth.JWTSecret == "" {
			return fmt.Errorf("JWT_SECRET cannot be empty in %s environment. Please set a secure JWT secret", config.Environment)
		}

		// Validate JWT secret complexity (minimum 32 characters for security)
		if len(config.Auth.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters long in %s environment for security", config.Environment)
		}

		// Check for common insecure JWT secrets
		insecureSecrets := []string{
			"test-jwt-secret",
			"secret",
			"jwt-secret",
			"default-secret",
			"test-jwt-secret-for-ci",
			"test-jwt-secret-for-ci-only",
			"changeme",
			"password",
			"123456",
		}

		for _, insecure := range insecureSecrets {
			if config.Auth.JWTSecret == insecure {
				return fmt.Errorf("JWT_SECRET '%s' is insecure and not allowed in %s environment. Please use a secure, randomly generated secret", insecure, config.Environment)
			}
		}
	}

	return nil
}
