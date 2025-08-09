package database

import (
	"context"
	"fmt"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type PostgresDB struct {
	Pool *pgxpool.Pool
}

func NewPostgresConnection(cfg *config.DatabaseConfig) (*PostgresDB, error) {
	var dsn string

	// Use DATABASE_URL if provided (common for cloud deployments like Digital Ocean)
	if cfg.DatabaseURL != "" {
		dsn = cfg.DatabaseURL
	} else {
		// Build DSN from individual components
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	}

	// Parse and configure connection pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool settings with bounds checking
	if cfg.MaxOpenConns > 0 && cfg.MaxOpenConns <= 2147483647 {
		poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 && cfg.MaxIdleConns <= 2147483647 {
		poolConfig.MinConns = int32(cfg.MaxIdleConns)
	}

	// Parse duration strings
	if cfg.ConnMaxLifetime != "" {
		if duration, err := time.ParseDuration(cfg.ConnMaxLifetime); err == nil {
			poolConfig.MaxConnLifetime = duration
		}
	}

	if cfg.ConnMaxIdleTime != "" {
		if duration, err := time.ParseDuration(cfg.ConnMaxIdleTime); err == nil {
			poolConfig.MaxConnIdleTime = duration
		}
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
