package interfaces

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// Test configuration loading
	config, err := Load()
	
	// Should not crash even without config file
	if err != nil {
		// This is expected in test environment
		assert.Contains(t, err.Error(), "Config File")
	}
	
	if config != nil {
		// Verify default values are set
		assert.NotEmpty(t, config.Environment)
		assert.NotEmpty(t, config.LogLevel)
		assert.Greater(t, config.Server.Port, 0)
	}
}

func TestSetDefaults(t *testing.T) {
	// Test that defaults are properly set
	// This is tested indirectly through Load(), but we can verify specific defaults
	
	config, err := Load()
	if err != nil && config == nil {
		t.Skip("Cannot test defaults without config loading")
		return
	}
	
	// Verify some key defaults
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, true, config.Arbitrage.Enabled)
	assert.Equal(t, 0.5, config.Arbitrage.MinProfitThreshold)
}

func TestNewInMemoryBlacklistCache(t *testing.T) {
	cache := NewInMemoryBlacklistCache()
	assert.NotNil(t, cache)
	
	// Test interface compliance
	assert.Implements(t, (*BlacklistCache)(nil), cache)
}

func TestMockBlacklistCache_IsBlacklisted(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Test with empty cache
	isBlacklisted, reason := cache.IsBlacklisted("BTC/USDT")
	assert.False(t, isBlacklisted)
	assert.Empty(t, reason)
	
	// Add a symbol and test
	cache.Add("BTC/USDT", "test reason", time.Hour)
	isBlacklisted, reason = cache.IsBlacklisted("BTC/USDT")
	assert.True(t, isBlacklisted)
	assert.Equal(t, "test reason", reason)
	
	// Test with expired entry (using very small TTL to simulate expiration)
	cache.Add("ETH/USDT", "expired reason", time.Nanosecond)
	// Small sleep to ensure expiration
	time.Sleep(time.Millisecond)
	isBlacklisted, reason = cache.IsBlacklisted("ETH/USDT")
	assert.False(t, isBlacklisted)
	assert.Empty(t, reason)
}

func TestMockBlacklistCache_Add(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Add symbol with TTL
	cache.Add("BTC/USDT", "test reason", time.Hour)
	assert.Contains(t, cache.entries, "BTC/USDT")
	assert.Equal(t, "test reason", cache.entries["BTC/USDT"].Reason)
	assert.NotNil(t, cache.entries["BTC/USDT"].ExpiresAt)
	
	// Add symbol without TTL (permanent)
	cache.Add("ETH/USDT", "permanent reason", 0)
	assert.Contains(t, cache.entries, "ETH/USDT")
	assert.Equal(t, "permanent reason", cache.entries["ETH/USDT"].Reason)
	assert.Nil(t, cache.entries["ETH/USDT"].ExpiresAt)
}

func TestMockBlacklistCache_Remove(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Add symbol first
	cache.Add("BTC/USDT", "test reason", time.Hour)
	assert.Contains(t, cache.entries, "BTC/USDT")
	
	// Remove it
	cache.Remove("BTC/USDT")
	assert.NotContains(t, cache.entries, "BTC/USDT")
	
	// Remove non-existent symbol (should not panic)
	cache.Remove("NONEXISTENT")
	assert.NotContains(t, cache.entries, "NONEXISTENT")
}

func TestMockBlacklistCache_Clear(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Add multiple symbols
	cache.Add("BTC/USDT", "reason 1", time.Hour)
	cache.Add("ETH/USDT", "reason 2", time.Hour)
	cache.Add("BNB/USDT", "reason 3", time.Hour)
	assert.Equal(t, 3, len(cache.entries))
	
	// Clear all
	cache.Clear()
	assert.Empty(t, cache.entries)
}

func TestMockBlacklistCache_GetStats(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Test with empty cache
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.TotalEntries)
	assert.False(t, stats.LastCleanup.IsZero())
	
	// Add some entries
	cache.Add("BTC/USDT", "reason 1", time.Hour)
	cache.Add("ETH/USDT", "reason 2", time.Hour)
	
	stats = cache.GetStats()
	assert.Equal(t, int64(2), stats.TotalEntries)
}

func TestMockBlacklistCache_LogStats(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Should not panic
	cache.LogStats()
}

func TestMockBlacklistCache_Close(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Add some data
	cache.Add("BTC/USDT", "test reason", time.Hour)
	
	// Close should not panic and should clear data
	err := cache.Close()
	assert.NoError(t, err)
	assert.Nil(t, cache.entries)
}

func TestMockBlacklistCache_LoadFromDatabase(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Mock context
	ctx := context.Background()
	
	// Should not panic and return nil error
	err := cache.LoadFromDatabase(ctx)
	assert.NoError(t, err)
}

func TestMockBlacklistCache_GetBlacklistedSymbols(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Test with empty cache
	symbols, err := cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	assert.Empty(t, symbols)
	
	// Add some entries
	cache.Add("BTC/USDT", "reason 1", time.Hour)
	cache.Add("ETH/USDT", "reason 2", time.Nanosecond) // Will expire quickly
	cache.Add("BNB/USDT", "reason 3", time.Hour)
	
	// Let the ETH/USDT entry expire
	time.Sleep(time.Millisecond)
	
	symbols, err = cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	
	// Should only return non-expired entries
	assert.Len(t, symbols, 2)
	
	// Verify entries
	symbolMap := make(map[string]BlacklistCacheEntry)
	for _, symbol := range symbols {
		symbolMap[symbol.Symbol] = symbol
	}
	
	assert.Contains(t, symbolMap, "BTC/USDT")
	assert.Contains(t, symbolMap, "BNB/USDT")
	assert.NotContains(t, symbolMap, "ETH/USDT")
}

