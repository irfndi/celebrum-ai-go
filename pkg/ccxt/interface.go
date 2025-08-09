package ccxt

import (
	"context"

	"github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/shopspring/decimal"
)

// CCXTService defines the interface for CCXT operations
type CCXTService interface {
	// Service lifecycle
	Initialize(ctx context.Context) error
	IsHealthy(ctx context.Context) bool
	Close() error
	GetServiceURL() string

	// Exchange information
	GetSupportedExchanges() []string
	GetExchangeInfo(exchangeID string) (ExchangeInfo, bool)

	// Market data operations
	FetchMarketData(ctx context.Context, exchanges []string, symbols []string) ([]models.MarketPrice, error)
	FetchSingleTicker(ctx context.Context, exchange, symbol string) (*models.MarketPrice, error)
	FetchOrderBook(ctx context.Context, exchange, symbol string, limit int) (*OrderBookResponse, error)
	FetchOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error)
	FetchTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error)
	FetchMarkets(ctx context.Context, exchange string) (*MarketsResponse, error)

	// Funding rate operations
	FetchFundingRate(ctx context.Context, exchange, symbol string) (*FundingRate, error)
	FetchFundingRates(ctx context.Context, exchange string, symbols []string) ([]FundingRate, error)
	FetchAllFundingRates(ctx context.Context, exchange string) ([]FundingRate, error)

	// Arbitrage operations
	CalculateArbitrageOpportunities(ctx context.Context, exchanges []string, symbols []string, minProfitPercent decimal.Decimal) ([]models.ArbitrageOpportunityResponse, error)
	CalculateFundingRateArbitrage(ctx context.Context, symbols []string, exchanges []string, minProfit float64) ([]FundingArbitrageOpportunity, error)
}

// CCXTClient defines the interface for low-level CCXT HTTP operations
type CCXTClient interface {
	// Health and status
	HealthCheck(ctx context.Context) (*HealthResponse, error)

	// Exchange operations
	GetExchanges(ctx context.Context) (*ExchangesResponse, error)

	// Market data operations
	GetTicker(ctx context.Context, exchange, symbol string) (*TickerResponse, error)
	GetTickers(ctx context.Context, req *TickersRequest) (*TickersResponse, error)
	GetOrderBook(ctx context.Context, exchange, symbol string, limit int) (*OrderBookResponse, error)
	GetTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error)
	GetOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error)
	GetMarkets(ctx context.Context, exchange string) (*MarketsResponse, error)

	// Lifecycle
	Close() error
}

// Ensure our implementations satisfy the interfaces
var (
	_ CCXTService = (*Service)(nil)
	_ CCXTClient  = (*Client)(nil)
)
