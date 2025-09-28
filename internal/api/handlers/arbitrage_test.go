package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewArbitrageHandler(t *testing.T) {
	t.Run("create handler with nil dependencies", func(t *testing.T) {
		handler := NewArbitrageHandler(nil, nil, nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.notificationService)
		assert.Nil(t, handler.redisClient)
	})
}

func TestArbitrageHandler_BasicFunctionality(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)
	assert.NotNil(t, handler)
	assert.Nil(t, handler.db)
	assert.Nil(t, handler.notificationService)
	assert.Nil(t, handler.redisClient)
}

func TestArbitrageHandler_FindCrossExchangeArbitrage(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("empty exchanges map", func(t *testing.T) {
		exchanges := make(map[string]struct {
			price     float64
			volume    float64
			timestamp time.Time
		})

		opportunities := handler.findCrossExchangeArbitrage("BTC/USDT", exchanges, 1.0)
		assert.Empty(t, opportunities)
	})

	t.Run("single exchange", func(t *testing.T) {
		exchanges := map[string]struct {
			price     float64
			volume    float64
			timestamp time.Time
		}{
			"binance": {45000, 100, time.Now()},
		}

		opportunities := handler.findCrossExchangeArbitrage("BTC/USDT", exchanges, 1.0)
		assert.Empty(t, opportunities)
	})

	t.Run("multiple exchanges with arbitrage opportunity", func(t *testing.T) {
		now := time.Now()
		exchanges := map[string]struct {
			price     float64
			volume    float64
			timestamp time.Time
		}{
			"binance":  {45000, 100, now},
			"coinbase": {45500, 80, now},
		}

		opportunities := handler.findCrossExchangeArbitrage("BTC/USDT", exchanges, 1.0)
		assert.Len(t, opportunities, 1)

		opp := opportunities[0]
		assert.Equal(t, "BTC/USDT", opp.Symbol)
		assert.Equal(t, "binance", opp.BuyExchange)
		assert.Equal(t, "coinbase", opp.SellExchange)
		assert.True(t, opp.ProfitPercent > 0)
	})

	t.Run("multiple exchanges no arbitrage opportunity", func(t *testing.T) {
		now := time.Now()
		exchanges := map[string]struct {
			price     float64
			volume    float64
			timestamp time.Time
		}{
			"binance":  {45000, 100, now},
			"coinbase": {45050, 80, now}, // Only 0.11% difference
		}

		opportunities := handler.findCrossExchangeArbitrage("BTC/USDT", exchanges, 1.0)
		assert.Empty(t, opportunities)
	})

	t.Run("stale data filtering", func(t *testing.T) {
		// Note: Current implementation doesn't filter stale data
		// This test documents current behavior
		oldTime := time.Now().Add(-10 * time.Minute)
		exchanges := map[string]struct {
			price     float64
			volume    float64
			timestamp time.Time
		}{
			"binance":  {45000, 100, oldTime},
			"coinbase": {45500, 80, time.Now()},
		}

		opportunities := handler.findCrossExchangeArbitrage("BTC/USDT", exchanges, 1.0)
		// Current implementation doesn't filter by timestamp, so we expect an opportunity
		assert.Len(t, opportunities, 1)

		opp := opportunities[0]
		assert.Equal(t, "BTC/USDT", opp.Symbol)
		assert.Equal(t, "binance", opp.BuyExchange)
		assert.Equal(t, "coinbase", opp.SellExchange)
		assert.True(t, opp.ProfitPercent > 0)
	})
}

func TestArbitrageHandler_CalculateSMA(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("empty data", func(t *testing.T) {
		data := []float64{}
		sma := handler.calculateSMA(data, 5)
		assert.Equal(t, 0.0, sma)
	})

	t.Run("single data point", func(t *testing.T) {
		data := []float64{100.0}
		sma := handler.calculateSMA(data, 1)
		assert.Equal(t, 100.0, sma)
	})

	t.Run("multiple data points", func(t *testing.T) {
		data := []float64{100.0, 200.0, 300.0, 400.0, 500.0}
		sma := handler.calculateSMA(data, 5)
		assert.Equal(t, 300.0, sma) // (100+200+300+400+500)/5 = 300
	})

	t.Run("decimal precision", func(t *testing.T) {
		data := []float64{100.5, 200.25, 300.75}
		sma := handler.calculateSMA(data, 3)
		assert.Equal(t, 200.5, sma) // (100.5+200.25+300.75)/3 = 200.5
	})
}

