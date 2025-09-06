package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/redis/go-redis/v9"
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

// BlacklistCache interface defines the contract for blacklist caching
type BlacklistCache interface {
	IsBlacklisted(symbol string) (bool, string)
	Add(symbol, reason string, ttl time.Duration)
	Remove(symbol string)
	Clear()
	GetStats() BlacklistCacheStats
	LogStats()
	Close() error
	// New methods for database persistence
	LoadFromDatabase(ctx context.Context) error
	GetBlacklistedSymbols() ([]BlacklistCacheEntry, error)
}

// BlacklistRepository interface defines the contract for database operations
// This allows for dependency injection and testing with mock implementations
type BlacklistRepository interface {
	AddExchange(ctx context.Context, exchangeName, reason string, expiresAt *time.Time) (*database.ExchangeBlacklistEntry, error)
	RemoveExchange(ctx context.Context, exchangeName string) error
	IsBlacklisted(ctx context.Context, exchangeName string) (bool, string, error)
	GetAllBlacklisted(ctx context.Context) ([]database.ExchangeBlacklistEntry, error)
	CleanupExpired(ctx context.Context) (int64, error)
	GetBlacklistHistory(ctx context.Context, limit int) ([]database.ExchangeBlacklistEntry, error)
	ClearAll(ctx context.Context) (int64, error)
}

// RedisBlacklistCache implements BlacklistCache using Redis with database persistence
type RedisBlacklistCache struct {
	client redis.Cmdable
	ctx    context.Context
	stats  BlacklistCacheStats
	mu     sync.RWMutex
	prefix string
	repo   BlacklistRepository
}

// NewRedisBlacklistCache creates a new Redis-based blacklist cache with database persistence
func NewRedisBlacklistCache(client redis.Cmdable, repo BlacklistRepository) *RedisBlacklistCache {
	return &RedisBlacklistCache{
		client: client,
		ctx:    context.Background(),
		prefix: "blacklist:",
		stats:  BlacklistCacheStats{},
		repo:   repo,
	}
}

// IsBlacklisted checks if a symbol is blacklisted
func (rbc *RedisBlacklistCache) IsBlacklisted(symbol string) (bool, string) {
	rbc.mu.Lock()
	defer rbc.mu.Unlock()

	key := rbc.prefix + symbol
	val, err := rbc.client.Get(rbc.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			rbc.stats.Misses++
			return false, ""
		}
		// Log error but don't fail the check
		log.Printf("Redis blacklist check error for %s: %v", symbol, err)
		rbc.stats.Misses++
		return false, ""
	}

	var entry BlacklistCacheEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		log.Printf("Failed to unmarshal blacklist entry for %s: %v", symbol, err)
		rbc.stats.Misses++
		return false, ""
	}

	// Check if entry has expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		// Remove expired entry
		rbc.client.Del(rbc.ctx, key)
		rbc.stats.ExpiredEntries++
		rbc.stats.Misses++
		return false, ""
	}

	rbc.stats.Hits++
	return true, entry.Reason
}

// Add adds a symbol to the blacklist with a TTL
func (rbc *RedisBlacklistCache) Add(symbol, reason string, ttl time.Duration) {
	rbc.mu.Lock()
	defer rbc.mu.Unlock()

	entry := BlacklistCacheEntry{
		Symbol:    symbol,
		Reason:    reason,
		CreatedAt: time.Now(),
	}

	// Set expiration time if TTL is provided
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		entry.ExpiresAt = &expiresAt
	}

	// Persist to database first
	if rbc.repo != nil {
		ctx := context.Background()
		_, err := rbc.repo.AddExchange(ctx, symbol, reason, entry.ExpiresAt)
		if err != nil {
			log.Printf("Error persisting to database: %v", err)
			// Continue with Redis cache even if database fails
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal blacklist entry for %s: %v", symbol, err)
		return
	}

	key := rbc.prefix + symbol
	err = rbc.client.Set(rbc.ctx, key, data, ttl).Err()
	if err != nil {
		log.Printf("Failed to set blacklist entry for %s: %v", symbol, err)
		return
	}

	rbc.stats.Adds++
	rbc.stats.TotalEntries++
	log.Printf("Blacklisted symbol %s for %s (TTL: %v)", symbol, reason, ttl)
}

