package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewArbitrageHandler(t *testing.T) {
	t.Run("create handler with nil dependencies", func(t *testing.T) {
		handler := NewArbitrageHandler(nil, nil, nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Nil(t, handler.ccxtService)
		assert.Nil(t, handler.notificationService)
	})

	t.Run("create handler with mock dependencies", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.db)
		assert.Equal(t, mockCCXT, handler.ccxtService)
		assert.Nil(t, handler.notificationService)
	})
}

func TestArbitrageHandler_GetArbitrageOpportunities_InvalidMinProfit(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/arbitrage", handler.GetArbitrageOpportunities)

	req, _ := http.NewRequest("GET", "/arbitrage?min_profit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid min_profit parameter", response["error"])
}

func TestArbitrageHandler_GetArbitrageOpportunities_InvalidLimit(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/arbitrage", handler.GetArbitrageOpportunities)

	req, _ := http.NewRequest("GET", "/arbitrage?limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid limit parameter", response["error"])
}

func TestArbitrageHandler_GetArbitrageHistory_InvalidPage(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/arbitrage/history", handler.GetArbitrageHistory)

	req, _ := http.NewRequest("GET", "/arbitrage/history?page=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid page parameter", response["error"])
}

func TestArbitrageHandler_GetArbitrageHistory_InvalidLimit(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/arbitrage/history", handler.GetArbitrageHistory)

	req, _ := http.NewRequest("GET", "/arbitrage/history?limit=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid limit parameter (1-100)", response["error"])
}

func TestArbitrageHandler_GetArbitrageHistory_LimitTooHigh(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/arbitrage/history", handler.GetArbitrageHistory)

	req, _ := http.NewRequest("GET", "/arbitrage/history?limit=101", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid limit parameter (1-100)", response["error"])
}

func TestArbitrageHandler_GetArbitrageHistory_PageZero(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/arbitrage/history", handler.GetArbitrageHistory)

	req, _ := http.NewRequest("GET", "/arbitrage/history?page=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid page parameter", response["error"])
}

func TestArbitrageOpportunity_Struct(t *testing.T) {
	now := time.Now()
	opp := ArbitrageOpportunity{
		Symbol:          "BTC/USD",
		BuyExchange:     "binance",
		SellExchange:    "coinbase",
		BuyPrice:        50000.0,
		SellPrice:       50500.0,
		ProfitPercent:   1.0,
		ProfitAmount:    500.0,
		Volume:          1.0,
		Timestamp:       now,
		OpportunityType: "arbitrage",
	}

	assert.Equal(t, "BTC/USD", opp.Symbol)
	assert.Equal(t, "binance", opp.BuyExchange)
	assert.Equal(t, "coinbase", opp.SellExchange)
	assert.Equal(t, 50000.0, opp.BuyPrice)
	assert.Equal(t, 50500.0, opp.SellPrice)
	assert.Equal(t, 1.0, opp.ProfitPercent)
	assert.Equal(t, 500.0, opp.ProfitAmount)
	assert.Equal(t, 1.0, opp.Volume)
	assert.Equal(t, now, opp.Timestamp)
	assert.Equal(t, "arbitrage", opp.OpportunityType)
}

func TestArbitrageResponse_Struct(t *testing.T) {
	now := time.Now()
	opportunities := []ArbitrageOpportunity{
		{
			Symbol:          "BTC/USD",
			BuyExchange:     "binance",
			SellExchange:    "coinbase",
			BuyPrice:        50000.0,
			SellPrice:       50500.0,
			ProfitPercent:   1.0,
			ProfitAmount:    500.0,
			Volume:          1.0,
			Timestamp:       now,
			OpportunityType: "arbitrage",
		},
	}

	response := ArbitrageResponse{
		Opportunities: opportunities,
		Count:         1,
		Timestamp:     now,
	}

	assert.Equal(t, opportunities, response.Opportunities)
	assert.Equal(t, 1, response.Count)
	assert.Equal(t, now, response.Timestamp)
}

func TestArbitrageHistoryItem_Struct(t *testing.T) {
	now := time.Now()
	item := ArbitrageHistoryItem{
		ID:            1,
		Symbol:        "BTC/USD",
		BuyExchange:   "binance",
		SellExchange:  "coinbase",
		BuyPrice:      50000.0,
		SellPrice:     50500.0,
		ProfitPercent: 1.0,
		DetectedAt:    now,
	}

	assert.Equal(t, 1, item.ID)
	assert.Equal(t, "BTC/USD", item.Symbol)
	assert.Equal(t, "binance", item.BuyExchange)
	assert.Equal(t, "coinbase", item.SellExchange)
	assert.Equal(t, 50000.0, item.BuyPrice)
	assert.Equal(t, 50500.0, item.SellPrice)
	assert.Equal(t, 1.0, item.ProfitPercent)
	assert.Equal(t, now, item.DetectedAt)
}

func TestArbitrageHistoryResponse_Struct(t *testing.T) {
	now := time.Now()
	history := []ArbitrageHistoryItem{
		{
			ID:            1,
			Symbol:        "BTC/USD",
			BuyExchange:   "binance",
			SellExchange:  "coinbase",
			BuyPrice:      50000.0,
			SellPrice:     50500.0,
			ProfitPercent: 1.0,
			DetectedAt:    now,
		},
	}

	response := ArbitrageHistoryResponse{
		History: history,
		Count:   1,
		Page:    1,
		Limit:   20,
	}

	assert.Equal(t, history, response.History)
	assert.Equal(t, 1, response.Count)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 20, response.Limit)
}

