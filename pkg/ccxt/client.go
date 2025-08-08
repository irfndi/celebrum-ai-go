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

	"github.com/irfndi/celebrum-ai-go/internal/config"
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
	path := fmt.Sprintf("/api/ticker/%s/%s", exchange, symbol)
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
	path := fmt.Sprintf("/api/orderbook/%s/%s", exchange, symbol)
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var response OrderBookResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetTrades retrieves recent trades for a specific exchange and symbol
func (c *Client) GetTrades(ctx context.Context, exchange, symbol string, limit int) (*TradesResponse, error) {
	path := fmt.Sprintf("/api/trades/%s/%s", exchange, symbol)
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}
	var response TradesResponse
	err := c.makeRequest(ctx, "GET", path, nil, &response)
	return &response, err
}

// GetOHLCV retrieves OHLCV data for a specific exchange and symbol
func (c *Client) GetOHLCV(ctx context.Context, exchange, symbol, timeframe string, limit int) (*OHLCVResponse, error) {
	path := fmt.Sprintf("/api/ohlcv/%s/%s", exchange, symbol)
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
