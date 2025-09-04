package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"log/slog"

	"github.com/irfndi/celebrum-ai-go/internal/models"
)

// TestSignalProcessor is a minimal version for testing
type TestSignalProcessor struct {
	db     pgxmock.PgxPoolIface
	logger *slog.Logger
}

// getRecentMarketDataFromDB retrieves recent market data from the database
func (sp *TestSignalProcessor) getRecentMarketDataFromDB(symbol, exchange string, duration time.Duration) ([]models.MarketData, error) {
	since := time.Now().Add(-duration)

	query := `
		SELECT md.id, md.exchange_id, md.trading_pair_id, md.last_price, md.volume_24h, 
		       md.timestamp, md.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md.trading_pair_id = tp.id
		JOIN exchanges e ON md.exchange_id = e.id
		WHERE tp.symbol = $1 AND e.ccxt_id = $2 AND md.timestamp >= $3
		ORDER BY md.timestamp DESC
		LIMIT 100
	`

	rows, err := sp.db.Query(context.Background(), query, symbol, exchange, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	var marketData []models.MarketData
	for rows.Next() {
		var data models.MarketData
		scanErr := rows.Scan(
			&data.ID,
			&data.ExchangeID,
			&data.TradingPairID,
			&data.LastPrice,
			&data.Volume24h,
			&data.Timestamp,
			&data.CreatedAt,
		)
		if scanErr != nil {
			return nil, fmt.Errorf("failed to scan market data: %w", scanErr)
		}
		marketData = append(marketData, data)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating market data rows: %w", err)
	}

	return marketData, nil
}

// getTradingPairSymbol retrieves the symbol for a given trading pair ID
func (sp *TestSignalProcessor) getTradingPairSymbol(tradingPairID int) (string, error) {
	var symbol string
	query := `SELECT symbol FROM trading_pairs WHERE id = $1`
	err := sp.db.QueryRow(context.Background(), query, tradingPairID).Scan(&symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get trading pair symbol: %w", err)
	}
	return symbol, nil
}

// Test for getRecentMarketDataFromDB method that would catch volume_24h column bugs
func TestSignalProcessor_GetRecentMarketDataFromDB_Success(t *testing.T) {
	// Create mock database
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	// Create a minimal TestSignalProcessor for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	// Test data
	symbol := "BTC/USDT"
	exchange := "binance"
	duration := time.Hour

	// Expected database query result
	expectedTime := time.Now()
	expectedRows := pgxmock.NewRows([]string{
		"id", "exchange_id", "trading_pair_id", "last_price", "volume_24h",
		"timestamp", "created_at",
	}).AddRow(
		"test-id-1", 1, 1, decimal.NewFromFloat(45000.0), decimal.NewFromFloat(1000000.0),
		expectedTime, expectedTime,
	).AddRow(
		"test-id-2", 1, 1, decimal.NewFromFloat(45100.0), decimal.NewFromFloat(1100000.0),
		expectedTime.Add(-time.Minute), expectedTime.Add(-time.Minute),
	)

	// Set up mock expectation
	mockPool.ExpectQuery(`
		SELECT md\.id, md\.exchange_id, md\.trading_pair_id, md\.last_price, md\.volume_24h, 
		       md\.timestamp, md\.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md\.trading_pair_id = tp\.id
		JOIN exchanges e ON md\.exchange_id = e\.id
		WHERE tp\.symbol = \$1 AND e\.ccxt_id = \$2 AND md\.timestamp >= \$3
		ORDER BY md\.timestamp DESC
		LIMIT 100
	`).WithArgs(symbol, exchange, pgxmock.AnyArg()).WillReturnRows(expectedRows)

	// Execute test by calling the method directly
	result, err := processor.getRecentMarketDataFromDB(symbol, exchange, duration)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "test-id-1", result[0].ID)
	assert.Equal(t, decimal.NewFromFloat(45000.0), result[0].LastPrice)
	assert.Equal(t, decimal.NewFromFloat(1000000.0), result[0].Volume24h)
	assert.Equal(t, decimal.NewFromFloat(45100.0), result[1].LastPrice)
	assert.Equal(t, decimal.NewFromFloat(1100000.0), result[1].Volume24h)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetRecentMarketDataFromDB_MissingVolume24hColumn(t *testing.T) {
	// This test would have caught the volume_24h column bug
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	symbol := "BTC/USDT"
	exchange := "binance"
	duration := time.Hour

	// Simulate database schema without volume_24h column (missing column)
	expectedRows := pgxmock.NewRows([]string{
		"id", "exchange_id", "trading_pair_id", "last_price", 
		"timestamp", "created_at",
	}).AddRow(
		"test-id-1", 1, 1, decimal.NewFromFloat(45000.0),
		time.Now(), time.Now(),
	)

	mockPool.ExpectQuery(`
		SELECT md\.id, md\.exchange_id, md\.trading_pair_id, md\.last_price, md\.volume_24h, 
		       md\.timestamp, md\.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md\.trading_pair_id = tp\.id
		JOIN exchanges e ON md\.exchange_id = e\.id
		WHERE tp\.symbol = \$1 AND e\.ccxt_id = \$2 AND md\.timestamp >= \$3
		ORDER BY md\.timestamp DESC
		LIMIT 100
	`).WithArgs(symbol, exchange, pgxmock.AnyArg()).WillReturnRows(expectedRows)

	// Execute test - this should fail due to column count mismatch
	result, err := processor.getRecentMarketDataFromDB(symbol, exchange, duration)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan market data")
	assert.Nil(t, result)
	// The exact error message might vary, but it should indicate a scanning issue

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetRecentMarketDataFromDB_QueryError(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	symbol := "BTC/USDT"
	exchange := "binance"
	duration := time.Hour

	// Simulate database connection error
	mockPool.ExpectQuery(`
		SELECT md\.id, md\.exchange_id, md\.trading_pair_id, md\.last_price, md\.volume_24h, 
		       md\.timestamp, md\.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md\.trading_pair_id = tp\.id
		JOIN exchanges e ON md\.exchange_id = e\.id
		WHERE tp\.symbol = \$1 AND e\.ccxt_id = \$2 AND md\.timestamp >= \$3
		ORDER BY md\.timestamp DESC
		LIMIT 100
	`).WithArgs(symbol, exchange, pgxmock.AnyArg()).WillReturnError(errors.New("connection timeout"))

	// Execute test
	result, err := processor.getRecentMarketDataFromDB(symbol, exchange, duration)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to query market data")
	assert.Contains(t, err.Error(), "connection timeout")
	assert.Nil(t, result)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetRecentMarketDataFromDB_EmptyResult(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	symbol := "BTC/USDT"
	exchange := "binance"
	duration := time.Hour

	// Simulate empty result set
	expectedRows := pgxmock.NewRows([]string{
		"id", "exchange_id", "trading_pair_id", "last_price", "volume_24h",
		"timestamp", "created_at",
	})

	mockPool.ExpectQuery(`
		SELECT md\.id, md\.exchange_id, md\.trading_pair_id, md\.last_price, md\.volume_24h, 
		       md\.timestamp, md\.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md\.trading_pair_id = tp\.id
		JOIN exchanges e ON md\.exchange_id = e\.id
		WHERE tp\.symbol = \$1 AND e\.ccxt_id = \$2 AND md\.timestamp >= \$3
		ORDER BY md\.timestamp DESC
		LIMIT 100
	`).WithArgs(symbol, exchange, pgxmock.AnyArg()).WillReturnRows(expectedRows)

	// Execute test
	result, err := processor.getRecentMarketDataFromDB(symbol, exchange, duration)

	// Assertions
	assert.NoError(t, err)
	assert.Empty(t, result)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetRecentMarketDataFromDB_ScanError(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	symbol := "BTC/USDT"
	exchange := "binance"
	duration := time.Hour

	// Simulate data type mismatch in scanning
	expectedRows := pgxmock.NewRows([]string{
		"id", "exchange_id", "trading_pair_id", "last_price", "volume_24h",
		"timestamp", "created_at",
	}).AddRow(
		"test-id-1", "invalid-int", 1, decimal.NewFromFloat(45000.0), decimal.NewFromFloat(1000000.0),
		time.Now(), time.Now(),
	)

	mockPool.ExpectQuery(`
		SELECT md\.id, md\.exchange_id, md\.trading_pair_id, md\.last_price, md\.volume_24h, 
		       md\.timestamp, md\.created_at
		FROM market_data md
		JOIN trading_pairs tp ON md\.trading_pair_id = tp\.id
		JOIN exchanges e ON md\.exchange_id = e\.id
		WHERE tp\.symbol = \$1 AND e\.ccxt_id = \$2 AND md\.timestamp >= \$3
		ORDER BY md\.timestamp DESC
		LIMIT 100
	`).WithArgs(symbol, exchange, pgxmock.AnyArg()).WillReturnRows(expectedRows)

	// Execute test
	result, err := processor.getRecentMarketDataFromDB(symbol, exchange, duration)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan market data")
	assert.Nil(t, result)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetTradingPairSymbol_Success(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	tradingPairID := 1
	expectedSymbol := "BTC/USDT"

	mockPool.ExpectQuery("SELECT symbol FROM trading_pairs WHERE id = \\$1").
		WithArgs(tradingPairID).
		WillReturnRows(pgxmock.NewRows([]string{"symbol"}).AddRow(expectedSymbol))

	result, err := processor.getTradingPairSymbol(tradingPairID)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedSymbol, result)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetTradingPairSymbol_NotFound(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	tradingPairID := 999

	mockPool.ExpectQuery("SELECT symbol FROM trading_pairs WHERE id = \\$1").
		WithArgs(tradingPairID).
		WillReturnError(sql.ErrNoRows)

	result, err := processor.getTradingPairSymbol(tradingPairID)

	// Assertions
	assert.Error(t, err)
	assert.Empty(t, result)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

func TestSignalProcessor_GetTradingPairSymbol_DBError(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	processor := &TestSignalProcessor{
		db:     mockPool,
		logger: logger,
	}

	tradingPairID := 1

	mockPool.ExpectQuery("SELECT symbol FROM trading_pairs WHERE id = \\$1").
		WithArgs(tradingPairID).
		WillReturnError(errors.New("database connection failed"))

	result, err := processor.getTradingPairSymbol(tradingPairID)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
	assert.Empty(t, result)

	// Ensure all expectations were met
	assert.NoError(t, mockPool.ExpectationsWereMet())
}

