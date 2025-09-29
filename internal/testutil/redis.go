package testutil

import (
	"os"

	"github.com/redis/go-redis/v9"
)

// GetTestRedisOptions returns Redis options for testing with configurable address
func GetTestRedisOptions() *redis.Options {
	redisAddr := os.Getenv("REDIS_TEST_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // fallback for local development
	}

	return &redis.Options{
		Addr: redisAddr,
		DB:   1, // Use test database
	}
}

// GetTestRedisClient returns a Redis client configured for testing
func GetTestRedisClient() *redis.Client {
	return redis.NewClient(GetTestRedisOptions())
}
