package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// FuturesArbitrageOpportunity represents a detected opportunity for arbitrage using futures contracts.
// It leverages funding rate differences between exchanges to generate profit.
type FuturesArbitrageOpportunity struct {
	ID              string `json:"id" db:"id"`
	Symbol          string `json:"symbol"`
	BaseCurrency    string `json:"base_currency"`
	QuoteCurrency   string `json:"quote_currency"`
	LongExchange    string `json:"long_exchange"`
	ShortExchange   string `json:"short_exchange"`
	LongExchangeID  int    `json:"long_exchange_id" db:"long_exchange_id"`
	ShortExchangeID int    `json:"short_exchange_id" db:"short_exchange_id"`

	// Funding Rate Data
	LongFundingRate  decimal.Decimal `json:"long_funding_rate"`
	ShortFundingRate decimal.Decimal `json:"short_funding_rate"`
	NetFundingRate   decimal.Decimal `json:"net_funding_rate"`
	FundingInterval  int             `json:"funding_interval"` // Hours between funding payments (usually 8)

	// Price Data
	LongMarkPrice             decimal.Decimal `json:"long_mark_price"`
	ShortMarkPrice            decimal.Decimal `json:"short_mark_price"`
	PriceDifference           decimal.Decimal `json:"price_difference"`
	PriceDifferencePercentage decimal.Decimal `json:"price_difference_percentage"`

	// Profit Calculations
	HourlyRate             decimal.Decimal `json:"hourly_rate"`
	DailyRate              decimal.Decimal `json:"daily_rate"`
	APY                    decimal.Decimal `json:"apy"`
	EstimatedProfit8h      decimal.Decimal `json:"estimated_profit_8h"`
	EstimatedProfitDaily   decimal.Decimal `json:"estimated_profit_daily"`
	EstimatedProfitWeekly  decimal.Decimal `json:"estimated_profit_weekly"`
	EstimatedProfitMonthly decimal.Decimal `json:"estimated_profit_monthly"`

	// Risk Management
	RiskScore               decimal.Decimal `json:"risk_score"`
	VolatilityScore         decimal.Decimal `json:"volatility_score"`
	LiquidityScore          decimal.Decimal `json:"liquidity_score"`
	RecommendedPositionSize decimal.Decimal `json:"recommended_position_size"`
	MaxLeverage             decimal.Decimal `json:"max_leverage"`
	RecommendedLeverage     decimal.Decimal `json:"recommended_leverage"`
	StopLossPercentage      decimal.Decimal `json:"stop_loss_percentage"`

	// Position Sizing Recommendations
	MinPositionSize     decimal.Decimal `json:"min_position_size"`
	MaxPositionSize     decimal.Decimal `json:"max_position_size"`
	OptimalPositionSize decimal.Decimal `json:"optimal_position_size"`

	// Timing and Validity
	DetectedAt        time.Time `json:"detected_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	NextFundingTime   time.Time `json:"next_funding_time"`
	TimeToNextFunding int       `json:"time_to_next_funding"` // Minutes
	IsActive          bool      `json:"is_active"`

	// Market Conditions
	MarketTrend        string                    `json:"market_trend"` // "bullish", "bearish", "neutral"
	Volume24h          decimal.Decimal           `json:"volume_24h"`
	OpenInterest       decimal.Decimal           `json:"open_interest"`
	FundingRateHistory []FundingRateHistoryPoint `json:"funding_rate_history,omitempty"`
}

// FundingRateHistoryPoint captures a snapshot of the funding rate at a specific time.
type FundingRateHistoryPoint struct {
	Timestamp   time.Time       `json:"timestamp"`
	FundingRate decimal.Decimal `json:"funding_rate"`
	MarkPrice   decimal.Decimal `json:"mark_price"`
}

// FuturesArbitrageCalculationInput defines the parameters required to calculate potential futures arbitrage profits and risks.
type FuturesArbitrageCalculationInput struct {
	Symbol             string          `json:"symbol"`
	LongExchange       string          `json:"long_exchange"`
	ShortExchange      string          `json:"short_exchange"`
	LongFundingRate    decimal.Decimal `json:"long_funding_rate"`
	ShortFundingRate   decimal.Decimal `json:"short_funding_rate"`
	LongMarkPrice      decimal.Decimal `json:"long_mark_price"`
	ShortMarkPrice     decimal.Decimal `json:"short_mark_price"`
	BaseAmount         decimal.Decimal `json:"base_amount"`         // Amount to calculate profits for
	UserRiskTolerance  string          `json:"user_risk_tolerance"` // "low", "medium", "high"
	MaxLeverageAllowed decimal.Decimal `json:"max_leverage_allowed"`
	AvailableCapital   decimal.Decimal `json:"available_capital"`
	FundingInterval    int             `json:"funding_interval"` // Hours
}

// FuturesArbitrageRiskMetrics encapsulates various risk factors associated with a futures arbitrage opportunity.
type FuturesArbitrageRiskMetrics struct {
	// Price Risk
	PriceCorrelation decimal.Decimal `json:"price_correlation"`
	PriceVolatility  decimal.Decimal `json:"price_volatility"`
	MaxDrawdown      decimal.Decimal `json:"max_drawdown"`

	// Funding Rate Risk
	FundingRateVolatility decimal.Decimal `json:"funding_rate_volatility"`
	FundingRateStability  decimal.Decimal `json:"funding_rate_stability"`

	// Liquidity Risk
	BidAskSpread decimal.Decimal `json:"bid_ask_spread"`
	MarketDepth  decimal.Decimal `json:"market_depth"`
	SlippageRisk decimal.Decimal `json:"slippage_risk"`

	// Exchange Risk
	ExchangeReliability decimal.Decimal `json:"exchange_reliability"`
	CounterpartyRisk    decimal.Decimal `json:"counterparty_risk"`

	// Overall Risk Score (0-100)
	OverallRiskScore decimal.Decimal `json:"overall_risk_score"`
	RiskCategory     string          `json:"risk_category"` // "low", "medium", "high", "extreme"
	Recommendation   string          `json:"recommendation"`
}

// FuturesPositionSizing provides recommendations for position sizes and leverage based on risk analysis.
type FuturesPositionSizing struct {
	// Kelly Criterion based sizing
	KellyPercentage   decimal.Decimal `json:"kelly_percentage"`
	KellyPositionSize decimal.Decimal `json:"kelly_position_size"`

	// Risk-based sizing
	ConservativeSize decimal.Decimal `json:"conservative_size"`
	ModerateSize     decimal.Decimal `json:"moderate_size"`
	AggressiveSize   decimal.Decimal `json:"aggressive_size"`

	// Leverage recommendations
	MinLeverage     decimal.Decimal `json:"min_leverage"`
	OptimalLeverage decimal.Decimal `json:"optimal_leverage"`
	MaxSafeLeverage decimal.Decimal `json:"max_safe_leverage"`

	// Risk management
	StopLossPrice     decimal.Decimal `json:"stop_loss_price"`
	TakeProfitPrice   decimal.Decimal `json:"take_profit_price"`
	MaxLossPercentage decimal.Decimal `json:"max_loss_percentage"`
}

// FuturesArbitrageStrategy defines a complete strategy for executing a futures arbitrage trade.
// It includes the opportunity details, risk assessment, position sizing, and execution plan.
type FuturesArbitrageStrategy struct {
	ID             string                      `json:"id"`
	Name           string                      `json:"name"`
	Description    string                      `json:"description"`
	Opportunity    FuturesArbitrageOpportunity `json:"opportunity"`
	RiskMetrics    FuturesArbitrageRiskMetrics `json:"risk_metrics"`
	PositionSizing FuturesPositionSizing       `json:"position_sizing"`

	// Execution Plan
	LongPosition           PositionDetails `json:"long_position"`
	ShortPosition          PositionDetails `json:"short_position"`
	ExecutionOrder         []string        `json:"execution_order"`
	EstimatedExecutionTime int             `json:"estimated_execution_time"` // Seconds

	// Performance Tracking
	ExpectedReturn      decimal.Decimal `json:"expected_return"`
	SharpeRatio         decimal.Decimal `json:"sharpe_ratio"`
	MaxDrawdownExpected decimal.Decimal `json:"max_drawdown_expected"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`
}

// PositionDetails describes the parameters for a single side (long or short) of an arbitrage trade.
type PositionDetails struct {
	Exchange       string          `json:"exchange"`
	Symbol         string          `json:"symbol"`
	Side           string          `json:"side"` // "long" or "short"
	Size           decimal.Decimal `json:"size"`
	Leverage       decimal.Decimal `json:"leverage"`
	EntryPrice     decimal.Decimal `json:"entry_price"`
	StopLoss       decimal.Decimal `json:"stop_loss"`
	TakeProfit     decimal.Decimal `json:"take_profit"`
	MarginRequired decimal.Decimal `json:"margin_required"`
	EstimatedFees  decimal.Decimal `json:"estimated_fees"`
}

// FuturesArbitrageRequest represents the parameters for querying or calculating futures arbitrage opportunities.
type FuturesArbitrageRequest struct {
	Symbols               []string        `json:"symbols,omitempty"`
	Exchanges             []string        `json:"exchanges,omitempty"`
	MinAPY                decimal.Decimal `json:"min_apy,omitempty"`
	MaxRiskScore          decimal.Decimal `json:"max_risk_score,omitempty"`
	RiskTolerance         string          `json:"risk_tolerance,omitempty"` // "low", "medium", "high"
	AvailableCapital      decimal.Decimal `json:"available_capital,omitempty"`
	MaxLeverage           decimal.Decimal `json:"max_leverage,omitempty"`
	TimeHorizon           string          `json:"time_horizon,omitempty"` // "short", "medium", "long"
	IncludeRiskMetrics    bool            `json:"include_risk_metrics,omitempty"`
	IncludePositionSizing bool            `json:"include_position_sizing,omitempty"`
	Limit                 int             `json:"limit,omitempty"`
	Page                  int             `json:"page,omitempty"`
}

// FuturesArbitrageResponse contains the list of found opportunities and strategies, along with pagination and summary data.
type FuturesArbitrageResponse struct {
	Opportunities []FuturesArbitrageOpportunity `json:"opportunities"`
	Strategies    []FuturesArbitrageStrategy    `json:"strategies,omitempty"`
	Count         int                           `json:"count"`
	TotalCount    int                           `json:"total_count"`
	Page          int                           `json:"page"`
	Limit         int                           `json:"limit"`
	Timestamp     time.Time                     `json:"timestamp"`
	MarketSummary FuturesMarketSummary          `json:"market_summary"`
}

// FuturesMarketSummary provides a high-level overview of the current futures market conditions.
type FuturesMarketSummary struct {
	TotalOpportunities  int             `json:"total_opportunities"`
	AverageAPY          decimal.Decimal `json:"average_apy"`
	HighestAPY          decimal.Decimal `json:"highest_apy"`
	AverageRiskScore    decimal.Decimal `json:"average_risk_score"`
	MarketVolatility    decimal.Decimal `json:"market_volatility"`
	FundingRateTrend    string          `json:"funding_rate_trend"` // "increasing", "decreasing", "stable"
	RecommendedStrategy string          `json:"recommended_strategy"`
	MarketCondition     string          `json:"market_condition"` // "favorable", "neutral", "unfavorable"
}
