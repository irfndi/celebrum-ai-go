# Celebrum AI - Makefile for development and deployment

# Variables
APP_NAME=celebrum-ai
GO_VERSION=1.25
DOCKER_REGISTRY=ghcr.io/irfndi
DOCKER_IMAGE_APP=$(DOCKER_REGISTRY)/app:latest
DOCKER_IMAGE_CCXT=$(DOCKER_REGISTRY)/ccxt-service:latest
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_COMPOSE_PROD_FILE=docker-compose.single-droplet.yml
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
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Building CCXT service...$(NC)"; \
		cd ccxt-service && bun run build; \
	else \
		echo "$(YELLOW)Skipping CCXT service build - ccxt-service directory or bun not found$(NC)"; \
	fi
	@echo "$(GREEN)Build complete!$(NC)"

test: ## Run tests across all languages
	@echo "$(GREEN)Running tests across all languages...$(NC)"
	@# Run Go tests
	go test -v ./...
	@# Run TypeScript/JavaScript tests in ccxt-service
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		cd ccxt-service && bun test; \
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

# E2E trace validation script (requires SigNoz collector running)
test-traces: ## Verify OTLP traces end-to-end using Bun script
	@echo "$(GREEN)Running OTLP traces validation script...$(NC)"
	bun run ./test-traces-bun.ts

lint: go-env-setup ## Run linter across all languages
	@echo "$(GREEN)Running linter across all languages...$(NC)"
	@# Lint Go code
	$(GO_ENV) golangci-lint run
	@# Lint TypeScript/JavaScript in ccxt-service
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Linting TypeScript...$(NC)"; \
		cd ccxt-service && bunx oxlint .; \
	else \
		echo "$(YELLOW)Skipping TypeScript linting - ccxt-service directory or bun not found$(NC)"; \
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
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Type checking TypeScript...$(NC)"; \
		cd ccxt-service && bun tsc --noEmit; \
	else \
		echo "$(YELLOW)Skipping TypeScript type checking - ccxt-service directory or bun not found$(NC)"; \
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
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Formatting TypeScript...$(NC)"; \
		cd ccxt-service && bunx prettier --write . || bun format --write . || echo "$(YELLOW)Could not format TypeScript - prettier or bun format not available$(NC)"; \
	else \
		echo "$(YELLOW)Skipping TypeScript formatting - ccxt-service directory or bun not found$(NC)"; \
	fi
	@# Format shell scripts
	@if command -v shfmt >/dev/null 2>&1; then \
		find . -name "*.sh" -not -path "./node_modules/*" -not -path "./.git/*" -not -path "./bin/*" -not -path "./build/*" -exec shfmt -w {} \; 2>/dev/null || true; \
	fi
	@# Format YAML files
	@if command -v bun >/dev/null 2>&1 && [ -d "ccxt-service" ]; then \
		cd ccxt-service && bunx prettier --write . 2>/dev/null || true; \
	fi

fmt-check: ## Check code formatting
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@UNFORMATTED="$$(gofmt -s -l .)"; \
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
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d postgres redis
	@echo "$(GREEN)Development environment ready!$(NC)"

dev-down: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) down

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
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Scanning TypeScript dependencies...$(NC)"; \
		cd ccxt-service && bun audit || echo "$(YELLOW)bun audit completed with warnings$(NC)"; \
	else \
		echo "$(YELLOW)Skipping TypeScript security scan - ccxt-service directory or bun not found$(NC)"; \
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
	@# Build main application image
	docker build -t $(DOCKER_IMAGE_APP) .
	@# Build CCXT service image if directory exists
	@if [ -d "ccxt-service" ]; then \
		echo "$(GREEN)Building CCXT service image...$(NC)"; \
		docker build -t $(DOCKER_IMAGE_CCXT) ./ccxt-service; \
	else \
		echo "$(YELLOW)Skipping CCXT service image - ccxt-service directory not found$(NC)"; \
	fi
	@echo "$(GREEN)Docker images built!$(NC)"

