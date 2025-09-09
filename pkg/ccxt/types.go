package ccxt

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/irfndi/celebrum-ai-go/pkg/interfaces"
		"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// Service provides high-level CCXT operations
type Service struct {
	client             *Client
	supportedExchanges map[string]ExchangeInfo
	blacklistCache     interfaces.BlacklistCache
	mu                 sync.RWMutex
	lastUpdate         time.Time
	logger             *logrus.Logger
}

// NewService creates a new CCXT service instance
func NewService(cfg *interfaces.CCXTConfig, logger *logrus.Logger, blacklistCache interfaces.BlacklistCache) *Service {
	s := &Service{
		client:             NewClient(cfg),
		supportedExchanges: make(map[string]ExchangeInfo),
		blacklistCache:     blacklistCache,
		logger:             logger,
	}

	return s
}

// Initialize initializes the service and loads supported exchanges
func (s *Service) Initialize(ctx context.Context) error {
	exchanges, err := s.client.GetExchanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to get exchanges: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Load supported exchanges
	for _, exchange := range exchanges.Exchanges {
		s.supportedExchanges[exchange.ID] = exchange
	}

	s.lastUpdate = time.Now()
	s.logger.Infof("Initialized %d exchanges", len(exchanges.Exchanges))

	return nil
}

// GetSupportedExchanges returns the list of supported exchange IDs
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

// FetchSingleTicker fetches ticker data for a single exchange and symbol
func (s *Service) FetchSingleTicker(ctx context.Context, exchange, symbol string) (interfaces.MarketPriceInterface, error) {
	// Check if symbol is blacklisted
	if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
		return nil, fmt.Errorf("symbol %s is blacklisted: %s", symbol, reason)
	}

	// Check if exchange is supported
	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	ticker, err := s.client.GetTicker(ctx, exchange, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticker from %s: %w", exchange, err)
	}

	// Convert to our MarketPrice model
	marketPrice := &interfaces.MarketPrice{
		ExchangeName: exchange,
		Symbol:       symbol,
		Price:        ticker.Ticker.Last,
		Volume:       ticker.Ticker.Volume,
		Timestamp:    ticker.Ticker.Timestamp.Time(),
	}

	return marketPrice, nil
}

// FetchMarketData fetches market data for multiple exchanges and symbols
func (s *Service) FetchMarketData(ctx context.Context, exchanges, symbols []string) ([]interfaces.MarketPriceInterface, error) {
	if len(exchanges) == 0 || len(symbols) == 0 {
		return nil, fmt.Errorf("exchanges and symbols cannot be empty")
	}

	var marketData []interfaces.MarketPriceInterface

	for _, exchange := range exchanges {
		// Check if exchange is supported
		if _, exists := s.GetExchangeInfo(exchange); !exists {
			s.logger.Warnf("Exchange %s is not supported, skipping", exchange)
			continue
		}

		for _, symbol := range symbols {
			// Check if symbol is blacklisted
			if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
				s.logger.Debugf("Symbol %s is blacklisted: %s", symbol, reason)
				continue
			}

			ticker, err := s.client.GetTicker(ctx, exchange, symbol)
			if err != nil {
				s.logger.Errorf("Failed to fetch ticker from %s for %s: %v", exchange, symbol, err)
				continue
			}

			// Convert to our MarketPrice model
			marketPrice := &interfaces.MarketPrice{
				ExchangeName: exchange,
				Symbol:       symbol,
				Price:        ticker.Ticker.Last,
				Volume:       ticker.Ticker.Volume,
				Timestamp:    ticker.Ticker.Timestamp.Time(),
			}

			marketData = append(marketData, marketPrice)
		}
	}

	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data fetched for exchanges %v and symbols %v", exchanges, symbols)
	}

	return marketData, nil
}

