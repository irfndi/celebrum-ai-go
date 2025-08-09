package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test ArbitrageOpportunity struct
func TestArbitrageOpportunity_Struct(t *testing.T) {
	now := time.Now()
	expiry := now.Add(time.Hour)
	buyPrice := decimal.NewFromFloat(100.50)
	sellPrice := decimal.NewFromFloat(102.00)
	profitPercentage := decimal.NewFromFloat(1.49)

	arb := ArbitrageOpportunity{
		ID:               "arb-123",
		TradingPairID:    1,
		BuyExchangeID:    1,
		SellExchangeID:   2,
		BuyPrice:         buyPrice,
		SellPrice:        sellPrice,
		ProfitPercentage: profitPercentage,
		DetectedAt:       now,
		ExpiresAt:        expiry,
	}

	assert.Equal(t, "arb-123", arb.ID)
	assert.Equal(t, 1, arb.TradingPairID)
	assert.Equal(t, 1, arb.BuyExchangeID)
	assert.Equal(t, 2, arb.SellExchangeID)
	assert.True(t, buyPrice.Equal(arb.BuyPrice))
	assert.True(t, sellPrice.Equal(arb.SellPrice))
	assert.True(t, profitPercentage.Equal(arb.ProfitPercentage))
	assert.Equal(t, now, arb.DetectedAt)
	assert.Equal(t, expiry, arb.ExpiresAt)
}

// Test ArbitrageOpportunityRequest struct
func TestArbitrageOpportunityRequest_Struct(t *testing.T) {
	minProfit := decimal.NewFromFloat(1.5)
	req := ArbitrageOpportunityRequest{
		MinProfit: minProfit,
		Symbol:    "BTC/USDT",
		Limit:     10,
	}

	assert.True(t, minProfit.Equal(req.MinProfit))
	assert.Equal(t, "BTC/USDT", req.Symbol)
	assert.Equal(t, 10, req.Limit)
}

// Test ArbitrageOpportunityResponse struct
func TestArbitrageOpportunityResponse_Struct(t *testing.T) {
	now := time.Now()
	expiry := now.Add(time.Hour)
	buyPrice := decimal.NewFromFloat(100.50)
	sellPrice := decimal.NewFromFloat(102.00)
	profitPercentage := decimal.NewFromFloat(1.49)

	resp := ArbitrageOpportunityResponse{
		ID:               "arb-123",
		Symbol:           "BTC/USDT",
		BuyExchange:      "binance",
		SellExchange:     "coinbase",
		BuyPrice:         buyPrice,
		SellPrice:        sellPrice,
		ProfitPercentage: profitPercentage,
		DetectedAt:       now,
		ExpiresAt:        expiry,
	}

	assert.Equal(t, "arb-123", resp.ID)
	assert.Equal(t, "BTC/USDT", resp.Symbol)
	assert.Equal(t, "binance", resp.BuyExchange)
	assert.Equal(t, "coinbase", resp.SellExchange)
	assert.True(t, buyPrice.Equal(resp.BuyPrice))
	assert.True(t, sellPrice.Equal(resp.SellPrice))
	assert.True(t, profitPercentage.Equal(resp.ProfitPercentage))
	assert.Equal(t, now, resp.DetectedAt)
	assert.Equal(t, expiry, resp.ExpiresAt)
}

// Test ArbitrageOpportunitiesResponse struct
func TestArbitrageOpportunitiesResponse_Struct(t *testing.T) {
	now := time.Now()
	opportunities := []ArbitrageOpportunityResponse{
		{ID: "arb-1", Symbol: "BTC/USDT"},
		{ID: "arb-2", Symbol: "ETH/USDT"},
	}

	resp := ArbitrageOpportunitiesResponse{
		Opportunities: opportunities,
		Count:         2,
		Timestamp:     now,
	}

	assert.Equal(t, 2, len(resp.Opportunities))
	assert.Equal(t, "arb-1", resp.Opportunities[0].ID)
	assert.Equal(t, "arb-2", resp.Opportunities[1].ID)
	assert.Equal(t, 2, resp.Count)
	assert.Equal(t, now, resp.Timestamp)
}

// Test Exchange struct
func TestExchange_Struct(t *testing.T) {
	now := time.Now()
	exchange := Exchange{
		ID:        1,
		Name:      "Binance",
		APIURL:    "https://api.binance.com",
		IsActive:  true,
		LastPing:  &now,
		CreatedAt: now,
	}

	assert.Equal(t, 1, exchange.ID)
	assert.Equal(t, "Binance", exchange.Name)
	assert.Equal(t, "https://api.binance.com", exchange.APIURL)
	assert.True(t, exchange.IsActive)
	assert.Equal(t, &now, exchange.LastPing)
	assert.Equal(t, now, exchange.CreatedAt)
}