docker-build-hybrid: ## Build hybrid Docker image for binary deployment
	@echo "$(GREEN)Building hybrid Docker image...$(NC)"
	docker build --target binary -o bin/$(APP_NAME) .
	@echo "$(GREEN)Hybrid build completed: bin/$(APP_NAME)$(NC)"

docker-clean: ## Clean Docker artifacts
	@echo "$(YELLOW)Cleaning Docker artifacts...$(NC)"
	docker system prune -f
	@echo "$(GREEN)Docker cleanup completed$(NC)"

docker-build-all: ## Build all Docker images with version tags
	@echo "$(GREEN)Building all Docker images with version tags...$(NC)"
	docker build -t $(DOCKER_REGISTRY)/app:$(VERSION) -t $(DOCKER_REGISTRY)/app:latest .
	docker build -t $(DOCKER_REGISTRY)/ccxt-service:$(VERSION) -t $(DOCKER_REGISTRY)/ccxt-service:latest ./ccxt-service
	@echo "$(GREEN)All images built with version: $(VERSION)$(NC)"

docker-build-app: ## Build main app image with version
	@echo "$(GREEN)Building main app image...$(NC)"
	docker build -t $(DOCKER_REGISTRY)/app:$(VERSION) -t $(DOCKER_REGISTRY)/app:latest .

docker-build-ccxt: ## Build CCXT service image with version
	@echo "$(GREEN)Building CCXT service image...$(NC)"
	docker build -t $(DOCKER_REGISTRY)/ccxt-service:$(VERSION) -t $(DOCKER_REGISTRY)/ccxt-service:latest ./ccxt-service

docker-run: docker-build ## Run with Docker
	@echo "$(GREEN)Running with Docker...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) up --build

docker-prod: ## Run production Docker setup
	@echo "$(GREEN)Running production Docker setup...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_PROD_FILE) up -d --build

## Database
db-migrate: ## Run database migrations
	@echo "$(GREEN)Running database migrations...$(NC)"
	./bin/$(APP_NAME) migrate

db-seed: ## Seed database with sample data
	@echo "$(GREEN)Seeding database...$(NC)"
	./bin/$(APP_NAME) seed

## Deployment
deploy: ## Deploy to production
	@echo "$(GREEN)Deploying to production...$(NC)"
	./scripts/deploy-enhanced.sh production

deploy-staging: ## Deploy to staging
	@echo "$(GREEN)Deploying to staging...$(NC)"
	./scripts/deploy-enhanced.sh staging

deploy-manual: build ## Manual deployment with rsync
	@echo "$(GREEN)Manual deployment with rsync...$(NC)"
	./scripts/deploy.sh production

deploy-binary: build ## Deploy using binary deployment
	@echo "$(GREEN)Deploying using binary deployment...$(NC)"
	./scripts/deploy-binary.sh production

deploy-binary-staging: build ## Deploy to staging using binary deployment
	@echo "$(GREEN)Deploying to staging using binary deployment...$(NC)"
	./scripts/deploy-binary.sh staging

setup-staging: ## Set up staging environment
	@echo "$(GREEN)Setting up staging environment...$(NC)"
	./scripts/setup-staging.sh

deploy-rollback: ## Rollback deployment
	@echo "$(GREEN)Rolling back deployment...$(NC)"
	./scripts/deploy-enhanced.sh --rollback

## CI/CD
ci-test: ## Run CI tests with proper environment
	@echo "$(GREEN)Running CI tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...

ci-lint: ## Run linter for CI
	@echo "$(GREEN)Running CI linter...$(NC)"
	golangci-lint run --timeout=5m

