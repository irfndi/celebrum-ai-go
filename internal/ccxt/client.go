package ccxt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/config"
)

// Client represents the CCXT HTTP client
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	timeout    time.Duration
}

// NewClient creates a new CCXT client instance
func NewClient(cfg *config.CCXTConfig) *Client {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		BaseURL: strings.TrimSuffix(cfg.ServiceURL, "/"),
		timeout: timeout,
	}
}

// HealthCheck checks if the CCXT service is healthy
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	var response HealthResponse
	err := c.makeRequest(ctx, "GET", "/health", nil, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetExchanges retrieves all supported exchanges
func (c *Client) GetExchanges(ctx context.Context) (*ExchangesResponse, error) {
	var response ExchangesResponse
	err := c.makeRequest(ctx, "GET", "/api/exchanges", nil, &response)
	return &response, err
}

// GetTicker retrieves ticker data for a specific exchange and symbol
func (c *Client) GetTicker(ctx context.Context, exchange, symbol string) (*TickerResponse, error) {
	// Convert symbol format based on exchange requirements
	ccxtSymbol := c.formatSymbolForExchange(exchange, symbol)
	path := fmt.Sprintf("/api/ticker/%s/%s", exchange, ccxtSymbol)
	var response TickerResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetTickers retrieves multiple tickers in a single request
func (c *Client) GetTickers(ctx context.Context, req *TickersRequest) (*TickersResponse, error) {
	var response TickersResponse
	err := c.makeRequest(ctx, "POST", "/api/tickers", req, &response)
	return &response, err
}

// GetOrderBook retrieves order book data for a specific exchange and symbol
func (c *Client) GetOrderBook(ctx context.Context, exchange, symbol string, limit int) (*OrderBookResponse, error) {
	// Convert symbol format based on exchange requirements
	ccxtSymbol := c.formatSymbolForExchange(exchange, symbol)
	path := fmt.Sprintf("/api/orderbook/%s/%s", exchange, ccxtSymbol)
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var response OrderBookResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetTrades retrieves recent trades for a specific exchange and symbol
func (c *Client) GetTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error) {
	// Convert symbol format based on exchange requirements
	ccxtSymbol := c.formatSymbolForExchange(exchange, symbol)
	path := fmt.Sprintf("/api/trades/%s/%s", exchange, ccxtSymbol)
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var response TradesResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetOHLCV retrieves OHLCV data for a specific exchange and symbol
func (c *Client) GetOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error) {
	// Convert symbol format based on exchange requirements
	ccxtSymbol := c.formatSymbolForExchange(exchange, symbol)
	path := fmt.Sprintf("/api/ohlcv/%s/%s", exchange, ccxtSymbol)
	params := url.Values{}
	if timeframe != "" {
		params.Set("timeframe", timeframe)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	var response OHLCVResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetMarkets retrieves all trading pairs for a specific exchange
func (c *Client) GetMarkets(ctx context.Context, exchange string) (*MarketsResponse, error) {
	path := fmt.Sprintf("/api/markets/%s", exchange)
	var response MarketsResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetFundingRate retrieves funding rate for a specific symbol on an exchange
func (c *Client) GetFundingRate(ctx context.Context, exchange, symbol string) (*FundingRate, error) {
	ccxtSymbol := c.formatSymbolForExchange(exchange, symbol)
	path := fmt.Sprintf("/api/funding-rate/%s/%s", exchange, ccxtSymbol)
	var response FundingRate
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetFundingRates retrieves funding rates for multiple symbols on an exchange
func (c *Client) GetFundingRates(ctx context.Context, exchange string, symbols []string) ([]FundingRate, error) {
	if len(symbols) == 0 {
		return []FundingRate{}, nil
	}

	// Format symbols
	formattedSymbols := make([]string, len(symbols))
	for i, symbol := range symbols {
		formattedSymbols[i] = c.formatSymbolForExchange(exchange, symbol)
	}

	// Join symbols with comma
	symbolsParam := strings.Join(formattedSymbols, ",")
	path := fmt.Sprintf("/api/funding-rates/%s?symbols=%s", exchange, url.QueryEscape(symbolsParam))

	var response FundingRateResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, err
	}
	return response.FundingRates, nil
}

// GetAllFundingRates retrieves all available funding rates for an exchange
func (c *Client) GetAllFundingRates(ctx context.Context, exchange string) ([]FundingRate, error) {
	path := fmt.Sprintf("/api/funding-rates/%s", exchange)
	var response FundingRateResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	if err != nil {
		return nil, err
	}
	return response.FundingRates, nil
}

// Exchange Management Methods

// GetExchangeConfig retrieves the current exchange configuration
func (c *Client) GetExchangeConfig(ctx context.Context) (*ExchangeConfigResponse, error) {
	var response ExchangeConfigResponse
	err := c.makeRequest(ctx, "GET", "/api/admin/exchanges/config", nil, &response)
	return &response, err
}

// AddExchangeToBlacklist adds an exchange to the blacklist
func (c *Client) AddExchangeToBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error) {
	path := fmt.Sprintf("/api/admin/exchanges/blacklist/%s", exchange)
	var response ExchangeManagementResponse
	err := c.makeRequest(ctx, "POST", path, nil, &response)
	return &response, err
}

// RemoveExchangeFromBlacklist removes an exchange from the blacklist
func (c *Client) RemoveExchangeFromBlacklist(ctx context.Context, exchange string) (*ExchangeManagementResponse, error) {
	path := fmt.Sprintf("/api/admin/exchanges/blacklist/%s", exchange)
	var response ExchangeManagementResponse
	err := c.makeRequest(ctx, "DELETE", path, nil, &response)
	return &response, err
}

// RefreshExchanges refreshes all exchanges (re-initializes non-blacklisted exchanges)
func (c *Client) RefreshExchanges(ctx context.Context) (*ExchangeManagementResponse, error) {
	var response ExchangeManagementResponse
	err := c.makeRequest(ctx, "POST", "/api/admin/exchanges/refresh", nil, &response)
	return &response, err
}

// AddExchange dynamically adds a new exchange
func (c *Client) AddExchange(ctx context.Context, exchange string) (*ExchangeManagementResponse, error) {
	path := fmt.Sprintf("/api/admin/exchanges/add/%s", exchange)
	var response ExchangeManagementResponse
	err := c.makeRequest(ctx, "POST", path, nil, &response)
	return &response, err
}

// formatSymbolForExchange formats the symbol based on exchange requirements
func (c *Client) formatSymbolForExchange(exchange, symbol string) string {
	switch strings.ToLower(exchange) {
	case "kraken", "okx":
		// Kraken and OKX use URL-encoded slash format
		return url.QueryEscape(symbol)
	case "coinbase", "coinbasepro":
		// Coinbase uses dash format (BTC-USDT)
		return strings.ReplaceAll(symbol, "/", "-")
	default:
		// Most exchanges (like Binance) use concatenated format
		return strings.ReplaceAll(symbol, "/", "")
	}
}

// makeRequest is a helper method to make HTTP requests to the CCXT service
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Celebrum-AI-Go/1.0")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Error closing response body: %v", err)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			return fmt.Errorf("CCXT service error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return fmt.Errorf("CCXT service error (%d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// Close closes the HTTP client (if needed for cleanup)
func (c *Client) Close() error {
	// HTTP client doesn't need explicit closing, but this method
	// is provided for interface compatibility
	return nil
}
