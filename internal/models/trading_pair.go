.PHONY: build test lint run clean docker-build docker-run dev-setup

# Variables
APP_NAME=celebrum-ai
GO_VERSION=1.21
DOCKER_IMAGE=$(APP_NAME):latest

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(APP_NAME) cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	go mod tidy

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	go run cmd/server/main.go

# Run with hot reload (requires air)
dev:
	@echo "Starting development server with hot reload..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

# Setup development environment
dev-setup:
	@echo "Setting up development environment..."
	docker-compose up -d postgres redis
	@echo "Waiting for services to start..."
	sleep 5
	@echo "Running database migrations..."
	@# TODO: Add migration command when implemented
	@echo "Development environment ready!"

# Stop development environment
dev-down:
	@echo "Stopping development environment..."
	docker-compose down

# Database migrations (placeholder)
migrate-up:
	@echo "Running database migrations..."
	@# TODO: Implement migration tool

migrate-down:
	@echo "Rolling back database migrations..."
	@# TODO: Implement migration tool

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	@echo "Development tools installed!"

# Generate API documentation (placeholder)
docs:
	@echo "Generating API documentation..."
	@# TODO: Add swagger/openapi generation

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  run           - Run the application"
	@echo "  dev           - Run with hot reload"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  dev-setup     - Setup development environment"
	@echo "  dev-down      - Stop development environment"
	@echo "  install-tools - Install development tools"
	@echo "  security      - Run security scan"
	@echo "  help          - Show this help message"