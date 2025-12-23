# Continuation Plan: Monorepo Restructuring & CI/CD Enhancement

## 1. Environment & Dependencies Update
- [ ] **Go Version Upgrade**: Update `go.mod` in `services/backend-api` and project root (if applicable) to `go 1.25.5`.
- [ ] **Dockerfile Updates**: Update `services/backend-api/Dockerfile` to use `golang:1.25.5` as the builder image.
- [ ] **Dependency Cleanup**: Run `go mod tidy` in `services/backend-api` to remove unused dependencies (e.g., `github.com/go-telegram/bot`).
- [ ] **Bun & TypeScript**: Ensure `services/telegram-service` and `services/ccxt-service` are using the latest stable Bun version in their Dockerfiles/CI.

## 2. Infrastructure & Docker Compose
- [ ] **Service Separation**: Update `docker-compose.yaml` to define 3 distinct services:
    - `backend-api` (Go)
    - `ccxt-service` (TypeScript/Bun)
    - `telegram-service` (TypeScript/Bun)
- [ ] **Networking**: Ensure all services share the `coolify` network and can communicate internally via service names (e.g., `http://telegram-service:3002`).
- [ ] **Database & Redis**: Verify `docker-compose.yaml` uses the latest stable images for `postgres` and `redis`, ensuring persistence and correct port mappings.

## 3. CI/CD Pipeline Implementation
- [ ] **Unified Makefile**: Enhance the root `Makefile` to orchestrate tasks across the monorepo:
    - `make ci-check`: Runs lint, format, typecheck, and tests for all services.
    - `make build`: Builds all Docker images (dry-run for verification).
    - `make lint`: Runs `golangci-lint` for Go and `oxlint`/`eslint` for TypeScript.
    - `make test`: Runs unit and integration tests with coverage.
- [ ] **GitHub Actions/CI**: (If applicable) Update `.github/workflows/validation.yml` to utilize these Makefile targets.

## 4. Testing & Coverage
- [ ] **Coverage Goals**: Target 60% coverage across unit, integration, and e2e tests.
- [ ] **Test Refactoring**: Update Go tests to mock the external `telegram-service` HTTP calls instead of the internal bot library.
- [ ] **Seeding**: Implement a seeding script (e.g., `make db-seed`) or migration to populate the database with initial test data (users, exchange keys, etc.) for integration testing.

## 5. Database Consistency
- [ ] **Migration Workflow**: Enforce usage of `.sql` migration files in `services/backend-api/database/migrations`.
- [ ] **Auto-Migration**: Ensure the entrypoint script (`migrate.sh`) runs migrations on container startup but skips already applied ones (idempotency).

## 6. TypeScript Telegram Bot
- [ ] **Implementation**: Verify `services/telegram-service` uses `grammy` and `effect` (if desired) for the bot logic.
- [ ] **API Endpoint**: Ensure it exposes a `/send-message` (or similar) endpoint for the Go backend to consume.

## Execution Order
1.  **Update Go Version & Cleanup**: `go.mod`, `Dockerfile`.
2.  **Refactor Docker Compose**: Split services.
3.  **Enhance Makefile**: Add CI targets.
4.  **Implement Seeding**: Create SQL/script for test data.
5.  **Run CI/Tests**: Verify coverage and build stability.
