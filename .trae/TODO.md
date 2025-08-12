# TODO:

- [x] 49: Debug and fix /api/futures-arbitrage/opportunities endpoint response issues (priority: High)
- [x] 56: Create FuturesArbitrageService to generate opportunities from collected funding rates (priority: High)
- [x] 57: Add FuturesArbitrageService initialization to main.go (priority: High)
- [x] 58: Implement periodic opportunity calculation scheduler in FuturesArbitrageService (priority: High)
- [x] 59: Build and run the application to test the new FuturesArbitrageService (priority: High)
- [x] 67: Create new migration file 036_add_missing_funding_rate_columns.sql (priority: High)
- [x] 68: Add mark_price DECIMAL(20, 8) column to funding_rates table with IF NOT EXISTS check (priority: High)
- [x] 69: Add index_price DECIMAL(20, 8) column to funding_rates table with IF NOT EXISTS check (priority: High)
- [x] 70: Apply the new migration to fix the PostgreSQL column error (priority: High)
- [x] 71: Test the futures arbitrage endpoint after fixing the missing columns (priority: High)
- [x] 72: Verify PostgreSQL error is resolved and endpoint returns proper JSON response (priority: High)
