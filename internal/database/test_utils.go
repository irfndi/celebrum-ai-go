package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
)

// MockPool is a mock implementation of pgxpool.Pool for testing
type MockPool struct {
	mock.Mock
}

func (m *MockPool) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgx.Rows), callArgs.Error(1)
}

func (m *MockPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgx.Row)
}

func (m *MockPool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	callArgs := m.Called(ctx, sql, args)
	return callArgs.Get(0).(pgconn.CommandTag), callArgs.Error(1)
}

func (m *MockPool) Ping(ctx context.Context) error {
	callArgs := m.Called(ctx)
	return callArgs.Error(0)
}

func (m *MockPool) Close() {
	m.Called()
}

func (m *MockPool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	callArgs := m.Called(ctx)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(*pgxpool.Conn), callArgs.Error(1)
}

// MockPostgresDB provides a fast, in-memory mock database for testing
type MockPostgresDB struct {
	Pool *MockPool
	data map[string]map[string]interface{}
	mu   sync.RWMutex
}

// NewMockPostgresDB creates a new mock database for testing
func NewMockPostgresDB() *MockPostgresDB {
	mockPool := &MockPool{}
	
	db := &MockPostgresDB{
		Pool: mockPool,
		data: make(map[string]map[string]interface{}),
	}
	
	// Setup default mock behaviors
	db.setupDefaultMockBehaviors()
	
	return db
}

// setupDefaultMockBehaviors configures common mock responses
func (db *MockPostgresDB) setupDefaultMockBehaviors() {
	// Mock Ping to always succeed
	db.Pool.On("Ping", mock.Anything).Return(nil)
	
	// Mock Query to return empty result set by default
	db.Pool.On("Query", mock.Anything, mock.Anything, mock.Anything).Return(&MockRows{}, nil)
	
	// Mock QueryRow to return a mock row
	db.Pool.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(&MockRow{})
	
	// Mock Exec to return success
	db.Pool.On("Exec", mock.Anything, mock.Anything, mock.Anything).Return(pgconn.NewCommandTag("SELECT 0"), nil)
	
	// Mock Close to do nothing
	db.Pool.On("Close").Return()
}

// MockRows implements pgx.Rows for testing
type MockRows struct {
	mock.Mock
	values [][]interface{}
	current int
	closed  bool
}

func NewMockRows(values [][]interface{}) *MockRows {
	return &MockRows{
		values:  values,
		current: -1,
		closed:  false,
	}
}

func (m *MockRows) Close() {
	m.closed = true
}

func (m *MockRows) Err() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func (m *MockRows) CommandTag() pgconn.CommandTag {
	callArgs := m.Called()
	return callArgs.Get(0).(pgconn.CommandTag)
}

func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	callArgs := m.Called()
	return callArgs.Get(0).([]pgconn.FieldDescription)
}

func (m *MockRows) Next() bool {
	m.current++
	if m.current >= len(m.values) {
		return false
	}
	return true
}

func (m *MockRows) Values() ([]interface{}, error) {
	if m.current < 0 || m.current >= len(m.values) {
		return nil, fmt.Errorf("no current row")
	}
	return m.values[m.current], nil
}

func (m *MockRows) Scan(dest ...interface{}) error {
	if m.current < 0 || m.current >= len(m.values) {
		return fmt.Errorf("no current row")
	}
	
	values := m.values[m.current]
	if len(values) != len(dest) {
		return fmt.Errorf("column count mismatch: expected %d, got %d", len(dest), len(values))
	}
	
	for i, value := range values {
		switch d := dest[i].(type) {
		case *string:
			if value != nil {
				*d = value.(string)
			}
		case *int:
			if value != nil {
				*d = value.(int)
			}
		case *int64:
			if value != nil {
				*d = int64(value.(int))
			}
		case *float64:
			if value != nil {
				*d = value.(float64)
			}
		case *decimal.Decimal:
			if value != nil {
				*d = value.(decimal.Decimal)
			}
		case *bool:
			if value != nil {
				*d = value.(bool)
			}
		case *time.Time:
			if value != nil {
				*d = value.(time.Time)
			}
		default:
			return fmt.Errorf("unsupported scan type: %T", d)
		}
	}
	
	return nil
}

