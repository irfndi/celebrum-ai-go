# TODO List

## High Priority Security Tasks

- [ ] **Secure admin endpoints** - Add authentication and authorization middleware to /api/admin/ endpoints (e.g., exchange blacklist management) to prevent unauthorized access
- [ ] **Fix GitGuardian security alerts**:
  - [ ] Generic High Entropy Secret in `internal/middleware/auth_test.go` (commit 58fdeb8)
  - [ ] Generic Password in `docker-compose.yml` (commit 76611de) 
  - [ ] Generic Password in `scripts/robust-migrate.sh` (commit 76611de)
- [ ] **Make exchange blacklist persistent** - Store blacklist changes in configuration file or database instead of in-memory modifications that are lost on restart
- [ ] **Optimize production Docker image** - Remove debugging tools (procfs, htop) from production image to minimize attack surface and image size. Consider multi-stage build with debug stage

## Completed Tasks

- [x] ~~Fix failing tests with timeout issues in `internal/services` package~~ - **COMPLETED**: All tests in internal/services are now passing. Tests that had goroutine leaks in external library (github.com/cinar/indicator/v2) are properly skipped. No timeout issues remain.
- [x] ~~Analyze cache metrics showing zero hits/misses despite Redis keys~~
- [x] ~~Fix MarketHandler integration with CacheAnalyticsService~~
- [x] ~~Create comprehensive tests for cache analytics~~
- [x] ~~Create comprehensive tests for Telegram bot webhook functionality~~
- [x] ~~Document architectural recommendations for cache and Telegram systems~~

## Medium Priority Tasks

- [ ] Add proper cache category tracking for remaining handlers
- [ ] Implement rate limiting for API endpoints
- [ ] Add monitoring and alerting for critical services
- [ ] Optimize database queries for better performance

## Low Priority Tasks

- [ ] Add more comprehensive logging
- [ ] Implement graceful shutdown handling
- [ ] Add health check endpoints
- [ ] Consider implementing circuit breaker pattern for external API calls
- [ ] Run Make fmt, lint, test. etc (all language)