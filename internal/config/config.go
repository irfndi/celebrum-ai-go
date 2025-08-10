package config

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Environment string           `mapstructure:"environment"`
	LogLevel    string           `mapstructure:"log_level"`
	Server      ServerConfig     `mapstructure:"server"`
	Database    DatabaseConfig   `mapstructure:"database"`
	Redis       RedisConfig      `mapstructure:"redis"`
	CCXT        CCXTConfig       `mapstructure:"ccxt"`
	Telegram    TelegramConfig   `mapstructure:"telegram"`
	Cleanup     CleanupConfig    `mapstructure:"cleanup"`
	MarketData  MarketDataConfig `mapstructure:"market_data"`
	Arbitrage   ArbitrageConfig  `mapstructure:"arbitrage"`
	Security    SecurityConfig   `mapstructure:"security"`
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
	BotToken   string `mapstructure:"bot_token"`
	WebhookURL string `mapstructure:"webhook_url"`
}

type CleanupConfig struct {
	MarketDataRetentionHours  int `mapstructure:"market_data_retention_hours"`
	FundingRateRetentionHours int `mapstructure:"funding_rate_retention_hours"`
	ArbitrageRetentionHours   int `mapstructure:"arbitrage_retention_hours"`
	CleanupIntervalMinutes    int `mapstructure:"cleanup_interval_minutes"`
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
}

type SecurityConfig struct {
	JWTSecret  string `mapstructure:"jwt_secret" json:"-" yaml:"-"`
	JWTExpiry  string `mapstructure:"jwt_expiry"`
	BcryptCost int    `mapstructure:"bcrypt_cost"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Enable environment variable support
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Bind specific environment variables
	if err := viper.BindEnv("security.jwt_secret", "JWT_SECRET"); err != nil {
		return nil, fmt.Errorf("failed to bind JWT_SECRET environment variable: %w", err)
	}
	if err := viper.BindEnv("ccxt.service_url", "CCXT_SERVICE_URL"); err != nil {
		return nil, fmt.Errorf("failed to bind CCXT_SERVICE_URL environment variable: %w", err)
	}

	// Set default values (after environment variables are bound)
	setDefaults()

	// Configuration sources will be logged after config is loaded
	if err := viper.BindEnv("database.host", "DATABASE_HOST"); err != nil {
		return nil, fmt.Errorf("failed to bind DATABASE_HOST environment variable: %w", err)
	}
	if err := viper.BindEnv("database.port", "DATABASE_PORT"); err != nil {
		return nil, fmt.Errorf("failed to bind DATABASE_PORT environment variable: %w", err)
	}
	if err := viper.BindEnv("database.user", "DATABASE_USER"); err != nil {
		return nil, fmt.Errorf("failed to bind DATABASE_USER environment variable: %w", err)
	}
	if err := viper.BindEnv("database.password", "DATABASE_PASSWORD"); err != nil {
		return nil, fmt.Errorf("failed to bind DATABASE_PASSWORD environment variable: %w", err)
	}
	if err := viper.BindEnv("database.dbname", "DATABASE_DBNAME"); err != nil {
		return nil, fmt.Errorf("failed to bind DATABASE_DBNAME environment variable: %w", err)
	}
	if err := viper.BindEnv("database.sslmode", "DATABASE_SSLMODE"); err != nil {
		return nil, fmt.Errorf("failed to bind DATABASE_SSLMODE environment variable: %w", err)
	}
	if err := viper.BindEnv("redis.host", "REDIS_HOST"); err != nil {
		return nil, fmt.Errorf("failed to bind REDIS_HOST environment variable: %w", err)
	}
	if err := viper.BindEnv("redis.port", "REDIS_PORT"); err != nil {
		return nil, fmt.Errorf("failed to bind REDIS_PORT environment variable: %w", err)
	}
	if err := viper.BindEnv("ccxt.timeout", "CCXT_TIMEOUT"); err != nil {
		return nil, fmt.Errorf("failed to bind CCXT_TIMEOUT environment variable: %w", err)
	}
	if err := viper.BindEnv("telegram.bot_token", "TELEGRAM_BOT_TOKEN"); err != nil {
		return nil, fmt.Errorf("failed to bind TELEGRAM_BOT_TOKEN environment variable: %w", err)
	}

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found, use defaults and environment variables
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("DEBUG: Error reading config file: %v", err)
			return nil, err
		}
		log.Printf("DEBUG: Config file not found, using defaults and environment variables")
	} else {
		log.Printf("DEBUG: Config file loaded: %s", viper.ConfigFileUsed())
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Normalize environment to lowercase for consistent comparison
	environment := strings.ToLower(config.Environment)

	// Validate JWT secret in non-development environments
	if environment != "development" && config.Security.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET environment variable is required in non-development environments")
	}

	// Validate JWT expiry duration
	if config.Security.JWTExpiry != "" {
		if _, err := time.ParseDuration(config.Security.JWTExpiry); err != nil {
			return nil, fmt.Errorf("invalid JWT expiry duration: %w", err)
		}
	}

	// Validate bcrypt cost parameter
	if config.Security.BcryptCost < bcrypt.MinCost || config.Security.BcryptCost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost must be between %d and %d, got %d",
			bcrypt.MinCost, bcrypt.MaxCost, config.Security.BcryptCost)
	}

	// Update config with normalized environment
	config.Environment = environment

	// Debug: Log final configuration
	log.Printf("DEBUG: CCXT ServiceURL=%s timeout=%d", config.CCXT.ServiceURL, config.CCXT.Timeout)

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
	viper.SetDefault("database.password", "postgres")
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
	viper.SetDefault("ccxt.service_url", "http://ccxt-service:3001")
	viper.SetDefault("ccxt.timeout", 30)

	// Configuration defaults set silently to avoid side effects

	// Telegram
	viper.SetDefault("telegram.bot_token", "")
	viper.SetDefault("telegram.webhook_url", "")

	// Cleanup
	viper.SetDefault("cleanup.market_data_retention_hours", 24)
	viper.SetDefault("cleanup.funding_rate_retention_hours", 24)
	viper.SetDefault("cleanup.arbitrage_retention_hours", 72)
	viper.SetDefault("cleanup.cleanup_interval_minutes", 60)

	// Market Data
	viper.SetDefault("market_data.collection_interval", "5m")
	viper.SetDefault("market_data.batch_size", 100)
	viper.SetDefault("market_data.max_retries", 3)
	viper.SetDefault("market_data.timeout", "15s")
	viper.SetDefault("market_data.exchanges", []string{"binance", "coinbase", "kraken", "bitfinex", "huobi"})

	// Arbitrage
	viper.SetDefault("arbitrage.min_profit_threshold", 0.5)
	viper.SetDefault("arbitrage.max_trade_amount", 1000.0)
	viper.SetDefault("arbitrage.check_interval", "2m")
	viper.SetDefault("arbitrage.enabled_pairs", []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT"})

	// Security
	viper.SetDefault("security.jwt_secret", "")
	viper.SetDefault("security.jwt_expiry", "24h")
	viper.SetDefault("security.bcrypt_cost", 12)
}