// Test CCXTExchange struct
func TestCCXTExchange_Struct(t *testing.T) {
	now := time.Now()
	ccxtExchange := CCXTExchange{
		ID:               1,
		ExchangeID:       1,
		CCXTID:           "binance",
		IsTestnet:        false,
		APIKeyRequired:   true,
		RateLimit:        1200,
		HasFutures:       true,
		WebsocketEnabled: true,
		LastHealthCheck:  &now,
		Status:           "active",
	}

	assert.Equal(t, 1, ccxtExchange.ID)
	assert.Equal(t, 1, ccxtExchange.ExchangeID)
	assert.Equal(t, "binance", ccxtExchange.CCXTID)
	assert.False(t, ccxtExchange.IsTestnet)
	assert.True(t, ccxtExchange.APIKeyRequired)
	assert.Equal(t, 1200, ccxtExchange.RateLimit)
	assert.True(t, ccxtExchange.HasFutures)
	assert.True(t, ccxtExchange.WebsocketEnabled)
	assert.Equal(t, &now, ccxtExchange.LastHealthCheck)
	assert.Equal(t, "active", ccxtExchange.Status)
}

// Test ExchangeInfo struct
func TestExchangeInfo_Struct(t *testing.T) {
	exchangeInfo := ExchangeInfo{
		ID:               1,
		Name:             "Binance",
		CCXTID:           "binance",
		IsActive:         true,
		HasFutures:       true,
		WebsocketEnabled: true,
		Status:           "active",
	}

	assert.Equal(t, 1, exchangeInfo.ID)
	assert.Equal(t, "Binance", exchangeInfo.Name)
	assert.Equal(t, "binance", exchangeInfo.CCXTID)
	assert.True(t, exchangeInfo.IsActive)
	assert.True(t, exchangeInfo.HasFutures)
	assert.True(t, exchangeInfo.WebsocketEnabled)
	assert.Equal(t, "active", exchangeInfo.Status)
}

// Test MarketData struct
func TestMarketData_Struct(t *testing.T) {
	now := time.Now()
	price := decimal.NewFromFloat(50000.50)
	volume := decimal.NewFromFloat(1000.25)

	marketData := MarketData{
		ID:            "md-123",
		ExchangeID:    1,
		TradingPairID: 1,
		LastPrice:     price,
		Volume24h:     volume,
		Timestamp:     now,
		CreatedAt:     now,
	}

	assert.Equal(t, "md-123", marketData.ID)
	assert.Equal(t, 1, marketData.ExchangeID)
	assert.Equal(t, 1, marketData.TradingPairID)
	assert.True(t, price.Equal(marketData.LastPrice))
	assert.True(t, volume.Equal(marketData.Volume24h))
	assert.Equal(t, now, marketData.Timestamp)
	assert.Equal(t, now, marketData.CreatedAt)
}

// Test MarketPrice struct
func TestMarketPrice_Struct(t *testing.T) {
	now := time.Now()
	price := decimal.NewFromFloat(50000.50)
	volume := decimal.NewFromFloat(1000.25)

	marketPrice := MarketPrice{
		ExchangeID:   1,
		ExchangeName: "Binance",
		Symbol:       "BTC/USDT",
		Price:        price,
		Volume:       volume,
		Timestamp:    now,
	}

	assert.Equal(t, 1, marketPrice.ExchangeID)
	assert.Equal(t, "Binance", marketPrice.ExchangeName)
	assert.Equal(t, "BTC/USDT", marketPrice.Symbol)
	assert.True(t, price.Equal(marketPrice.Price))
	assert.True(t, volume.Equal(marketPrice.Volume))
	assert.Equal(t, now, marketPrice.Timestamp)
}

