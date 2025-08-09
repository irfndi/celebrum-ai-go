package config

import (
	"errors"
	"fmt"
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

	// Set default values
	setDefaults()

	// Enable environment variable support
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Bind specific environment variables
	if err := viper.BindEnv("security.jwt_secret", "JWT_SECRET"); err != nil {
		return nil, fmt.Errorf("failed to bind JWT_SECRET environment variable: %w", err)
	}

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
	viper.SetDefault("ccxt.service_url", "http://localhost:3001")
	viper.SetDefault("ccxt.timeout", 30)

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