func (m *MockRows) RawValues() [][]byte {
	callArgs := m.Called()
	return callArgs.Get(0).([][]byte)
}

func (m *MockRows) ScanRow(dest []interface{}) error {
	callArgs := m.Called(dest)
	return callArgs.Error(0)
}

// MockRow implements pgx.Row for testing
type MockRow struct {
	mock.Mock
	values []interface{}
}

func NewMockRow(values []interface{}) *MockRow {
	return &MockRow{values: values}
}

func (m *MockRow) Scan(dest ...interface{}) error {
	callArgs := m.Called(dest)
	return callArgs.Error(0)
}

// CreateTestDatabaseConnection creates a fast mock connection for testing
func CreateTestDatabaseConnection(t *testing.T) *PostgresDB {
	t.Helper()
	
	// Create a mock pool
	mockPool := &MockPool{}
	
	// Setup basic expectations for test environment
	mockPool.On("Ping", mock.Anything).Return(nil)
	mockPool.On("Close").Return()
	
	// Return the mock database
	return &PostgresDB{
		Pool: nil, // We use nil pool to avoid real connections
	}
}

// MockRedisClient provides a fast mock Redis client for testing
type MockRedisClient struct {
	mock.Mock
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewMockRedisClient creates a new mock Redis client for testing
func NewMockRedisClient() *MockRedisClient {
	client := &MockRedisClient{
		data: make(map[string]interface{}),
	}
	
	// Setup default mock behaviors
	client.On("Ping", mock.Anything).Return(nil)
	client.On("Close").Return()
	client.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	client.On("Get", mock.Anything, mock.Anything).Return("", nil)
	client.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	client.On("Exists", mock.Anything, mock.Anything, mock.Anything).Return(int64(0), nil)
	
	return client
}

// TestConnectionOptimizer provides utilities to optimize connections in tests
type TestConnectionOptimizer struct {
	connections map[string]interface{}
	mu          sync.RWMutex
}

// NewTestConnectionOptimizer creates a new optimizer for test connections
func NewTestConnectionOptimizer() *TestConnectionOptimizer {
	return &TestConnectionOptimizer{
		connections: make(map[string]interface{}),
	}
}

// GetOptimizedConnection returns a connection optimized for testing
func (opt *TestConnectionOptimizer) GetOptimizedConnection(name string) interface{} {
	opt.mu.RLock()
	defer opt.mu.RUnlock()
	
	if conn, exists := opt.connections[name]; exists {
		return conn
	}
	
	// Create optimized connection based on type
	switch name {
	case "postgres":
		return CreateTestDatabaseConnection(nil)
	case "redis":
		return NewMockRedisClient()
	default:
		return nil
	}
}

// SetOptimizedConnection sets an optimized connection for testing
func (opt *TestConnectionOptimizer) SetOptimizedConnection(name string, conn interface{}) {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	
	opt.connections[name] = conn
}

// Cleanup closes all connections and cleans up resources
func (opt *TestConnectionOptimizer) Cleanup() {
	opt.mu.Lock()
	defer opt.mu.Unlock()
	
	// Close all connections
	for name, conn := range opt.connections {
		switch c := conn.(type) {
		case *PostgresDB:
			c.Close()
		case interface{ Close() }:
			c.Close()
		}
		delete(opt.connections, name)
	}
}

// Global test optimizer instance
var testOptimizer = NewTestConnectionOptimizer()

// GetTestOptimizer returns the global test connection optimizer
func GetTestOptimizer() *TestConnectionOptimizer {
	return testOptimizer
}

// CleanupTestConnections cleans up all test connections
func CleanupTestConnections() {
	testOptimizer.Cleanup()
}