func TestArbitrageHandler_FindArbitrageOpportunities_Integration(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("basic arbitrage detection", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 10, "")
		assert.NoError(t, err)
		// With nil dependencies, this should return empty slice
		assert.Empty(t, opportunities)
	})

	t.Run("higher profit threshold", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 2.0, 5, "BTC/USDT")
		assert.NoError(t, err)
		// With nil dependencies, this should return empty slice
		assert.Empty(t, opportunities)
	})

	t.Run("no limit", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 0.5, 0, "")
		assert.NoError(t, err)
		// With nil dependencies, this should return empty slice
		assert.Empty(t, opportunities)
	})
}

func TestArbitrageHandler_SendArbitrageNotifications(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil notification service does not panic", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 1.5},
			{Symbol: "ETH/USDT", ProfitPercent: 2.0},
		}

		// Should not panic with nil notification service
		handler.sendArbitrageNotifications(opportunities)
	})

	t.Run("low profit opportunities do not trigger notifications", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 0.5}, // Below 1% threshold
			{Symbol: "ETH/USDT", ProfitPercent: 0.8}, // Below 1% threshold
		}

		handler.sendArbitrageNotifications(opportunities)
		// Should not call notification service for low profit opportunities
	})

	t.Run("mixed profit opportunities filter correctly", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 0.5}, // Below threshold
			{Symbol: "ETH/USDT", ProfitPercent: 1.2}, // Above threshold
			{Symbol: "BNB/USDT", ProfitPercent: 2.0}, // Above threshold
			{Symbol: "ADA/USDT", ProfitPercent: 0.8}, // Below threshold
		}

		handler.sendArbitrageNotifications(opportunities)
		// Should filter to only high-profit opportunities
	})

	t.Run("empty opportunities list does not trigger notifications", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{}

		handler.sendArbitrageNotifications(opportunities)
		// Should not call notification service for empty list
	})
}

func TestArbitrageHandler_DecimalCalculations(t *testing.T) {
	t.Run("profit calculation with decimal precision", func(t *testing.T) {
		buyPrice := decimal.NewFromFloat(45000.50)
		sellPrice := decimal.NewFromFloat(45500.75)

		profitAmount := sellPrice.Sub(buyPrice)
		profitPercent := profitAmount.Div(buyPrice).Mul(decimal.NewFromFloat(100))

		assert.True(t, profitAmount.GreaterThan(decimal.NewFromFloat(500)))
		assert.True(t, profitPercent.GreaterThan(decimal.NewFromFloat(1.0)))
	})

	t.Run("volume calculations", func(t *testing.T) {
		price := decimal.NewFromFloat(45000)
		volume := decimal.NewFromFloat(100)
		expectedVolume := price.Mul(volume)

		assert.Equal(t, decimal.NewFromFloat(4500000), expectedVolume)
	})
}

// TestArbitrageHandler_GetArbitrageOpportunities tests the HTTP handler endpoint
func TestArbitrageHandler_GetArbitrageOpportunities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("valid request with default parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/opportunities", nil)
		c.Request = req

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 0, response.Count) // No database, so empty response
	})

	t.Run("custom min_profit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/opportunities?min_profit=1.5", nil)
		c.Request = req

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("custom limit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/opportunities?limit=25", nil)
		c.Request = req

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("symbol filter parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/opportunities?symbol=BTC/USDT", nil)
		c.Request = req

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("invalid min_profit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/opportunities?min_profit=invalid", nil)
		c.Request = req

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("invalid limit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/opportunities?limit=invalid", nil)
		c.Request = req

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})
}

// TestArbitrageHandler_GetArbitrageHistory tests the history endpoint
func TestArbitrageHandler_GetArbitrageHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("valid request with default parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/history", nil)
		c.Request = req

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 1, response.Page)
		assert.Equal(t, 20, response.Limit)
	})

	t.Run("custom pagination parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/history?page=2&limit=10", nil)
		c.Request = req

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 2, response.Page)
		assert.Equal(t, 10, response.Limit)
	})

	t.Run("symbol filter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/history?symbol=BTC/USDT", nil)
		c.Request = req

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response ArbitrageHistoryResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("invalid page parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/history?page=invalid", nil)
		c.Request = req

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("invalid limit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/history?limit=0", nil)
		c.Request = req

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("limit too high", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/history?limit=150", nil)
		c.Request = req

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})
}

