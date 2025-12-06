# Celebrum AI - Makefile for development and deployment

# Variables
APP_NAME=celebrum-ai
GO_VERSION=1.25
DOCKER_REGISTRY=ghcr.io/irfndi
DOCKER_IMAGE_APP=$(DOCKER_REGISTRY)/app:latest
DOCKER_COMPOSE_FILE?=docker-compose.dev.yml
DOCKER_COMPOSE_ENV_FILE=.env
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO_CACHE_DIR=$(PWD)/.cache/go-build
GO_MOD_CACHE_DIR=$(PWD)/.cache/go-mod
GO_ENV=GOCACHE=$(GO_CACHE_DIR)

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build test test-coverage coverage-check lint fmt run dev dev-setup dev-down install-tools security docker-build docker-run deploy clean dev-up-orchestrated prod-up-orchestrated webhook-enable webhook-disable webhook-status startup-status down-orchestrated go-env-setup

# Default target
all: build

## Help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

go-env-setup:
	@mkdir -p $(GO_CACHE_DIR) $(GO_MOD_CACHE_DIR)

## Development
build: ## Build the application across all languages
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	@# Build Go application
	go build -o bin/$(APP_NAME) cmd/server/main.go
	@# Build TypeScript/CCXT service
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Building CCXT service...$(NC)"; \
		cd services/ccxt && bun run build; \
	else \
		echo "$(YELLOW)Skipping CCXT service build - services/ccxt directory or bun not found$(NC)"; \
	fi
	@echo "$(GREEN)Build complete!$(NC)"

test: ## Run tests across all languages
	@echo "$(GREEN)Running tests across all languages...$(NC)"
	@# Run Go tests
	go test -v ./...
	@# Run TypeScript/JavaScript tests in ccxt-service
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		cd services/ccxt && bun test; \
	fi
	@# Run shell script tests if available
	@if [ -f "scripts/test.sh" ]; then \
		bash scripts/test.sh; \
	else \
		true; \
	fi

test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./cmd/... ./internal/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

coverage-check: ## Run coverage gate (warn by default, STRICT=true to fail)
	@echo "$(GREEN)Running coverage check (threshold $${MIN_COVERAGE:-80}%)...$(NC)"
	MIN_COVERAGE=$${MIN_COVERAGE:-80} \
	STRICT=$${STRICT:-false} \
	bash scripts/coverage-check.sh

lint: go-env-setup ## Run linter across all languages
	@echo "$(GREEN)Running linter across all languages...$(NC)"
	@# Lint Go code
	$(GO_ENV) golangci-lint run
	@# Lint TypeScript/JavaScript in ccxt-service
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Linting TypeScript...$(NC)"; \
		cd services/ccxt && bunx oxlint .; \
	else \
		echo "$(YELLOW)Skipping TypeScript linting - services/ccxt directory or bun not found$(NC)"; \
	fi
	@# Lint YAML files
	@if command -v yamllint >/dev/null 2>&1; then \
		find . -name "*.yml" -o -name "*.yaml" | grep -v node_modules | grep -v .git | grep -v build | xargs yamllint 2>/dev/null || true; \
	fi

typecheck: ## Run type checking across all languages
	@echo "$(GREEN)Running type checking across all languages...$(NC)"
	@# Type check Go code
	go vet ./...
	@# Type check TypeScript in ccxt-service
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Type checking TypeScript...$(NC)"; \
		cd services/ccxt && bun tsc --noEmit; \
	else \
		echo "$(YELLOW)Skipping TypeScript type checking - services/ccxt directory or bun not found$(NC)"; \
	fi

