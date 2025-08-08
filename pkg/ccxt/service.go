package ccxt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/shopspring/decimal"
)

// Service provides high-level CCXT operations
type Service struct {
	client          *Client
	supportedExchanges map[string]ExchangeInfo
	mu              sync.RWMutex
	lastUpdate      time.Time
}

// NewService creates a new CCXT service instance
func NewService(cfg *config.CCXTConfig) *Service {
	return &Service{
		client:             NewClient(cfg),
		supportedExchanges: make(map[string]ExchangeInfo),
	}
}

// Initialize initializes the service by fetching supported exchanges
func (s *Service) Initialize(ctx context.Context) error {
	exchangesResp, err := s.client.GetExchanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch supported exchanges: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, exchange := range exchangesResp.Exchanges {
		s.supportedExchanges[exchange.ID] = exchange
	}
	s.lastUpdate = time.Now()

	return nil
}

// IsHealthy checks if the CCXT service is healthy
func (s *Service) IsHealthy(ctx context.Context) bool {
	_, err := s.client.HealthCheck(ctx)
	return err == nil
}

// GetSupportedExchanges returns a list of supported exchange IDs
func (s *Service) GetSupportedExchanges() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exchanges := make([]string, 0, len(s.supportedExchanges))
	for id := range s.supportedExchanges {
		exchanges = append(exchanges, id)
	}
	return exchanges
}

// GetExchangeInfo returns information about a specific exchange
func (s *Service) GetExchangeInfo(exchangeID string) (ExchangeInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.supportedExchanges[exchangeID]
	return info, exists
}

// FetchMarketData fetches market data for multiple exchanges and symbols
func (s *Service) FetchMarketData(ctx context.Context, exchanges []string, symbols []string) ([]models.MarketPrice, error) {
	if len(exchanges) == 0 || len(symbols) == 0 {
		return nil, fmt.Errorf("exchanges and symbols cannot be empty")
	}

	// Use the bulk tickers endpoint for efficiency
	req := &TickersRequest{
		Symbols:   symbols,
		Exchanges: exchanges,
	}

	resp, err := s.client.GetTickers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tickers: %w", err)
	}

	marketData := make([]models.MarketPrice, 0, len(resp.Tickers))
	for _, tickerData := range resp.Tickers {
		md := models.MarketPrice{
			ExchangeName: tickerData.Exchange,
			Symbol:       tickerData.Ticker.Symbol,
			Price:        tickerData.Ticker.Last,
			Volume:       tickerData.Ticker.Volume,
			Timestamp:    tickerData.Ticker.Timestamp,
		}
		marketData = append(marketData, md)
	}

	return marketData, nil
}

// FetchSingleTicker fetches ticker data for a single exchange and symbol
func (s *Service) FetchSingleTicker(ctx context.Context, exchange, symbol string) (*models.MarketPrice, error) {
	resp, err := s.client.GetTicker(ctx, exchange, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticker for %s:%s: %w", exchange, symbol, err)
	}

	return &models.MarketPrice{
		ExchangeName: resp.Exchange,
		Symbol:       resp.Ticker.Symbol,
		Price:        resp.Ticker.Last,
		Volume:       resp.Ticker.Volume,
		Timestamp:    resp.Ticker.Timestamp,
	}, nil
}

// FetchOrderBook fetches order book data for a specific exchange and symbol
func (s *Service) FetchOrderBook(ctx context.Context, exchange, symbol string, limit int) (*OrderBookResponse, error) {
	return s.client.GetOrderBook(ctx, exchange, symbol, limit)
}

// FetchOHLCV fetches OHLCV data for technical analysis
func (s *Service) FetchOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error) {
	return s.client.GetOHLCV(ctx, exchange, symbol, timeframe, limit)
}

// FetchTrades fetches recent trades for a specific exchange and symbol
func (s *Service) FetchTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error) {
	return s.client.GetTrades(ctx, exchange, symbol, limit)
}

// FetchMarkets fetches all available trading pairs for an exchange
func (s *Service) FetchMarkets(ctx context.Context, exchange string) (*MarketsResponse, error) {
	return s.client.GetMarkets(ctx, exchange)
}

// CalculateArbitrageOpportunities identifies arbitrage opportunities from market data
// This function takes ticker data with bid/ask prices to find arbitrage opportunities
func (s *Service) CalculateArbitrageOpportunities(ctx context.Context, exchanges []string, symbols []string, minProfitPercent decimal.Decimal) ([]models.ArbitrageOpportunityResponse, error) {
	if len(exchanges) == 0 || len(symbols) == 0 {
		return nil, fmt.Errorf("exchanges and symbols cannot be empty")
	}

	// Fetch detailed ticker data with bid/ask prices
	req := &TickersRequest{
		Symbols:   symbols,
		Exchanges: exchanges,
	}

	resp, err := s.client.GetTickers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tickers for arbitrage calculation: %w", err)
	}

	// Group ticker data by symbol
	symbolData := make(map[string][]TickerData)
	for _, tickerData := range resp.Tickers {
		symbol := tickerData.Ticker.Symbol
		symbolData[symbol] = append(symbolData[symbol], tickerData)
	}

	var opportunities []models.ArbitrageOpportunityResponse

	// Find arbitrage opportunities for each symbol
	for symbol, data := range symbolData {
		if len(data) < 2 {
			continue // Need at least 2 exchanges for arbitrage
		}

		// Find the lowest ask and highest bid
		var lowestAsk, highestBid TickerData
		lowestAskPrice := decimal.NewFromFloat(1e10) // Large initial value
		highestBidPrice := decimal.Zero

		for _, td := range data {
			if td.Ticker.Ask.GreaterThan(decimal.Zero) && td.Ticker.Ask.LessThan(lowestAskPrice) {
				lowestAsk = td
				lowestAskPrice = td.Ticker.Ask
			}
			if td.Ticker.Bid.GreaterThan(highestBidPrice) {
				highestBid = td
				highestBidPrice = td.Ticker.Bid
			}
		}

		// Skip if we don't have valid bid/ask data or same exchange
		if lowestAsk.Exchange == "" || highestBid.Exchange == "" ||
			lowestAsk.Exchange == highestBid.Exchange {
			continue
		}

		// Calculate profit percentage
		if lowestAskPrice.GreaterThan(decimal.Zero) && highestBidPrice.GreaterThan(lowestAskPrice) {
			profitAmount := highestBidPrice.Sub(lowestAskPrice)
			profitPercent := profitAmount.Div(lowestAskPrice).Mul(decimal.NewFromInt(100))

			if profitPercent.GreaterThanOrEqual(minProfitPercent) {
				opportunity := models.ArbitrageOpportunityResponse{
					Symbol:           symbol,
					BuyExchange:      lowestAsk.Exchange,
					SellExchange:     highestBid.Exchange,
					BuyPrice:         lowestAskPrice,
					SellPrice:        highestBidPrice,
					ProfitPercentage: profitPercent,
					DetectedAt:       time.Now(),
					ExpiresAt:        time.Now().Add(5 * time.Minute), // 5-minute window
				}
				opportunities = append(opportunities, opportunity)
			}
		}
	}

	return opportunities, nil
}

// Close closes the CCXT service
func (s *Service) Close() error {
	return s.client.Close()
}