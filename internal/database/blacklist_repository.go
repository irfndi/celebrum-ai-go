package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ExchangeBlacklistEntry represents a blacklisted exchange in the database
type ExchangeBlacklistEntry struct {
	ID           int64      `json:"id" db:"id"`
	ExchangeName string     `json:"exchange_name" db:"exchange_name"`
	Reason       string     `json:"reason" db:"reason"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive     bool       `json:"is_active" db:"is_active"`
}

// BlacklistRepository handles database operations for exchange blacklist
type BlacklistRepository struct {
	pool *pgxpool.Pool
}

// NewBlacklistRepository creates a new blacklist repository
func NewBlacklistRepository(pool *pgxpool.Pool) *BlacklistRepository {
	return &BlacklistRepository{
		pool: pool,
	}
}

// AddExchange adds an exchange to the blacklist
func (r *BlacklistRepository) AddExchange(ctx context.Context, exchangeName, reason string, expiresAt *time.Time) (*ExchangeBlacklistEntry, error) {
	query := `
		INSERT INTO exchange_blacklist (exchange_name, reason, expires_at, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (exchange_name) WHERE is_active = true
		DO UPDATE SET 
			reason = EXCLUDED.reason,
			expires_at = EXCLUDED.expires_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, exchange_name, reason, created_at, updated_at, expires_at, is_active
	`

	var entry ExchangeBlacklistEntry
	err := r.pool.QueryRow(ctx, query, exchangeName, reason, expiresAt).Scan(
		&entry.ID,
		&entry.ExchangeName,
		&entry.Reason,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&entry.ExpiresAt,
		&entry.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add exchange to blacklist: %w", err)
	}

	return &entry, nil
}

// RemoveExchange removes an exchange from the blacklist
func (r *BlacklistRepository) RemoveExchange(ctx context.Context, exchangeName string) error {
	query := `
		UPDATE exchange_blacklist 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE exchange_name = $1 AND is_active = true
	`

	result, err := r.pool.Exec(ctx, query, exchangeName)
	if err != nil {
		return fmt.Errorf("failed to remove exchange from blacklist: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("exchange %s not found in blacklist or already inactive", exchangeName)
	}

	return nil
}

// IsBlacklisted checks if an exchange is currently blacklisted
func (r *BlacklistRepository) IsBlacklisted(ctx context.Context, exchangeName string) (bool, string, error) {
	query := `
		SELECT reason, expires_at
		FROM exchange_blacklist 
		WHERE exchange_name = $1 AND is_active = true
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`

	var reason string
	var expiresAt *time.Time
	err := r.pool.QueryRow(ctx, query, exchangeName).Scan(&reason, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("failed to check blacklist status: %w", err)
	}

	return true, reason, nil
}

// GetAllBlacklisted returns all currently blacklisted exchanges
func (r *BlacklistRepository) GetAllBlacklisted(ctx context.Context) ([]ExchangeBlacklistEntry, error) {
	query := `
		SELECT id, exchange_name, reason, created_at, updated_at, expires_at, is_active
		FROM exchange_blacklist 
		WHERE is_active = true
		AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get blacklisted exchanges: %w", err)
	}
	defer rows.Close()

	var entries []ExchangeBlacklistEntry
	for rows.Next() {
		var entry ExchangeBlacklistEntry
		err := rows.Scan(
			&entry.ID,
			&entry.ExchangeName,
			&entry.Reason,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.ExpiresAt,
			&entry.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blacklist entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blacklist entries: %w", err)
	}

	return entries, nil
}

// CleanupExpired removes expired blacklist entries
func (r *BlacklistRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `
		UPDATE exchange_blacklist 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE is_active = true 
		AND expires_at IS NOT NULL 
		AND expires_at <= CURRENT_TIMESTAMP
	`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired blacklist entries: %w", err)
	}

	return result.RowsAffected(), nil
}

// GetBlacklistHistory returns the history of blacklist changes
func (r *BlacklistRepository) GetBlacklistHistory(ctx context.Context, limit int) ([]ExchangeBlacklistEntry, error) {
	query := `
		SELECT id, exchange_name, reason, created_at, updated_at, expires_at, is_active
		FROM exchange_blacklist 
		ORDER BY updated_at DESC
		LIMIT $1
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get blacklist history: %w", err)
	}
	defer rows.Close()

	var entries []ExchangeBlacklistEntry
	for rows.Next() {
		var entry ExchangeBlacklistEntry
		err := rows.Scan(
			&entry.ID,
			&entry.ExchangeName,
			&entry.Reason,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&entry.ExpiresAt,
			&entry.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan blacklist history entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating blacklist history: %w", err)
	}

	return entries, nil
}

// ClearAll deactivates all blacklist entries
func (r *BlacklistRepository) ClearAll(ctx context.Context) (int64, error) {
	query := `
		UPDATE exchange_blacklist 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE is_active = true
	`

	result, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to clear all blacklist entries: %w", err)
	}

	return result.RowsAffected(), nil
}
