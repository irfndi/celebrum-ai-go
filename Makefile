# Celebrum AI - Makefile for development and deployment

# Variables
APP_NAME=celebrum-ai
GO_VERSION=1.24
DOCKER_IMAGE=$(APP_NAME):latest
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_COMPOSE_DEV_FILE=docker-compose.override.yml
DOCKER_COMPOSE_STAGING_FILE=docker-compose.staging.yml
DOCKER_COMPOSE_PROD_FILE=docker-compose.prod.yml

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: all help build test test-coverage lint fmt go-fmt fmt-check run dev dev-setup dev-down install-tools security docker-build docker-run docker-prod docker-push db-migrate db-seed deploy deploy-staging deploy-manual deploy-rollback ci-test ci-lint ci-build ci-check ts-fmt ts-lint ts-test ts-build ensure-bun go-typecheck all-fmt all-lint all-test all-build setup-all health-all clean docker-prune clean-deep mod-tidy mod-download ccxt-setup health

# Default target
all: build

## Help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(BLUE)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Development
build: ## Build the application
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	go build -o bin/$(APP_NAME) cmd/server/main.go
	@echo "$(GREEN)Build complete!$(NC)"

test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

lint: ## Run linter
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run

go-fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	go fmt ./...
	goimports -w .

fmt: go-fmt ## Alias for go-fmt (backward compatibility)

fmt-check: ## Check code formatting
	@echo "$(GREEN)Checking code formatting...$(NC)"
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "$(RED)The following files are not formatted:$(NC)"; \
		gofmt -s -l .; \
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
setup-dev: ## Setup development environment with new config
	@echo "$(GREEN)Setting up development environment...$(NC)"
	./scripts/setup-environment.sh -e development

dev-up: ## Start development environment
	@echo "$(GREEN)Starting development environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -f $(DOCKER_COMPOSE_DEV_FILE) up -d

dev-down: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -f $(DOCKER_COMPOSE_DEV_FILE) down

setup-staging: ## Setup staging environment
	@echo "$(GREEN)Setting up staging environment...$(NC)"
	./scripts/setup-environment.sh -e staging

staging-up: ## Start staging environment
	@echo "$(GREEN)Starting staging environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -f $(DOCKER_COMPOSE_STAGING_FILE) up -d

staging-down: ## Stop staging environment
	@echo "$(YELLOW)Stopping staging environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -f $(DOCKER_COMPOSE_STAGING_FILE) down

setup-prod: ## Setup production environment
	@echo "$(GREEN)Setting up production environment...$(NC)"
	./scripts/setup-environment.sh -e production

prod-up: ## Start production environment
	@echo "$(GREEN)Starting production environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -f $(DOCKER_COMPOSE_PROD_FILE) up -d

prod-down: ## Stop production environment
	@echo "$(YELLOW)Stopping production environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) -f $(DOCKER_COMPOSE_PROD_FILE) down

install-tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(NC)"
	go install github.com/air-verse/air@v1.52.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.0
	go install golang.org/x/tools/cmd/goimports@v0.24.0
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@v2.20.0
	@echo "$(GREEN)Tools installed!$(NC)"

security: ## Run security scan
	@echo "$(GREEN)Running security scan...$(NC)"
	gosec ./...

## Docker
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE)$(NC)"

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

ci-build: ## Build for CI
	@echo "$(GREEN)Building for CI...$(NC)"
	CGO_ENABLED=0 go build -v -ldflags "-X main.version=$(shell git describe --tags --always --dirty) -X main.buildTime=$(shell date -u '+%Y-%m-%d_%H:%M:%S')" -o bin/$(APP_NAME) cmd/server/main.go

ci-check: ci-lint ci-test ci-build ## Run all CI checks
	@echo "$(GREEN)All CI checks completed!$(NC)"

docker-push: ## Push Docker image to registry
	@echo "$(GREEN)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE)

## TypeScript/Go Combined Commands
ensure-bun:
	@command -v bun >/dev/null || { echo "$(RED)Bun is not installed. See https://bun.sh/docs/installation$(NC)"; exit 1; }

ts-fmt: ensure-bun ## Format TypeScript code
	@echo "$(GREEN)Formatting TypeScript code...$(NC)"
	@if [ -d "ccxt-service" ]; then \
		cd ccxt-service && bun run format; \
	else \
		echo "$(YELLOW)ccxt-service directory not found, skipping TypeScript formatting$(NC)"; \
	fi

ts-lint: ensure-bun ## Lint TypeScript code
	@echo "$(GREEN)Linting TypeScript code...$(NC)"
	@if [ -d "ccxt-service" ]; then \
		cd ccxt-service && bun run lint; \
	else \
		echo "$(YELLOW)ccxt-service directory not found, skipping TypeScript linting$(NC)"; \
	fi

ts-test: ensure-bun ## Run TypeScript tests
	@echo "$(GREEN)Running TypeScript tests...$(NC)"
	cd ccxt-service && bun test

ts-build: ensure-bun ## Build TypeScript service
	@echo "$(GREEN)Building TypeScript service...$(NC)"
	cd ccxt-service && bun run build

## Type Checking
go-typecheck: ## Run Go type checking
	@echo "$(GREEN)Running Go type checking...$(NC)"
	go vet ./...
	@echo "$(GREEN)Type checking complete!$(NC)"

## Combined Commands
all-fmt: go-fmt ts-fmt ## Format both Go and TypeScript code
all-lint: lint go-typecheck ts-lint ## Lint both Go and TypeScript code
all-test: test ts-test ## Run both Go and TypeScript tests
all-build: build ts-build ## Build both Go and TypeScript services

## Setup and Installation
setup-all: install-tools ensure-bun mod-download ccxt-setup ## Install all development tools
	@echo "$(GREEN)All development tools installed!$(NC)"

## Health Checks
health-all: health ## Check health of all services
	@echo "$(GREEN)Checking CCXT service health...$(NC)"
	curl -sf http://localhost/ccxt/health >/dev/null || echo "$(RED)CCXT service health check failed$(NC)"

## Utilities
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	@if [ -d "ccxt-service" ]; then rm -rf ccxt-service/dist/; fi
	@echo "$(GREEN)Clean complete!$(NC)"

docker-prune: ## Clean up Docker resources
	@echo "$(YELLOW)Cleaning up Docker resources...$(NC)"
	docker system prune -f
	@echo "$(GREEN)Docker cleanup complete!$(NC)"

clean-deep: clean docker-prune ## Deep clean including Docker resources

mod-tidy: ## Tidy Go modules
	@echo "$(GREEN)Tidying Go modules...$(NC)"
	go mod tidy

mod-download: ## Download Go modules
	@echo "$(GREEN)Downloading Go modules...$(NC)"
	go mod download

## CCXT Service
ccxt-setup: ## Setup CCXT Node.js service
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