ci-build: ## Build for CI across all languages
	@echo "$(GREEN)Building for CI...$(NC)"
	@# Build Go application for CI
	CGO_ENABLED=0 go build -v -ldflags "-X main.version=$(shell git describe --tags --always --dirty) -X main.buildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')" -o bin/$(APP_NAME) cmd/server/main.go
	@# Build TypeScript/CCXT service for CI
	@if [ -d "ccxt-service" ] && command -v bun >/dev/null 2>&1; then \
		echo "$(GREEN)Building CCXT service for CI...$(NC)"; \
		cd ccxt-service && bun run build; \
	else \
		echo "$(YELLOW)Skipping CCXT service CI build - ccxt-service directory or bun not found$(NC)"; \
	fi

ci-check: ci-lint ci-test ci-build ## Run all CI checks
	@echo "$(GREEN)All CI checks completed!$(NC)"

docker-push: ## Push Docker image to registry
	@echo "$(GREEN)Pushing Docker images to registry...$(NC)"
	docker push $(DOCKER_REGISTRY)/app:$(VERSION)
	docker push $(DOCKER_REGISTRY)/app:latest
	docker push $(DOCKER_REGISTRY)/ccxt-service:$(VERSION)
	docker push $(DOCKER_REGISTRY)/ccxt-service:latest

docker-push-app: ## Push main app image
	@echo "$(GREEN)Pushing main app image...$(NC)"
	docker push $(DOCKER_REGISTRY)/app:$(VERSION)
	docker push $(DOCKER_REGISTRY)/app:latest

docker-push-ccxt: ## Push CCXT service image
	@echo "$(GREEN)Pushing CCXT service image...$(NC)"
	docker push $(DOCKER_REGISTRY)/ccxt-service:$(VERSION)
	docker push $(DOCKER_REGISTRY)/ccxt-service:latest

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
	@echo "$(YELLOW)Using secure migration script...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) exec app ./scripts/migrate-optimized.sh migrate

.PHONY: auto-migrate dev-up prod-up

# Automatic migration for all environments
auto-migrate: ## Run automatic migration sync
	@echo "$(GREEN)Starting automatic migration sync...$(NC)"
	@docker compose -f $(DOCKER_COMPOSE_FILE) up --build migrate
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
	@chmod +x ./webhook-control.sh
	@./webhook-control.sh enable

webhook-disable: ## Disable external connections and Telegram webhooks
	@echo "$(YELLOW)Disabling external connections and webhooks...$(NC)"
	@chmod +x ./webhook-control.sh
	@./webhook-control.sh disable

webhook-status: ## Check webhook and external connection status
	@echo "$(GREEN)Checking webhook status...$(NC)"
	@chmod +x ./webhook-control.sh
	@./webhook-control.sh status

startup-status: ## Check sequential startup status
	@echo "$(GREEN)Checking startup status...$(NC)"
	@if [ -f ".env.startup" ]; then \
		echo "$(GREEN)Startup configuration found:$(NC)"; \
		cat .env.startup | grep -E "(STARTUP_PHASE|EXTERNAL_CONNECTIONS_ENABLED|WARMUP_ENABLED)"; \
	else \
		echo "$(YELLOW)No startup configuration found$(NC)"; \
	fi

down-orchestrated: ## Stop all services gracefully with orchestrated shutdown
	@echo "$(YELLOW)Stopping services with orchestrated shutdown...$(NC)"
	@./webhook-control.sh disable 2>/dev/null || true
	@docker compose -f $(DOCKER_COMPOSE_FILE) down
	@docker compose -f $(DOCKER_COMPOSE_PROD_FILE) down 2>/dev/null || true
	@echo "$(GREEN)All services stopped$(NC)"

## Observability (SigNoz)
observability-deploy: ## Deploy SigNoz observability stack
	@echo "$(GREEN)Deploying SigNoz observability stack...$(NC)"
	@chmod +x ./observability/scripts/deploy.sh
	@./observability/scripts/deploy.sh

observability-health: ## Check SigNoz services health
	@echo "$(GREEN)Checking SigNoz services health...$(NC)"
	@chmod +x ./observability/scripts/health-check.sh
	@./observability/scripts/health-check.sh

