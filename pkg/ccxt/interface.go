package ccxt

import (
	"context"
	"time"

	"github.com/irfandi/celebrum-ai-go/pkg/interfaces"
	"github.com/shopspring/decimal"
)

// MarketPriceInterface defines the interface for market price data
type MarketPriceInterface interface {
	GetPrice() float64
	GetVolume() float64
	GetTimestamp() time.Time
	GetExchangeName() string
	GetSymbol() string
}

// ArbitrageOpportunityInterface defines the interface for arbitrage opportunity data
type ArbitrageOpportunityInterface interface {
	GetSymbol() string
	GetBuyExchange() string
	GetSellExchange() string
	GetBuyPrice() decimal.Decimal
	GetSellPrice() decimal.Decimal
	GetProfitPercentage() decimal.Decimal
	GetDetectedAt() time.Time
	GetExpiresAt() time.Time
}

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

	// Exchange management
	GetExchangeConfig(ctx context.Context) (*ExchangeConfigResponse, error)
	AddExchangeToBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error)
	RemoveExchangeFromBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error)
	RefreshExchanges(ctx context.Context) (*ExchangeManagementResponse, error)
	AddExchange(ctx context.Context, exchange string) (*ExchangeManagementResponse, error)

	// Market data operations
	FetchMarketData(ctx context.Context, exchanges []string, symbols []string) ([]interfaces.MarketPriceInterface, error)
	FetchSingleTicker(ctx context.Context, exchange, symbol string) (interfaces.MarketPriceInterface, error)
	FetchOrderBook(ctx context.Context, exchange, symbol string, limit int) (*OrderBookResponse, error)
	FetchOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error)
	FetchTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error)
	FetchMarkets(ctx context.Context, exchange string) (*MarketsResponse, error)

	// Funding rate operations
	FetchFundingRate(ctx context.Context, exchange, symbol string) (*FundingRate, error)
	FetchFundingRates(ctx context.Context, exchange string, symbols []string) ([]FundingRate, error)
	FetchAllFundingRates(ctx context.Context, exchange string) ([]FundingRate, error)

	// Arbitrage operations
	CalculateArbitrageOpportunities(ctx context.Context, exchanges []string, symbols []string, minProfitPercent decimal.Decimal) ([]interfaces.ArbitrageOpportunityInterface, error)
	CalculateFundingRateArbitrage(ctx context.Context, symbols []string, exchanges []string, minProfit float64) ([]FundingArbitrageOpportunity, error)
}

// CCXTClient defines the interface for low-level CCXT HTTP operations
type CCXTClient interface {
	// Health and status
	HealthCheck(ctx context.Context) (*HealthResponse, error)

	// Exchange operations
	GetExchanges(ctx context.Context) (*ExchangesResponse, error)

	// Exchange management operations
	GetExchangeConfig(ctx context.Context) (*ExchangeConfigResponse, error)
	AddExchangeToBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error)
	RemoveExchangeFromBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error)
	RefreshExchanges(ctx context.Context) (*ExchangeManagementResponse, error)
	AddExchange(ctx context.Context, exchange string) (*ExchangeManagementResponse, error)

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
