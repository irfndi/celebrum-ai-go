package interfaces

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewInMemoryBlacklistCache tests the NewInMemoryBlacklistCache factory function
func TestNewInMemoryBlacklistCache(t *testing.T) {
	cache := NewInMemoryBlacklistCache()
	require.NotNil(t, cache)

	// Verify it implements the BlacklistCache interface
	var _ BlacklistCache = cache
}

// TestNewRedisSymbolCache tests the NewRedisSymbolCache factory function
func TestNewRedisSymbolCache(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)
	require.NotNil(t, cache)

	// Verify it implements the SymbolCacheInterface interface
	var _ SymbolCacheInterface = cache
}

// TestMockBlacklistCache_IsBlacklisted tests the IsBlacklisted method
func TestMockBlacklistCache_IsBlacklisted(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Initially, symbol should not be blacklisted
	isBlacklisted, reason := cache.IsBlacklisted("BTC/USDT")
	assert.False(t, isBlacklisted)
	assert.Empty(t, reason)

	// Add symbol to blacklist
	cache.Add("BTC/USDT", "Test reason", time.Hour)

	// Now it should be blacklisted
	isBlacklisted, reason = cache.IsBlacklisted("BTC/USDT")
	assert.True(t, isBlacklisted)
	assert.Equal(t, "Test reason", reason)
}

// TestMockBlacklistCache_IsBlacklisted_Expired tests expired entries
func TestMockBlacklistCache_IsBlacklisted_Expired(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add symbol with very short TTL
	cache.Add("ETH/USDT", "Short TTL", time.Millisecond)

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	// Should not be blacklisted anymore
	isBlacklisted, reason := cache.IsBlacklisted("ETH/USDT")
	assert.False(t, isBlacklisted)
	assert.Empty(t, reason)
}

// TestMockBlacklistCache_Add tests the Add method
func TestMockBlacklistCache_Add(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add with TTL
	cache.Add("BTC/USDT", "Test reason", time.Hour)
	isBlacklisted, _ := cache.IsBlacklisted("BTC/USDT")
	assert.True(t, isBlacklisted)

	// Add without TTL (0 duration)
	cache.Add("ETH/USDT", "Permanent blacklist", 0)
	isBlacklisted, _ = cache.IsBlacklisted("ETH/USDT")
	assert.True(t, isBlacklisted)
}

// TestMockBlacklistCache_Remove tests the Remove method
func TestMockBlacklistCache_Remove(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add and remove
	cache.Add("BTC/USDT", "Test reason", time.Hour)
	isBlacklisted, _ := cache.IsBlacklisted("BTC/USDT")
	assert.True(t, isBlacklisted)

	cache.Remove("BTC/USDT")
	isBlacklisted, _ = cache.IsBlacklisted("BTC/USDT")
	assert.False(t, isBlacklisted)
}

// TestMockBlacklistCache_Clear tests the Clear method
func TestMockBlacklistCache_Clear(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add multiple entries
	cache.Add("BTC/USDT", "Reason 1", time.Hour)
	cache.Add("ETH/USDT", "Reason 2", time.Hour)
	cache.Add("XRP/USDT", "Reason 3", time.Hour)

	// Clear all
	cache.Clear()

	// All should be gone
	isBlacklisted, _ := cache.IsBlacklisted("BTC/USDT")
	assert.False(t, isBlacklisted)
	isBlacklisted, _ = cache.IsBlacklisted("ETH/USDT")
	assert.False(t, isBlacklisted)
	isBlacklisted, _ = cache.IsBlacklisted("XRP/USDT")
	assert.False(t, isBlacklisted)
}

// TestMockBlacklistCache_GetStats tests the GetStats method
func TestMockBlacklistCache_GetStats(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Initially empty
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.TotalEntries)

	// Add entries
	cache.Add("BTC/USDT", "Reason 1", time.Hour)
	cache.Add("ETH/USDT", "Reason 2", time.Hour)

	stats = cache.GetStats()
	assert.Equal(t, int64(2), stats.TotalEntries)
}