observability-stop: ## Stop SigNoz observability stack
	@echo "$(YELLOW)Stopping SigNoz observability stack...$(NC)"
	cd observability && docker compose --env-file .env.observability down

observability-logs: ## Show SigNoz services logs
	@echo "$(GREEN)Showing SigNoz services logs...$(NC)"
	cd observability && docker compose --env-file .env.observability logs -f

observability-logs-query: ## Show SigNoz Query Service logs
	@echo "$(GREEN)Showing SigNoz Query Service logs...$(NC)"
	cd observability && docker compose --env-file .env.observability logs -f signoz-query-service

observability-logs-frontend: ## Show SigNoz Frontend logs
	@echo "$(GREEN)Showing SigNoz Frontend logs...$(NC)"
	cd observability && docker compose --env-file .env.observability logs -f signoz-frontend

observability-logs-collector: ## Show OpenTelemetry Collector logs
	@echo "$(GREEN)Showing OpenTelemetry Collector logs...$(NC)"
	cd observability && docker compose --env-file .env.observability logs -f signoz-otel-collector

observability-logs-clickhouse: ## Show ClickHouse logs
	@echo "$(GREEN)Showing ClickHouse logs...$(NC)"
	cd observability && docker compose --env-file .env.observability logs -f signoz-clickhouse

observability-restart: ## Restart SigNoz observability stack
	@echo "$(YELLOW)Restarting SigNoz observability stack...$(NC)"
	cd observability && docker compose --env-file .env.observability restart

observability-clean: ## Clean up SigNoz stack (containers only)
	@echo "$(YELLOW)Cleaning up SigNoz stack...$(NC)"
	@chmod +x ./observability/scripts/cleanup.sh
	@./observability/scripts/cleanup.sh

observability-clean-all: ## Clean up SigNoz stack (including volumes and data)
	@echo "$(RED)Cleaning up SigNoz stack completely...$(NC)"
	@chmod +x ./observability/scripts/cleanup.sh
	@./observability/scripts/cleanup.sh --volumes --data --force

observability-status: ## Show SigNoz services status
	@echo "$(GREEN)SigNoz Services Status:$(NC)"
	cd observability && docker compose --env-file .env.observability ps

observability-ui: ## Open SigNoz UI in browser
	@echo "$(GREEN)Opening SigNoz UI...$(NC)"
	@echo "$(BLUE)SigNoz UI: http://localhost:3301$(NC)"
	@command -v open >/dev/null 2>&1 && open http://localhost:3301 || echo "$(YELLOW)Please open http://localhost:3301 in your browser$(NC)"

## Utilities
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker system prune -f
	@echo "$(GREEN)Clean complete!$(NC)"

## Artifact Management
artifacts-setup: ## Setup artifacts directory structure
	@echo "$(GREEN)Setting up artifacts directory...$(NC)"
	./scripts/artifact-manager.sh setup

artifacts-store: ## Store deployment package
	@echo "$(GREEN)Storing deployment package...$(NC)"
	@if [ -z "$(PACKAGE)" ]; then echo "$(RED)Usage: make artifacts-store PACKAGE=<package> VERSION=<version> ENV=<env>$(NC)"; exit 1; fi
	@if [ -z "$(VERSION)" ]; then echo "$(RED)Usage: make artifacts-store PACKAGE=<package> VERSION=<version> ENV=<env>$(NC)"; exit 1; fi
	@if [ -z "$(ENV)" ]; then echo "$(RED)Usage: make artifacts-store PACKAGE=<package> VERSION=<version> ENV=<env>$(NC)"; exit 1; fi
	./scripts/artifact-manager.sh store "$(PACKAGE)" "$(VERSION)" "$(ENV)"

artifacts-list: ## List available artifacts
	@echo "$(GREEN)Listing available artifacts...$(NC)"
	./scripts/artifact-manager.sh list

