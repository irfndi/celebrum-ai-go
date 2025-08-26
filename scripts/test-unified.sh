#!/bin/bash

# Unified Testing Script for Celebrum AI
# Consolidates functionality from test-deployment.sh and test-core-services.sh
# Supports both CI/CD validation and manual testing

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${PROJECT_ROOT}/logs/testing.log"
HEALTH_CHECK_TIMEOUT=300
HEALTH_CHECK_INTERVAL=10
TEST_TIMEOUT=60

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}[TEST]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${GREEN}[PASS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${RED}[FAIL]${NC} $1" | tee -a "$LOG_FILE"
}

# Initialize logging
init_logging() {
    mkdir -p "$(dirname "$LOG_FILE")"
    log "Starting unified testing process at $(date)"
    log "Script directory: $SCRIPT_DIR"
    log "Project root: $PROJECT_ROOT"
}

# Check prerequisites
check_prerequisites() {
    log "Checking testing prerequisites..."
    
    local missing_tools=()
    
    # Check required tools
    for tool in docker docker-compose curl jq; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        return 1
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        return 1
    fi
    
    log_success "All prerequisites satisfied"
    return 0
}

# Setup test environment
setup_test_environment() {
    local test_type="${1:-full}"
    
    log "Setting up test environment for $test_type testing..."
    
    cd "$PROJECT_ROOT"
    
    # Choose appropriate docker-compose file
    local compose_file="docker-compose.yml"
    case "$test_type" in
        "ci")
            compose_file="docker-compose.ci.yml"
            ;;
        "core")
            compose_file="docker-compose.yml"
            ;;
        "full")
            compose_file="docker-compose.yml"
            ;;
    esac
    
    if [ ! -f "$compose_file" ]; then
        log_error "Docker compose file not found: $compose_file"
        return 1
    fi
    
    export COMPOSE_FILE="$compose_file"
    log_success "Test environment configured with $compose_file"
}

# Build test services
build_test_services() {
    local test_type="${1:-full}"
    
    log "Building services for $test_type testing..."
    
    case "$test_type" in
        "ci")
            if docker-compose -f "${COMPOSE_FILE:-docker-compose.ci.yml}" build --no-cache; then
                log_success "CI services built successfully"
            else
                log_error "Failed to build CI services"
                return 1
            fi
            ;;
        "core")
            # Build only core services
            local core_services=("postgres" "redis" "app")
            for service in "${core_services[@]}"; do
                if docker-compose build "$service"; then
                    log_success "Service $service built successfully"
                else
                    log_error "Failed to build service: $service"
                    return 1
                fi
            done
            ;;
        "full")
            if docker-compose build; then
                log_success "All services built successfully"
            else
                log_error "Failed to build services"
                return 1
            fi
            ;;
    esac
}

# Start test services
start_test_services() {
    local test_type="${1:-full}"
    
    log "Starting services for $test_type testing..."
    
    case "$test_type" in
        "ci")
            if docker-compose -f "${COMPOSE_FILE:-docker-compose.ci.yml}" up -d; then
                log_success "CI services started"
            else
                log_error "Failed to start CI services"
                return 1
            fi
            ;;
        "core")
            # Start only core services
            local core_services=("postgres" "redis" "app")
            if docker-compose up -d "${core_services[@]}"; then
                log_success "Core services started"
            else
                log_error "Failed to start core services"
                return 1
            fi
            ;;
        "full")
            # Use startup orchestrator for full testing
            if bash "$SCRIPT_DIR/startup-orchestrator.sh"; then
                log_success "All services started with orchestrator"
            else
                log_error "Failed to start services with orchestrator"
                return 1
            fi
            ;;
    esac
}

# Wait for services to be healthy
wait_for_services() {
    local test_type="${1:-full}"
    local timeout="${2:-$HEALTH_CHECK_TIMEOUT}"
    
    log "Waiting for services to become healthy (timeout: ${timeout}s)..."
    
    local services_to_check=()
    case "$test_type" in
        "ci")
            services_to_check=($(docker-compose -f "${COMPOSE_FILE:-docker-compose.ci.yml}" config --services))
            ;;
        "core")
            services_to_check=("postgres" "redis" "app")
            ;;
        "full")
            services_to_check=($(docker-compose config --services))
            ;;
    esac
    
    local elapsed=0
    local all_healthy=false
    
    while [ $elapsed -lt $timeout ] && [ "$all_healthy" = false ]; do
        all_healthy=true
        
        for service in "${services_to_check[@]}"; do
            local status=$(docker-compose ps -q "$service" | xargs -r docker inspect --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
            
            if [ "$status" != "healthy" ]; then
                # Check if service is running without health check
                local running=$(docker-compose ps "$service" | grep -c "Up" || echo "0")
                if [ "$running" -eq 0 ]; then
                    all_healthy=false
                    break
                fi
            fi
        done
        
        if [ "$all_healthy" = true ]; then
            log_success "All services are healthy"
            return 0
        fi
        
        sleep $HEALTH_CHECK_INTERVAL
        elapsed=$((elapsed + HEALTH_CHECK_INTERVAL))
        log "Waiting for services... (${elapsed}s/${timeout}s)"
    done
    
    log_error "Services failed to become healthy within ${timeout}s"
    show_service_status
    return 1
}

