---
applyTo: "internal/services/**/*"
---

# Services Layer Instructions

## Service Design

- Services contain business logic; keep handlers thin
- Use dependency injection for external dependencies
- Define service interfaces for testability
- Implement circuit breaker pattern for external service calls

## Financial Services

### Arbitrage Detection

- Use `decimal.Decimal` for all price and quantity calculations
- Consider exchange fees in profit calculations
- Account for slippage in trade execution estimates
- Validate opportunities before returning

### Technical Analysis

- Implement indicators following standard formulas
- Handle insufficient data gracefully
- Cache computed indicators when appropriate
- Document calculation methodology

## Error Handling

- Wrap errors with context
- Use custom error types for business logic errors
- Log errors with sufficient context for debugging
- Implement graceful degradation

## Concurrency

- Use mutexes for shared state
- Implement proper timeout handling
- Use context for cancellation propagation
- Avoid blocking operations in hot paths

## Testing

- Unit test all service methods
- Mock external dependencies (database, cache, APIs)
- Test edge cases in financial calculations
- Test concurrent access patterns

## Code Example

```go
type ArbitrageService struct {
    mu           sync.RWMutex
    exchangeRepo ExchangeRepository
    cache        Cache
}

func (s *ArbitrageService) FindOpportunities(ctx context.Context) ([]Opportunity, error) {
    // Use context for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Business logic here
    return opportunities, nil
}
```