func TestArbitrageHandler_GetArbitrageOpportunities_Success(t *testing.T) {
	// Test with nil database (returns empty opportunities)
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/arbitrage?min_profit=0.5&limit=10", nil)

	handler.GetArbitrageOpportunities(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response ArbitrageResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Count) // Expect 0 opportunities with nil database
	assert.Len(t, response.Opportunities, 0)
	assert.NotZero(t, response.Timestamp)
}

func TestArbitrageHandler_GetArbitrageHistory_Success(t *testing.T) {
	// Test with nil database (returns empty history)
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/arbitrage/history?page=1&limit=20", nil)

	handler.GetArbitrageHistory(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var response ArbitrageHistoryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Count) // Expect 0 history items with nil database
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, 20, response.Limit)
	assert.Len(t, response.History, 0)
}

func TestArbitrageHandler_GetFundingRateArbitrage_InvalidMinProfit(t *testing.T) {
	handler := NewArbitrageHandler(nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/arbitrage/funding?min_profit=invalid", nil)

	handler.GetFundingRateArbitrage(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid min_profit parameter", response["error"])
}

func TestArbitrageHandler_sendArbitrageNotifications(t *testing.T) {
	t.Run("nil notification service", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		// Should not panic with nil notification service
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 2.0},
		}
		handler.sendArbitrageNotifications(opportunities)
	})

	t.Run("low profit opportunities", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		// Low profit opportunities should not trigger notifications
		opportunities := []ArbitrageOpportunity{
			{Symbol: "BTC/USDT", ProfitPercent: 0.5}, // Below 1% threshold
		}
		handler.sendArbitrageNotifications(opportunities)
	})
}

func TestGetArbitrageOpportunities(t *testing.T) {
	t.Run("successful request with default parameters", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/opportunities", nil)

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "opportunities")
		assert.Contains(t, response, "count")
		assert.Contains(t, response, "timestamp")
	})

	t.Run("request with custom parameters", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/opportunities?min_profit=1.0&limit=10&symbol=BTC/USDT", nil)
		c.Request.URL.RawQuery = "min_profit=1.0&limit=10&symbol=BTC/USDT"

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "opportunities")
		assert.Contains(t, response, "count")
		assert.Contains(t, response, "timestamp")
	})

	t.Run("invalid min_profit parameter", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/opportunities?min_profit=invalid", nil)
		c.Request.URL.RawQuery = "min_profit=invalid"

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"], "Invalid min_profit parameter")
	})

	t.Run("invalid limit parameter", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/opportunities?limit=invalid", nil)
		c.Request.URL.RawQuery = "limit=invalid"

		handler.GetArbitrageOpportunities(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"], "Invalid limit parameter")
	})
}

func TestGetArbitrageHistory(t *testing.T) {
	t.Run("successful request with default parameters", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/history", nil)

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "history")
		assert.Contains(t, response, "count")
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "limit")
	})

	t.Run("request with custom pagination", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/history?page=2&limit=10&symbol=BTC/USDT", nil)
		c.Request.URL.RawQuery = "page=2&limit=10&symbol=BTC/USDT"

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(2), response["page"])
		assert.Equal(t, float64(10), response["limit"])
	})

	t.Run("invalid page parameter", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/history?page=invalid", nil)
		c.Request.URL.RawQuery = "page=invalid"

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"], "Invalid page parameter")
	})

	t.Run("invalid limit parameter", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/history?limit=invalid", nil)
		c.Request.URL.RawQuery = "limit=invalid"

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"], "Invalid limit parameter")
	})

	t.Run("limit too high", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/arbitrage/history?limit=200", nil)
		c.Request.URL.RawQuery = "limit=200"

		handler.GetArbitrageHistory(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.Contains(t, response["error"], "Invalid limit parameter")
	})
}

func TestArbitrageHandler_findCrossExchangeArbitrage(t *testing.T) {
	t.Run("no opportunities", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		// Test with no market data
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 1.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})

	t.Run("valid opportunity", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		// Test finding opportunities (would need proper mock setup)
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 0.5, "BTC/USDT")
		assert.NoError(t, err)
		// Additional assertions would depend on mock data setup
		_ = opportunities
	})

	t.Run("edge cases", func(t *testing.T) {
		mockCCXT := &MockCCXTService{}
		handler := NewArbitrageHandler(nil, mockCCXT, nil, nil)

		// Test with very high minimum profit
		opportunities, err := handler.findCrossExchangeOpportunities(context.Background(), 100.0, "")
		assert.NoError(t, err)
		assert.Empty(t, opportunities)
	})
}