# Show service status
show_service_status() {
    log "Current service status:"
    docker-compose ps
    
    log "Service logs (last 20 lines each):"
    local services=($(docker-compose config --services))
    for service in "${services[@]}"; do
        echo "\n=== $service logs ==="
        docker-compose logs --tail=20 "$service" 2>/dev/null || echo "No logs available for $service"
    done
}

# Test application health
test_application_health() {
    local base_url="${1:-http://localhost:8080}"
    
    log "Testing application health at $base_url..."
    
    local health_endpoints=("/health" "/api/health")
    local healthy=false
    
    for endpoint in "${health_endpoints[@]}"; do
        local url="$base_url$endpoint"
        
        if curl -sf "$url" > /dev/null 2>&1; then
            log_success "Health endpoint $endpoint is responding"
            healthy=true
            break
        else
            log_warn "Health endpoint $endpoint is not responding"
        fi
    done
    
    if [ "$healthy" = false ]; then
        log_error "No health endpoints are responding"
        return 1
    fi
    
    return 0
}

# Test database connectivity
test_database_connectivity() {
    log "Testing database connectivity..."
    
    # Test PostgreSQL
    if docker-compose exec -T postgres pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
        log_success "PostgreSQL is ready"
    else
        log_error "PostgreSQL is not ready"
        return 1
    fi
    
    # Test database connection from app
    local db_test_url="http://localhost:8080/api/health/database"
    if curl -sf "$db_test_url" > /dev/null 2>&1; then
        log_success "Database connectivity from app verified"
    else
        log_warn "Database connectivity test endpoint not available"
    fi
    
    return 0
}

# Test Redis connectivity
test_redis_connectivity() {
    log "Testing Redis connectivity..."
    
    # Test Redis directly
    if docker-compose exec -T redis redis-cli ping | grep -q PONG; then
        log_success "Redis is responding"
    else
        log_error "Redis is not responding"
        return 1
    fi
    
    # Test Redis connection from app
    local redis_test_url="http://localhost:8080/api/health/redis"
    if curl -sf "$redis_test_url" > /dev/null 2>&1; then
        log_success "Redis connectivity from app verified"
    else
        log_warn "Redis connectivity test endpoint not available"
    fi
    
    return 0
}

# Test API endpoints
test_api_endpoints() {
    local base_url="${1:-http://localhost:8080}"
    
    log "Testing API endpoints at $base_url..."
    
    local endpoints=(
        "/api/exchanges"
        "/api/trading-pairs"
        "/api/market-data"
    )
    
    local passed=0
    local total=${#endpoints[@]}
    
    for endpoint in "${endpoints[@]}"; do
        local url="$base_url$endpoint"
        
        if timeout $TEST_TIMEOUT curl -sf "$url" > /dev/null 2>&1; then
            log_success "API endpoint $endpoint is responding"
            passed=$((passed + 1))
        else
            log_warn "API endpoint $endpoint is not responding or timed out"
        fi
    done
    
    log "API endpoint test results: $passed/$total endpoints responding"
    
    if [ $passed -eq 0 ]; then
        log_error "No API endpoints are responding"
        return 1
    fi
    
    return 0
}

# Test CCXT service
test_ccxt_service() {
    log "Testing CCXT service..."
    
    # Check if CCXT service is running
    if docker-compose ps ccxt | grep -q "Up"; then
        log_success "CCXT service is running"
        
        # Test CCXT endpoints if available
        local ccxt_url="http://localhost:3000/health"
        if curl -sf "$ccxt_url" > /dev/null 2>&1; then
            log_success "CCXT service health endpoint is responding"
        else
            log_warn "CCXT service health endpoint not available"
        fi
    else
        log_warn "CCXT service is not running"
    fi
    
    return 0
}

# Test webhook functionality
test_webhook_functionality() {
    log "Testing webhook functionality..."
    
    # Test webhook control script
    if [ -f "$SCRIPT_DIR/webhook-control.sh" ]; then
        if bash "$SCRIPT_DIR/webhook-control.sh" status > /dev/null 2>&1; then
            log_success "Webhook control script is functional"
        else
            log_warn "Webhook control script test failed"
        fi
    else
        log_warn "Webhook control script not found"
    fi
    
    return 0
}

# Run comprehensive tests
run_comprehensive_tests() {
    local test_type="${1:-full}"
    local base_url="${2:-http://localhost:8080}"
    
    log "Running comprehensive tests for $test_type..."
    
    local tests_passed=0
    local tests_total=0
    
    # Test application health
    tests_total=$((tests_total + 1))
    if test_application_health "$base_url"; then
        tests_passed=$((tests_passed + 1))
    fi
    
    # Test database connectivity
    tests_total=$((tests_total + 1))
    if test_database_connectivity; then
        tests_passed=$((tests_passed + 1))
    fi
    
    # Test Redis connectivity
    tests_total=$((tests_total + 1))
    if test_redis_connectivity; then
        tests_passed=$((tests_passed + 1))
    fi
    
    # Test API endpoints (only for full tests)
    if [ "$test_type" = "full" ]; then
        tests_total=$((tests_total + 1))
        if test_api_endpoints "$base_url"; then
            tests_passed=$((tests_passed + 1))
        fi
        
        # Test CCXT service
        tests_total=$((tests_total + 1))
        if test_ccxt_service; then
            tests_passed=$((tests_passed + 1))
        fi
        
        # Test webhook functionality
        tests_total=$((tests_total + 1))
        if test_webhook_functionality; then
            tests_passed=$((tests_passed + 1))
        fi
    fi
    
    log "Test results: $tests_passed/$tests_total tests passed"
    
    if [ $tests_passed -eq $tests_total ]; then
        log_success "All tests passed!"
        return 0
    else
        log_error "Some tests failed ($((tests_total - tests_passed)) failures)"
        return 1
    fi
}

# Cleanup test environment
cleanup_test_environment() {
    local cleanup_type="${1:-stop}"
    
    log "Cleaning up test environment ($cleanup_type)..."
    
    case "$cleanup_type" in
        "stop")
            docker-compose stop
            log_success "Services stopped"
            ;;
        "down")
            docker-compose down
            log_success "Services stopped and removed"
            ;;
        "full")
            docker-compose down -v --remove-orphans
            log_success "Services, volumes, and orphans removed"
            ;;
        "none")
            log "Skipping cleanup (services left running)"
            ;;
        *)
            log_warn "Unknown cleanup type: $cleanup_type, stopping services"
            docker-compose stop
            ;;
    esac
}

