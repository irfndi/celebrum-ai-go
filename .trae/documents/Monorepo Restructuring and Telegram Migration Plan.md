# Restructure to Monorepo and Migrate Telegram Service

This plan involves reorganizing the project into a monorepo structure, moving the Go application and TypeScript services into a `services/` directory, and migrating the Telegram Bot from Go to a new TypeScript service (Grammy + EffectTS). The deployment will remain unified (Single Container Monolith) for now, but with the flexibility to split later.

## 1. Directory Restructuring (Monorepo)
Create a `services/` directory to house all applications.

-   **`services/backend-api/`**: Will contain the Go application.
    -   Move `cmd/`, `internal/`, `pkg/`, `go.mod`, `go.sum`, `config.yml` here.
    -   Move `database/` here (as it's primarily the backend's responsibility).
    -   Create a new `Dockerfile` for independent building.
-   **`services/ccxt-service/`**:
    -   Move existing `ccxt-service/` content here.
-   **`services/telegram-service/`**:
    -   Move existing `telegram-service/` content here.
-   **Root**:
    -   Update `Dockerfile` to build from the new `services/` paths.
    -   Update `docker-compose.yaml` to reflect the new structure (but keep running as a single service for now, or split into 3 services if preferred for "future k8s" readiness). *I will configure 3 separate services in Docker Compose to fulfill the "each services using each docker" requirement while keeping deployment simple.*
    -   Keep `Makefile` and `scripts/` at root for orchestration.

## 2. Telegram Service Migration (Go -> TS)
The new TypeScript service will handle all Telegram interactions. The Go backend will become a pure API provider for the bot.

### TypeScript Service (`services/telegram-service`)
-   **Enhance `index.ts`**:
    -   Add a `POST /send-message` endpoint (protected by `ADMIN_API_KEY`) to allow the backend to send alerts via the bot.
    -   Ensure it connects to the `backend-api` service for fetching data (`/opportunities`, `/users`).

### Go Backend (`services/backend-api`)
-   **Refactor `NotificationService`**:
    -   Remove `go-telegram/bot` dependency.
    -   Update `SendMessage` logic to make an HTTP POST request to the `telegram-service`'s new `/send-message` endpoint.
-   **Clean up Handlers**:
    -   Delete `internal/api/handlers/telegram.go` (the old bot logic).
    -   Keep `internal/api/handlers/telegram_internal.go` (API endpoints for the bot).
    -   Remove the webhook route from `internal/api/routes.go`.

## 3. Configuration & Deployment
-   **Docker Compose**: Define 3 services (`backend-api`, `ccxt-service`, `telegram-service`) sharing a network.
-   **Makefile**: Update build and run targets to support the new directory structure.
-   **Cleanup**: Remove the old Go Telegram library from `go.mod`.

## Execution Steps
1.  **Create Folders**: Set up `services/` and move files.
2.  **Update Configs**: Modify `docker-compose.yaml` and `Makefile`.
3.  **Update TS Service**: Add `/send-message` endpoint.
4.  **Refactor Go Backend**: Switch notification logic to HTTP and remove old bot code.
5.  **Verify**: Ensure all services build and communicate correctly.
