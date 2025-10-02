package interfaces

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	// Test that configuration loading works without panicking
	config, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Test some default values
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestBlacklistCacheEntry(t *testing.T) {
	// Test BlacklistCacheEntry struct
	entry := BlacklistCacheEntry{
		Symbol:    "TEST/USDT",
		Reason:    "Test blacklist",
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "TEST/USDT", entry.Symbol)
	assert.Equal(t, "Test blacklist", entry.Reason)
	assert.False(t, entry.CreatedAt.IsZero())
}

func TestExchangeBlacklistEntry(t *testing.T) {
	// Test ExchangeBlacklistEntry struct
	entry := ExchangeBlacklistEntry{
		ID:           1,
		ExchangeName: "test_exchange",
		Reason:       "Test reason",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	assert.Equal(t, int64(1), entry.ID)
	assert.Equal(t, "test_exchange", entry.ExchangeName)
	assert.Equal(t, "Test reason", entry.Reason)
}

func TestSymbolCacheEntry(t *testing.T) {
	// Test SymbolCacheEntry struct
	entry := SymbolCacheEntry{
		Symbols:   []string{"BTC/USDT", "ETH/USDT"},
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	assert.Len(t, entry.Symbols, 2)
	assert.Equal(t, "BTC/USDT", entry.Symbols[0])
	assert.Equal(t, "ETH/USDT", entry.Symbols[1])
	assert.False(t, entry.CachedAt.IsZero())
	assert.False(t, entry.ExpiresAt.IsZero())
}

func TestBlacklistCacheStats(t *testing.T) {
	// Test BlacklistCacheStats struct
	stats := BlacklistCacheStats{
		TotalEntries:   10,
		ExpiredEntries: 2,
		Hits:           100,
		Misses:         20,
		Adds:           15,
	}

	assert.Equal(t, int64(10), stats.TotalEntries)
	assert.Equal(t, int64(2), stats.ExpiredEntries)
	assert.Equal(t, int64(100), stats.Hits)
	assert.Equal(t, int64(20), stats.Misses)
	assert.Equal(t, int64(15), stats.Adds)
}

func TestSymbolCacheStats(t *testing.T) {
	// Test SymbolCacheStats struct
	stats := SymbolCacheStats{
		Hits:   50,
		Misses: 10,
		Sets:   25,
	}

	assert.Equal(t, int64(50), stats.Hits)
	assert.Equal(t, int64(10), stats.Misses)
	assert.Equal(t, int64(25), stats.Sets)
}
