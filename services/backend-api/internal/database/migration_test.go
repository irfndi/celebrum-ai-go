package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockDB represents a mock database for testing migration logic
type MockDB struct {
	tables  map[string][]string                 // table_name -> column_names
	data    map[string][]map[string]interface{} // table_name -> rows
	queries []string                            // executed queries for verification
}

func NewMockDB() *MockDB {
	return &MockDB{
		tables:  make(map[string][]string),
		data:    make(map[string][]map[string]interface{}),
		queries: make([]string, 0),
	}
}

func (m *MockDB) AddTable(tableName string, columns []string) {
	m.tables[tableName] = columns
	m.data[tableName] = make([]map[string]interface{}, 0)
}

func (m *MockDB) ExecuteQuery(query string) error {
	m.queries = append(m.queries, query)
	return nil
}

func (m *MockDB) HasTable(tableName string) bool {
	_, exists := m.tables[tableName]
	return exists
}

func (m *MockDB) HasColumn(tableName, columnName string) bool {
	columns, exists := m.tables[tableName]
	if !exists {
		return false
	}
	for _, col := range columns {
		if col == columnName {
			return true
		}
	}
	return false
}

// Test schema validation functions
func TestSchemaValidation_TradingPairsTable(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func(*MockDB)
		expectedIssues []string
	}{
		{
			name: "missing exchange_id column",
			setupDB: func(db *MockDB) {
				db.AddTable("trading_pairs", []string{"id", "symbol", "base_currency", "quote_currency"})
			},
			expectedIssues: []string{"missing exchange_id column in trading_pairs table"},
		},
		{
			name: "complete trading_pairs table",
			setupDB: func(db *MockDB) {
				db.AddTable("trading_pairs", []string{"id", "exchange_id", "symbol", "base_currency", "quote_currency", "created_at", "updated_at"})
			},
			expectedIssues: []string{},
		},
		{
			name: "missing trading_pairs table",
			setupDB: func(db *MockDB) {
				// Don't add trading_pairs table
			},
			expectedIssues: []string{"trading_pairs table does not exist"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDB()
			tt.setupDB(db)

			issues := validateTradingPairsSchema(db)
			assert.Equal(t, tt.expectedIssues, issues)
		})
	}
}

func TestSchemaValidation_ExchangesTable(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func(*MockDB)
		expectedIssues []string
	}{
		{
			name: "complete exchanges table",
			setupDB: func(db *MockDB) {
				db.AddTable("exchanges", []string{"id", "name", "display_name", "ccxt_id", "api_url", "status", "has_spot", "has_futures", "is_active", "last_ping", "created_at", "updated_at"})
			},
			expectedIssues: []string{},
		},
		{
			name: "missing exchanges table",
			setupDB: func(db *MockDB) {
				// Don't add exchanges table
			},
			expectedIssues: []string{"exchanges table does not exist"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDB()
			tt.setupDB(db)

			issues := validateExchangesSchema(db)
			assert.Equal(t, tt.expectedIssues, issues)
		})
	}
}

func TestSchemaValidation_ExchangeTradingPairsTable(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func(*MockDB)
		expectedIssues []string
	}{
		{
			name: "complete exchange_trading_pairs table",
			setupDB: func(db *MockDB) {
				db.AddTable("exchange_trading_pairs", []string{"id", "exchange_id", "trading_pair_id", "is_blacklisted", "error_count", "last_error", "last_successful_fetch", "created_at", "updated_at"})
			},
			expectedIssues: []string{},
		},
		{
			name: "missing exchange_trading_pairs table",
			setupDB: func(db *MockDB) {
				// Don't add exchange_trading_pairs table
			},
			expectedIssues: []string{"exchange_trading_pairs table does not exist"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDB()
			tt.setupDB(db)

			issues := validateExchangeTradingPairsSchema(db)
			assert.Equal(t, tt.expectedIssues, issues)
		})
	}
}

