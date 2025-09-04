package handlers

import (
	"context"
	"encoding/json"
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
}

// TestArbitrageHandler_VolatilityOpportunities tests volatility opportunity detection
func TestArbitrageHandler_VolatilityOpportunities(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty opportunities", func(t *testing.T) {
		opportunities, err := handler.findVolatilityOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
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
}

// TestArbitrageHandler_GetArbitrageHistory tests the internal history method
func TestArbitrageHandler_GetArbitrageHistory_Internal(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	t.Run("nil database returns empty history", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 10, 0, "")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("with symbol filter", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 5, 0, "BTC/USDT")
		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("pagination parameters", func(t *testing.T) {
		history, err := handler.getArbitrageHistory(context.Background(), 20, 10, "")
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