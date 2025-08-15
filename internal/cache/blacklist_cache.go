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
	Symbol    string    `json:"symbol"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
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

// RedisBlacklistCache implements BlacklistCache using Redis with database persistence
type RedisBlacklistCache struct {
	client redis.Cmdable
	ctx    context.Context
	stats  BlacklistCacheStats
	mu     sync.RWMutex
	prefix string
	repo   *database.BlacklistRepository
}

// NewRedisBlacklistCache creates a new Redis-based blacklist cache with database persistence
func NewRedisBlacklistCache(client redis.Cmdable, repo *database.BlacklistRepository) *RedisBlacklistCache {
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
	if time.Now().After(entry.ExpiresAt) {
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
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}

	// Persist to database first
	if rbc.repo != nil {
		ctx := context.Background()
		_, err := rbc.repo.AddExchange(ctx, symbol, reason, &entry.ExpiresAt)
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
	keys, err := rbc.client.Keys(rbc.ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get blacklist keys: %v", err)
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
	keys, err := rbc.client.Keys(rbc.ctx, pattern).Result()
	if err == nil {
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
			ExpiresAt: time.Time{}, // Set to zero if no expiration
		}

		if entry.ExpiresAt != nil {
			cacheEntry.ExpiresAt = *entry.ExpiresAt
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
	keys, err := rbc.client.Keys(rbc.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get blacklist keys: %w", err)
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
		if time.Now().After(entry.ExpiresAt) {
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
	keys, err := rbc.client.Keys(rbc.ctx, pattern).Result()
	if err != nil {
		log.Printf("Failed to get blacklist keys for cleanup: %v", err)
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
		if time.Now().After(entry.ExpiresAt) {
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
	ibc.mu.RLock()
	defer ibc.mu.RUnlock()

	entry, exists := ibc.cache[symbol]
	if !exists {
		ibc.stats.Misses++
		return false, ""
	}

	// Check if entry has expired (skip check for far future dates which indicate no expiration)
	if entry.ExpiresAt.Year() < 9999 && time.Now().After(entry.ExpiresAt) {
		// Remove expired entry
		delete(ibc.cache, symbol)
		ibc.stats.ExpiredEntries++
		ibc.stats.TotalEntries--
		ibc.stats.Misses++
		return false, ""
	}

	ibc.stats.Hits++
	return true, entry.Reason
}

// Add adds a symbol to the blacklist (in-memory implementation)
func (ibc *InMemoryBlacklistCache) Add(symbol, reason string, ttl time.Duration) {
	ibc.mu.Lock()
	defer ibc.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	} else {
		// TTL of 0 means no expiration - set to far future
		expiresAt = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}

	entry := &BlacklistCacheEntry{
		Symbol:    symbol,
		Reason:    reason,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

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
		// Check if entry has expired (skip check for far future dates which indicate no expiration)
		if entry.ExpiresAt.Year() < 9999 && time.Now().After(entry.ExpiresAt) {
			continue // Skip expired entries
		}
		entries = append(entries, *entry)
	}

	return entries, nil
}
