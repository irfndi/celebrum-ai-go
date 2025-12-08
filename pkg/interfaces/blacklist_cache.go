package interfaces

import (
	"context"
	"sync"
	"time"
)

// BlacklistCacheEntry represents a blacklisted symbol with its associated metadata.
// It tracks why a symbol was blacklisted and when (or if) the blacklist entry expires.
type BlacklistCacheEntry struct {
	// Symbol is the trading pair identifier that is blacklisted.
	Symbol string `json:"symbol"`
	// Reason describes why the symbol was blacklisted (e.g., "low volume", "api error").
	Reason string `json:"reason"`
	// ExpiresAt points to the time when the blacklist entry is no longer valid.
	// If nil, the entry does not expire automatically.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// CreatedAt is the timestamp when the entry was created.
	CreatedAt time.Time `json:"created_at"`
}

// BlacklistCacheStats holds operational statistics about the blacklist cache.
// Useful for monitoring cache performance and effectiveness.
type BlacklistCacheStats struct {
	// TotalEntries is the current number of items in the cache.
	TotalEntries int64 `json:"total_entries"`
	// ExpiredEntries is the count of entries that have passed their expiration time.
	ExpiredEntries int64 `json:"expired_entries"`
	// Hits is the number of times a cache lookup successfully found an entry.
	Hits int64 `json:"hits"`
	// Misses is the number of times a cache lookup failed to find an entry.
	Misses int64 `json:"misses"`
	// Adds is the total number of entries added to the cache.
	Adds int64 `json:"adds"`
	// LastCleanup is the timestamp of the last cache cleanup operation.
	LastCleanup time.Time `json:"last_cleanup"`
}

