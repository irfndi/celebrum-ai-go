# Celebrum AI Go Style Guide

This style guide defines coding standards and best practices for the Celebrum AI Go cryptocurrency arbitrage trading system.

## General Go Conventions

### Naming Conventions

- **Packages**: Use short, lowercase names without underscores
- **Functions**: Use camelCase, start with lowercase for private, uppercase for public
- **Variables**: Use camelCase, descriptive names
- **Constants**: Use camelCase or ALL_CAPS for package-level constants
- **Interfaces**: Use `-er` suffix when possible (e.g., `Trader`, `Exchanger`)

### Code Organization

- Keep functions under 50 lines when possible
- Group related functionality in the same package
- Use clear package boundaries (internal/, pkg/, cmd/)
- Separate business logic from infrastructure concerns

## Financial Trading System Specific Guidelines

### Precision and Accuracy

```go
// GOOD: Use decimal.Decimal for financial calculations
import "github.com/shopspring/decimal"

type Price struct {
    Value    decimal.Decimal `json:"value"`
    Currency string          `json:"currency"`
}

// BAD: Never use float64 for money
type BadPrice struct {
    Value    float64 `json:"value"` // Precision loss!
    Currency string  `json:"currency"`
}
```

### Error Handling

```go
// GOOD: Wrap errors with context
func (s *ExchangeService) GetBalance(ctx context.Context, exchange string) (*Balance, error) {
    balance, err := s.client.FetchBalance(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch balance from %s: %w", exchange, err)
    }
    return balance, nil
}

// GOOD: Use custom error types for business logic
type InsufficientFundsError struct {
    Required decimal.Decimal
    Available decimal.Decimal
}

func (e InsufficientFundsError) Error() string {
    return fmt.Sprintf("insufficient funds: required %s, available %s", 
        e.Required.String(), e.Available.String())
}
```

### Concurrency and Thread Safety

```go
// GOOD: Use mutex for shared state
type OrderBook struct {
    mu    sync.RWMutex
    bids  []Order
    asks  []Order
}

func (ob *OrderBook) GetBestBid() (Order, bool) {
    ob.mu.RLock()
    defer ob.mu.RUnlock()
    
    if len(ob.bids) == 0 {
        return Order{}, false
    }
    return ob.bids[0], true
}

// GOOD: Use channels for communication
type ArbitrageEngine struct {
    opportunities chan ArbitrageOpportunity
    shutdown      chan struct{}
}
```

### API Key and Secrets Management

```go
// GOOD: Never log sensitive data
func (c *ExchangeClient) authenticate(apiKey, secret string) error {
    // Log without exposing secrets
    log.Info("authenticating with exchange", "exchange", c.name)
    
    // Use the secrets...
    return nil
}

// BAD: Never do this
func badAuthenticate(apiKey, secret string) error {
    log.Info("auth", "key", apiKey, "secret", secret) // Exposed!
    return nil
}
```

### Rate Limiting

```go
// GOOD: Implement proper rate limiting
type RateLimiter struct {
    limiter *rate.Limiter
    burst   int
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    return rl.limiter.Wait(ctx)
}

func (c *ExchangeClient) makeRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit wait failed: %w", err)
    }
    
    return c.httpClient.Do(req)
}
```

## Testing Standards

### Unit Tests

```go
// GOOD: Test financial calculations with precision
func TestPriceCalculation(t *testing.T) {
    tests := []struct {
        name     string
        price1   decimal.Decimal
        price2   decimal.Decimal
        expected decimal.Decimal
    }{
        {
            name:     "basic addition",
            price1:   decimal.NewFromString("100.50"),
            price2:   decimal.NewFromString("50.25"),
            expected: decimal.NewFromString("150.75"),
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.price1.Add(tt.price2)
            assert.True(t, tt.expected.Equal(result), 
                "expected %s, got %s", tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
// GOOD: Use testify/suite for integration tests
type ExchangeIntegrationSuite struct {
    suite.Suite
    client *ExchangeClient
    server *httptest.Server
}

func (s *ExchangeIntegrationSuite) SetupTest() {
    s.server = httptest.NewServer(http.HandlerFunc(s.mockHandler))
    s.client = NewExchangeClient(s.server.URL, "test-key", "test-secret")
}
```

## Documentation Standards

### Function Documentation

