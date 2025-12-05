# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Celebrum AI is a comprehensive cryptocurrency arbitrage detection and technical analysis platform built with Go. It features real-time market data collection, arbitrage opportunity identification, and technical indicator calculations with support for 100+ cryptocurrency exchanges via CCXT.

## Development Commands

### Essential Development Commands
```bash
# Build and run
make build                    # Build the application
make run                      # Run the application
make dev                      # Run with hot reload (requires air)

# Development environment
make dev-setup                # Start PostgreSQL and Redis services
make dev-down                 # Stop development environment
make dev-up-orchestrated      # Start with robust sequential startup

# Testing and quality
make test                     # Run tests across all languages
make test-coverage            # Run tests with coverage report
make lint                     # Run linter across all languages
make typecheck                # Run type checking
make fmt                      # Format code across all languages
make ci-check                 # Run all CI checks locally

# Database operations
make migrate                  # Run database migrations
make migrate-status           # Check migration status
make migrate-list             # List available migrations

# Docker operations
make docker-build             # Build Docker image
make docker-run               # Run with Docker
make docker-prod              # Run production Docker setup

# Deployment
make deploy                   # Deploy to production (containers)
make deploy-staging           # Deploy to staging (containers)
make health-prod              # Check production health

```

## Architecture Overview

### Core Components

1. **Main Application** (`cmd/server/main.go`)
   - Entry point for the Go application
   - Initializes configuration, database, and services
   - Sets up HTTP routes and middleware

2. **API Layer** (`internal/api/`)
   - HTTP handlers for REST API endpoints
   - Telegram bot integration
   - Authentication and authorization middleware
   - Market data, analysis, and arbitrage endpoints

3. **Services Layer** (`internal/services/`)
   - Business logic for arbitrage detection
   - Technical analysis calculations
   - Market data collection and processing
   - Signal aggregation and processing

4. **Data Models** (`internal/models/`)
   - Database entities and data structures
   - Exchange, market data, and arbitrage models
   - Technical indicator models

5. **Database Layer** (`internal/database/`)
   - PostgreSQL connection and operations
   - Redis caching integration
   - Database migrations and seeding

6. **CCXT Service** (`ccxt-service/`)
   - Bun/TypeScript service for exchange integration
   - Real-time market data collection
   - WebSocket connections for live data

### Key Design Patterns

- **Clean Architecture**: Separation of concerns with clear layers
- **Microservices**: Go backend + TypeScript CCXT service
- **Caching Strategy**: Redis for performance optimization
- **Circuit Breaker**: Fault tolerance for external services
- **Background Processing**: Workers for data collection and analysis

## Technology Stack

- **Backend**: Go 1.25+ with Gin web framework
- **Database**: PostgreSQL 15+ with Redis 7+ for caching
- **Market Data**: CCXT via Bun service for exchange integration
- **Testing**: testify, pgxmock, miniredis
- **Deployment**: Docker containers on Coolify
- **CI/CD**: GitHub Actions with automated deployment via Coolify

## Development Environment Setup

### Prerequisites
- Go 1.25 or higher
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+
- Bun 1.0+ (for CCXT service)

### Quick Start
```bash
# Clone and setup
git clone https://github.com/irfndi/celebrum-ai-go.git
cd celebrum-ai-go
cp .env.example .env

# Start development environment
make dev-setup

# Install dependencies
go mod download
make ccxt-setup

# Run the application
make run
```

## Testing Strategy

### Test Structure
- Unit tests: Individual component testing
- Integration tests: Database and external service testing
- E2E tests: Full workflow testing
- Performance tests: Load and stress testing

### Running Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test -v ./internal/services/...

# Run with race detection
go test -race ./...
```

## Configuration Management

### Environment Variables
- Database connections (PostgreSQL, Redis)
- API keys and exchange credentials
- JWT secrets and authentication
- External service configurations

### Configuration Files
- `config.yml`: Default configuration
- `.env`: Environment-specific overrides
- `docker-compose.yml`: Single Docker configuration

## Deployment Architecture

### Coolify + Docker

The project uses Coolify for deployment management:

1. **Automated Deployment**: Pushing to the repository triggers deployment via Coolify webhooks.
2. **Single Dockerfile**: The root `Dockerfile` builds both the Go backend and CCXT service.
3. **Environment Management**: Managed through Coolify's UI.


## Observability and Monitoring

### Health Checks
- `/health` endpoint for service monitoring
- Individual service health monitoring
- Database connectivity checks
- External service availability

## Security Considerations

- JWT-based authentication with middleware
- Rate limiting on API endpoints
- Environment variable management for secrets
- SSL/TLS encryption for production
- Input validation and sanitization
- Database connection pooling and security

## Performance Optimization

### Caching Strategy
- Redis for frequently accessed data
- Symbol cache for exchange information
- Blacklist cache for filtering
- Cache warming for critical data

### Database Optimization
- Connection pooling with pgx
- Query optimization and indexing
- Read/write splitting for scaling
- Backup and recovery procedures

### Resource Management
- Circuit breaker pattern for external services
- Timeout management for API calls
- Resource pooling and reuse
- Performance monitoring and alerting

## Common Development Patterns

### Error Handling
- Custom error types with context
- Structured error responses
- Graceful degradation
- Error recovery mechanisms

### Logging
- Structured JSON logging
- Correlation IDs for request tracing
- Log levels and filtering
- Centralized log aggregation

### Testing Patterns
- Table-driven tests for comprehensive coverage
- Mock implementations for external dependencies
- Integration tests with real databases
- Benchmark tests for performance critical paths

## Troubleshooting

### Common Issues
- Database connection issues: Check PostgreSQL and Redis status
- CCXT service problems: Verify Bun installation and dependencies
- Performance issues: Monitor resource usage and query performance
- Deployment failures: Check Coolify logs and server configuration

### Debug Commands
```bash
# Check logs
make logs

# Database operations
make migrate-status
make migrate
```