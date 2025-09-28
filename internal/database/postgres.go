package database

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

const maxAllowedPoolConns int32 = 10000

func NewPostgresConnection(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	poolConfig, err := buildPGXPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Create connection pool with retry logic
	var pool *pgxpool.Pool
	for attempts := 0; attempts < 3; attempts++ {
		pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			break
		}
		logrus.Warnf("Database connection attempt %d failed: %v", attempts+1, err)
		if attempts < 2 {
			time.Sleep(time.Duration(attempts+1) * time.Second)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool after retries: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logrus.Info("Successfully connected to PostgreSQL")

	return &PostgresDB{Pool: pool}, nil
}

func (db *PostgresDB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		logrus.Info("PostgreSQL connection closed")
	}
}

func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func buildPGXPoolConfig(cfg *config.DatabaseConfig) (*pgxpool.Config, error) {
	var dsn string

	if cfg.DatabaseURL != "" {
		dsn = cfg.DatabaseURL
	} else {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		poolConfig.MaxConns = clampToSafePoolSize(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		poolConfig.MinConns = clampToSafePoolSize(cfg.MaxIdleConns)
	}

	if cfg.ConnMaxLifetime != "" {
		duration, err := time.ParseDuration(cfg.ConnMaxLifetime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ConnMaxLifetime: %w", err)
		}
		poolConfig.MaxConnLifetime = duration
	}

	if cfg.ConnMaxIdleTime != "" {
		duration, err := time.ParseDuration(cfg.ConnMaxIdleTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ConnMaxIdleTime: %w", err)
		}
		poolConfig.MaxConnIdleTime = duration
	}

	return poolConfig, nil
}

func clampToSafePoolSize(value int) int32 {
	requested := int64(value)
	if requested <= 0 {
		return 0
	}

	if requested > int64(math.MaxInt32) {
		logrus.Warnf("Configured pool size %d exceeds int32 limit; clamping to %d", value, maxAllowedPoolConns)
		return maxAllowedPoolConns
	}

	if requested > int64(maxAllowedPoolConns) {
		logrus.Warnf("Configured pool size %d exceeds safe limit %d; clamping", value, maxAllowedPoolConns)
		return maxAllowedPoolConns
	}

	return int32(requested)
}
