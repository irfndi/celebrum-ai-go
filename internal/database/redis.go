package database

import (
	"context"
	"fmt"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// ErrorRecoveryManager interface for dependency injection
type ErrorRecoveryManager interface {
	ExecuteWithRetry(ctx context.Context, operationName string, operation func() error) error
}

type RedisClient struct {
	Client *redis.Client
	logger *logrus.Logger
}

func NewRedisConnection(cfg config.RedisConfig) (*RedisClient, error) {
	return NewRedisConnectionWithRetry(cfg, nil)
}

func NewRedisConnectionWithRetry(cfg config.RedisConfig, errorRecoveryManager ErrorRecoveryManager) (*RedisClient, error) {
	logger := logrus.New()

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test the connection with retry logic if available
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var connectionErr error
	if errorRecoveryManager != nil {
		// Use retry mechanism
		connectionErr = errorRecoveryManager.ExecuteWithRetry(ctx, "redis_operation", func() error {
			return rdb.Ping(ctx).Err()
		})
	} else {
		// Fallback to simple connection test
		connectionErr = rdb.Ping(ctx).Err()
	}

	if connectionErr != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", connectionErr)
	}

	logger.Info("Successfully connected to Redis")

	return &RedisClient{
		Client: rdb,
		logger: logger,
	}, nil
}

func (r *RedisClient) Close() {
	if r.Client != nil {
		if err := r.Client.Close(); err != nil {
			// Log error but don't return it since this is a cleanup function
			if r.logger != nil {
				r.logger.Errorf("Error closing Redis client: %v", err)
			} else {
				logrus.Errorf("Error closing Redis client: %v", err)
			}
		}
		if r.logger != nil {
			r.logger.Info("Redis connection closed")
		} else {
			logrus.Info("Redis connection closed")
		}
	}
}

func (r *RedisClient) HealthCheck(ctx context.Context) error {
	if r.Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Ping(ctx).Err()
}

// Cache operations
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r.Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	if r.Client == nil {
		return "", fmt.Errorf("redis client is nil")
	}
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if r.Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	return r.Client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if r.Client == nil {
		return 0, fmt.Errorf("redis client is nil")
	}
	return r.Client.Exists(ctx, keys...).Result()
}