func TestMockBlacklistCache_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			symbol := "SYMBOL"
			cache.Add(symbol, "reason", time.Hour)
			cache.IsBlacklisted(symbol)
			cache.Remove(symbol)
			cache.GetStats()
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should not panic
	assert.NotNil(t, cache)
}

func TestNewRedisSymbolCache(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour)
	assert.NotNil(t, cache)
	
	// Test interface compliance
	assert.Implements(t, (*SymbolCacheInterface)(nil), cache)
}

func TestMockSymbolCache_Get(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour).(*mockSymbolCache)
	
	// Test with empty cache
	symbols, found := cache.Get("binance")
	assert.False(t, found)
	assert.Nil(t, symbols)
	
	// Add a symbol and test
	cache.Set("binance", []string{"BTC/USDT", "ETH/USDT"})
	symbols, found = cache.Get("binance")
	assert.True(t, found)
	assert.Equal(t, []string{"BTC/USDT", "ETH/USDT"}, symbols)
	
	// Test with expired entry
	cache.Set("coinbase", []string{"BTC/USD"})
	cache.entries["coinbase"] = SymbolCacheEntry{
		Symbols:   []string{"BTC/USD"},
		CachedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	symbols, found = cache.Get("coinbase")
	assert.False(t, found)
	assert.Nil(t, symbols)
}

func TestMockSymbolCache_Set(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour).(*mockSymbolCache)
	
	// Set symbols for an exchange
	symbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT"}
	cache.Set("binance", symbols)
	
	assert.Contains(t, cache.entries, "binance")
	assert.Equal(t, symbols, cache.entries["binance"].Symbols)
	assert.False(t, cache.entries["binance"].CachedAt.IsZero())
	assert.True(t, cache.entries["binance"].ExpiresAt.After(time.Now()))
}

func TestMockSymbolCache_GetStats(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour).(*mockSymbolCache)
	
	stats := cache.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)
}

func TestMockSymbolCache_LogStats(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour).(*mockSymbolCache)
	
	// Should not panic
	cache.LogStats()
}

func TestMockSymbolCache_ConcurrentAccess(t *testing.T) {
	cache := NewRedisSymbolCache(nil, time.Hour).(*mockSymbolCache)
	
	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			cache.Set("exchange", []string{"SYMBOL"})
			cache.Get("exchange")
			cache.GetStats()
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Should not panic
	assert.NotNil(t, cache)
}

func TestBlacklistCacheEntry_Expiration(t *testing.T) {
	// Test behavior with expired entries
	cache := NewInMemoryBlacklistCache().(*mockBlacklistCache)
	
	// Initialize the entries map if it's nil
	if cache.entries == nil {
		cache.entries = make(map[string]BlacklistCacheEntry)
	}
	
	// Add an entry that expires in the future
	futureTime := time.Now().Add(time.Hour)
	cache.entries["FUTURE"] = BlacklistCacheEntry{
		Symbol:    "FUTURE",
		Reason:    "future reason",
		ExpiresAt: &futureTime,
		CreatedAt: time.Now(),
	}
	
	// Add an entry that expires in the past
	pastTime := time.Now().Add(-time.Hour)
	cache.entries["PAST"] = BlacklistCacheEntry{
		Symbol:    "PAST",
		Reason:    "past reason",
		ExpiresAt: &pastTime,
		CreatedAt: time.Now(),
	}
	
	// Add an entry with no expiration (nil)
	cache.entries["PERMANENT"] = BlacklistCacheEntry{
		Symbol:    "PERMANENT",
		Reason:    "permanent reason",
		ExpiresAt: nil,
		CreatedAt: time.Now(),
	}
	
	// Test IsBlacklisted behavior
	isBlacklisted, _ := cache.IsBlacklisted("FUTURE")
	assert.True(t, isBlacklisted)
	
	isBlacklisted, _ = cache.IsBlacklisted("PAST")
	assert.False(t, isBlacklisted)
	
	isBlacklisted, _ = cache.IsBlacklisted("PERMANENT")
	assert.True(t, isBlacklisted)
	
	// Test GetBlacklistedSymbols behavior
	symbols, err := cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	assert.Len(t, symbols, 2) // FUTURE and PERMANENT, but not PAST
}

func TestBlacklistCacheInterface_Compliance(t *testing.T) {
	// Test that mock implementation fully satisfies the interface
	var cache BlacklistCache = NewInMemoryBlacklistCache()
	
	// All these methods should be available and not panic
	_, _ = cache.IsBlacklisted("test")
	cache.Add("test", "reason", time.Hour)
	cache.Remove("test")
	cache.Clear()
	_ = cache.GetStats()
	cache.LogStats()
	_ = cache.Close()
	
	ctx := context.Background()
	_ = cache.LoadFromDatabase(ctx)
	_, _ = cache.GetBlacklistedSymbols()
}

func TestSymbolCacheInterface_Compliance(t *testing.T) {
	// Test that mock implementation fully satisfies the interface
	var cache SymbolCacheInterface = NewRedisSymbolCache(nil, time.Hour)
	
	// All these methods should be available and not panic
	_, _ = cache.Get("test")
	cache.Set("test", []string{"test"})
	_ = cache.GetStats()
	cache.LogStats()
}