// Test TickerData struct
func TestTickerData_Struct(t *testing.T) {
	now := time.Now()
	bid := decimal.NewFromFloat(49999.99)
	ask := decimal.NewFromFloat(50000.01)
	last := decimal.NewFromFloat(50000.00)
	high := decimal.NewFromFloat(51000.00)
	low := decimal.NewFromFloat(49000.00)
	volume := decimal.NewFromFloat(1000.25)

	ticker := TickerData{
		Symbol:    "BTC/USDT",
		Bid:       bid,
		Ask:       ask,
		Last:      last,
		High:      high,
		Low:       low,
		Volume:    volume,
		Timestamp: now,
	}

	assert.Equal(t, "BTC/USDT", ticker.Symbol)
	assert.True(t, bid.Equal(ticker.Bid))
	assert.True(t, ask.Equal(ticker.Ask))
	assert.True(t, last.Equal(ticker.Last))
	assert.True(t, high.Equal(ticker.High))
	assert.True(t, low.Equal(ticker.Low))
	assert.True(t, volume.Equal(ticker.Volume))
	assert.Equal(t, now, ticker.Timestamp)
}

// Test OrderBookData struct
func TestOrderBookData_Struct(t *testing.T) {
	now := time.Now()
	bids := [][]decimal.Decimal{
		{decimal.NewFromFloat(49999.99), decimal.NewFromFloat(1.5)},
		{decimal.NewFromFloat(49999.98), decimal.NewFromFloat(2.0)},
	}
	asks := [][]decimal.Decimal{
		{decimal.NewFromFloat(50000.01), decimal.NewFromFloat(1.2)},
		{decimal.NewFromFloat(50000.02), decimal.NewFromFloat(1.8)},
	}

	orderBook := OrderBookData{
		Symbol:    "BTC/USDT",
		Bids:      bids,
		Asks:      asks,
		Timestamp: now,
	}

	assert.Equal(t, "BTC/USDT", orderBook.Symbol)
	assert.Equal(t, 2, len(orderBook.Bids))
	assert.Equal(t, 2, len(orderBook.Asks))
	assert.True(t, decimal.NewFromFloat(49999.99).Equal(orderBook.Bids[0][0]))
	assert.True(t, decimal.NewFromFloat(50000.01).Equal(orderBook.Asks[0][0]))
	assert.Equal(t, now, orderBook.Timestamp)
}

// Test MarketDataRequest struct
func TestMarketDataRequest_Struct(t *testing.T) {
	req := MarketDataRequest{
		Symbols:  []string{"BTC/USDT", "ETH/USDT"},
		Exchange: "binance",
		Limit:    100,
	}

	assert.Equal(t, []string{"BTC/USDT", "ETH/USDT"}, req.Symbols)
	assert.Equal(t, "binance", req.Exchange)
	assert.Equal(t, 100, req.Limit)
}

// Test TradingPair struct and methods
func TestTradingPair_Struct(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	tp := TradingPair{
		ID:            id,
		Symbol:        "BTC/USDT",
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, id, tp.ID)
	assert.Equal(t, "BTC/USDT", tp.Symbol)
	assert.Equal(t, "BTC", tp.BaseCurrency)
	assert.Equal(t, "USDT", tp.QuoteCurrency)
	assert.True(t, tp.IsActive)
	assert.Equal(t, now, tp.CreatedAt)
	assert.Equal(t, now, tp.UpdatedAt)
}

func TestTradingPair_TableName(t *testing.T) {
	tp := TradingPair{}
	assert.Equal(t, "trading_pairs", tp.TableName())
}

func TestTradingPair_String(t *testing.T) {
	tp := TradingPair{Symbol: "BTC/USDT"}
	assert.Equal(t, "BTC/USDT", tp.String())
}

func TestTradingPair_GetFullSymbol(t *testing.T) {
	tp := TradingPair{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
	}
	assert.Equal(t, "BTC/USDT", tp.GetFullSymbol())
}

// Test User struct
func TestUser_Struct(t *testing.T) {
	now := time.Now()
	chatID := "123456789"

	user := User{
		ID:               "user-123",
		Email:            "test@example.com",
		PasswordHash:     "hashed_password",
		TelegramChatID:   &chatID,
		SubscriptionTier: "premium",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.PasswordHash)
	assert.Equal(t, &chatID, user.TelegramChatID)
	assert.Equal(t, "premium", user.SubscriptionTier)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}

// Test UserAlert struct
func TestUserAlert_Struct(t *testing.T) {
	now := time.Now()
	conditions := json.RawMessage(`{"price_threshold": 50000}`)

	alert := UserAlert{
		ID:         "alert-123",
		UserID:     "user-123",
		AlertType:  "price",
		Conditions: conditions,
		IsActive:   true,
		CreatedAt:  now,
	}

	assert.Equal(t, "alert-123", alert.ID)
	assert.Equal(t, "user-123", alert.UserID)
	assert.Equal(t, "price", alert.AlertType)
	assert.Equal(t, conditions, alert.Conditions)
	assert.True(t, alert.IsActive)
	assert.Equal(t, now, alert.CreatedAt)
}