```go
// CalculateArbitrageProfit calculates the potential profit from an arbitrage opportunity.
// It takes into account trading fees, slippage, and minimum profit thresholds.
//
// Parameters:
//   - buyPrice: The price to buy the asset on the source exchange
//   - sellPrice: The price to sell the asset on the target exchange
//   - amount: The amount of the asset to trade
//   - fees: The trading fees for both exchanges
//
// Returns the net profit after all costs, or an error if the calculation fails.
func CalculateArbitrageProfit(buyPrice, sellPrice, amount decimal.Decimal, fees TradingFees) (decimal.Decimal, error) {
    // Implementation...
}
```

### Package Documentation

```go
// Package arbitrage provides functionality for detecting and executing
// cryptocurrency arbitrage opportunities across multiple exchanges.
//
// The package includes:
//   - Real-time price monitoring
//   - Arbitrage opportunity detection
//   - Risk management and position sizing
//   - Execution engine with proper error handling
//
// Example usage:
//
//	engine := arbitrage.NewEngine(config)
//	engine.Start(ctx)
//
// For more details, see the README.md file.
package arbitrage
```

## Security Guidelines

### Input Validation

```go
// GOOD: Validate all inputs
func ValidateTradeRequest(req *TradeRequest) error {
    if req.Amount.LessThanOrEqual(decimal.Zero) {
        return errors.New("trade amount must be positive")
    }
    
    if req.Symbol == "" {
        return errors.New("symbol is required")
    }
    
    if !isValidSymbol(req.Symbol) {
        return errors.New("invalid symbol format")
    }
    
    return nil
}
```

### SQL Injection Prevention

```go
// GOOD: Use parameterized queries
func (r *TradeRepository) GetTradesByUser(ctx context.Context, userID int64) ([]Trade, error) {
    query := `SELECT id, user_id, symbol, amount, price, created_at 
              FROM trades WHERE user_id = $1 ORDER BY created_at DESC`
    
    rows, err := r.db.QueryContext(ctx, query, userID)
    // Handle results...
}
```

## Performance Guidelines

### Memory Management

```go
// GOOD: Reuse objects to reduce allocations
type OrderBookProcessor struct {
    orderPool sync.Pool
}

func (p *OrderBookProcessor) processOrder(data []byte) {
    order := p.orderPool.Get().(*Order)
    defer p.orderPool.Put(order)
    
    // Reset and use the order
    order.Reset()
    // Process...
}
```

### Database Queries

```go
// GOOD: Use prepared statements for repeated queries
type TradeRepository struct {
    insertStmt *sql.Stmt
    selectStmt *sql.Stmt
}

func (r *TradeRepository) InsertTrade(ctx context.Context, trade *Trade) error {
    _, err := r.insertStmt.ExecContext(ctx, trade.UserID, trade.Symbol, 
        trade.Amount, trade.Price, trade.CreatedAt)
    return err
}
```

## Logging Standards

### Structured Logging

```go
// GOOD: Use structured logging with context
func (s *ArbitrageService) ExecuteTrade(ctx context.Context, opp *Opportunity) error {
    logger := s.logger.With(
        "opportunity_id", opp.ID,
        "source_exchange", opp.SourceExchange,
        "target_exchange", opp.TargetExchange,
        "symbol", opp.Symbol,
    )
    
    logger.Info("starting arbitrage execution")
    
    // Execute trade...
    
    logger.Info("arbitrage execution completed", 
        "profit", opp.ExpectedProfit.String())
    
    return nil
}
```

### Sensitive Data Handling

```go
// GOOD: Mask sensitive data in logs
func logAPIRequest(endpoint string, apiKey string) {
    maskedKey := apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
    log.Info("making API request", 
        "endpoint", endpoint, 
        "api_key", maskedKey)
}
```

## Code Review Checklist

When reviewing code, ensure:

1. **Financial Precision**: No float64 for monetary values
2. **Error Handling**: All errors are properly wrapped and handled
3. **Security**: No secrets in logs, proper input validation
4. **Concurrency**: Proper synchronization for shared state
5. **Testing**: Adequate test coverage, especially for edge cases
6. **Documentation**: Public functions and complex logic are documented
7. **Performance**: No obvious performance bottlenecks
8. **Rate Limiting**: API calls are properly rate limited
9. **Monitoring**: Important operations are logged with context
10. **Configuration**: No hardcoded values, use configuration files

This style guide should be followed consistently across the entire codebase to maintain high code quality and system reliability.