fmt: ## Format code across all languages
	@echo "$(GREEN)Formatting code across all languages...$(NC)"
	@# Format Go code
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "$(YELLOW)goimports not found, skipping Go imports formatting$(NC)"; \
	fi
	@# Format TypeScript/JavaScript in ccxt-service
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Formatting TypeScript...$(NC)"; \
		cd services/ccxt && bunx prettier --write . || bun format --write . || echo "$(YELLOW)Could not format TypeScript - prettier or bun format not available$(NC)"; \
	else \
		echo "$(YELLOW)Skipping TypeScript formatting - services/ccxt directory or bun not found$(NC)"; \
	fi
	@# Format shell scripts
	@if command -v shfmt >/dev/null 2>&1; then \
		find . -name "*.sh" -not -path "./node_modules/*" -not -path "./.git/*" -not -path "./bin/*" -not -path "./build/*" -exec shfmt -w {} \; 2>/dev/null || true; \
	fi
	@# Format YAML files
	@if command -v bun >/dev/null 2>&1 && [ -d "services/ccxt" ]; then \
		cd services/ccxt && bunx prettier --write . 2>/dev/null || true; \
	fi

fmt-check: ## Check code formatting
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@UNFORMATTED="$$(find . -name "*.go" -not -path "./.cache/*" -not -path "./vendor/*" -not -path "./.git/*" | xargs gofmt -s -l)"; \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "$(RED)The following files are not formatted:$(NC)"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	else \
		echo "$(GREEN)All files are properly formatted$(NC)"; \
	fi

run: build ## Run the application
	@echo "$(GREEN)Starting $(APP_NAME)...$(NC)"
	./bin/$(APP_NAME)

dev: ## Run with hot reload (requires air)
	@echo "$(GREEN)Starting development server with hot reload...$(NC)"
	air

## Environment Setup
dev-setup: ## Setup development environment
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@if [ ! -f .env ]; then cp .env.example .env; echo "$(YELLOW)Created .env from .env.example$(NC)"; fi
	docker compose -f docker-compose.dev.yml --env-file .env up -d postgres
	@echo "$(GREEN)Development environment ready!$(NC)"

dev-down: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	docker compose -f docker-compose.dev.yml --env-file .env down
	@echo "$(GREEN)Development environment stopped$(NC)"

install-tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(NC)"
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Tools installed!$(NC)"

security: ## Run security scan across all languages
	@echo "$(GREEN)Running security scan across all languages...$(NC)"
	@# Security scan for Go
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)gosec not found, skipping Go security scan$(NC)"; \
	fi
	@# Security scan for TypeScript/JavaScript dependencies
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Scanning TypeScript dependencies...$(NC)"; \
		cd services/ccxt && bun audit || echo "$(YELLOW)bun audit completed with warnings$(NC)"; \
	else \
		echo "$(YELLOW)Skipping TypeScript security scan - services/ccxt directory or bun not found$(NC)"; \
	fi
	@# Security scan for Docker images
	@if command -v docker >/dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/app" -w /app securecodewarrior/docker-security-scanner . 2>/dev/null || echo "$(YELLOW)Docker security scanner not available$(NC)"; \
	fi
	@# Check for secrets in code
	@if command -v gitleaks >/dev/null 2>&1; then \
		gitleaks detect --source . --verbose 2>/dev/null || echo "$(YELLOW)gitleaks not available$(NC)"; \
	fi

## Docker
docker-build: ## Build Docker images for all services
	@echo "$(GREEN)Building Docker images...$(NC)"
	@# Build main application image (includes CCXT service via multi-stage build)
	docker build -t $(DOCKER_IMAGE_APP) .
	@echo "$(GREEN)Docker images built!$(NC)"

docker-clean: ## Clean Docker artifacts
	@echo "$(YELLOW)Cleaning Docker artifacts...$(NC)"
	docker system prune -f
	@echo "$(GREEN)Docker cleanup completed$(NC)"

docker-build-all: ## Build all Docker images with version tags
	@echo "$(GREEN)Building all Docker images with version tags...$(NC)"
	docker build -t $(DOCKER_REGISTRY)/app:$(VERSION) -t $(DOCKER_REGISTRY)/app:latest .
	@echo "$(GREEN)All images built with version: $(VERSION)$(NC)"

docker-build-app: ## Build main app image with version
	@echo "$(GREEN)Building main app image...$(NC)"
	docker build -t $(DOCKER_REGISTRY)/app:$(VERSION) -t $(DOCKER_REGISTRY)/app:latest .

docker-run: docker-build ## Run with Docker
	@echo "$(GREEN)Running with Docker...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) --env-file .env up --build

docker-prod: ## Run production Docker setup
	@echo "$(GREEN)Running production Docker setup...$(NC)"
	docker compose -f docker-compose.yml --env-file .env up -d --build
	@echo "$(GREEN)Production environment started!$(NC)"

## Database
db-migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@chmod +x database/migrate.sh
	@./database/migrate.sh

db-seed: ## Seed database with sample data
	@echo "$(GREEN)Seeding database...$(NC)"
	./bin/$(APP_NAME) seed

## CI/CD
ci-test: ## Run CI tests with proper environment
	@echo "$(GREEN)Running CI tests...$(NC)"
	@# Exclude only internal/api/handlers/testmocks and internal/observability packages (no tests, trigger Go 1.25 covdata bug)
	go test -v -race -coverprofile=coverage.out $$(go list ./... | grep -v -E '(internal/api/handlers/testmocks|internal/observability)')
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Running CCXT service tests...$(NC)"; \
		cd services/ccxt && bun test; \
	fi

ci-lint: ## Run linter for CI
	@echo "$(GREEN)Running CI linter...$(NC)"
	./bin/golangci-lint run --timeout=5m

ci-build: ## Build for CI across all languages
	@echo "$(GREEN)Building for CI...$(NC)"
	@# Build Go application for CI
	CGO_ENABLED=0 go build -v -ldflags "-X main.version=$(shell git describe --tags --always --dirty) -X main.buildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')" -o bin/$(APP_NAME) cmd/server/main.go
	@# Build TypeScript/CCXT service for CI
	@if [ -d "services/ccxt" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Building CCXT service for CI...$(NC)"; \
		cd services/ccxt && bun run build; \
	else \
		echo "$(YELLOW)Skipping CCXT service CI build - services/ccxt directory or bun not found$(NC)"; \
	fi

ci-check: ci-lint ci-test ci-build ## Run all CI checks
	@echo "$(GREEN)All CI checks completed!$(NC)"

docker-push: ## Push Docker image to registry
	@echo "$(GREEN)Pushing Docker images to registry...$(NC)"
	docker push $(DOCKER_REGISTRY)/app:$(VERSION)
	docker push $(DOCKER_REGISTRY)/app:latest

docker-push-app: ## Push main app image
	@echo "$(GREEN)Pushing main app image...$(NC)"
	docker push $(DOCKER_REGISTRY)/app:$(VERSION)
	docker push $(DOCKER_REGISTRY)/app:latest

## Database Migration Targets
.PHONY: migrate migrate-status migrate-list migrate-docker

migrate: ## Run all pending database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	@cd database && ./migrate.sh

migrate-status: ## Check database migration status
	@echo "$(GREEN)Checking migration status...$(NC)"
	@cd database && ./migrate.sh status

migrate-list: ## List available database migrations
	@echo "$(GREEN)Listing available migrations...$(NC)"
	@cd database && ./migrate.sh list

migrate-docker: ## Run migrations in Docker environment
	@echo "$(GREEN)Running migrations in Docker...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) --env-file .env exec celebrum /app/database/migrate.sh

.PHONY: auto-migrate dev-up prod-up

# Automatic migration for all environments
auto-migrate: ## Run automatic migration sync
	@echo "$(GREEN)Starting automatic migration sync...$(NC)"
	@docker compose -f $(DOCKER_COMPOSE_FILE) --env-file .env exec celebrum /app/database/migrate.sh
	@echo "$(GREEN)All migrations applied successfully!$(NC)"

# Development environment with orchestrated sequential startup
dev-up-orchestrated: ## Start development environment with robust sequential startup
	@echo "$(GREEN)Starting development environment with orchestrated sequential startup...$(NC)"
	@chmod +x scripts/startup-orchestrator.sh
	@./scripts/startup-orchestrator.sh dev

# Production environment with orchestrated sequential startup
prod-up-orchestrated: ## Start production environment with robust sequential startup
	@echo "$(GREEN)Starting production environment with orchestrated sequential startup...$(NC)"
	@chmod +x scripts/startup-orchestrator.sh
	@./scripts/startup-orchestrator.sh prod

# Development environment with auto-migration (legacy - use dev-up-orchestrated for robust startup)
dev-up: dev-up-orchestrated ## Start development environment with automatic migrations

# Production-like environment with auto-migration (legacy - use prod-up-orchestrated for robust startup)
prod-up: prod-up-orchestrated ## Start production environment with automatic migrations

## Sequential Startup Control
webhook-enable: ## Enable external connections and Telegram webhooks
	@echo "$(GREEN)Enabling external connections and webhooks...$(NC)"
	@chmod +x scripts/webhook-control.sh
	@scripts/webhook-control.sh enable

webhook-disable: ## Disable external connections and Telegram webhooks
	@echo "$(YELLOW)Disabling external connections and webhooks...$(NC)"
	@chmod +x scripts/webhook-control.sh
	@scripts/webhook-control.sh disable

webhook-status: ## Check webhook and external connection status
	@echo "$(GREEN)Checking webhook status...$(NC)"
	@chmod +x scripts/webhook-control.sh
	@scripts/webhook-control.sh status

startup-status: ## Check sequential startup status
	@echo "$(GREEN)Checking startup status...$(NC)"
	@if [ -f ".env" ]; then \
		echo "$(GREEN)Startup configuration found:$(NC)"; \
		cat .env | grep -E "(STARTUP_PHASE|EXTERNAL_CONNECTIONS_ENABLED|WARMUP_ENABLED)" || echo "No startup vars found in .env"; \
	else \
		echo "$(YELLOW)No startup configuration found$(NC)"; \
	fi

down-orchestrated: ## Stop all services gracefully with orchestrated shutdown
	@echo "$(YELLOW)Stopping services with orchestrated shutdown...$(NC)"
	@scripts/webhook-control.sh disable 2>/dev/null || true
	@docker compose -f $(DOCKER_COMPOSE_FILE) --env-file .env down
	@echo "$(GREEN)All services stopped$(NC)"

## Utilities
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker system prune -f
	@echo "$(GREEN)Clean complete!$(NC)"

mod-tidy: ## Tidy Go modules
	@echo "$(GREEN)Tidying Go modules...$(NC)"
	go mod tidy

mod-download: ## Download Go modules
	@echo "$(GREEN)Downloading Go modules...$(NC)"
	go mod download

## CCXT Service
ccxt-setup: ## Setup CCXT Bun service
	@echo "$(GREEN)Setting up CCXT service...$(NC)"
	cd services/ccxt && bun install

ccxt-dev: ## Run CCXT service in development
	@echo "$(GREEN)Starting CCXT service...$(NC)"
	cd services/ccxt && bun run dev

ccxt-build: ## Build CCXT service
	@echo "$(GREEN)Building CCXT service...$(NC)"
	cd services/ccxt && bun run build

## Health Checks
health: ## Check application health
	@echo "$(GREEN)Checking application health...$(NC)"
	curl -f http://localhost:8080/health || echo "$(RED)Health check failed$(NC)"

health-prod: ## Check production health
	@echo "$(GREEN)Checking production health...$(NC)"
	curl -f https://localhost/health || echo "$(RED)Production health check failed$(NC)"

status: ## Show service status
	@echo "$(GREEN)Service Status:$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) --env-file .env ps

status-prod: ## Show production service status
	@echo "$(GREEN)Production Service Status:$(NC)"
	@if [ -f ".env" ]; then \
		docker compose -f docker-compose.yml --env-file .env ps; \
	else \
		echo "$(YELLOW)No .env file found. Run 'make docker-prod' first.$(NC)"; \
	fi

## Logs
logs: ## Show application logs
	docker compose -f $(DOCKER_COMPOSE_FILE) --env-file .env logs -f celebrum

logs-all: ## Show all service logs
	docker compose --env-file .env logs -f

## Environment Management
env-current: ## Show current environment
	@if [ -f ".env" ]; then \
		if grep -q "ENVIRONMENT=development" .env; then \
			echo "$(GREEN)Current environment: Development$(NC)"; \
		elif grep -q "ENVIRONMENT=production" .env; then \
			echo "$(GREEN)Current environment: Production$(NC)"; \
		else \
			echo "$(YELLOW)Current environment: Custom$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)No .env file found.$(NC)"; \
	fi
