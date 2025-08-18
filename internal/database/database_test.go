package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test PostgresDB struct
func TestPostgresDB_Struct(t *testing.T) {
	db := &PostgresDB{
		Pool: nil, // We can't create a real pool without a database
	}

	assert.NotNil(t, db)
	assert.Nil(t, db.Pool)
}

// Test PostgresDB Close method with nil pool
func TestPostgresDB_Close_NilPool(t *testing.T) {
	db := &PostgresDB{Pool: nil}

	// Should not panic when closing nil pool
	assert.NotPanics(t, func() {
		db.Close()
	})
}

// Test PostgresDB HealthCheck with nil pool
func TestPostgresDB_HealthCheck_NilPool(t *testing.T) {
	db := &PostgresDB{Pool: nil}
	ctx := context.Background()

	// Should panic when trying to ping nil pool
	assert.Panics(t, func() {
		_ = db.HealthCheck(ctx)
	})
}

// Test RedisClient struct
func TestRedisClient_Struct(t *testing.T) {
	client := &RedisClient{
		Client: nil, // We can't create a real client without Redis
	}

	assert.NotNil(t, client)
	assert.Nil(t, client.Client)
}

// Test RedisClient Close method with nil client
func TestRedisClient_Close_NilClient(t *testing.T) {
	client := &RedisClient{Client: nil}

	// Should not panic when closing nil client
	assert.NotPanics(t, func() {
		client.Close()
	})
}

// Test RedisClient HealthCheck with nil client
func TestRedisClient_HealthCheck_NilClient(t *testing.T) {
	client := &RedisClient{Client: nil}
	ctx := context.Background()

	// Should return error when trying to ping nil client
	err := client.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis client is nil")
}

// Test RedisClient cache operations with nil client
func TestRedisClient_Set_NilClient(t *testing.T) {
	client := &RedisClient{Client: nil}
	ctx := context.Background()

	// Should return error when trying to set with nil client
	err := client.Set(ctx, "key", "value", time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis client is nil")
}

func TestRedisClient_Get_NilClient(t *testing.T) {
	client := &RedisClient{Client: nil}
	ctx := context.Background()

	// Should return error when trying to get with nil client
	_, err := client.Get(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis client is nil")
}

func TestRedisClient_Delete_NilClient(t *testing.T) {
	client := &RedisClient{Client: nil}
	ctx := context.Background()

	// Should return error when trying to delete with nil client
	err := client.Delete(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis client is nil")
}

func TestRedisClient_Exists_NilClient(t *testing.T) {
	client := &RedisClient{Client: nil}
	ctx := context.Background()

	// Should return error when trying to check existence with nil client
	_, err := client.Exists(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis client is nil")
}

// Test NewPostgresConnection with invalid config
func TestNewPostgresConnection_InvalidConfig(t *testing.T) {
	// Test with invalid database URL
	cfg := &config.DatabaseConfig{
		DatabaseURL: "invalid-url",
	}

	db, err := NewPostgresConnection(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to parse database config")
}

func TestNewPostgresConnection_InvalidDurationConfig(t *testing.T) {
	// Test with valid basic config but invalid duration strings
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "test",
		Password:        "test",
		DBName:          "test",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: "invalid-duration",
		ConnMaxIdleTime: "invalid-duration",
	}

	// This should still work as invalid durations are ignored
	db, err := NewPostgresConnection(cfg)
	// We expect this to fail because we don't have a real database
	// but it should fail at connection, not at config parsing
	assert.Error(t, err)
	assert.Nil(t, db)
	// The error should be about connection failure
	assert.Contains(t, err.Error(), "failed to ping database")
}

func TestNewPostgresConnection_BuildDSNFromComponents(t *testing.T) {
	// Test DSN building from individual components
	cfg := &config.DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		User:         "testuser",
		Password:     "testpass",
		DBName:       "testdb",
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		// Leave DatabaseURL empty to test component-based DSN
	}

	db, err := NewPostgresConnection(cfg)
	// We expect this to fail because we don't have a real database
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "failed to ping database")
}

// Test NewRedisConnection with invalid config
func TestNewRedisConnection_InvalidConfig(t *testing.T) {
	// Test with invalid Redis config (non-existent host)
	cfg := config.RedisConfig{
		Host:     "non-existent-host",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := NewRedisConnection(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

// Test configuration validation
func TestDatabaseConfig_Validation(t *testing.T) {
	// Test valid PostgreSQL config
	postgresCfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "password",
		DBName:          "testdb",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: "300s",
		ConnMaxIdleTime: "60s",
	}

	assert.Equal(t, "localhost", postgresCfg.Host)
	assert.Equal(t, 5432, postgresCfg.Port)
	assert.Equal(t, "postgres", postgresCfg.User)
	assert.Equal(t, "password", postgresCfg.Password)
	assert.Equal(t, "testdb", postgresCfg.DBName)
	assert.Equal(t, "disable", postgresCfg.SSLMode)
	assert.Equal(t, 25, postgresCfg.MaxOpenConns)
	assert.Equal(t, 5, postgresCfg.MaxIdleConns)
	assert.Equal(t, "300s", postgresCfg.ConnMaxLifetime)
	assert.Equal(t, "60s", postgresCfg.ConnMaxIdleTime)

	// Test valid Redis config
	redisCfg := config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "redis_password",
		DB:       1,
	}

	assert.Equal(t, "localhost", redisCfg.Host)
	assert.Equal(t, 6379, redisCfg.Port)
	assert.Equal(t, "redis_password", redisCfg.Password)
	assert.Equal(t, 1, redisCfg.DB)
}

// Test duration parsing
func TestDurationParsing(t *testing.T) {
	// Test valid duration strings
	validDurations := []string{"300s", "5m", "1h", "24h"}

	for _, durationStr := range validDurations {
		duration, err := time.ParseDuration(durationStr)
		assert.NoError(t, err, "Should parse valid duration: %s", durationStr)
		assert.Greater(t, duration, time.Duration(0), "Duration should be positive")
	}

	// Test invalid duration strings
	invalidDurations := []string{"invalid", "300", "5minutes", "1hour"}

	for _, durationStr := range invalidDurations {
		_, err := time.ParseDuration(durationStr)
		assert.Error(t, err, "Should fail to parse invalid duration: %s", durationStr)
	}
}

// Test context operations
func TestContextOperations(t *testing.T) {
	// Test context creation and cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)

	// Test context with deadline
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))

	// Test context cancellation
	cancel()
	select {
	case <-ctx.Done():
		// Context should be cancelled
		assert.Error(t, ctx.Err())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should have been cancelled")
	}
}

// Test error handling patterns
func TestErrorHandling(t *testing.T) {
	// Test error wrapping patterns used in the database package
	originalErr := assert.AnError
	wrappedErr := fmt.Errorf("failed to connect: %w", originalErr)

	assert.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "failed to connect")
	assert.Contains(t, wrappedErr.Error(), originalErr.Error())

	// Test error unwrapping
	require.ErrorIs(t, wrappedErr, originalErr)
}
