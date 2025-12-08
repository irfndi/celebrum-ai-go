package interfaces

import (
	"time"

	"github.com/shopspring/decimal"
)

// ArbitrageOpportunityResponse represents an arbitrage opportunity detected between two exchanges.
// It contains all the necessary details to execute or analyze the arbitrage trade, including
// prices, profit margins, and timestamps.
type ArbitrageOpportunityResponse struct {
	// Symbol is the trading pair symbol (e.g., "BTC/USD").
	Symbol           string          `json:"symbol"`
	// BuyExchange is the name of the exchange where the asset is bought.
	BuyExchange      string          `json:"buy_exchange"`
	// SellExchange is the name of the exchange where the asset is sold.
	SellExchange     string          `json:"sell_exchange"`
	// BuyPrice is the price at which the asset can be bought on the BuyExchange.
	BuyPrice         decimal.Decimal `json:"buy_price"`
	// SellPrice is the price at which the asset can be sold on the SellExchange.
	SellPrice        decimal.Decimal `json:"sell_price"`
	// ProfitPercentage is the potential profit percentage from the arbitrage trade.
	ProfitPercentage decimal.Decimal `json:"profit_percentage"`
	// DetectedAt is the timestamp when the arbitrage opportunity was first detected.
	DetectedAt       time.Time       `json:"detected_at"`
	// ExpiresAt is the timestamp when the opportunity is expected to no longer be valid.
	ExpiresAt        time.Time       `json:"expires_at"`
}

// ArbitrageOpportunityInterface defines the contract for accessing arbitrage opportunity data.
// Any type implementing this interface can be used as an arbitrage opportunity source.
type ArbitrageOpportunityInterface interface {
	// GetSymbol retrieves the trading pair symbol associated with the arbitrage opportunity.
	//
	// Returns:
	//   string: The trading pair symbol.
	GetSymbol() string

	// GetBuyExchange retrieves the name of the exchange where the asset should be bought.
	//
	// Returns:
	//   string: The name of the buying exchange.
	GetBuyExchange() string

	// GetSellExchange retrieves the name of the exchange where the asset should be sold.
	//
	// Returns:
	//   string: The name of the selling exchange.
	GetSellExchange() string

	// GetBuyPrice retrieves the buying price of the asset.
	//
	// Returns:
	//   decimal.Decimal: The buy price.
	GetBuyPrice() decimal.Decimal

	// GetSellPrice retrieves the selling price of the asset.
	//
	// Returns:
	//   decimal.Decimal: The sell price.
	GetSellPrice() decimal.Decimal

	// GetProfitPercentage retrieves the calculated profit percentage for the opportunity.
	//
	// Returns:
	//   decimal.Decimal: The profit percentage.
	GetProfitPercentage() decimal.Decimal

	// GetDetectedAt retrieves the time when the opportunity was detected.
	//
	// Returns:
	//   time.Time: The detection timestamp.
	GetDetectedAt() time.Time

	// GetExpiresAt retrieves the estimated expiration time of the opportunity.
	//
	// Returns:
	//   time.Time: The expiration timestamp.
	GetExpiresAt() time.Time
}

// GetSymbol returns the trading pair symbol for this arbitrage opportunity.
//
// Returns:
//   string: The symbol (e.g., "BTC/USDT").
func (ao *ArbitrageOpportunityResponse) GetSymbol() string {
	return ao.Symbol
}

// GetBuyExchange returns the name of the exchange where the asset is to be bought.
//
// Returns:
//   string: The buy exchange name.
func (ao *ArbitrageOpportunityResponse) GetBuyExchange() string {
	return ao.BuyExchange
}

// GetSellExchange returns the name of the exchange where the asset is to be sold.
//
// Returns:
//   string: The sell exchange name.
func (ao *ArbitrageOpportunityResponse) GetSellExchange() string {
	return ao.SellExchange
}

// GetBuyPrice returns the price at which the asset is bought.
//
// Returns:
//   decimal.Decimal: The buy price.
func (ao *ArbitrageOpportunityResponse) GetBuyPrice() decimal.Decimal {
	return ao.BuyPrice
}

// GetSellPrice returns the price at which the asset is sold.
//
// Returns:
//   decimal.Decimal: The sell price.
func (ao *ArbitrageOpportunityResponse) GetSellPrice() decimal.Decimal {
	return ao.SellPrice
}

// GetProfitPercentage returns the calculated profit percentage.
//
// Returns:
//   decimal.Decimal: The profit percentage.
func (ao *ArbitrageOpportunityResponse) GetProfitPercentage() decimal.Decimal {
	return ao.ProfitPercentage
}

// GetDetectedAt returns the timestamp when the opportunity was detected.
//
// Returns:
//   time.Time: The detection time.
func (ao *ArbitrageOpportunityResponse) GetDetectedAt() time.Time {
	return ao.DetectedAt
}

// GetExpiresAt returns the timestamp when the opportunity expires.
//
// Returns:
//   time.Time: The expiration time.
func (ao *ArbitrageOpportunityResponse) GetExpiresAt() time.Time {
	return ao.ExpiresAt
}
