package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

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
	mu     sync.RWMutex
}

// RedisSymbolCache implements symbol caching using Redis
type RedisSymbolCache struct {
	redis  *redis.Client
	ttl    time.Duration
	stats  *SymbolCacheStats
	prefix string
}

// NewRedisSymbolCache creates a new Redis-based symbol cache
func NewRedisSymbolCache(redisClient *redis.Client, ttl time.Duration) *RedisSymbolCache {
	return &RedisSymbolCache{
		redis:  redisClient,
		ttl:    ttl,
		stats:  &SymbolCacheStats{},
		prefix: "symbol_cache:",
	}
}

// Get retrieves symbols for an exchange from Redis cache
func (c *RedisSymbolCache) Get(exchangeID string) ([]string, bool) {
	ctx := context.Background()
	cacheKey := c.prefix + exchangeID

	// Get cached data from Redis
	data, err := c.redis.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		// Cache miss
		c.stats.mu.Lock()
		c.stats.Misses++
		c.stats.mu.Unlock()
		return nil, false
	}
	if err != nil {
		log.Printf("Redis error getting symbols for %s: %v", exchangeID, err)
		c.stats.mu.Lock()
		c.stats.Misses++
		c.stats.mu.Unlock()
		return nil, false
	}

	// Deserialize cached entry
	var entry SymbolCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		log.Printf("Error deserializing cached symbols for %s: %v", exchangeID, err)
		c.stats.mu.Lock()
		c.stats.Misses++
		c.stats.mu.Unlock()
		return nil, false
	}

	// Check if entry has expired (additional check beyond Redis TTL)
	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, but return it anyway to prevent API calls during runtime
		// This matches the behavior of the original in-memory cache
		log.Printf("Cached symbols for %s expired but returning to prevent API calls", exchangeID)
	}

	// Cache hit
	c.stats.mu.Lock()
	c.stats.Hits++
	c.stats.mu.Unlock()

	return entry.Symbols, true
}

// Set stores symbols for an exchange in Redis cache
func (c *RedisSymbolCache) Set(exchangeID string, symbols []string) {
	ctx := context.Background()
	cacheKey := c.prefix + exchangeID

	// Create cache entry with metadata
	now := time.Now()
	entry := SymbolCacheEntry{
		Symbols:   symbols,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}

	// Serialize entry
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error serializing symbols for %s: %v", exchangeID, err)
		return
	}

	// Store in Redis with TTL
	err = c.redis.Set(ctx, cacheKey, data, c.ttl).Err()
	if err != nil {
		log.Printf("Redis error setting symbols for %s: %v", exchangeID, err)
		return
	}

	// Update stats
	c.stats.mu.Lock()
	c.stats.Sets++
	c.stats.mu.Unlock()

	log.Printf("Cached %d symbols for %s (TTL: %v)", len(symbols), exchangeID, c.ttl)
}

// GetStats returns current cache statistics
func (c *RedisSymbolCache) GetStats() SymbolCacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()
	return SymbolCacheStats{
		Hits:   c.stats.Hits,
		Misses: c.stats.Misses,
		Sets:   c.stats.Sets,
	}
}

// LogStats logs current cache performance statistics
func (c *RedisSymbolCache) LogStats() {
	stats := c.GetStats()
	total := stats.Hits + stats.Misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(stats.Hits) / float64(total) * 100
	}

	log.Printf("Redis Symbol Cache Stats - Hits: %d, Misses: %d, Sets: %d, Hit Rate: %.2f%%",
		stats.Hits, stats.Misses, stats.Sets, hitRate)
}

// Clear removes all cached symbols (useful for testing or cache invalidation)
func (c *RedisSymbolCache) Clear() error {
	ctx := context.Background()
	pattern := c.prefix + "*"

	// Get all keys matching the pattern using SCAN for better performance
	var keys []string
	iter := c.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("error scanning cache keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all matching keys
	if err := c.redis.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("error clearing cache: %w", err)
	}

	log.Printf("Cleared %d symbol cache entries", len(keys))
	return nil
}

// GetCachedExchanges returns a list of exchanges that have cached symbols
func (c *RedisSymbolCache) GetCachedExchanges() ([]string, error) {
	ctx := context.Background()
	pattern := c.prefix + "*"

	// Get all keys matching the pattern using SCAN for better performance
	var keys []string
	iter := c.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error scanning cache keys: %w", err)
	}

	// Extract exchange IDs from keys
	var exchanges []string
	prefixLen := len(c.prefix)
	for _, key := range keys {
		if len(key) > prefixLen {
			exchangeID := key[prefixLen:]
			exchanges = append(exchanges, exchangeID)
		}
	}

	return exchanges, nil
}

// WarmCache pre-loads symbols for specified exchanges
func (c *RedisSymbolCache) WarmCache(exchanges []string, symbolFetcher func(string) ([]string, error)) error {
	log.Printf("Warming symbol cache for %d exchanges...", len(exchanges))

	for _, exchangeID := range exchanges {
		// Check if already cached
		if _, exists := c.Get(exchangeID); exists {
			log.Printf("Symbols for %s already cached, skipping", exchangeID)
			continue
		}

		// Fetch and cache symbols
		symbols, err := symbolFetcher(exchangeID)
		if err != nil {
			log.Printf("Warning: Failed to warm cache for %s: %v", exchangeID, err)
			continue
		}

		c.Set(exchangeID, symbols)
		log.Printf("Warmed cache for %s with %d symbols", exchangeID, len(symbols))
	}

	log.Printf("Symbol cache warming completed")
	return nil
}