// Test Portfolio struct
func TestPortfolio_Struct(t *testing.T) {
	now := time.Now()
	quantity := decimal.NewFromFloat(1.5)
	avgPrice := decimal.NewFromFloat(50000.00)

	portfolio := Portfolio{
		ID:        "portfolio-123",
		UserID:    "user-123",
		Symbol:    "BTC",
		Quantity:  quantity,
		AvgPrice:  avgPrice,
		UpdatedAt: now,
	}

	assert.Equal(t, "portfolio-123", portfolio.ID)
	assert.Equal(t, "user-123", portfolio.UserID)
	assert.Equal(t, "BTC", portfolio.Symbol)
	assert.True(t, quantity.Equal(portfolio.Quantity))
	assert.True(t, avgPrice.Equal(portfolio.AvgPrice))
	assert.Equal(t, now, portfolio.UpdatedAt)
}

// Test UserRequest struct
func TestUserRequest_Struct(t *testing.T) {
	req := UserRequest{
		Email:            "test@example.com",
		Password:         "password123",
		TelegramChatID:   "123456789",
		SubscriptionTier: "premium",
	}

	assert.Equal(t, "test@example.com", req.Email)
	assert.Equal(t, "password123", req.Password)
	assert.Equal(t, "123456789", req.TelegramChatID)
	assert.Equal(t, "premium", req.SubscriptionTier)
}

// Test UserResponse struct
func TestUserResponse_Struct(t *testing.T) {
	now := time.Now()

	resp := UserResponse{
		ID:               "user-123",
		Email:            "test@example.com",
		TelegramChatID:   "123456789",
		SubscriptionTier: "premium",
		CreatedAt:        now,
	}

	assert.Equal(t, "user-123", resp.ID)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "123456789", resp.TelegramChatID)
	assert.Equal(t, "premium", resp.SubscriptionTier)
	assert.Equal(t, now, resp.CreatedAt)
}

// Test AlertConditions struct
func TestAlertConditions_Struct(t *testing.T) {
	priceThreshold := decimal.NewFromFloat(50000.00)
	profitThreshold := decimal.NewFromFloat(5.0)
	volumeThreshold := decimal.NewFromFloat(1000.0)

	conditions := AlertConditions{
		PriceThreshold:  &priceThreshold,
		ProfitThreshold: &profitThreshold,
		VolumeThreshold: &volumeThreshold,
		Symbol:          "BTC/USDT",
		Exchange:        "binance",
	}

	assert.True(t, priceThreshold.Equal(*conditions.PriceThreshold))
	assert.True(t, profitThreshold.Equal(*conditions.ProfitThreshold))
	assert.True(t, volumeThreshold.Equal(*conditions.VolumeThreshold))
	assert.Equal(t, "BTC/USDT", conditions.Symbol)
	assert.Equal(t, "binance", conditions.Exchange)
}

// Test TechnicalIndicator struct
func TestTechnicalIndicator_Struct(t *testing.T) {
	now := time.Now()
	values := json.RawMessage(`{"rsi": 65.5, "signal": "neutral"}`)

	indicator := TechnicalIndicator{
		ID:            "indicator-123",
		TradingPairID: 1,
		IndicatorType: "RSI",
		Timeframe:     "1h",
		Values:        values,
		CalculatedAt:  now,
	}

	assert.Equal(t, "indicator-123", indicator.ID)
	assert.Equal(t, 1, indicator.TradingPairID)
	assert.Equal(t, "RSI", indicator.IndicatorType)
	assert.Equal(t, "1h", indicator.Timeframe)
	assert.Equal(t, values, indicator.Values)
	assert.Equal(t, now, indicator.CalculatedAt)
}

// Test IndicatorData struct
func TestIndicatorData_Struct(t *testing.T) {
	now := time.Now()
	indicators := map[string]interface{}{
		"rsi":  65.5,
		"macd": map[string]float64{"macd": 1.2, "signal": 1.1},
	}

	data := IndicatorData{
		Symbol:     "BTC/USDT",
		Timeframe:  "1h",
		Indicators: indicators,
		Timestamp:  now,
	}

	assert.Equal(t, "BTC/USDT", data.Symbol)
	assert.Equal(t, "1h", data.Timeframe)
	assert.Equal(t, indicators, data.Indicators)
	assert.Equal(t, now, data.Timestamp)
}