// CalculateArbitrageOpportunities calculates arbitrage opportunities across exchanges
func (s *Service) CalculateArbitrageOpportunities(ctx context.Context, exchanges, symbols []string, minProfit decimal.Decimal) ([]interfaces.ArbitrageOpportunityInterface, error) {
	if len(exchanges) == 0 || len(symbols) == 0 {
		return nil, fmt.Errorf("exchanges and symbols cannot be empty")
	}

	// Fetch market data for all exchanges and symbols
	marketData, err := s.FetchMarketData(ctx, exchanges, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data: %w", err)
	}

	// Group market data by symbol
	symbolPrices := make(map[string][]interfaces.MarketPriceInterface)
	for _, data := range marketData {
		symbolPrices[data.GetSymbol()] = append(symbolPrices[data.GetSymbol()], data)
	}

	var opportunities []interfaces.ArbitrageOpportunityInterface

	// Calculate arbitrage opportunities for each symbol
	for symbol, prices := range symbolPrices {
		if len(prices) < 2 {
			continue // Need at least 2 exchanges for arbitrage
		}

		// Find the lowest and highest prices
		var lowestPrice, highestPrice interfaces.MarketPriceInterface
		for _, price := range prices {
			if lowestPrice == nil || decimal.NewFromFloat(price.GetPrice()).Cmp(decimal.NewFromFloat(lowestPrice.GetPrice())) < 0 {
				lowestPrice = price
			}
			if highestPrice == nil || decimal.NewFromFloat(price.GetPrice()).Cmp(decimal.NewFromFloat(highestPrice.GetPrice())) > 0 {
				highestPrice = price
			}
		}

		// Calculate profit percentage
		if lowestPrice.GetExchangeName() == highestPrice.GetExchangeName() {
			continue // Skip if same exchange
		}

		lowestPriceDecimal := decimal.NewFromFloat(lowestPrice.GetPrice())
		highestPriceDecimal := decimal.NewFromFloat(highestPrice.GetPrice())
		profitPercentage := highestPriceDecimal.Sub(lowestPriceDecimal).Div(lowestPriceDecimal).Mul(decimal.NewFromInt(100))

		if profitPercentage.Cmp(minProfit) >= 0 {
			opportunity := &interfaces.ArbitrageOpportunityResponse{
				Symbol:           symbol,
				BuyExchange:      lowestPrice.GetExchangeName(),
				SellExchange:     highestPrice.GetExchangeName(),
				BuyPrice:         lowestPriceDecimal,
				SellPrice:        highestPriceDecimal,
				ProfitPercentage: profitPercentage,
				DetectedAt:       time.Now(),
				ExpiresAt:        time.Now().Add(5 * time.Minute), // Opportunity expires in 5 minutes
			}

			opportunities = append(opportunities, opportunity)
		}
	}

	return opportunities, nil
}

// GetExchangeHealth checks the health of an exchange
func (s *Service) GetExchangeHealth(ctx context.Context, exchangeID string) (bool, error) {
	if _, exists := s.GetExchangeInfo(exchangeID); !exists {
		return false, fmt.Errorf("exchange %s is not supported", exchangeID)
	}

	// Try to fetch a common symbol like BTC/USDT
	_, err := s.FetchSingleTicker(ctx, exchangeID, "BTC/USDT")
	if err != nil {
		return false, err
	}

	return true, nil
}

// IsHealthy checks if the service is healthy
func (s *Service) IsHealthy(ctx context.Context) bool {
	// Try to fetch a common symbol like BTC/USDT from binance
	_, err := s.FetchSingleTicker(ctx, "binance", "BTC/USDT")
	return err == nil
}

// Close cleans up service resources
func (s *Service) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// GetServiceURL returns the service URL
func (s *Service) GetServiceURL() string {
	return s.client.BaseURL
}

// GetExchangeConfig gets the exchange configuration
func (s *Service) GetExchangeConfig(ctx context.Context) (*ExchangeConfigResponse, error) {
	return s.client.GetExchangeConfig(ctx)
}

// AddExchangeToBlacklist adds an exchange to the blacklist
func (s *Service) AddExchangeToBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error) {
	return s.client.AddExchangeToBlacklist(ctx, exchange)
}

// RemoveExchangeFromBlacklist removes an exchange from the blacklist
func (s *Service) RemoveExchangeFromBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error) {
	return s.client.RemoveExchangeFromBlacklist(ctx, exchange)
}

// RefreshExchanges refreshes all exchanges
func (s *Service) RefreshExchanges(ctx context.Context) (*ExchangeManagementResponse, error) {
	return s.client.RefreshExchanges(ctx)
}

// AddExchange adds a new exchange
func (s *Service) AddExchange(ctx context.Context, exchange string) (*ExchangeManagementResponse, error) {
	return s.client.AddExchange(ctx, exchange)
}

// FetchOrderBook fetches order book data
func (s *Service) FetchOrderBook(ctx context.Context, exchange, symbol string, limit int) (*OrderBookResponse, error) {
	if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
		return nil, fmt.Errorf("symbol %s is blacklisted: %s", symbol, reason)
	}

	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetOrderBook(ctx, exchange, symbol, limit)
}

// FetchOHLCV fetches OHLCV data
func (s *Service) FetchOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error) {
	if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
		return nil, fmt.Errorf("symbol %s is blacklisted: %s", symbol, reason)
	}

	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetOHLCV(ctx, exchange, symbol, timeframe, limit)
}

// FetchTrades fetches trade data
func (s *Service) FetchTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error) {
	if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
		return nil, fmt.Errorf("symbol %s is blacklisted: %s", symbol, reason)
	}

	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetTrades(ctx, exchange, symbol, limit)
}

// FetchMarkets fetches market data
func (s *Service) FetchMarkets(ctx context.Context, exchange string) (*MarketsResponse, error) {
	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetMarkets(ctx, exchange)
}