// Remove removes a symbol from the blacklist
func (rbc *RedisBlacklistCache) Remove(symbol string) {
	rbc.mu.Lock()
	defer rbc.mu.Unlock()

	// Remove from database first
	if rbc.repo != nil {
		ctx := context.Background()
		err := rbc.repo.RemoveExchange(ctx, symbol)
		if err != nil {
			log.Printf("Error removing from database: %v", err)
			// Continue with Redis cache even if database fails
		}
	}

	key := rbc.prefix + symbol
	result := rbc.client.Del(rbc.ctx, key)
	if result.Err() != nil {
		log.Printf("Failed to remove blacklist entry for %s: %v", symbol, result.Err())
		return
	}

	if result.Val() > 0 {
		rbc.stats.TotalEntries--
		log.Printf("Removed symbol %s from blacklist", symbol)
	}
}

// Clear removes all blacklisted symbols
func (rbc *RedisBlacklistCache) Clear() {
	rbc.mu.Lock()
	defer rbc.mu.Unlock()

	// Note: ClearAll is not implemented in repository as it's a dangerous operation
	// For now, we'll only clear the Redis cache
	// TODO: Implement a safe clear operation in repository if needed

	pattern := rbc.prefix + "*"
	var keys []string
	iter := rbc.client.Scan(rbc.ctx, 0, pattern, 0).Iterator()
	for iter.Next(rbc.ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("Failed to scan blacklist keys: %v", err)
		return
	}

	if len(keys) > 0 {
		result := rbc.client.Del(rbc.ctx, keys...)
		if result.Err() != nil {
			log.Printf("Failed to clear blacklist: %v", result.Err())
			return
		}
		rbc.stats.TotalEntries = 0
		log.Printf("Cleared %d blacklisted symbols", result.Val())
	}
}

// GetStats returns current cache statistics
func (rbc *RedisBlacklistCache) GetStats() BlacklistCacheStats {
	rbc.mu.RLock()
	defer rbc.mu.RUnlock()

	// Update total entries count from Redis
	pattern := rbc.prefix + "*"
	var keys []string
	iter := rbc.client.Scan(rbc.ctx, 0, pattern, 0).Iterator()
	for iter.Next(rbc.ctx) {
		keys = append(keys, iter.Val())
	}
	if iter.Err() == nil {
		rbc.stats.TotalEntries = int64(len(keys))
	}

	return rbc.stats
}

// LogStats logs current cache statistics
func (rbc *RedisBlacklistCache) LogStats() {
	stats := rbc.GetStats()
	log.Printf("Blacklist Cache Stats - Total: %d, Hits: %d, Misses: %d, Adds: %d, Expired: %d",
		stats.TotalEntries, stats.Hits, stats.Misses, stats.Adds, stats.ExpiredEntries)
}

// Close closes the Redis connection (if needed)
func (rbc *RedisBlacklistCache) Close() error {
	// Redis client is managed externally, so we don't close it here
	return nil
}