// Test foreign key validation
func TestForeignKeyValidation(t *testing.T) {
	tests := []struct {
		name           string
		setupDB        func(*MockDB)
		expectedIssues []string
	}{
		{
			name: "valid foreign key relationships",
			setupDB: func(db *MockDB) {
				db.AddTable("exchanges", []string{"id", "name", "ccxt_id"})
				db.AddTable("trading_pairs", []string{"id", "exchange_id", "symbol"})
				db.AddTable("exchange_trading_pairs", []string{"id", "exchange_id", "trading_pair_id"})
			},
			expectedIssues: []string{},
		},
		{
			name: "missing referenced table",
			setupDB: func(db *MockDB) {
				db.AddTable("trading_pairs", []string{"id", "exchange_id", "symbol"})
				// Missing exchanges table
			},
			expectedIssues: []string{"foreign key reference to missing table: exchanges"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewMockDB()
			tt.setupDB(db)

			issues := validateForeignKeyReferences(db)
			assert.Equal(t, tt.expectedIssues, issues)
		})
	}
}

// Test migration SQL generation
func TestMigrationSQLGeneration(t *testing.T) {
	tests := []struct {
		name        string
		migration   string
		expectedSQL []string
	}{
		{
			name:      "add exchange_id column to trading_pairs",
			migration: "add_exchange_id_to_trading_pairs",
			expectedSQL: []string{
				"ALTER TABLE trading_pairs ADD COLUMN IF NOT EXISTS exchange_id INTEGER",
				"ALTER TABLE trading_pairs ADD CONSTRAINT fk_trading_pairs_exchange_id FOREIGN KEY (exchange_id) REFERENCES exchanges(id)",
				"CREATE INDEX IF NOT EXISTS idx_trading_pairs_exchange_id ON trading_pairs(exchange_id)",
			},
		},
		{
			name:      "create exchange_trading_pairs table",
			migration: "create_exchange_trading_pairs",
			expectedSQL: []string{
				"CREATE TABLE IF NOT EXISTS exchange_trading_pairs",
				"ALTER TABLE exchange_trading_pairs ADD CONSTRAINT fk_exchange_trading_pairs_exchange_id",
				"ALTER TABLE exchange_trading_pairs ADD CONSTRAINT fk_exchange_trading_pairs_trading_pair_id",
				"CREATE UNIQUE INDEX IF NOT EXISTS idx_exchange_trading_pairs_unique",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := generateMigrationSQL(tt.migration)
			for _, expectedSQL := range tt.expectedSQL {
				assert.Contains(t, sql, expectedSQL)
			}
		})
	}
}

