package interfaces

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// BlacklistCacheEntry represents a blacklisted symbol with metadata
type BlacklistCacheEntry struct {
	Symbol    string     `json:"symbol"`
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // nil means no expiration
	CreatedAt time.Time  `json:"created_at"`
}

// BlacklistCacheStats holds statistics about the blacklist cache
type BlacklistCacheStats struct {
	TotalEntries   int64     `json:"total_entries"`
	ExpiredEntries int64     `json:"expired_entries"`
	Hits           int64     `json:"hits"`
	Misses         int64     `json:"misses"`
	Adds           int64     `json:"adds"`
	LastCleanup    time.Time `json:"last_cleanup"`
}

// Config types moved from internal/config to avoid internal package import violations
type Config struct {
	Environment string           `mapstructure:"environment"`
	LogLevel    string           `mapstructure:"log_level"`
	Server      ServerConfig     `mapstructure:"server"`
	Database    DatabaseConfig   `mapstructure:"database"`
	Redis       RedisConfig      `mapstructure:"redis"`
	CCXT        CCXTConfig       `mapstructure:"ccxt"`
	Telegram    TelegramConfig   `mapstructure:"telegram"`
	Telemetry   TelemetryConfig  `mapstructure:"telemetry"`
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
	BotToken       string `mapstructure:"bot_token"`
	WebhookURL     string `mapstructure:"webhook_url"`
	UsePolling     bool   `mapstructure:"use_polling"`
	PollingOffset  int    `mapstructure:"polling_offset"`
	PollingLimit   int    `mapstructure:"polling_limit"`
	PollingTimeout int    `mapstructure:"polling_timeout"`
}

type TelemetryConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	OTLPEndpoint   string `mapstructure:"otlp_endpoint"`
	ServiceName    string `mapstructure:"service_name"`
	ServiceVersion string `mapstructure:"service_version"`
	LogLevel       string `mapstructure:"log_level"`
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
	// Remove hardcoded default password for security
	// viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "celebrum_ai")
	viper.SetDefault("database.sslmode", "disable")
	// Remove hardcoded database URL for security
	// viper.SetDefault("database.database_url", "postgres://postgres:password@localhost:5432/celebrum_ai?sslmode=disable")
	viper.SetDefault("database.database_url", "")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "300s")
	viper.SetDefault("database.conn_max_idle_time", "60s")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	// Remove hardcoded Redis password for security
	// viper.SetDefault("redis.password", "your_redis_password")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// CCXT
	viper.SetDefault("ccxt.service_url", "http://localhost:3000")
	viper.SetDefault("ccxt.timeout", 30)

	// Telegram
	// Remove hardcoded Telegram bot token for security
	// viper.SetDefault("telegram.bot_token", "your_telegram_bot_token")
	viper.SetDefault("telegram.bot_token", "")
	viper.SetDefault("telegram.webhook_url", "")
	viper.SetDefault("telegram.use_polling", false)
	viper.SetDefault("telegram.polling_offset", 0)
	viper.SetDefault("telegram.polling_limit", 100)
	viper.SetDefault("telegram.polling_timeout", 60)

	// Telemetry
	viper.SetDefault("telemetry.enabled", true)
	viper.SetDefault("telemetry.otlp_endpoint", "http://localhost:4318")
	viper.SetDefault("telemetry.service_name", "celebrum-ai-go")
	viper.SetDefault("telemetry.service_version", "1.0.0")
	viper.SetDefault("telemetry.log_level", "info")

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
}

// ExchangeBlacklistEntry represents a blacklisted exchange in the database
type ExchangeBlacklistEntry struct {
	ID           int64      `json:"id"`
	ExchangeName string     `json:"exchange_name"`
	Reason       string     `json:"reason"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SymbolCacheEntry represents a cached symbol entry with metadata
type SymbolCacheEntry struct {
	Symbols   []string  `json:"symbols"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SymbolCacheStats tracks cache performance metrics
type SymbolCacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Sets   int64 `json:"sets"`
}

// SymbolCacheInterface defines the interface for symbol caching
type SymbolCacheInterface interface {
	Get(exchangeID string) ([]string, bool)
	Set(exchangeID string, symbols []string)
	GetStats() SymbolCacheStats
	LogStats()
}