// LoadFromDatabase loads blacklist entries from the database into Redis cache
func (rbc *RedisBlacklistCache) LoadFromDatabase(ctx context.Context) error {
	if rbc.repo == nil {
		return fmt.Errorf("database repository not configured")
	}

	rbc.mu.Lock()
	defer rbc.mu.Unlock()

	// Get all active blacklist entries from database
	entries, err := rbc.repo.GetAllBlacklisted(ctx)
	if err != nil {
		return fmt.Errorf("failed to load blacklist from database: %w", err)
	}

	// Load each entry into Redis cache
	for _, entry := range entries {
		// Skip expired entries
		if entry.ExpiresAt != nil && entry.ExpiresAt.Before(time.Now()) {
			continue
		}

		cacheEntry := BlacklistCacheEntry{
			Symbol:    entry.ExchangeName,
			Reason:    entry.Reason,
			CreatedAt: entry.CreatedAt,
			ExpiresAt: entry.ExpiresAt, // Use the pointer directly
		}

		data, err := json.Marshal(cacheEntry)
		if err != nil {
			log.Printf("Error marshaling blacklist entry for %s: %v", entry.ExchangeName, err)
			continue
		}

		key := rbc.prefix + entry.ExchangeName
		var ttl time.Duration
		if entry.ExpiresAt != nil {
			ttl = time.Until(*entry.ExpiresAt)
			if ttl <= 0 {
				continue // Skip expired entries
			}
		} else {
			ttl = 0 // No expiration
		}

		err = rbc.client.Set(ctx, key, data, ttl).Err()
		if err != nil {
			log.Printf("Error loading blacklist entry to cache for %s: %v", entry.ExchangeName, err)
			continue
		}
	}

	log.Printf("Loaded %d blacklist entries from database to cache", len(entries))
	return nil
}

// GetBlacklistedSymbols returns all currently blacklisted symbols
func (rbc *RedisBlacklistCache) GetBlacklistedSymbols() ([]BlacklistCacheEntry, error) {
	rbc.mu.RLock()
	defer rbc.mu.RUnlock()

	pattern := rbc.prefix + "*"
	var keys []string
	iter := rbc.client.Scan(rbc.ctx, 0, pattern, 0).Iterator()
	for iter.Next(rbc.ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan blacklist keys: %w", err)
	}

	var entries []BlacklistCacheEntry
	for _, key := range keys {
		val, err := rbc.client.Get(rbc.ctx, key).Result()
		if err != nil {
			continue // Skip errors, key might have expired
		}

		var entry BlacklistCacheEntry
		if err := json.Unmarshal([]byte(val), &entry); err != nil {
			continue // Skip malformed entries
		}

		// Check if entry has expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			// Remove expired entry
			rbc.client.Del(rbc.ctx, key)
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// CleanupExpired removes all expired entries from the blacklist
func (rbc *RedisBlacklistCache) CleanupExpired() int {
	rbc.mu.Lock()
	defer rbc.mu.Unlock()

	pattern := rbc.prefix + "*"
	var keys []string
	iter := rbc.client.Scan(rbc.ctx, 0, pattern, 0).Iterator()
	for iter.Next(rbc.ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Printf("Failed to scan blacklist keys for cleanup: %v", err)
		return 0
	}

	expiredCount := 0
	for _, key := range keys {
		val, err := rbc.client.Get(rbc.ctx, key).Result()
		if err != nil {
			continue
		}

		var entry BlacklistCacheEntry
		if err := json.Unmarshal([]byte(val), &entry); err != nil {
			continue
		}

		// Check if entry has expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			rbc.client.Del(rbc.ctx, key)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		rbc.stats.ExpiredEntries += int64(expiredCount)
		rbc.stats.TotalEntries -= int64(expiredCount)
		rbc.stats.LastCleanup = time.Now()
		log.Printf("Cleaned up %d expired blacklist entries", expiredCount)
	}

	return expiredCount
}

// InMemoryBlacklistCache provides a fallback in-memory implementation
type InMemoryBlacklistCache struct {
	cache map[string]*BlacklistCacheEntry
	mu    sync.RWMutex
	stats BlacklistCacheStats
}

// NewInMemoryBlacklistCache creates a new in-memory blacklist cache
func NewInMemoryBlacklistCache() *InMemoryBlacklistCache {
	return &InMemoryBlacklistCache{
		cache: make(map[string]*BlacklistCacheEntry),
		stats: BlacklistCacheStats{},
	}
}

// IsBlacklisted checks if a symbol is blacklisted (in-memory implementation)
func (ibc *InMemoryBlacklistCache) IsBlacklisted(symbol string) (bool, string) {
	// First, try with read lock for the common case
	ibc.mu.RLock()
	entry, exists := ibc.cache[symbol]
	if !exists {
		ibc.mu.RUnlock()
		// Need write lock to update stats
		ibc.mu.Lock()
		ibc.stats.Misses++
		ibc.mu.Unlock()
		return false, ""
	}

	// Check if entry has expired
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		ibc.mu.RUnlock()
		// Need write lock to remove expired entry and update stats
		ibc.mu.Lock()
		// Double-check the entry still exists and is expired
		if entry, exists := ibc.cache[symbol]; exists && entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			delete(ibc.cache, symbol)
			ibc.stats.ExpiredEntries++
			ibc.stats.TotalEntries--
		}
		ibc.stats.Misses++
		ibc.mu.Unlock()
		return false, ""
	}

	ibc.mu.RUnlock()
	// Need write lock to update stats
	ibc.mu.Lock()
	ibc.stats.Hits++
	ibc.mu.Unlock()
	return true, entry.Reason
}