# Main testing function
main() {
    local command="${1:-test}"
    local test_type="${2:-full}"
    local cleanup="${3:-stop}"
    local base_url="${4:-http://localhost:8080}"
    
    # Initialize
    init_logging
    
    case "$command" in
        "test")
            check_prerequisites
            setup_test_environment "$test_type"
            build_test_services "$test_type"
            start_test_services "$test_type"
            
            if wait_for_services "$test_type"; then
                if run_comprehensive_tests "$test_type" "$base_url"; then
                    log_success "All tests completed successfully!"
                    cleanup_test_environment "$cleanup"
                    exit 0
                else
                    log_error "Tests failed!"
                    show_service_status
                    cleanup_test_environment "$cleanup"
                    exit 1
                fi
            else
                log_error "Services failed to start properly!"
                show_service_status
                cleanup_test_environment "$cleanup"
                exit 1
            fi
            ;;
        "build")
            check_prerequisites
            setup_test_environment "$test_type"
            build_test_services "$test_type"
            log_success "Build test completed successfully!"
            ;;
        "start")
            check_prerequisites
            setup_test_environment "$test_type"
            start_test_services "$test_type"
            wait_for_services "$test_type"
            log_success "Services started successfully!"
            ;;
        "health")
            run_comprehensive_tests "$test_type" "$base_url"
            ;;
        "status")
            show_service_status
            ;;
        "cleanup")
            cleanup_test_environment "$test_type"
            log_success "Cleanup completed successfully!"
            ;;
        "help")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show help
show_help() {
    cat << EOF
Unified Testing Script for Celebrum AI

Usage: $0 [COMMAND] [TEST_TYPE] [CLEANUP] [BASE_URL]

Commands:
  test       - Run full test suite (default)
  build      - Build test services only
  start      - Start test services only
  health     - Run health tests only
  status     - Show service status
  cleanup    - Cleanup test environment
  help       - Show this help message

Test Types:
  full       - Full application testing (default)
  core       - Core services only (postgres, redis, app)
  ci         - CI/CD testing with docker-compose.ci.yml

Cleanup Options:
  stop       - Stop services (default)
  down       - Stop and remove services
  full       - Stop, remove services, volumes, and orphans
  none       - Leave services running

Examples:
  $0 test full stop
  $0 test core down
  $0 test ci full
  $0 build full
  $0 health core
  $0 status
  $0 cleanup full

Environment Variables:
  HEALTH_CHECK_TIMEOUT    - Health check timeout in seconds (default: 300)
  HEALTH_CHECK_INTERVAL   - Health check interval in seconds (default: 10)
  TEST_TIMEOUT           - Individual test timeout in seconds (default: 60)
EOF
}

# Execute main function with all arguments
main "$@"