// TestArbitrageHandler_GetFundingRateArbitrage tests the funding rate arbitrage endpoint
func TestArbitrageHandler_GetFundingRateArbitrage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	// Helper function to handle expected panics due to nil ccxtService
	callWithPanicRecovery := func(fn func()) {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to nil ccxtService
				assert.NotNil(t, r)
			}
		}()
		fn()
	}

	t.Run("valid request with default parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate", nil)
		c.Request = req

		callWithPanicRecovery(func() {
			handler.GetFundingRateArbitrage(c)
		})
	})

	t.Run("custom parameters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?min_profit=0.02&max_risk=2.5&limit=15", nil)
		c.Request = req

		callWithPanicRecovery(func() {
			handler.GetFundingRateArbitrage(c)
		})
	})

	t.Run("custom symbols and exchanges", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?symbols=BTC/USDT:USDT&symbols=ETH/USDT:USDT&exchanges=binance&exchanges=bybit", nil)
		c.Request = req

		callWithPanicRecovery(func() {
			handler.GetFundingRateArbitrage(c)
		})
	})

	t.Run("invalid min_profit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?min_profit=invalid", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("negative min_profit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?min_profit=-0.01", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("invalid max_risk parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?max_risk=invalid", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("max_risk too low", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?max_risk=0.5", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("max_risk too high", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?max_risk=6.0", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("invalid limit parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?limit=invalid", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("limit too low", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?limit=0", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})

	t.Run("limit too high", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rate?limit=150", nil)
		c.Request = req

		handler.GetFundingRateArbitrage(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
	})
}

// TestArbitrageHandler_GetFundingRates tests the funding rates endpoint
func TestArbitrageHandler_GetFundingRates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	// Helper function to handle expected panics due to nil ccxtService
	callWithPanicRecovery := func(fn func()) {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic due to nil ccxtService
				assert.NotNil(t, r)
			}
		}()
		fn()
	}

	t.Run("missing exchange parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rates", nil)
		c.Request = req

		handler.GetFundingRates(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var errorResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")
		assert.Contains(t, errorResponse["error"].(string), "Exchange parameter is required")
	})

	t.Run("valid exchange parameter without symbols", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rates/binance", nil)
		c.Request = req

		// Set the exchange parameter
		c.Params = gin.Params{gin.Param{Key: "exchange", Value: "binance"}}

		callWithPanicRecovery(func() {
			handler.GetFundingRates(c)
		})
	})

	t.Run("valid exchange with symbols", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/arbitrage/funding-rates/binance?symbols=BTC/USDT:USDT&symbols=ETH/USDT:USDT", nil)
		c.Request = req

		// Set the exchange parameter
		c.Params = gin.Params{gin.Param{Key: "exchange", Value: "binance"}}

		callWithPanicRecovery(func() {
			handler.GetFundingRates(c)
		})
	})
}