// BlacklistRepository interface defines the contract for database operations
// This allows for dependency injection and testing with mock implementations
type BlacklistRepository interface {
	AddExchange(ctx context.Context, exchangeName, reason string, expiresAt *time.Time) (*ExchangeBlacklistEntry, error)
	RemoveExchange(ctx context.Context, exchangeName string) error
	IsBlacklisted(ctx context.Context, exchangeName string) (bool, string, error)
	GetAllBlacklisted(ctx context.Context) ([]ExchangeBlacklistEntry, error)
	CleanupExpired(ctx context.Context) (int64, error)
	GetBlacklistHistory(ctx context.Context, limit int) ([]ExchangeBlacklistEntry, error)
	ClearAll(ctx context.Context) (int64, error)
}

// BlacklistCache defines the interface for blacklist cache operations
type BlacklistCache interface {
	IsBlacklisted(symbol string) (bool, string)
	Add(symbol, reason string, ttl time.Duration)
	Remove(symbol string)
	Clear()
	GetStats() BlacklistCacheStats
	LogStats()
	Close() error
	// New methods for database persistence
	LoadFromDatabase(ctx interface{}) error
	GetBlacklistedSymbols() ([]BlacklistCacheEntry, error)
}

// NewInMemoryBlacklistCache creates a new in-memory blacklist cache instance
// This function serves as a factory for creating blacklist cache instances
func NewInMemoryBlacklistCache() BlacklistCache {
	// This is a placeholder - the actual implementation should be in the internal/cache package
	// For now, we'll return a simple mock implementation
	return &mockBlacklistCache{}
}

// NewRedisSymbolCache creates a new Redis-based symbol cache
// This function serves as a factory for creating symbol cache instances
func NewRedisSymbolCache(redisClient interface{}, ttl time.Duration) SymbolCacheInterface {
	// This is a placeholder - the actual implementation should be in the internal/cache package
	// For now, we'll return a simple mock implementation
	return &mockSymbolCache{}
}

// mockBlacklistCache is a simple implementation for testing
type mockBlacklistCache struct {
	entries map[string]BlacklistCacheEntry
	mu      sync.RWMutex
}

func (m *mockBlacklistCache) IsBlacklisted(symbol string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if entry, exists := m.entries[symbol]; exists {
		if entry.ExpiresAt == nil || entry.ExpiresAt.After(time.Now()) {
			return true, entry.Reason
		}
	}
	return false, ""
}

func (m *mockBlacklistCache) Add(symbol, reason string, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil {
		m.entries = make(map[string]BlacklistCacheEntry)
	}

	var expiresAt *time.Time
	if ttl > 0 {
		exp := time.Now().Add(ttl)
		expiresAt = &exp
	}

	m.entries[symbol] = BlacklistCacheEntry{
		Symbol:    symbol,
		Reason:    reason,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}

func (m *mockBlacklistCache) Remove(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, symbol)
}

func (m *mockBlacklistCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]BlacklistCacheEntry)
}

func (m *mockBlacklistCache) GetStats() BlacklistCacheStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return BlacklistCacheStats{
		TotalEntries: int64(len(m.entries)),
		LastCleanup:  time.Now(),
	}
}

func (m *mockBlacklistCache) LogStats() {
	// Mock implementation
}

func (m *mockBlacklistCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = nil
	return nil
}

func (m *mockBlacklistCache) LoadFromDatabase(ctx interface{}) error {
	// Mock implementation
	return nil
}

func (m *mockBlacklistCache) GetBlacklistedSymbols() ([]BlacklistCacheEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []BlacklistCacheEntry
	for _, entry := range m.entries {
		if entry.ExpiresAt == nil || entry.ExpiresAt.After(time.Now()) {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// mockSymbolCache is a simple implementation for testing
type mockSymbolCache struct {
	entries map[string]SymbolCacheEntry
	mu      sync.RWMutex
}

func (m *mockSymbolCache) Get(exchangeID string) ([]string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if entry, exists := m.entries[exchangeID]; exists {
		if time.Now().After(entry.ExpiresAt) {
			return nil, false
		}
		return entry.Symbols, true
	}
	return nil, false
}

func (m *mockSymbolCache) Set(exchangeID string, symbols []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil {
		m.entries = make(map[string]SymbolCacheEntry)
	}

	m.entries[exchangeID] = SymbolCacheEntry{
		Symbols:   symbols,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
}

func (m *mockSymbolCache) GetStats() SymbolCacheStats {
	return SymbolCacheStats{}
}

func (m *mockSymbolCache) LogStats() {
	// Mock implementation
}
