package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a test Redis instance using miniredis
func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	s, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	cleanup := func() {
		client.Close()
		s.Close()
	}

	return client, cleanup
}

func TestNewRedisSymbolCache(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	ttl := 5 * time.Minute
	cache := NewRedisSymbolCache(client, ttl)

	assert.NotNil(t, cache)
	assert.Equal(t, client, cache.redis)
	assert.Equal(t, ttl, cache.ttl)
	assert.NotNil(t, cache.stats)
	assert.Equal(t, "symbol_cache:", cache.prefix)
}

func TestRedisSymbolCache_Get_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// First, set some data
	symbols := []string{"BTC/USD", "ETH/USD", "LTC/USD"}
	cache.Set("binance", symbols)

	// Now get it back
	retrieved, found := cache.Get("binance")

	assert.True(t, found)
	assert.Equal(t, symbols, retrieved)

	// Check stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(1), stats.Sets)
}

func TestRedisSymbolCache_Get_Miss(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Try to get non-existent data
	retrieved, found := cache.Get("nonexistent")

	assert.False(t, found)
	assert.Nil(t, retrieved)

	// Check stats
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)
}

func TestRedisSymbolCache_Get_InvalidJSON(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Manually set invalid JSON data
	client.Set(context.Background(), "symbol_cache:test", "invalid json", 5*time.Minute)

	// Try to get it
	retrieved, found := cache.Get("test")

	assert.False(t, found)
	assert.Nil(t, retrieved)

	// Check stats - should be a miss due to JSON error
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestRedisSymbolCache_Get_ExpiredEntry(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Create an expired entry
	expiredEntry := SymbolCacheEntry{
		Symbols:   []string{"BTC/USD"},
		CachedAt:  time.Now().Add(-10 * time.Minute),
		ExpiresAt: time.Now().Add(-5 * time.Minute), // Expired 5 minutes ago
	}

	data, _ := json.Marshal(expiredEntry)
	client.Set(context.Background(), "symbol_cache:expired", string(data), 5*time.Minute)

	// Get the expired entry - should still return it to prevent API calls
	retrieved, found := cache.Get("expired")

	assert.True(t, found) // Should still return expired data
	assert.Equal(t, []string{"BTC/USD"}, retrieved)

	// Check stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
}

func TestRedisSymbolCache_Set(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	symbols := []string{"BTC/USD", "ETH/USD", "LTC/USD"}
	cache.Set("binance", symbols)

	// Verify data was stored correctly
	data, err := client.Get(context.Background(), "symbol_cache:binance").Result()
	require.NoError(t, err)

	var entry SymbolCacheEntry
	err = json.Unmarshal([]byte(data), &entry)
	require.NoError(t, err)

	assert.Equal(t, symbols, entry.Symbols)
	assert.True(t, time.Since(entry.CachedAt) < time.Minute)
	assert.True(t, entry.ExpiresAt.After(time.Now()))

	// Check stats
	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Sets)
}

func TestRedisSymbolCache_GetStats(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Initial stats should be zero
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)

	// Perform some operations
	cache.Set("binance", []string{"BTC/USD"})
	cache.Get("binance") // Hit
	cache.Get("nonexistent") // Miss

	// Check updated stats
	stats = cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(1), stats.Sets)
}

func TestRedisSymbolCache_LogStats(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// This test just ensures LogStats doesn't panic
	cache.LogStats()

	// Add some data and log again
	cache.Set("binance", []string{"BTC/USD"})
	cache.Get("binance")
	cache.LogStats()
}

func TestRedisSymbolCache_Clear_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Add some data
	cache.Set("binance", []string{"BTC/USD"})
	cache.Set("coinbase", []string{"ETH/USD"})

	// Clear the cache
	err := cache.Clear()
	assert.NoError(t, err)

	// Verify data is gone
	_, found1 := cache.Get("binance")
	_, found2 := cache.Get("coinbase")
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestRedisSymbolCache_Clear_NoKeys(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Clear empty cache
	err := cache.Clear()
	assert.NoError(t, err)
}