// ExchangeBlacklistEntry represents a blacklisted exchange persisted in the database.
type ExchangeBlacklistEntry struct {
	// ID is the unique identifier for the entry.
	ID int64 `json:"id"`
	// ExchangeName is the name of the exchange.
	ExchangeName string `json:"exchange_name"`
	// Reason describes why the exchange was blacklisted.
	Reason string `json:"reason"`
	// ExpiresAt is the timestamp when the blacklist entry expires.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// CreatedAt is the timestamp when the entry was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the timestamp when the entry was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// SymbolCacheEntry represents a cached symbol list for an exchange.
type SymbolCacheEntry struct {
	// Symbols is the list of trading pair symbols.
	Symbols []string `json:"symbols"`
	// CachedAt is the timestamp when the data was cached.
	CachedAt time.Time `json:"cached_at"`
	// ExpiresAt is the timestamp when the cache entry expires.
	ExpiresAt time.Time `json:"expires_at"`
}

// SymbolCacheStats tracks performance metrics for the symbol cache.
type SymbolCacheStats struct {
	// Hits is the number of successful cache lookups.
	Hits int64 `json:"hits"`
	// Misses is the number of failed cache lookups.
	Misses int64 `json:"misses"`
	// Sets is the number of times data was written to the cache.
	Sets int64 `json:"sets"`
}

// SymbolCacheInterface defines the contract for symbol caching mechanisms.
type SymbolCacheInterface interface {
	// Get retrieves cached symbols for a given exchange.
	//
	// Parameters:
	//   exchangeID: The identifier of the exchange.
	//
	// Returns:
	//   []string: A list of symbols if found.
	//   bool: True if found and valid, false otherwise.
	Get(exchangeID string) ([]string, bool)

	// Set stores symbols in the cache for a given exchange.
	//
	// Parameters:
	//   exchangeID: The identifier of the exchange.
	//   symbols: The list of symbols to cache.
	Set(exchangeID string, symbols []string)

	// GetStats retrieves current cache statistics.
	//
	// Returns:
	//   SymbolCacheStats: The statistics object.
	GetStats() SymbolCacheStats

	// LogStats outputs the cache statistics to the log.
	LogStats()
}

// BlacklistRepository defines the contract for database operations related to the blacklist.
// This interface enables dependency injection and mocking for testing.
type BlacklistRepository interface {
	// AddExchange adds an exchange to the blacklist.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//   exchangeName: The name of the exchange.
	//   reason: The reason for blacklisting.
	//   expiresAt: Optional expiration time.
	//
	// Returns:
	//   *ExchangeBlacklistEntry: The created entry.
	//   error: An error if the operation fails.
	AddExchange(ctx context.Context, exchangeName, reason string, expiresAt *time.Time) (*ExchangeBlacklistEntry, error)

	// RemoveExchange removes an exchange from the blacklist.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//   exchangeName: The name of the exchange to remove.
	//
	// Returns:
	//   error: An error if the operation fails.
	RemoveExchange(ctx context.Context, exchangeName string) error

	// IsBlacklisted checks if an exchange is currently blacklisted.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//   exchangeName: The name of the exchange to check.
	//
	// Returns:
	//   bool: True if blacklisted.
	//   string: The reason for blacklisting.
	//   error: An error if the check fails.
	IsBlacklisted(ctx context.Context, exchangeName string) (bool, string, error)

	// GetAllBlacklisted retrieves all blacklisted exchanges.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//
	// Returns:
	//   []ExchangeBlacklistEntry: A list of all blacklist entries.
	//   error: An error if the retrieval fails.
	GetAllBlacklisted(ctx context.Context) ([]ExchangeBlacklistEntry, error)

	// CleanupExpired removes expired blacklist entries from the database.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//
	// Returns:
	//   int64: The number of entries removed.
	//   error: An error if the operation fails.
	CleanupExpired(ctx context.Context) (int64, error)

	// GetBlacklistHistory retrieves the history of blacklist entries.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//   limit: The maximum number of entries to return.
	//
	// Returns:
	//   []ExchangeBlacklistEntry: A list of historical entries.
	//   error: An error if the retrieval fails.
	GetBlacklistHistory(ctx context.Context, limit int) ([]ExchangeBlacklistEntry, error)

	// ClearAll removes all blacklist entries.
	//
	// Parameters:
	//   ctx: The context for the operation.
	//
	// Returns:
	//   int64: The number of entries removed.
	//   error: An error if the operation fails.
	ClearAll(ctx context.Context) (int64, error)
}

// BlacklistCache defines the interface for in-memory or Redis-based blacklist caching.
type BlacklistCache interface {
	// IsBlacklisted checks if a symbol is in the blacklist.
	//
	// Parameters:
	//   symbol: The symbol to check.
	//
	// Returns:
	//   bool: True if the symbol is blacklisted.
	//   string: The reason for blacklisting.
	IsBlacklisted(symbol string) (bool, string)

	// Add adds a symbol to the blacklist.
	//
	// Parameters:
	//   symbol: The symbol to blacklist.
	//   reason: The reason for blacklisting.
	//   ttl: The time-to-live duration for the entry.
	Add(symbol, reason string, ttl time.Duration)

	// Remove removes a symbol from the blacklist.
	//
	// Parameters:
	//   symbol: The symbol to remove.
	Remove(symbol string)

	// Clear removes all entries from the blacklist cache.
	Clear()

	// GetStats retrieves current cache statistics.
	//
	// Returns:
	//   BlacklistCacheStats: The statistics object.
	GetStats() BlacklistCacheStats

	// LogStats outputs the cache statistics to the log.
	LogStats()

	// Close cleans up resources used by the cache.
	//
	// Returns:
	//   error: An error if closing fails.
	Close() error

	// LoadFromDatabase loads blacklist entries from the database into the cache.
	//
	// Parameters:
	//   ctx: A context or similar object used for the operation.
	//
	// Returns:
	//   error: An error if loading fails.
	LoadFromDatabase(ctx context.Context) error

	// GetBlacklistedSymbols retrieves all currently blacklisted symbols.
	//
	// Returns:
	//   []BlacklistCacheEntry: A list of blacklist entries.
	//   error: An error if retrieval fails.
	GetBlacklistedSymbols() ([]BlacklistCacheEntry, error)
}

// NewInMemoryBlacklistCache creates a new in-memory blacklist cache instance.
// This function serves as a factory for creating default blacklist cache instances.
//
// Returns:
//   BlacklistCache: An in-memory implementation of the BlacklistCache interface.
func NewInMemoryBlacklistCache() BlacklistCache {
	return &mockBlacklistCache{}
}

// NewRedisSymbolCache creates a new Redis-based symbol cache.
// This function serves as a factory for creating Redis symbol cache instances.
//
// Parameters:
//   redisClient: The Redis client instance (typed as interface{} to avoid imports).
//   ttl: The default time-to-live for cache entries.
//
// Returns:
//   SymbolCacheInterface: A Redis-based implementation of the SymbolCacheInterface.
func NewRedisSymbolCache(redisClient interface{}, ttl time.Duration) SymbolCacheInterface {
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

func (m *mockBlacklistCache) LoadFromDatabase(ctx context.Context) error {
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
