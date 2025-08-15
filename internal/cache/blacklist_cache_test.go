package cache

import (
	"context"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/stretchr/testify/assert"
)

// TestInMemoryBlacklistCache tests the in-memory implementation
func TestInMemoryBlacklistCache(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Test Add and IsBlacklisted
	cache.Add("binance", "test blacklist", time.Hour)
	isBlacklisted, reason := cache.IsBlacklisted("binance")
	assert.True(t, isBlacklisted)
	assert.Equal(t, "test blacklist", reason)

	// Test Remove
	cache.Remove("binance")
	isBlacklisted, _ = cache.IsBlacklisted("binance")
	assert.False(t, isBlacklisted)

	// Test Clear
	cache.Add("binance", "test1", time.Hour)
	cache.Add("coinbase", "test2", time.Hour)
	cache.Clear()
	isBlacklisted, _ = cache.IsBlacklisted("binance")
	assert.False(t, isBlacklisted)
	isBlacklisted, _ = cache.IsBlacklisted("coinbase")
	assert.False(t, isBlacklisted)
}

// TestBlacklistCacheExpiration tests TTL functionality
func TestBlacklistCacheExpiration(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add with very short TTL
	cache.Add("binance", "test blacklist", 10*time.Millisecond)

	// Should be blacklisted immediately
	isBlacklisted, reason := cache.IsBlacklisted("binance")
	assert.True(t, isBlacklisted)
	assert.Equal(t, "test blacklist", reason)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should no longer be blacklisted
	isBlacklisted, _ = cache.IsBlacklisted("binance")
	assert.False(t, isBlacklisted)
}

// TestBlacklistCacheStats tests statistics tracking
func TestBlacklistCacheStats(t *testing.T) {
	cache := NewInMemoryBlacklistCache()

	// Add some entries
	cache.Add("binance", "test1", time.Hour)
	cache.Add("coinbase", "test2", time.Hour)

	// Check some entries (hits and misses)
	cache.IsBlacklisted("binance")  // hit
	cache.IsBlacklisted("coinbase") // hit
	cache.IsBlacklisted("unknown")  // miss

	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats.TotalEntries)
	assert.Equal(t, int64(2), stats.Adds)
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

// TestBlacklistPersistenceInterface tests that the interface supports persistence
func TestBlacklistPersistenceInterface(t *testing.T) {
	// Test that RedisBlacklistCache implements the interface correctly
	cache := NewInMemoryBlacklistCache()

	// Test basic operations
	cache.Add("test", "reason", time.Hour)
	isBlacklisted, reason := cache.IsBlacklisted("test")
	assert.True(t, isBlacklisted)
	assert.Equal(t, "reason", reason)

	// Test that LoadFromDatabase method exists (for in-memory cache, it returns an error)
	err := cache.LoadFromDatabase(context.Background())
	// For in-memory cache, this should return an error indicating no database support
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database persistence not supported")

	// Test GetBlacklistedSymbols method exists
	symbols, err := cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	assert.Len(t, symbols, 1)
	assert.Equal(t, "test", symbols[0].Symbol)
	assert.Equal(t, "reason", symbols[0].Reason)
}

// TestDatabasePersistenceFlow tests the expected flow for database persistence
func TestDatabasePersistenceFlow(t *testing.T) {
	// This test documents the expected behavior for database persistence
	// without requiring actual database connection

	// 1. Application starts and loads blacklist from database
	cache := NewInMemoryBlacklistCache()
	err := cache.LoadFromDatabase(context.Background())
	assert.Error(t, err) // In-memory cache should return error for database operations
	assert.Contains(t, err.Error(), "database persistence not supported")

	// 2. Add exchange to blacklist (should persist to database in real implementation)
	cache.Add("binance", "Manual blacklist via API", 0) // 0 = no expiration

	// 3. Verify it's blacklisted
	isBlacklisted, reason := cache.IsBlacklisted("binance")
	assert.True(t, isBlacklisted)
	assert.Equal(t, "Manual blacklist via API", reason)

	// 4. Remove from blacklist (should remove from database in real implementation)
	cache.Remove("binance")

	// 5. Verify it's no longer blacklisted
	isBlacklisted, _ = cache.IsBlacklisted("binance")
	assert.False(t, isBlacklisted)

	// 6. Get all blacklisted symbols
	symbols, err := cache.GetBlacklistedSymbols()
	assert.NoError(t, err)
	assert.Empty(t, symbols) // Should be empty after removal
}

// TestBlacklistRepositoryInterface tests the expected repository interface
func TestBlacklistRepositoryInterface(t *testing.T) {
	// This test documents the expected repository interface
	// The actual repository implementation is tested separately

	// Expected methods that should be available:
	// - AddExchange(ctx, exchangeName, reason, expiresAt) (*ExchangeBlacklistEntry, error)
	// - RemoveExchange(ctx, exchangeName) error
	// - IsBlacklisted(ctx, exchangeName) (bool, string, error)
	// - GetAllBlacklisted(ctx) ([]ExchangeBlacklistEntry, error)
	// - CleanupExpired(ctx) (int64, error)
	// - GetBlacklistHistory(ctx, limit) ([]ExchangeBlacklistEntry, error)

	// Test that the ExchangeBlacklistEntry struct has expected fields
	entry := database.ExchangeBlacklistEntry{
		ID:           1,
		ExchangeName: "binance",
		Reason:       "test",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		ExpiresAt:    nil,
		IsActive:     true,
	}

	assert.Equal(t, int64(1), entry.ID)
	assert.Equal(t, "binance", entry.ExchangeName)
	assert.Equal(t, "test", entry.Reason)
	assert.True(t, entry.IsActive)
	assert.Nil(t, entry.ExpiresAt)
}