// FetchFundingRate fetches funding rate data
func (s *Service) FetchFundingRate(ctx context.Context, exchange, symbol string) (*FundingRate, error) {
	if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
		return nil, fmt.Errorf("symbol %s is blacklisted: %s", symbol, reason)
	}

	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetFundingRate(ctx, exchange, symbol)
}

// FetchFundingRates fetches multiple funding rates
func (s *Service) FetchFundingRates(ctx context.Context, exchange string, symbols []string) ([]FundingRate, error) {
	for _, symbol := range symbols {
		if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
			return nil, fmt.Errorf("symbol %s is blacklisted: %s", symbol, reason)
		}
	}

	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetFundingRates(ctx, exchange, symbols)
}

// FetchAllFundingRates fetches all funding rates
func (s *Service) FetchAllFundingRates(ctx context.Context, exchange string) ([]FundingRate, error) {
	if _, exists := s.GetExchangeInfo(exchange); !exists {
		return nil, fmt.Errorf("exchange %s is not supported", exchange)
	}

	return s.client.GetAllFundingRates(ctx, exchange)
}

// CalculateFundingRateArbitrage calculates funding rate arbitrage opportunities
func (s *Service) CalculateFundingRateArbitrage(ctx context.Context, symbols []string, exchanges []string, minProfit float64) ([]FundingArbitrageOpportunity, error) {
	if len(exchanges) == 0 || len(symbols) == 0 {
		return nil, fmt.Errorf("exchanges and symbols cannot be empty")
	}

	var opportunities []FundingArbitrageOpportunity

	// For each symbol, compare funding rates across exchanges
	for _, symbol := range symbols {
		if blacklisted, reason := s.blacklistCache.IsBlacklisted(symbol); blacklisted {
			s.logger.Debugf("Symbol %s is blacklisted: %s", symbol, reason)
			continue
		}

		var fundingRates []struct {
			exchange      string
			fundingRate   *FundingRate
		}

		// Collect funding rates from all exchanges for this symbol
		for _, exchange := range exchanges {
			if _, exists := s.GetExchangeInfo(exchange); !exists {
				s.logger.Warnf("Exchange %s is not supported, skipping", exchange)
				continue
			}

			rate, err := s.client.GetFundingRate(ctx, exchange, symbol)
			if err != nil {
				s.logger.Errorf("Failed to fetch funding rate from %s for %s: %v", exchange, symbol, err)
				continue
			}

			fundingRates = append(fundingRates, struct {
				exchange      string
				fundingRate   *FundingRate
			}{
				exchange:      exchange,
				fundingRate:   rate,
			})
		}

		// Calculate arbitrage opportunities if we have at least 2 exchanges
		if len(fundingRates) >= 2 {
			// Find the exchanges with the lowest and highest funding rates
			var lowest, highest *struct {
				exchange      string
				fundingRate   *FundingRate
			}

			for _, fr := range fundingRates {
				if lowest == nil || fr.fundingRate.FundingRate < lowest.fundingRate.FundingRate {
					lowest = &fr
				}
				if highest == nil || fr.fundingRate.FundingRate > highest.fundingRate.FundingRate {
					highest = &fr
				}
			}

			if lowest != nil && highest != nil && lowest.exchange != highest.exchange {
				netRate := highest.fundingRate.FundingRate - lowest.fundingRate.FundingRate
				estimatedProfitDaily := netRate * 3 // 3 funding periods per day
				estimatedProfitPercentage := (estimatedProfitDaily / 100) * 100 // Convert to percentage

				if estimatedProfitPercentage >= minProfit {
					opportunity := FundingArbitrageOpportunity{
						Symbol:                    symbol,
						LongExchange:              lowest.exchange,
						ShortExchange:             highest.exchange,
						LongFundingRate:           lowest.fundingRate.FundingRate,
						ShortFundingRate:          highest.fundingRate.FundingRate,
						NetFundingRate:            netRate,
						EstimatedProfit8h:         netRate,
						EstimatedProfitDaily:      estimatedProfitDaily,
						EstimatedProfitPercentage: estimatedProfitPercentage,
						LongMarkPrice:             lowest.fundingRate.MarkPrice,
						ShortMarkPrice:            highest.fundingRate.MarkPrice,
						PriceDifference:           highest.fundingRate.MarkPrice - lowest.fundingRate.MarkPrice,
						PriceDifferencePercentage: ((highest.fundingRate.MarkPrice - lowest.fundingRate.MarkPrice) / lowest.fundingRate.MarkPrice) * 100,
						RiskScore:                 0.5, // Default risk score
						Timestamp:                 highest.fundingRate.Timestamp,
					}

					opportunities = append(opportunities, opportunity)
				}
			}
		}
	}

	return opportunities, nil
}