func TestRedisSymbolCache_GetCachedExchanges_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Add data for multiple exchanges
	cache.Set("binance", []string{"BTC/USD"})
	cache.Set("coinbase", []string{"ETH/USD"})
	cache.Set("kraken", []string{"LTC/USD"})

	// Get cached exchanges
	exchanges, err := cache.GetCachedExchanges()
	assert.NoError(t, err)
	assert.Len(t, exchanges, 3)
	assert.Contains(t, exchanges, "binance")
	assert.Contains(t, exchanges, "coinbase")
	assert.Contains(t, exchanges, "kraken")
}

func TestRedisSymbolCache_GetCachedExchanges_Empty(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Get cached exchanges from empty cache
	exchanges, err := cache.GetCachedExchanges()
	assert.NoError(t, err)
	assert.Empty(t, exchanges)
}

func TestRedisSymbolCache_WarmCache_Success(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Mock symbol fetcher
	symbolFetcher := func(exchangeID string) ([]string, error) {
		switch exchangeID {
		case "binance":
			return []string{"BTC/USD", "ETH/USD"}, nil
		case "coinbase":
			return []string{"BTC/USD", "LTC/USD"}, nil
		default:
			return nil, assert.AnError
		}
	}

	// Warm cache
	exchanges := []string{"binance", "coinbase"}
	err := cache.WarmCache(exchanges, symbolFetcher)
	assert.NoError(t, err)

	// Verify data was cached
	binanceSymbols, found1 := cache.Get("binance")
	coinbaseSymbols, found2 := cache.Get("coinbase")

	assert.True(t, found1)
	assert.True(t, found2)
	assert.Equal(t, []string{"BTC/USD", "ETH/USD"}, binanceSymbols)
	assert.Equal(t, []string{"BTC/USD", "LTC/USD"}, coinbaseSymbols)
}

func TestRedisSymbolCache_WarmCache_AlreadyCached(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Pre-cache some data
	cache.Set("binance", []string{"EXISTING/DATA"})

	// Mock symbol fetcher that should not be called for binance
	symbolFetcher := func(exchangeID string) ([]string, error) {
		if exchangeID == "binance" {
			t.Error("Fetcher should not be called for already cached exchange")
		}
		return []string{"NEW/DATA"}, nil
	}

	// Warm cache
	exchanges := []string{"binance", "coinbase"}
	err := cache.WarmCache(exchanges, symbolFetcher)
	assert.NoError(t, err)

	// Verify binance data wasn't changed
	binanceSymbols, found := cache.Get("binance")
	assert.True(t, found)
	assert.Equal(t, []string{"EXISTING/DATA"}, binanceSymbols)
}

func TestRedisSymbolCache_WarmCache_FetcherError(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Mock symbol fetcher that returns error
	symbolFetcher := func(exchangeID string) ([]string, error) {
		return nil, assert.AnError
	}

	// Warm cache - should not fail even if fetcher returns error
	exchanges := []string{"binance"}
	err := cache.WarmCache(exchanges, symbolFetcher)
	assert.NoError(t, err) // WarmCache continues on individual errors

	// Verify no data was cached
	_, found := cache.Get("binance")
	assert.False(t, found)
}

func TestSymbolCacheStats_ThreadSafety(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Test concurrent access to stats
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cache.Set("test", []string{"BTC/USD"})
				cache.Get("test")
				cache.Get("nonexistent")
				cache.GetStats()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have reasonable stats
	stats := cache.GetStats()
	assert.True(t, stats.Sets > 0)
	assert.True(t, stats.Hits > 0)
	assert.True(t, stats.Misses > 0)
}

func TestRedisSymbolCache_EmptySymbols(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Set empty symbols list
	emptySymbols := []string{}
	cache.Set("empty", emptySymbols)

	// Get it back
	retrieved, found := cache.Get("empty")
	assert.True(t, found)
	assert.Equal(t, emptySymbols, retrieved)
}

func TestRedisSymbolCache_LargeSymbolsList(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cache := NewRedisSymbolCache(client, 5*time.Minute)

	// Create a large symbols list
	largeSymbols := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		largeSymbols[i] = fmt.Sprintf("SYMBOL%d/USD", i)
	}

	cache.Set("large", largeSymbols)

	// Get it back
	retrieved, found := cache.Get("large")
	assert.True(t, found)
	assert.Equal(t, largeSymbols, retrieved)
}