// Add adds a symbol to the blacklist (in-memory implementation)
func (ibc *InMemoryBlacklistCache) Add(symbol, reason string, ttl time.Duration) {
	ibc.mu.Lock()
	defer ibc.mu.Unlock()

	entry := &BlacklistCacheEntry{
		Symbol:    symbol,
		Reason:    reason,
		CreatedAt: time.Now(),
	}

	// Set expiration time if TTL is provided
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl)
		entry.ExpiresAt = &expiresAt
	}
	// If ttl <= 0, ExpiresAt remains nil (no expiration)

	ibc.cache[symbol] = entry
	ibc.stats.Adds++
	ibc.stats.TotalEntries++
	log.Printf("Blacklisted symbol %s for %s (TTL: %v)", symbol, reason, ttl)
}

// Remove removes a symbol from the blacklist (in-memory implementation)
func (ibc *InMemoryBlacklistCache) Remove(symbol string) {
	ibc.mu.Lock()
	defer ibc.mu.Unlock()

	if _, exists := ibc.cache[symbol]; exists {
		delete(ibc.cache, symbol)
		ibc.stats.TotalEntries--
		log.Printf("Removed symbol %s from blacklist", symbol)
	}
}

// Clear removes all blacklisted symbols (in-memory implementation)
func (ibc *InMemoryBlacklistCache) Clear() {
	ibc.mu.Lock()
	defer ibc.mu.Unlock()

	count := len(ibc.cache)
	ibc.cache = make(map[string]*BlacklistCacheEntry)
	ibc.stats.TotalEntries = 0
	log.Printf("Cleared %d blacklisted symbols", count)
}

// GetStats returns current cache statistics (in-memory implementation)
func (ibc *InMemoryBlacklistCache) GetStats() BlacklistCacheStats {
	ibc.mu.RLock()
	defer ibc.mu.RUnlock()
	return ibc.stats
}

// LogStats logs current cache statistics (in-memory implementation)
func (ibc *InMemoryBlacklistCache) LogStats() {
	stats := ibc.GetStats()
	log.Printf("Blacklist Cache Stats - Total: %d, Hits: %d, Misses: %d, Adds: %d, Expired: %d",
		stats.TotalEntries, stats.Hits, stats.Misses, stats.Adds, stats.ExpiredEntries)
}

// Close closes the cache (in-memory implementation)
func (ibc *InMemoryBlacklistCache) Close() error {
	return nil
}

// LoadFromDatabase loads blacklist entries from the database (in-memory implementation)
func (ibc *InMemoryBlacklistCache) LoadFromDatabase(ctx context.Context) error {
	// In-memory cache doesn't support database persistence
	return fmt.Errorf("database persistence not supported for in-memory cache")
}

// GetBlacklistedSymbols returns all blacklisted symbols (in-memory implementation)
func (ibc *InMemoryBlacklistCache) GetBlacklistedSymbols() ([]BlacklistCacheEntry, error) {
	ibc.mu.RLock()
	defer ibc.mu.RUnlock()

	var entries []BlacklistCacheEntry
	for _, entry := range ibc.cache {
		// Check if entry has expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			continue // Skip expired entries
		}
		entries = append(entries, *entry)
	}

	return entries, nil
}