artifacts-get: ## Get artifact path
	@echo "$(GREEN)Getting artifact path...$(NC)"
	@if [ -z "$(ARTIFACT_TYPE)" ]; then echo "$(RED)Usage: make artifacts-get ARTIFACT_TYPE=<type> VERSION=<version>$(NC)"; exit 1; fi
	@if [ -z "$(VERSION)" ]; then echo "$(RED)Usage: make artifacts-get ARTIFACT_TYPE=<type> VERSION=<version>$(NC)"; exit 1; fi
	./scripts/artifact-manager.sh get "$(ARTIFACT_TYPE)" "$(VERSION)"

artifacts-cleanup: ## Clean up old artifacts
	@echo "$(YELLOW)Cleaning up old artifacts...$(NC)"
	./scripts/artifact-manager.sh cleanup

## Deployment Benchmarking
benchmark: ## Run deployment benchmarks
	@echo "$(GREEN)Running deployment benchmarks...$(NC)"
	./scripts/benchmark-deployment.sh

benchmark-custom: ## Run deployment benchmarks with custom settings
	@echo "$(GREEN)Running deployment benchmarks with custom settings...$(NC)"
	./scripts/benchmark-deployment.sh --iterations "$(ITERATIONS)" --server "$(SERVER)"

benchmark-report: ## Generate benchmark report
	@echo "$(GREEN)Generating benchmark report...$(NC)"
	@if [ -z "$(REPORT_FILE)" ]; then echo "$(RED)Usage: make benchmark-report REPORT_FILE=<path>$(NC)"; exit 1; fi
	./scripts/benchmark-deployment.sh --report "$(REPORT_FILE)"

benchmark-clean: ## Clean up benchmark results directory
	@echo "$(YELLOW)Cleaning up benchmark results directory...$(NC)"
	@if [ -d "benchmark-results" ]; then \
		find benchmark-results -name "*.csv" -type f -delete 2>/dev/null || true; \
		find benchmark-results -name "*.md" -type f -delete 2>/dev/null || true; \
		rmdir benchmark-results 2>/dev/null || true; \
		echo "$(GREEN)Benchmark results cleaned up successfully$(NC)"; \
	else \
		echo "$(BLUE)No benchmark results directory found$(NC)"; \
	fi

mod-tidy: ## Tidy Go modules
	@echo "$(GREEN)Tidying Go modules...$(NC)"
	go mod tidy

mod-download: ## Download Go modules
	@echo "$(GREEN)Downloading Go modules...$(NC)"
	go mod download

## CCXT Service
ccxt-setup: ## Setup CCXT Bun service
	@echo "$(GREEN)Setting up CCXT service...$(NC)"
	cd ccxt-service && bun install

ccxt-dev: ## Run CCXT service in development
	@echo "$(GREEN)Starting CCXT service...$(NC)"
	cd ccxt-service && bun run dev

ccxt-build: ## Build CCXT service
	@echo "$(GREEN)Building CCXT service...$(NC)"
	cd ccxt-service && bun run build

## Health Checks
health: ## Check application health
	@echo "$(GREEN)Checking application health...$(NC)"
	curl -f http://localhost:8080/health || echo "$(RED)Health check failed$(NC)"

health-prod: ## Check production health
	@echo "$(GREEN)Checking production health...$(NC)"
	curl -f https://localhost/health || echo "$(RED)Production health check failed$(NC)"

status: ## Show service status
	@echo "$(GREEN)Service Status:$(NC)"
	docker compose -f $(DOCKER_COMPOSE_FILE) ps

status-prod: ## Show production service status
	@echo "$(GREEN)Production Service Status:$(NC)"
	docker compose -f $(DOCKER_COMPOSE_PROD_FILE) ps

## Logs
logs: ## Show application logs
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f app

logs-ccxt: ## Show CCXT service logs
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f ccxt-service

logs-all: ## Show all service logs
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f
