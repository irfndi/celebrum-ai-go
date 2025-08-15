package config

import (
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
	Cleanup     CleanupConfig    `mapstructure:"cleanup"`
	Backfill    BackfillConfig   `mapstructure:"backfill"`
	MarketData  MarketDataConfig `mapstructure:"market_data"`
	Arbitrage   ArbitrageConfig  `mapstructure:"arbitrage"`
	Blacklist   BlacklistConfig  `mapstructure:"blacklist"`
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
}

type BlacklistConfig struct {
	TTL             string `mapstructure:"ttl"`               // Default TTL for blacklisted symbols (e.g., "24h")
	ShortTTL        string `mapstructure:"short_ttl"`         // Short TTL for temporary issues (e.g., "1h")
	LongTTL         string `mapstructure:"long_ttl"`          // Long TTL for persistent issues (e.g., "72h")
	UseRedis        bool   `mapstructure:"use_redis"`         // Whether to use Redis for blacklist persistence
	RetryAfterClear bool   `mapstructure:"retry_after_clear"` // Whether to retry symbols after blacklist expires
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
	viper.SetDefault("arbitrage.min_profit_threshold", 0.5)
	viper.SetDefault("arbitrage.max_trade_amount", 1000.0)
	viper.SetDefault("arbitrage.check_interval", "2m")
	viper.SetDefault("arbitrage.enabled_pairs", []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT"})

	// Blacklist
	viper.SetDefault("blacklist.ttl", "24h")
	viper.SetDefault("blacklist.short_ttl", "1h")
	viper.SetDefault("blacklist.long_ttl", "72h")
	viper.SetDefault("blacklist.use_redis", true)
	viper.SetDefault("blacklist.retry_after_clear", true)
}
