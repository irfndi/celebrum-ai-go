# Celebrum AI - Makefile for development and deployment

# Variables
APP_NAME=celebrum-ai
GO_VERSION=1.21
DOCKER_IMAGE=$(APP_NAME):latest
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_COMPOSE_PROD_FILE=docker-compose.single-droplet.yml

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build test test-coverage lint fmt run dev dev-setup dev-down install-tools security docker-build docker-run deploy clean

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

fmt: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	go fmt ./...
	goimports -w .

run: build ## Run the application
	@echo "$(GREEN)Starting $(APP_NAME)...$(NC)"
	./bin/$(APP_NAME)

dev: ## Run with hot reload (requires air)
	@echo "$(GREEN)Starting development server with hot reload...$(NC)"
	air

## Environment Setup
dev-setup: ## Setup development environment
	@echo "$(GREEN)Setting up development environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d postgres redis
	@echo "$(GREEN)Development environment ready!$(NC)"

dev-down: ## Stop development environment
	@echo "$(YELLOW)Stopping development environment...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

install-tools: ## Install development tools
	@echo "$(GREEN)Installing development tools...$(NC)"
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
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
	docker-compose -f $(DOCKER_COMPOSE_FILE) up --build

docker-prod: ## Run production Docker setup
	@echo "$(GREEN)Running production Docker setup...$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_PROD_FILE) up -d --build

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
	./scripts/deploy.sh production

deploy-staging: ## Deploy to staging
	@echo "$(GREEN)Deploying to staging...$(NC)"
	./scripts/deploy.sh staging

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

status: ## Show service status
	@echo "$(GREEN)Service Status:$(NC)"
	docker-compose -f $(DOCKER_COMPOSE_FILE) ps

## Logs
logs: ## Show application logs
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f app

logs-ccxt: ## Show CCXT service logs
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f ccxt-service

logs-all: ## Show all service logs
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f
