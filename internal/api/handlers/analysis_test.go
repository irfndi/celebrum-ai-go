package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTechnicalIndicator_Struct(t *testing.T) {
	indicator := TechnicalIndicator{
		Symbol:     "BTC/USDT",
		Exchange:   "binance",
		Timeframe:  "1h",
		Timestamp:  time.Now(),
		Price:      50000.0,
		Indicators: map[string]interface{}{"rsi": 65.5, "macd": 0.25},
	}

	assert.Equal(t, "BTC/USDT", indicator.Symbol)
	assert.Equal(t, "binance", indicator.Exchange)
	assert.Equal(t, "1h", indicator.Timeframe)
	assert.Equal(t, 50000.0, indicator.Price)
	assert.NotZero(t, indicator.Timestamp)
	assert.Contains(t, indicator.Indicators, "rsi")
	assert.Contains(t, indicator.Indicators, "macd")
}

func TestTradingSignal_Struct(t *testing.T) {
	signal := TradingSignal{
		Symbol:     "BTC/USDT",
		Exchange:   "binance",
		SignalType: "BUY",
		Strength:   "STRONG",
		Price:      50000.0,
		Reason:     "RSI oversold",
		Confidence: 0.8,
		Timestamp:  time.Now(),
		Indicators: []string{"RSI", "MACD"},
	}

	assert.Equal(t, "BTC/USDT", signal.Symbol)
	assert.Equal(t, "binance", signal.Exchange)
	assert.Equal(t, "BUY", signal.SignalType)
	assert.Equal(t, "STRONG", signal.Strength)
	assert.Equal(t, 50000.0, signal.Price)
	assert.Equal(t, "RSI oversold", signal.Reason)
	assert.Equal(t, 0.8, signal.Confidence)
	assert.NotZero(t, signal.Timestamp)
	assert.Contains(t, signal.Indicators, "RSI")
	assert.Contains(t, signal.Indicators, "MACD")
}

func TestIndicatorsResponse_Struct(t *testing.T) {
	indicators := []TechnicalIndicator{
		{
			Symbol:     "BTC/USDT",
			Exchange:   "binance",
			Timeframe:  "1h",
			Timestamp:  time.Now(),
			Price:      50000.0,
			Indicators: map[string]interface{}{"rsi": 65.5},
		},
		{
			Symbol:     "ETH/USDT",
			Exchange:   "binance",
			Timeframe:  "1h",
			Timestamp:  time.Now(),
			Price:      3000.0,
			Indicators: map[string]interface{}{"macd": 0.25},
		},
	}

	response := IndicatorsResponse{
		Indicators: indicators,
		Count:      2,
		Timestamp:  time.Now(),
	}

	assert.Len(t, response.Indicators, 2)
	assert.Equal(t, 2, response.Count)
	assert.NotZero(t, response.Timestamp)
	assert.Equal(t, "BTC/USDT", response.Indicators[0].Symbol)
	assert.Equal(t, "ETH/USDT", response.Indicators[1].Symbol)
}

func TestSignalsResponse_Struct(t *testing.T) {
	signals := []TradingSignal{
		{
			Symbol:     "BTC/USDT",
			Exchange:   "binance",
			SignalType: "BUY",
			Strength:   "STRONG",
			Price:      50000.0,
			Reason:     "RSI oversold",
			Confidence: 0.8,
			Timestamp:  time.Now(),
			Indicators: []string{"RSI"},
		},
		{
			Symbol:     "ETH/USDT",
			Exchange:   "binance",
			SignalType: "SELL",
			Strength:   "MODERATE",
			Price:      3000.0,
			Reason:     "MACD bearish",
			Confidence: 0.6,
			Timestamp:  time.Now(),
			Indicators: []string{"MACD"},
		},
	}

	response := SignalsResponse{
		Signals:   signals,
		Count:     2,
		Timestamp: time.Now(),
	}

	assert.Len(t, response.Signals, 2)
	assert.Equal(t, 2, response.Count)
	assert.NotZero(t, response.Timestamp)
	assert.Equal(t, "BTC/USDT", response.Signals[0].Symbol)
	assert.Equal(t, "BUY", response.Signals[0].SignalType)
	assert.Equal(t, "ETH/USDT", response.Signals[1].Symbol)
	assert.Equal(t, "SELL", response.Signals[1].SignalType)
}

func TestOHLCV_Struct(t *testing.T) {
	ohlcv := OHLCV{
		Timestamp: time.Now(),
		Open:      50000.0,
		High:      51000.0,
		Low:       49500.0,
		Close:     50500.0,
		Volume:    1000.0,
	}

	assert.NotZero(t, ohlcv.Timestamp)
	assert.Equal(t, 50000.0, ohlcv.Open)
	assert.Equal(t, 51000.0, ohlcv.High)
	assert.Equal(t, 49500.0, ohlcv.Low)
	assert.Equal(t, 50500.0, ohlcv.Close)
	assert.Equal(t, 1000.0, ohlcv.Volume)
}

func TestNewAnalysisHandler(t *testing.T) {
	mockCCXT := &MockCCXTService{}
	handler := NewAnalysisHandler(nil, mockCCXT)

	assert.NotNil(t, handler)
	assert.Equal(t, mockCCXT, handler.ccxtService)
}

func TestAnalysisHandler_GetTechnicalIndicators(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		symbol         string
		exchange       string
		timeframe      string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "missing symbol",
			symbol:         "",
			exchange:       "binance",
			timeframe:      "1h",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "missing exchange",
			symbol:         "BTC/USDT",
			exchange:       "",
			timeframe:      "1h",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCCXT := &MockCCXTService{}
			handler := NewAnalysisHandler(nil, mockCCXT)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set query parameters
			c.Request = httptest.NewRequest("GET", "/analysis/indicators", nil)
			q := c.Request.URL.Query()
			if tt.symbol != "" {
				q.Add("symbol", tt.symbol)
			}
			if tt.exchange != "" {
				q.Add("exchange", tt.exchange)
			}
			if tt.timeframe != "" {
				q.Add("timeframe", tt.timeframe)
			}
			c.Request.URL.RawQuery = q.Encode()

			handler.GetTechnicalIndicators(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestAnalysisHandler_GetTradingSignals(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		symbol         string
		exchange       string
		timeframe      string
		minConfidence  string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "invalid min_confidence",
			symbol:         "BTC/USDT",
			exchange:       "binance",
			timeframe:      "1h",
			minConfidence:  "invalid",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCCXT := &MockCCXTService{}
			handler := NewAnalysisHandler(nil, mockCCXT)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set query parameters
			c.Request = httptest.NewRequest("GET", "/analysis/signals", nil)
			q := c.Request.URL.Query()
			if tt.symbol != "" {
				q.Add("symbol", tt.symbol)
			}
			if tt.exchange != "" {
				q.Add("exchange", tt.exchange)
			}
			if tt.timeframe != "" {
				q.Add("timeframe", tt.timeframe)
			}
			if tt.minConfidence != "" {
				q.Add("min_confidence", tt.minConfidence)
			}
			c.Request.URL.RawQuery = q.Encode()

			handler.GetTradingSignals(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectError {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			}
		})
	}
}
