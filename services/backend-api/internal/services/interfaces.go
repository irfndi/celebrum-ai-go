package services

import (
	"context"

	"github.com/irfandi/celebrum-ai-go/internal/ccxt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBPool defines the interface for database operations
type DBPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	Close()
}

// CCXTClient defines the interface for CCXT client operations used by services
type CCXTClient interface {
	GetFundingRates(ctx context.Context, exchange string, symbols []string) ([]ccxt.FundingRate, error)
}

// SignalAggregatorInterface defines the interface for signal aggregation
type SignalAggregatorInterface interface {
	AggregateArbitrageSignals(ctx context.Context, input ArbitrageSignalInput) ([]*AggregatedSignal, error)
	AggregateTechnicalSignals(ctx context.Context, input TechnicalSignalInput) ([]*AggregatedSignal, error)
	DeduplicateSignals(ctx context.Context, signals []*AggregatedSignal) ([]*AggregatedSignal, error)
}
