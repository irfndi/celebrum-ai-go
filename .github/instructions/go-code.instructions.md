---
applyTo: "**/*.go"
---

# Go Code Instructions

## Coding Standards

- Target Go 1.25; format with `gofmt`/`goimports`
- Use idiomatic Go naming:
  - Exported identifiers in `CamelCase`
  - Packages lowercase, short, descriptive
  - Interfaces with `-er` suffix when applicable (e.g., `Trader`, `Exchanger`)
- Keep functions under 50 lines when possible
- Use `decimal.Decimal` from `github.com/shopspring/decimal` for all financial calculations
- Never use `float64` for monetary values

## Error Handling

- Always wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return early on errors; avoid deep nesting
- Use custom error types for domain-specific errors

## Concurrency

- Use `sync.RWMutex` for shared state that's read more than written
- Use `sync.Mutex` for write-heavy shared state
- Always use `defer` for unlock to prevent deadlocks
- Use channels for goroutine communication; avoid shared memory when possible

## Testing

- Write table-driven tests for comprehensive coverage
- Place mocks in `internal/testutil` or `test/testmocks`
- Use `testify/assert` and `testify/require` for assertions
- Test edge cases for financial calculations (precision, rounding, overflow)

## Build and Test Commands

```bash
make build          # Build the application
make test           # Run tests
make lint           # Run golangci-lint
make fmt            # Format code
```
