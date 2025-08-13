package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/irfndi/celebrum-ai-go/internal/models"
)

func setupTestHandler(t *testing.T) (*FuturesArbitrageHandler, pgxmock.PgxPoolIface) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)

	// Use the NewFuturesArbitrageHandlerWithQuerier constructor
	handler := NewFuturesArbitrageHandlerWithQuerier(mock)
	return handler, mock
}

func TestCalculateFuturesArbitrage(t *testing.T) {
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
				// No database operations expected since storeFuturesOpportunity is a stub
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
			handler.CalculateFuturesArbitrage(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response struct {
					Opportunity *models.FuturesArbitrageOpportunity `json:"opportunity"`
					RiskMetrics *models.FuturesArbitrageRiskMetrics `json:"risk_metrics"`
					Timestamp   time.Time                           `json:"timestamp"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.requestBody.Symbol, response.Opportunity.Symbol)
				assert.Equal(t, tt.requestBody.LongExchange, response.Opportunity.LongExchange)
				assert.Equal(t, tt.requestBody.ShortExchange, response.Opportunity.ShortExchange)
			}

			// Ensure all expectations were met
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestGetFuturesArbitrageOpportunities(t *testing.T) {
	handler, mockDB := setupTestHandler(t)
	defer mockDB.Close()

	gin.SetMode(gin.TestMode)

	// Mock data
	mockOpportunities := []models.FuturesArbitrageOpportunity{
		{
			ID:               "1",
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
			DetectedAt:       time.Now(),
			ExpiresAt:        time.Now().Add(time.Hour * 8),
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
					"long_exchange", "short_exchange", "long_exchange_id", "short_exchange_id",
					"long_funding_rate", "short_funding_rate", "net_funding_rate", "funding_interval",
					"long_mark_price", "short_mark_price", "price_difference", "price_difference_percentage",
					"hourly_rate", "daily_rate", "apy",
					"estimated_profit_8h", "estimated_profit_daily", "estimated_profit_weekly", "estimated_profit_monthly",
					"risk_score", "volatility_score", "liquidity_score",
					"recommended_position_size", "max_leverage", "recommended_leverage", "stop_loss_percentage",
					"min_position_size", "max_position_size", "optimal_position_size",
					"detected_at", "expires_at", "next_funding_time", "time_to_next_funding", "is_active",
					"market_trend", "volume_24h", "open_interest",
				}).AddRow(
					mockOpportunities[0].ID,
					mockOpportunities[0].Symbol,
					mockOpportunities[0].BaseCurrency,
					mockOpportunities[0].QuoteCurrency,
					mockOpportunities[0].LongExchange,
					mockOpportunities[0].ShortExchange,
					1, 2, // Mock exchange IDs
					mockOpportunities[0].LongFundingRate,
					mockOpportunities[0].ShortFundingRate,
					mockOpportunities[0].NetFundingRate,
					8,                                                        // funding_interval
					decimal.NewFromFloat(50000), decimal.NewFromFloat(50100), // mark prices
					decimal.NewFromFloat(100), decimal.NewFromFloat(0.2), // price difference
					mockOpportunities[0].HourlyRate,
					mockOpportunities[0].DailyRate,
					mockOpportunities[0].APY,
					decimal.NewFromFloat(10), decimal.NewFromFloat(30), decimal.NewFromFloat(210), decimal.NewFromFloat(900), // estimated profits
					decimal.NewFromFloat(0.3), decimal.NewFromFloat(0.2), decimal.NewFromFloat(0.8), // risk scores
					decimal.NewFromFloat(1000), decimal.NewFromFloat(10), decimal.NewFromFloat(5), decimal.NewFromFloat(2), // position sizing
					decimal.NewFromFloat(100), decimal.NewFromFloat(5000), decimal.NewFromFloat(1000), // position limits
					mockOpportunities[0].DetectedAt,
					mockOpportunities[0].ExpiresAt,
					time.Now().Add(time.Hour), time.Duration(3600000000000), // next funding time
					mockOpportunities[0].IsActive,
					"bullish", decimal.NewFromFloat(1000000), decimal.NewFromFloat(50000000), // market data
				)

				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW\\(\\) ORDER BY apy DESC, risk_score ASC LIMIT \\$1 OFFSET \\$2").WithArgs(50, 0).WillReturnRows(rows)
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW\\(\\)").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
			},
		},
		{
			name:           "with symbol filter",
			queryParams:    "?symbols=BTC/USDT",
			expectedStatus: http.StatusOK,
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "symbol", "base_currency", "quote_currency",
					"long_exchange", "short_exchange", "long_exchange_id", "short_exchange_id",
					"long_funding_rate", "short_funding_rate", "net_funding_rate", "funding_interval",
					"long_mark_price", "short_mark_price", "price_difference", "price_difference_percentage",
					"hourly_rate", "daily_rate", "apy",
					"estimated_profit_8h", "estimated_profit_daily", "estimated_profit_weekly", "estimated_profit_monthly",
					"risk_score", "volatility_score", "liquidity_score",
					"recommended_position_size", "max_leverage", "recommended_leverage", "stop_loss_percentage",
					"min_position_size", "max_position_size", "optimal_position_size",
					"detected_at", "expires_at", "next_funding_time", "time_to_next_funding", "is_active",
					"market_trend", "volume_24h", "open_interest",
				})

				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW\\(\\) AND symbol = ANY\\(\\$1\\) ORDER BY apy DESC, risk_score ASC LIMIT \\$2 OFFSET \\$3").WithArgs([]string{"BTC/USDT"}, 50, 0).WillReturnRows(rows)
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW\\(\\)").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
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
			handler.GetFuturesArbitrageOpportunities(c)

			// Check response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.FuturesArbitrageResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				// Response should be a FuturesArbitrageResponse
				assert.IsType(t, models.FuturesArbitrageResponse{}, response)
			}

			// Ensure all expectations were met
			assert.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestGetFuturesMarketSummary(t *testing.T) {
	tests := []struct {
		name       string
		setupMock  func(mock pgxmock.PgxPoolIface)
		expectCode int
	}{
		{
			name: "successful market summary",
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "symbol", "base_currency", "quote_currency",
					"long_exchange", "short_exchange", "long_exchange_id", "short_exchange_id",
					"long_funding_rate", "short_funding_rate", "net_funding_rate", "funding_interval",
					"long_mark_price", "short_mark_price", "price_difference", "price_difference_percentage",
					"hourly_rate", "daily_rate", "apy",
					"estimated_profit_8h", "estimated_profit_daily", "estimated_profit_weekly", "estimated_profit_monthly",
					"risk_score", "volatility_score", "liquidity_score",
					"recommended_position_size", "max_leverage", "recommended_leverage", "stop_loss_percentage",
					"min_position_size", "max_position_size", "optimal_position_size",
					"detected_at", "expires_at", "next_funding_time", "time_to_next_funding", "is_active",
					"market_trend", "volume_24h", "open_interest",
				}).AddRow(
					"test-id", "BTC/USDT", "BTC", "USDT",
					"binance", "okx", 1, 2,
					decimal.NewFromFloat(0.01), decimal.NewFromFloat(-0.005), decimal.NewFromFloat(0.015), 8,
					decimal.NewFromFloat(50000), decimal.NewFromFloat(49950), decimal.NewFromFloat(50), decimal.NewFromFloat(0.1),
					decimal.NewFromFloat(0.1875), decimal.NewFromFloat(4.5), decimal.NewFromFloat(54.75),
					decimal.NewFromFloat(100), decimal.NewFromFloat(1000), decimal.NewFromFloat(10000), decimal.NewFromFloat(100000),
					decimal.NewFromFloat(25), decimal.NewFromFloat(30), decimal.NewFromFloat(80),
					decimal.NewFromFloat(1000), decimal.NewFromFloat(10), decimal.NewFromFloat(5), decimal.NewFromFloat(2),
					decimal.NewFromFloat(100), decimal.NewFromFloat(10000), decimal.NewFromFloat(5000),
					time.Now(), time.Now().Add(time.Hour), time.Now().Add(time.Hour), time.Hour, true,
					"bullish", decimal.NewFromFloat(1000000), decimal.NewFromFloat(50000000),
				)
				mock.ExpectQuery("SELECT (.+) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW\\(\\) ORDER BY apy DESC, risk_score ASC LIMIT \\$1 OFFSET \\$2").WithArgs(1000, 0).WillReturnRows(rows)
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM futures_arbitrage_opportunities WHERE is_active = true AND expires_at > NOW\\(\\)").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			assert.NoError(t, err)
			defer mock.Close()

			handler := NewFuturesArbitrageHandlerWithQuerier(mock)

			tt.setupMock(mock)

			req, _ := http.NewRequest("GET", "/api/futures-arbitrage/market-summary", nil)
			w := httptest.NewRecorder()
			router := gin.New()
			router.GET("/api/futures-arbitrage/market-summary", handler.GetFuturesMarketSummary)
			router.ServeHTTP(w, req)

			if w.Code != tt.expectCode {
				t.Logf("Response body: %s", w.Body.String())
			}
			assert.Equal(t, tt.expectCode, w.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