// TestMockBlacklistCache_LogStats tests the LogStats method
func TestMockBlacklistCache_LogStats(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Should not panic
	cache.LogStats()
}

// TestMockBlacklistCache_Close tests the Close method
func TestMockBlacklistCache_Close(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	cache.Add("BTC/USDT", "Reason", time.Hour)

	err := cache.Close()
	assert.NoError(t, err)

	// After close, should not be blacklisted (entries cleared)
	isBlacklisted, _ := cache.IsBlacklisted("BTC/USDT")
	assert.False(t, isBlacklisted)
}

// TestMockBlacklistCache_LoadFromDatabase tests the LoadFromDatabase method
func TestMockBlacklistCache_LoadFromDatabase(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	err := cache.LoadFromDatabase(context.Background())
	assert.NoError(t, err)
}

// TestMockBlacklistCache_GetBlacklistedSymbols tests the GetBlacklistedSymbols method
func TestMockBlacklistCache_GetBlacklistedSymbols(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Initially empty
	entries, err := cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	assert.Empty(t, entries)

	// Add entries
	cache.Add("BTC/USDT", "Reason 1", time.Hour)
	cache.Add("ETH/USDT", "Reason 2", time.Hour)

	entries, err = cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify entries contain the symbols
	symbols := make(map[string]bool)
	for _, entry := range entries {
		symbols[entry.Symbol] = true
	}
	assert.True(t, symbols["BTC/USDT"])
	assert.True(t, symbols["ETH/USDT"])
}

// TestMockBlacklistCache_GetBlacklistedSymbols_ExcludesExpired tests that expired entries are excluded
func TestMockBlacklistCache_GetBlacklistedSymbols_ExcludesExpired(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add one permanent and one short-lived entry
	cache.Add("BTC/USDT", "Permanent", 0)
	cache.Add("ETH/USDT", "Short-lived", time.Millisecond)

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	entries, err := cache.GetBlacklistedSymbols()
	assert.NoError(t, err)

	// Only the permanent one should be in the list
	assert.Len(t, entries, 1)
	assert.Equal(t, "BTC/USDT", entries[0].Symbol)
}

// TestMockSymbolCache_Get tests the Get method
func TestMockSymbolCache_Get(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)

	// Initially empty
	symbols, found := cache.Get("binance")
	assert.False(t, found)
	assert.Nil(t, symbols)
}

// TestMockSymbolCache_Set tests the Set method
func TestMockSymbolCache_Set(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)

	testSymbols := []string{"BTC/USDT", "ETH/USDT", "XRP/USDT"}
	cache.Set("binance", testSymbols)

	symbols, found := cache.Get("binance")
	assert.True(t, found)
	assert.Equal(t, testSymbols, symbols)
}

// TestMockSymbolCache_GetStats tests the GetStats method
func TestMockSymbolCache_GetStats(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)

	stats := cache.GetStats()
	// Should return empty stats for mock
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)
}

// TestMockSymbolCache_LogStats tests the LogStats method
func TestMockSymbolCache_LogStats(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)

	// Should not panic
	cache.LogStats()
}

// TestMockBlacklistCache_ConcurrentAccess tests thread safety
func TestMockBlacklistCache_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Concurrent writes and reads
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			symbol := fmt.Sprintf("TEST%d/USDT", idx)
			cache.Add(symbol, "Reason", time.Hour)
			cache.IsBlacklisted(symbol)
			cache.GetStats()
			cache.Remove(symbol)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestMockSymbolCache_ConcurrentAccess tests thread safety
func TestMockSymbolCache_ConcurrentAccess(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			exchangeID := fmt.Sprintf("exchange%d", idx)
			cache.Set(exchangeID, []string{"BTC/USDT"})
			cache.Get(exchangeID)
			cache.GetStats()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