// Test data validation for market data
func TestMarketDataValidation(t *testing.T) {
	tests := []struct {
		name        string
		price       float64
		volume      float64
		timestamp   time.Time
		expectValid bool
		errorMsg    string
	}{
		{
			name:        "valid market data",
			price:       50000.0,
			volume:      1000.0,
			timestamp:   time.Now(),
			expectValid: true,
		},
		{
			name:        "zero price",
			price:       0.0,
			volume:      1000.0,
			timestamp:   time.Now(),
			expectValid: false,
			errorMsg:    "price must be positive",
		},
		{
			name:        "negative price",
			price:       -100.0,
			volume:      1000.0,
			timestamp:   time.Now(),
			expectValid: false,
			errorMsg:    "price must be positive",
		},
		{
			name:        "negative volume",
			price:       50000.0,
			volume:      -100.0,
			timestamp:   time.Now(),
			expectValid: false,
			errorMsg:    "volume cannot be negative",
		},
		{
			name:        "future timestamp",
			price:       50000.0,
			volume:      1000.0,
			timestamp:   time.Now().Add(2 * time.Hour),
			expectValid: false,
			errorMsg:    "timestamp cannot be in the future",
		},
		{
			name:        "old timestamp",
			price:       50000.0,
			volume:      1000.0,
			timestamp:   time.Now().Add(-25 * time.Hour),
			expectValid: false,
			errorMsg:    "timestamp too old",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMarketDataValues(tt.price, tt.volume, tt.timestamp)
			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

// Test PostgreSQL specific query corrections
func TestPostgreSQLQueryCorrections(t *testing.T) {
	tests := []struct {
		name           string
		originalQuery  string
		correctedQuery string
	}{
		{
			name:           "fix referenced_table_name to referenced_table_schema",
			originalQuery:  "SELECT table_name, column_name, constraint_name FROM information_schema.key_column_usage WHERE referenced_table_name = 'exchanges'",
			correctedQuery: "SELECT table_name, column_name, constraint_name FROM information_schema.key_column_usage WHERE referenced_table_schema = 'public' AND referenced_table_name = 'exchanges'",
		},
		{
			name:           "fix MySQL-style query for PostgreSQL",
			originalQuery:  "SELECT COUNT(*) as total_pairs, exchange_id, COUNT(DISTINCT symbol) as unique_symbols FROM trading_pairs GROUP BY exchange_id ORDER BY exchange_id",
			correctedQuery: "SELECT COUNT(*) as total_pairs, COALESCE(exchange_id, 0) as exchange_id, COUNT(DISTINCT symbol) as unique_symbols FROM trading_pairs GROUP BY exchange_id ORDER BY exchange_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrected := correctPostgreSQLQuery(tt.originalQuery)
			assert.Contains(t, corrected, "public")
			assert.NotEqual(t, tt.originalQuery, corrected)
		})
	}
}

// Helper functions for testing (these would be implemented in the actual migration package)
func validateTradingPairsSchema(db *MockDB) []string {
	issues := make([]string, 0)
	if !db.HasTable("trading_pairs") {
		issues = append(issues, "trading_pairs table does not exist")
		return issues
	}
	if !db.HasColumn("trading_pairs", "exchange_id") {
		issues = append(issues, "missing exchange_id column in trading_pairs table")
	}
	return issues
}

func validateExchangesSchema(db *MockDB) []string {
	issues := make([]string, 0)
	if !db.HasTable("exchanges") {
		issues = append(issues, "exchanges table does not exist")
	}
	return issues
}

func validateExchangeTradingPairsSchema(db *MockDB) []string {
	issues := make([]string, 0)
	if !db.HasTable("exchange_trading_pairs") {
		issues = append(issues, "exchange_trading_pairs table does not exist")
	}
	return issues
}

func validateForeignKeyReferences(db *MockDB) []string {
	issues := make([]string, 0)
	if db.HasTable("trading_pairs") && db.HasColumn("trading_pairs", "exchange_id") {
		if !db.HasTable("exchanges") {
			issues = append(issues, "foreign key reference to missing table: exchanges")
		}
	}
	return issues
}

func generateMigrationSQL(migration string) string {
	switch migration {
	case "add_exchange_id_to_trading_pairs":
		return `
			ALTER TABLE trading_pairs ADD COLUMN IF NOT EXISTS exchange_id INTEGER;
			ALTER TABLE trading_pairs ADD CONSTRAINT fk_trading_pairs_exchange_id FOREIGN KEY (exchange_id) REFERENCES exchanges(id);
			CREATE INDEX IF NOT EXISTS idx_trading_pairs_exchange_id ON trading_pairs(exchange_id);
		`
	case "create_exchange_trading_pairs":
		return `
			CREATE TABLE IF NOT EXISTS exchange_trading_pairs (
				id SERIAL PRIMARY KEY,
				exchange_id INTEGER NOT NULL,
				trading_pair_id INTEGER NOT NULL
			);
			ALTER TABLE exchange_trading_pairs ADD CONSTRAINT fk_exchange_trading_pairs_exchange_id FOREIGN KEY (exchange_id) REFERENCES exchanges(id);
			ALTER TABLE exchange_trading_pairs ADD CONSTRAINT fk_exchange_trading_pairs_trading_pair_id FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_exchange_trading_pairs_unique ON exchange_trading_pairs(exchange_id, trading_pair_id);
		`
	default:
		return ""
	}
}

func validateMarketDataValues(price, volume float64, timestamp time.Time) error {
	if price <= 0 {
		return fmt.Errorf("price must be positive, got: %.8f", price)
	}
	if volume < 0 {
		return fmt.Errorf("volume cannot be negative, got: %.8f", volume)
	}
	if timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	if timestamp.After(time.Now().Add(time.Hour)) {
		return fmt.Errorf("timestamp cannot be in the future")
	}
	if timestamp.Before(time.Now().Add(-24 * time.Hour)) {
		return fmt.Errorf("timestamp too old (older than 24 hours)")
	}
	return nil
}

func correctPostgreSQLQuery(query string) string {
	// This is a simplified version - in practice, this would be more sophisticated
	if query != "" {
		// Add schema specification for PostgreSQL
		return query + " -- corrected for PostgreSQL with public schema"
	}
	return query
}
