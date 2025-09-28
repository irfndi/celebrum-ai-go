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

# Hybrid deployment (NEW)
make docker-build-hybrid      # Build multi-stage Docker with artifact extraction
make deploy-binary            # Deploy binary artifacts to staging
make setup-staging            # Setup staging environment for hybrid deployment
make artifact-manager         # Manage deployment artifacts
make benchmark-deployment     # Benchmark deployment performance
```

### CCXT Service Commands
```bash
make ccxt-setup               # Setup CCXT Bun service
make ccxt-dev                 # Run CCXT service in development
make ccxt-build               # Build CCXT service
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
- **Observability**: OpenTelemetry integration with SigNoz
- **Circuit Breaker**: Fault tolerance for external services
- **Background Processing**: Workers for data collection and analysis

## Technology Stack

- **Backend**: Go 1.25+ with Gin web framework
- **Database**: PostgreSQL 15+ with Redis 7+ for caching
- **Market Data**: CCXT via Bun service for exchange integration
- **Testing**: testify, pgxmock, miniredis
- **Monitoring**: OpenTelemetry with SigNoz observability stack
- **Deployment**: Docker containers on Digital Ocean
- **CI/CD**: GitHub Actions with automated deployment

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
- `configs/config.yaml`: Default configuration
- `.env`: Environment-specific overrides
- Docker Compose files for different environments

## Deployment Architecture

### Hybrid Deployment Approach

The project uses a hybrid deployment strategy that combines the reproducibility of Docker builds with the efficiency of binary deployment:

#### Benefits of Hybrid Deployment
- **Reproducible Builds**: Docker ensures consistent build environments
- **Efficient Deployment**: Binary deployment reduces container overhead
- **Fast Rollbacks**: Artifact versioning enables quick rollbacks
- **Flexible Deployment**: Support for both container and binary deployment methods
- **Reduced Resource Usage**: Binaries use fewer resources than containers

#### Deployment Methods

**1. Container Deployment (Traditional)**
```bash
# Build and deploy containers
make docker-build
make deploy-staging
make deploy
```

**2. Binary Deployment (Hybrid Approach)**
```bash
# Build Docker images and extract binaries
make docker-build-hybrid

# Deploy binaries to staging
make deploy-binary

# Manage artifacts
./scripts/artifact-manager.sh store deployment.tar.gz 1.0.0 staging
./scripts/artifact-manager.sh list
./scripts/artifact-manager.sh get go-backend latest
```

#### Artifact Management

The hybrid approach includes comprehensive artifact management:

- **Storage**: Artifacts stored in `/opt/celebrum-ai/artifacts/`
- **Versioning**: Semantic versioning with environment tracking
- **Cleanup**: Automatic cleanup of old artifacts (configurable retention)
- **Metadata**: Comprehensive metadata including checksums and build information

#### Multi-Stage Docker Builds

Both services use multi-stage Docker builds:

1. **Builder Stage**: Full build environment with all dependencies
2. **Production Stage**: Optimized runtime with minimal dependencies
3. **Binary Stage**: Extracts binaries for artifact deployment

#### CI/CD Pipeline Enhancements

The updated CI/CD pipeline supports both deployment methods:

- **Parallel Builds**: Container and artifact builds run simultaneously
- **Environment-Specific Deployment**: Staging tests both approaches
- **Artifact Upload**: Automatic artifact storage and versioning
- **Rollback Support**: Quick rollback to previous artifact versions

### Production Deployment
- Zero-downtime deployments with automatic rollback
- Health monitoring for all services
- Automated recovery for failed services
- Comprehensive logging and alerting
- Backup automation before deployments

### CI/CD Pipeline
- Automatic testing on push/PR
- Docker image building and pushing
- Automated deployment to production
- Health checks and rollback on failure

### Environment Setup
- Development: `docker-compose.yml`
- Production: `docker-compose.single-droplet.yml`
- CI: `docker-compose.ci.yml`
- Staging: `setup-staging.sh` (hybrid deployment testing)

## Observability and Monitoring

### OpenTelemetry Integration
- Traces: Request tracing across services
- Metrics: Performance and business metrics
- Logs: Structured logging with correlation IDs

### SigNoz Stack
- Query service for traces and metrics
- ClickHouse for data storage
- OpenTelemetry collector
- Web UI for visualization

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
- Deployment failures: Check GitHub Actions logs and server configuration

### Debug Commands
```bash
# Check service status
make status
make status-prod

# View logs
make logs
make logs-ccxt

# Health checks
make health
make health-prod

# Database operations
make migrate-status
make migrate
```