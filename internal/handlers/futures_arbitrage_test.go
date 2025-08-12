package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*FuturesArbitrageHandler, pgxmock.PgxPoolIface) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	// Type assertion to convert mock to *pgxpool.Pool
	pool, ok := mock.(*pgxpool.Pool)
	require.True(t, ok, "Failed to convert mock to *pgxpool.Pool")

	handler := NewFuturesArbitrageHandler(pool)
	return handler, mock
}

func TestCalculateArbitrage(t *testing.T) {
	handler, mockDB := setupTestHandler(t)
	defer mockDB.Close()

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    models.FuturesArbitrageCalculationInput
		expectedStatus int
		setupMock      func(pgxmock.PgxPoolIface)
	}{
		{
			name: "successful calculation",
			requestBody: models.FuturesArbitrageCalculationInput{
				Symbol:             "BTC/USDT",
				LongExchange:       "binance",
				ShortExchange:      "okx",
				LongFundingRate:    decimal.NewFromFloat(0.01),
				ShortFundingRate:   decimal.NewFromFloat(-0.005),
				LongMarkPrice:      decimal.NewFromFloat(50000),
				ShortMarkPrice:     decimal.NewFromFloat(50100),
				BaseAmount:         decimal.NewFromFloat(1.0),
				FundingInterval:    8,
				AvailableCapital:   decimal.NewFromFloat(10000),
				UserRiskTolerance:  "medium",
				MaxLeverageAllowed: decimal.NewFromFloat(10),
			},
			expectedStatus: http.StatusOK,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// Mock the insert query for saving the opportunity
				mock.ExpectExec("INSERT INTO futures_arbitrage_opportunities").WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
		},
		{
			name: "invalid request body",
			requestBody: models.FuturesArbitrageCalculationInput{
				// Missing required fields
				Symbol: "",
			},
			expectedStatus: http.StatusBadRequest,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// No database interaction expected for invalid input
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockDB)

			// Create request body
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/api/futures-arbitrage/calculate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Call handler
			handler.CalculateArbitrage(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.FuturesArbitrageOpportunity
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.requestBody.Symbol, response.Symbol)
				assert.Equal(t, tt.requestBody.LongExchange, response.LongExchange)
				assert.Equal(t, tt.requestBody.ShortExchange, response.ShortExchange)
			}

			// Ensure all expectations were met
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestGetOpportunities(t *testing.T) {
	handler, mockDB := setupTestHandler(t)
	defer mockDB.Close()

	gin.SetMode(gin.TestMode)

	// Mock data
	mockOpportunities := []models.FuturesArbitrageOpportunity{
		{
			ID:               1,
			Symbol:           "BTC/USDT",
			BaseCurrency:     "BTC",
			QuoteCurrency:    "USDT",
			LongExchange:     "binance",
			ShortExchange:    "okx",
			LongFundingRate:  decimal.NewFromFloat(0.01),
			ShortFundingRate: decimal.NewFromFloat(-0.005),
			NetFundingRate:   decimal.NewFromFloat(0.015),
			APY:              decimal.NewFromFloat(131.4),
			HourlyRate:       decimal.NewFromFloat(0.015),
			DailyRate:        decimal.NewFromFloat(0.36),
			IsActive:         true,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		setupMock      func(pgxmock.PgxPoolIface)
	}{
		{
			name:           "successful retrieval",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "symbol", "base_currency", "quote_currency",
					"long_exchange", "short_exchange", "long_funding_rate",
					"short_funding_rate", "net_funding_rate", "apy",
					"hourly_rate", "daily_rate", "is_active",
					"created_at", "updated_at",
				}).AddRow(
					mockOpportunities[0].ID,
					mockOpportunities[0].Symbol,
					mockOpportunities[0].BaseCurrency,
					mockOpportunities[0].QuoteCurrency,
					mockOpportunities[0].LongExchange,
					mockOpportunities[0].ShortExchange,
					mockOpportunities[0].LongFundingRate,
					mockOpportunities[0].ShortFundingRate,
					mockOpportunities[0].NetFundingRate,
					mockOpportunities[0].APY,
					mockOpportunities[0].HourlyRate,
					mockOpportunities[0].DailyRate,
					mockOpportunities[0].IsActive,
					mockOpportunities[0].CreatedAt,
					mockOpportunities[0].UpdatedAt,
				)

				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities").WillReturnRows(rows)
			},
		},
		{
			name:           "with symbol filter",
			queryParams:    "?symbol=BTC/USDT",
			expectedStatus: http.StatusOK,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "symbol", "base_currency", "quote_currency",
					"long_exchange", "short_exchange", "long_funding_rate",
					"short_funding_rate", "net_funding_rate", "apy",
					"hourly_rate", "daily_rate", "is_active",
					"created_at", "updated_at",
				})

				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities WHERE (.+) symbol").WillReturnRows(rows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockDB)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/futures-arbitrage/opportunities"+tt.queryParams, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Set query parameters if any
			if tt.queryParams != "" {
				c.Request.URL.RawQuery = tt.queryParams[1:] // Remove the '?' prefix
			}

			// Call handler
			handler.GetOpportunities(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response []models.FuturesArbitrageOpportunity
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				// Response should be an array (even if empty)
				assert.IsType(t, []models.FuturesArbitrageOpportunity{}, response)
			}

			// Ensure all expectations were met
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestGetOpportunityByID(t *testing.T) {
	handler, mockDB := setupTestHandler(t)
	defer mockDB.Close()

	gin.SetMode(gin.TestMode)

	mockOpportunity := models.FuturesArbitrageOpportunity{
		ID:               1,
		Symbol:           "BTC/USDT",
		BaseCurrency:     "BTC",
		QuoteCurrency:    "USDT",
		LongExchange:     "binance",
		ShortExchange:    "okx",
		LongFundingRate:  decimal.NewFromFloat(0.01),
		ShortFundingRate: decimal.NewFromFloat(-0.005),
		NetFundingRate:   decimal.NewFromFloat(0.015),
		APY:              decimal.NewFromFloat(131.4),
		HourlyRate:       decimal.NewFromFloat(0.015),
		DailyRate:        decimal.NewFromFloat(0.36),
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	tests := []struct {
		name           string
		id             string
		expectedStatus int
		setupMock      func(pgxmock.PgxPoolIface)
	}{
		{
			name:           "successful retrieval",
			id:             "1",
			expectedStatus: http.StatusOK,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "symbol", "base_currency", "quote_currency",
					"long_exchange", "short_exchange", "long_funding_rate",
					"short_funding_rate", "net_funding_rate", "apy",
					"hourly_rate", "daily_rate", "is_active",
					"created_at", "updated_at",
				}).AddRow(
					mockOpportunity.ID,
					mockOpportunity.Symbol,
					mockOpportunity.BaseCurrency,
					mockOpportunity.QuoteCurrency,
					mockOpportunity.LongExchange,
					mockOpportunity.ShortExchange,
					mockOpportunity.LongFundingRate,
					mockOpportunity.ShortFundingRate,
					mockOpportunity.NetFundingRate,
					mockOpportunity.APY,
					mockOpportunity.HourlyRate,
					mockOpportunity.DailyRate,
					mockOpportunity.IsActive,
					mockOpportunity.CreatedAt,
					mockOpportunity.UpdatedAt,
				)

				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities WHERE id").WithArgs(1).WillReturnRows(rows)
			},
		},
		{
			name:           "opportunity not found",
			id:             "999",
			expectedStatus: http.StatusNotFound,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities WHERE id").WithArgs(999).WillReturnError(pgx.ErrNoRows)
			},
		},
		{
			name:           "invalid ID",
			id:             "invalid",
			expectedStatus: http.StatusBadRequest,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				// No database interaction expected for invalid ID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockDB)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodGet, "/api/futures-arbitrage/opportunities/"+tt.id, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Create Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.id}}

			// Call handler
			handler.GetOpportunityByID(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.FuturesArbitrageOpportunity
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, mockOpportunity.ID, response.ID)
				assert.Equal(t, mockOpportunity.Symbol, response.Symbol)
			}

			// Ensure all expectations were met
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}
