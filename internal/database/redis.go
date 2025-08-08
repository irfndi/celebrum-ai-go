package database

import (
	"context"
	"fmt"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisConnection(cfg config.RedisConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logrus.Info("Successfully connected to Redis")

	return &RedisClient{Client: rdb}, nil
}

func (r *RedisClient) Close() {
	if r.Client != nil {
		r.Client.Close()
		logrus.Info("Redis connection closed")
	}
}

func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// Cache operations
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return r.Client.Exists(ctx, keys...).Result()
}