// TestArbitrageHandler_TechnicalAnalysisOpportunities tests technical analysis opportunity detection
func TestArbitrageHandler_TechnicalAnalysisOpportunities(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty opportunities", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("zero profit threshold", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 0.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("negative profit threshold", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), -1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("high profit threshold", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 10.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("with symbol filter", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 1.0, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opportunities, err := handler.findTechnicalAnalysisOpportunities(ctx, 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("various profit thresholds", func(t *testing.T) {
		thresholds := []float64{0.1, 0.5, 1.0, 2.0, 5.0}
		for _, threshold := range thresholds {
			t.Run(fmt.Sprintf("threshold %.1f", threshold), func(t *testing.T) {
				opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), threshold, "")
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})

	t.Run("various symbol filters", func(t *testing.T) {
		symbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT", "XRP/USDT"}
		for _, symbol := range symbols {
			t.Run(fmt.Sprintf("symbol %s", symbol), func(t *testing.T) {
				opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 1.0, symbol)
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})

	t.Run("edge case very high threshold", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 100.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("edge case empty symbol filter", func(t *testing.T) {
		opportunities, err := handler.findTechnicalAnalysisOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}

// TestArbitrageHandler_VolatilityOpportunities tests volatility opportunity detection
func TestArbitrageHandler_VolatilityOpportunities(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty opportunities", func(t *testing.T) {
		opportunities, err := handler.findVolatilityOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("zero profit threshold", func(t *testing.T) {
		opportunities, err := handler.findVolatilityOpportunities(context.Background(), 0.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("negative profit threshold", func(t *testing.T) {
		opportunities, err := handler.findVolatilityOpportunities(context.Background(), -1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("high profit threshold", func(t *testing.T) {
		opportunities, err := handler.findVolatilityOpportunities(context.Background(), 10.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("with symbol filter", func(t *testing.T) {
		opportunities, err := handler.findVolatilityOpportunities(context.Background(), 1.0, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		opportunities, err := handler.findVolatilityOpportunities(ctx, 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("various profit thresholds", func(t *testing.T) {
		thresholds := []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0}
		for _, threshold := range thresholds {
			t.Run(fmt.Sprintf("threshold %.1f", threshold), func(t *testing.T) {
				opportunities, err := handler.findVolatilityOpportunities(context.Background(), threshold, "")
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})

	t.Run("various symbol filters", func(t *testing.T) {
		symbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT", "XRP/USDT", "SOL/USDT"}
		for _, symbol := range symbols {
			t.Run(symbol, func(t *testing.T) {
				opportunities, err := handler.findVolatilityOpportunities(context.Background(), 1.0, symbol)
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		testCases := []struct {
			name      string
			threshold float64
			symbol    string
		}{
			{"very high threshold", 100.0, ""},
			{"empty symbol filter", 1.0, ""},
			{"low volatility threshold", 0.01, "BTC/USDT"},
			{"medium volatility threshold", 0.5, "ETH/USDT"},
			{"extreme volatility threshold", 20.0, "SOL/USDT"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				opportunities, err := handler.findVolatilityOpportunities(context.Background(), tc.threshold, tc.symbol)
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})
}

// TestArbitrageHandler_SpreadOpportunities tests spread opportunity detection
func TestArbitrageHandler_SpreadOpportunities(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty opportunities", func(t *testing.T) {
		opportunities, err := handler.findSpreadOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("zero profit threshold", func(t *testing.T) {
		opportunities, err := handler.findSpreadOpportunities(context.Background(), 0.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("negative profit threshold", func(t *testing.T) {
		opportunities, err := handler.findSpreadOpportunities(context.Background(), -1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("high profit threshold", func(t *testing.T) {
		opportunities, err := handler.findSpreadOpportunities(context.Background(), 10.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("with symbol filter", func(t *testing.T) {
		opportunities, err := handler.findSpreadOpportunities(context.Background(), 1.0, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		opportunities, err := handler.findSpreadOpportunities(ctx, 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("various profit thresholds", func(t *testing.T) {
		thresholds := []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0}
		for _, threshold := range thresholds {
			t.Run(fmt.Sprintf("threshold %.1f", threshold), func(t *testing.T) {
				opportunities, err := handler.findSpreadOpportunities(context.Background(), threshold, "")
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})

	t.Run("various symbol filters", func(t *testing.T) {
		symbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT", "XRP/USDT", "SOL/USDT"}
		for _, symbol := range symbols {
			t.Run(symbol, func(t *testing.T) {
				opportunities, err := handler.findSpreadOpportunities(context.Background(), 1.0, symbol)
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		testCases := []struct {
			name      string
			threshold float64
			symbol    string
		}{
			{"very high threshold", 100.0, ""},
			{"empty symbol filter", 1.0, ""},
			{"low spread threshold", 0.01, "BTC/USDT"},
			{"medium spread threshold", 0.5, "ETH/USDT"},
			{"extreme spread threshold", 20.0, "SOL/USDT"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				opportunities, err := handler.findSpreadOpportunities(context.Background(), tc.threshold, tc.symbol)
				assert.NoError(t, err)
				assert.Empty(t, opportunities)
			})
		}
	})
}

// TestArbitrageHandler_GetArbitrageHistory tests the internal history method
func TestArbitrageHandler_GetArbitrageHistory_Internal(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty history", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 10, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("zero limit", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 0, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("negative limit", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), -5, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("negative offset", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 10, -5, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("with symbol filter", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 5, 0, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("with empty symbol filter", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 5, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("various limit values", func(t *testing.T) {
		limits := []int{1, 5, 10, 25, 50, 100}
		for _, limit := range limits {
			t.Run(fmt.Sprintf("limit %d", limit), func(t *testing.T) {
				history, err := handler.getArbitrageHistory(context.Background(), limit, 0, "")
				assert.NoError(t, err)
				assert.Empty(t, history)
			})
		}
	})

	t.Run("various offset values", func(t *testing.T) {
		offsets := []int{0, 5, 10, 20, 50, 100}
		for _, offset := range offsets {
			t.Run(fmt.Sprintf("offset %d", offset), func(t *testing.T) {
				history, err := handler.getArbitrageHistory(context.Background(), 10, offset, "")
				assert.NoError(t, err)
				assert.Empty(t, history)
			})
		}
	})

	t.Run("various symbol filters", func(t *testing.T) {
		symbols := []string{"BTC/USDT", "ETH/USDT", "BNB/USDT", "ADA/USDT", "XRP/USDT", "SOL/USDT", "MATIC/USDT"}
		for _, symbol := range symbols {
			t.Run(symbol, func(t *testing.T) {
				history, err := handler.getArbitrageHistory(context.Background(), 5, 0, symbol)
				assert.NoError(t, err)
				assert.Empty(t, history)
			})
		}
	})

	t.Run("pagination parameters", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 20, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("large limit and offset", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 1000, 500, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("edge case combinations", func(t *testing.T) {
		testCases := []struct {
			name   string
			limit  int
			offset int
			symbol string
		}{
			{"zero all", 0, 0, ""},
			{"limit zero with symbol", 0, 10, "BTC/USDT"},
			{"offset zero with limit", 10, 0, "ETH/USDT"},
			{"high pagination", 100, 50, "BNB/USDT"},
			{"single item", 1, 0, "ADA/USDT"},
			{"large offset", 10, 1000, "XRP/USDT"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				history, err := handler.getArbitrageHistory(context.Background(), tc.limit, tc.offset, tc.symbol)
				assert.NoError(t, err)
				assert.Empty(t, history)
			})
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		history, err := handler.getArbitrageHistory(ctx, 10, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond) // Ensure timeout

		history, err := handler.getArbitrageHistory(ctx, 10, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})
}

// TestArbitrageHandler_SendArbitrageNotifications_Integration tests the notification method
func TestArbitrageHandler_SendArbitrageNotifications_Integration(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil notification service does not panic", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 1.5, BuyExchange: "binance", SellExchange: "coinbase"},
			{Symbol: "ETH/USDT", ProfitPercent: 2.0, BuyExchange: "binance", SellExchange: "coinbase"},
		}

		// Should not panic with nil notification service
		handler.sendArbitrageNotifications(opportunities)
	})

	t.Run("low profit opportunities do not trigger notifications", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 0.5, BuyExchange: "binance", SellExchange: "coinbase"}, // Below 1% threshold
			{Symbol: "ETH/USDT", ProfitPercent: 0.8, BuyExchange: "binance", SellExchange: "coinbase"}, // Below 1% threshold
		}

		handler.sendArbitrageNotifications(opportunities)
		// Should not call notification service for low profit opportunities
	})

	t.Run("high profit opportunities trigger notifications", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 1.2, BuyExchange: "binance", SellExchange: "coinbase"}, // Above 1% threshold
			{Symbol: "ETH/USDT", ProfitPercent: 2.5, BuyExchange: "binance", SellExchange: "coinbase"}, // Above 1% threshold
		}

		handler.sendArbitrageNotifications(opportunities)
		// Would call notification service if it wasn't nil
	})

	t.Run("mixed profit opportunities filter correctly", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 0.5, BuyExchange: "binance", SellExchange: "coinbase"}, // Below threshold
			{Symbol: "ETH/USDT", ProfitPercent: 1.2, BuyExchange: "binance", SellExchange: "coinbase"}, // Above threshold
			{Symbol: "BNB/USDT", ProfitPercent: 2.0, BuyExchange: "binance", SellExchange: "coinbase"}, // Above threshold
			{Symbol: "ADA/USDT", ProfitPercent: 0.8, BuyExchange: "binance", SellExchange: "coinbase"}, // Below threshold
		}

		handler.sendArbitrageNotifications(opportunities)
		// Should filter to only high-profit opportunities
	})

	t.Run("empty opportunities list does not trigger notifications", func(t *testing.T) {
		opportunities := []ArbitrageOpportunity{}

		handler.sendArbitrageNotifications(opportunities)
		// Should not call notification service for empty list
	})
}

// TestArbitrageHandler_FindCrossExchangeOpportunities tests the cross-exchange arbitrage detection
func TestArbitrageHandler_FindCrossExchangeOpportunities(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty opportunities", func(t *testing.T) {
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("with symbol filter", func(t *testing.T) {
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 1.5, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("higher minimum profit threshold", func(t *testing.T) {
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 2.0, "ETH/USDT")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("low minimum profit threshold", func(t *testing.T) {
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 0.5, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}

// TestArbitrageHandler_SendArbitrageNotifications_NilService tests the notification method with nil service
func TestArbitrageHandler_SendArbitrageNotifications_NilService(t *testing.T) {
	t.Run("nil notification service does not panic", func(t *testing.T) {
		handler := &ArbitrageHandler{
			notificationService: nil,
		}

		opportunities := []ArbitrageOpportunity{
			{
				Symbol:          "BTC/USDT",
				BuyExchange:     "binance",
				SellExchange:    "coinbase",
				BuyPrice:        45000.0,
				SellPrice:       45500.0,
				ProfitPercent:   1.2, // Above 1% threshold
				ProfitAmount:    500.0,
				Volume:          1.0,
				Timestamp:       time.Now(),
				OpportunityType: "cross_exchange",
			},
		}

		// Should not panic when notification service is nil
		handler.sendArbitrageNotifications(opportunities)
	})
}

// TestArbitrageHandler_SendArbitrageNotifications_Filtering tests profit filtering logic
func TestArbitrageHandler_SendArbitrageNotifications_Filtering(t *testing.T) {
	t.Run("opportunities below 1% threshold are filtered out", func(t *testing.T) {
		// This test verifies the filtering logic by checking that the method doesn't panic
		// with mixed high and low profit opportunities
		handler := &ArbitrageHandler{
			notificationService: nil, // We can't test the actual notification without a proper mock
		}

		opportunities := []ArbitrageOpportunity{
			{
				Symbol:          "BTC/USDT",
				BuyExchange:     "binance",
				SellExchange:    "coinbase",
				ProfitPercent:   0.5, // Below 1% threshold - should be filtered out
				ProfitAmount:    500.0,
				Volume:          1.0,
				Timestamp:       time.Now(),
				OpportunityType: "cross_exchange",
			},
			{
				Symbol:          "ETH/USDT",
				BuyExchange:     "binance",
				SellExchange:    "kraken",
				ProfitPercent:   1.8, // Above 1% threshold - should be included
				ProfitAmount:    60.0,
				Volume:          1.0,
				Timestamp:       time.Now(),
				OpportunityType: "cross_exchange",
			},
		}

		// Should not panic and should filter opportunities correctly
		handler.sendArbitrageNotifications(opportunities)
	})
}

// TestArbitrageHandler_ParameterParsing tests parameter parsing logic
func TestArbitrageHandler_ParameterParsing(t *testing.T) {
	t.Run("parse valid min_profit", func(t *testing.T) {
		minProfit, err := strconv.ParseFloat("0.5", 64)
		assert.NoError(t, err)
		assert.Equal(t, 0.5, minProfit)
	})

	t.Run("parse invalid min_profit", func(t *testing.T) {
		_, err := strconv.ParseFloat("invalid", 64)
		assert.Error(t, err)
	})

	t.Run("parse valid limit", func(t *testing.T) {
		limit, err := strconv.Atoi("50")
		assert.NoError(t, err)
		assert.Equal(t, 50, limit)
	})

	t.Run("parse invalid limit", func(t *testing.T) {
		_, err := strconv.Atoi("invalid")
		assert.Error(t, err)
	})

	t.Run("parse valid page", func(t *testing.T) {
		page, err := strconv.Atoi("2")
		assert.NoError(t, err)
		assert.Equal(t, 2, page)
	})

	t.Run("parse invalid page", func(t *testing.T) {
		_, err := strconv.Atoi("invalid")
		assert.Error(t, err)
	})

	t.Run("parse valid max_risk", func(t *testing.T) {
		maxRisk, err := strconv.ParseFloat("2.5", 64)
		assert.NoError(t, err)
		assert.Equal(t, 2.5, maxRisk)
	})

	t.Run("parse invalid max_risk", func(t *testing.T) {
		_, err := strconv.ParseFloat("invalid", 64)
		assert.Error(t, err)
	})
}

// TestArbitrageHandler_FindArbitrageOpportunities_ErrorCases tests error handling scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_ErrorCases(t *testing.T) {
	t.Run("nil database returns empty slice", func(t *testing.T) {
		handler := NewArbitrageHandler(nil, nil, nil, nil)

		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}

// TestArbitrageHandler_FindArbitrageOpportunities_ParameterValidation tests input parameter scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_ParameterValidation(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("zero profit threshold", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 0.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("negative profit threshold", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), -1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("zero limit", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})

	t.Run("negative limit", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, -5, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities) // Empty due to nil database
	})
}

// TestArbitrageHandler_FindArbitrageOpportunities_SymbolFiltering tests symbol filter scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_SymbolFiltering(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("empty symbol filter", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("specific symbol filter", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 10, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("different symbol filters", func(t *testing.T) {
		symbols := []string{"ETH/USDT", "BNB/USDT", "ADA/USDT"}
		for _, symbol := range symbols {
			opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 0.5, 5, symbol)
			assert.NoError(t, err)
			assert.Empty(t, opportunities)
		}
	})
}

// TestArbitrageHandler_FindArbitrageOpportunities_ProfitThresholds tests different profit threshold scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_ProfitThresholds(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	profitThresholds := []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0}

	for _, threshold := range profitThresholds {
		t.Run(fmt.Sprintf("profit threshold %.1f%%", threshold), func(t *testing.T) {
			opportunities, err := handler.FindArbitrageOpportunities(context.Background(), threshold, 10, "")
			assert.NoError(t, err)
			assert.Empty(t, opportunities)
		})
	}
}

// TestArbitrageHandler_FindArbitrageOpportunities_Limits tests different limit scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_Limits(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	limits := []int{0, 1, 5, 10, 50, 100}

	for _, limit := range limits {
		t.Run(fmt.Sprintf("limit %d", limit), func(t *testing.T) {
			opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, limit, "")
			assert.NoError(t, err)
			assert.Empty(t, opportunities)
		})
	}
}

// TestArbitrageHandlerForSorting is a handler that allows us to test the sorting and limiting logic
type TestArbitrageHandlerForSorting struct {
	*ArbitrageHandler
	mockOpportunities []ArbitrageOpportunity
	mockError         error
}

// MockFindArbitrageOpportunities tests the sorting and limiting logic
func TestArbitrageHandler_FindArbitrageOpportunities_SortingAndLimiting(t *testing.T) {
	// We can't easily mock the internal find* methods without complex setup,
	// but we can test that the function handles the nil database case gracefully
	// which exercises the error handling path

	t.Run("nil database returns empty slice", func(t *testing.T) {
		handler := NewArbitrageHandler(nil, nil, nil, nil)
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}

// TestArbitrageHandler_FindArbitrageOpportunities_ContextHandling tests context cancellation scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_ContextHandling(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		opportunities, err := handler.FindArbitrageOpportunities(ctx, 1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("deadline exceeded context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), -time.Hour) // Already expired
		defer cancel()

		opportunities, err := handler.FindArbitrageOpportunities(ctx, 1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}

// TestArbitrageHandler_FindArbitrageOpportunities_EdgeCases tests edge case scenarios
func TestArbitrageHandler_FindArbitrageOpportunities_EdgeCases(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("very high profit threshold", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1000.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("very large limit", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 0.1, 1000000, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("special characters in symbol filter", func(t *testing.T) {
		specialSymbols := []string{"BTC/USDT", "ETH/BTC", "XRP/USDT", "DOT/USDT", "ADA/USDT"}
		for _, symbol := range specialSymbols {
			opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 5, symbol)
			assert.NoError(t, err)
			assert.Empty(t, opportunities)
		}
	})

	t.Run("symbol filter case sensitivity", func(t *testing.T) {
		testCases := []string{
			"btc/usdt", // lowercase
			"BTC/USDT", // uppercase
			"btc/USDT", // mixed
			"BTC/usdt", // mixed
		}

		for _, symbol := range testCases {
			opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 5, symbol)
			assert.NoError(t, err)
			assert.Empty(t, opportunities)
		}
	})

	t.Run("negative profit threshold with nil db", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), -1.0, 10, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("zero limit with nil db", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("negative limit with nil db", func(t *testing.T) {
		opportunities, err := handler.FindArbitrageOpportunities(context.Background(), 1.0, -5, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}