// Test RSIData struct
func TestRSIData_Struct(t *testing.T) {
	now := time.Now()
	value := decimal.NewFromFloat(65.5)

	rsi := RSIData{
		Value:     value,
		Signal:    "neutral",
		Timestamp: now,
	}

	assert.True(t, value.Equal(rsi.Value))
	assert.Equal(t, "neutral", rsi.Signal)
	assert.Equal(t, now, rsi.Timestamp)
}

// Test MACDData struct
func TestMACDData_Struct(t *testing.T) {
	now := time.Now()
	macd := decimal.NewFromFloat(1.2)
	signal := decimal.NewFromFloat(1.1)
	histogram := decimal.NewFromFloat(0.1)

	macdData := MACDData{
		MACD:      macd,
		Signal:    signal,
		Histogram: histogram,
		Trend:     "bullish",
		Timestamp: now,
	}

	assert.True(t, macd.Equal(macdData.MACD))
	assert.True(t, signal.Equal(macdData.Signal))
	assert.True(t, histogram.Equal(macdData.Histogram))
	assert.Equal(t, "bullish", macdData.Trend)
	assert.Equal(t, now, macdData.Timestamp)
}

// Test SMAData struct
func TestSMAData_Struct(t *testing.T) {
	now := time.Now()
	value := decimal.NewFromFloat(50000.00)

	sma := SMAData{
		Period:    20,
		Value:     value,
		Trend:     "up",
		Timestamp: now,
	}

	assert.Equal(t, 20, sma.Period)
	assert.True(t, value.Equal(sma.Value))
	assert.Equal(t, "up", sma.Trend)
	assert.Equal(t, now, sma.Timestamp)
}

// Test Signal struct
func TestSignal_Struct(t *testing.T) {
	now := time.Now()
	price := decimal.NewFromFloat(50000.00)
	confidence := decimal.NewFromFloat(85.5)

	signal := Signal{
		Type:       "buy",
		Strength:   "strong",
		Price:      price,
		Indicator:  "RSI",
		Confidence: confidence,
		Timestamp:  now,
	}

	assert.Equal(t, "buy", signal.Type)
	assert.Equal(t, "strong", signal.Strength)
	assert.True(t, price.Equal(signal.Price))
	assert.Equal(t, "RSI", signal.Indicator)
	assert.True(t, confidence.Equal(signal.Confidence))
	assert.Equal(t, now, signal.Timestamp)
}

// Test TechnicalAnalysisRequest struct
func TestTechnicalAnalysisRequest_Struct(t *testing.T) {
	req := TechnicalAnalysisRequest{
		Symbol:     "BTC/USDT",
		Timeframe:  "1h",
		Indicators: []string{"RSI", "MACD", "SMA"},
		Period:     20,
	}

	assert.Equal(t, "BTC/USDT", req.Symbol)
	assert.Equal(t, "1h", req.Timeframe)
	assert.Equal(t, []string{"RSI", "MACD", "SMA"}, req.Indicators)
	assert.Equal(t, 20, req.Period)
}

// Test TechnicalAnalysisResponse struct
func TestTechnicalAnalysisResponse_Struct(t *testing.T) {
	now := time.Now()
	data := IndicatorData{
		Symbol:    "BTC/USDT",
		Timeframe: "1h",
	}
	signals := []Signal{
		{Type: "buy", Strength: "strong"},
		{Type: "sell", Strength: "weak"},
	}

	resp := TechnicalAnalysisResponse{
		Data:      data,
		Signals:   signals,
		Timestamp: now,
	}

	assert.Equal(t, "BTC/USDT", resp.Data.Symbol)
	assert.Equal(t, "1h", resp.Data.Timeframe)
	assert.Equal(t, 2, len(resp.Signals))
	assert.Equal(t, "buy", resp.Signals[0].Type)
	assert.Equal(t, "sell", resp.Signals[1].Type)
	assert.Equal(t, now, resp.Timestamp)
}

// Test JSON marshaling/unmarshaling for complex types
func TestAlertConditions_JSON(t *testing.T) {
	priceThreshold := decimal.NewFromFloat(50000.00)
	original := AlertConditions{
		PriceThreshold: &priceThreshold,
		Symbol:         "BTC/USDT",
		Exchange:       "binance",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaled AlertConditions
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Verify the data
	assert.True(t, priceThreshold.Equal(*unmarshaled.PriceThreshold))
	assert.Equal(t, "BTC/USDT", unmarshaled.Symbol)
	assert.Equal(t, "binance", unmarshaled.Exchange)
}
