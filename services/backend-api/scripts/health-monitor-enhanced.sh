#!/bin/bash

# Enhanced Health Monitoring Script for Celebrum AI
# Integrates with startup orchestrator and deployment processes
# Provides comprehensive health checks, monitoring, and auto-recovery

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${PROJECT_ROOT}/logs/health-monitor.log"
MONITOR_INTERVAL=30
HEALTH_CHECK_TIMEOUT=60
MAX_RESTART_ATTEMPTS=3
RESTART_COOLDOWN=60
ALERT_THRESHOLD=3

# Service configuration
SERVICES=("postgres" "celebrum")
CRITICAL_SERVICES=("postgres" "celebrum")
OPTIONAL_SERVICES=()

# Health check endpoints
declare -A HEALTH_ENDPOINTS=(
  ["celebrum"]="http://localhost:8080/health"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Logging functions
log() {
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  echo -e "${BLUE}[MONITOR]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  echo -e "${GREEN}[HEALTHY]${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  echo -e "${RED}[CRITICAL]${NC} $1" | tee -a "$LOG_FILE"
}

log_info() {
  local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
  echo -e "${MAGENTA}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

# Initialize logging
init_logging() {
  mkdir -p "$(dirname "$LOG_FILE")"
  log "Enhanced health monitoring started at $(date)"
  log "Monitor interval: ${MONITOR_INTERVAL}s"
  log "Health check timeout: ${HEALTH_CHECK_TIMEOUT}s"
  log "Max restart attempts: $MAX_RESTART_ATTEMPTS"
}

# Check if Docker is available
check_docker() {
  if ! command -v docker &>/dev/null; then
    log_error "Docker is not installed or not in PATH"
    return 1
  fi

  if ! docker info &>/dev/null; then
    log_error "Docker daemon is not running"
    return 1
  fi

  return 0
}

# Check if docker-compose is available
check_docker_compose() {
  if ! command -v docker-compose &>/dev/null; then
    log_error "docker-compose is not installed or not in PATH"
    return 1
  fi

  if [ ! -f "$PROJECT_ROOT/docker-compose.yml" ]; then
    log_error "docker-compose.yml not found in project root"
    return 1
  fi

  return 0
}

# Get service status
get_service_status() {
  local service="$1"

  # Check if service container exists and is running
  local container_id=$(docker-compose ps -q "$service" 2>/dev/null)

  if [ -z "$container_id" ]; then
    echo "not_found"
    return
  fi

  local status=$(docker inspect --format='{{.State.Status}}' "$container_id" 2>/dev/null || echo "unknown")
  local health=$(docker inspect --format='{{.State.Health.Status}}' "$container_id" 2>/dev/null || echo "none")

  if [ "$status" = "running" ]; then
    if [ "$health" = "healthy" ]; then
      echo "healthy"
    elif [ "$health" = "unhealthy" ]; then
      echo "unhealthy"
    elif [ "$health" = "starting" ]; then
      echo "starting"
    else
      echo "running"
    fi
  else
    echo "$status"
  fi
}

# Check HTTP endpoint health
check_http_endpoint() {
  local service="$1"
  local endpoint="${HEALTH_ENDPOINTS[$service]:-}"

  if [ -z "$endpoint" ]; then
    return 0 # No endpoint to check
  fi

  if timeout $HEALTH_CHECK_TIMEOUT curl -sf "$endpoint" >/dev/null 2>&1; then
    return 0
  else
    return 1
  fi
}

# Check database connectivity
check_database_health() {
  local service="postgres"

  # Check if PostgreSQL is ready
  if docker-compose exec -T "$service" pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
    return 0
  else
    return 1
  fi
}

# Comprehensive service health check
check_service_health() {
  local service="$1"
  local status=$(get_service_status "$service")

  case "$status" in
    "healthy")
      # Additional checks for specific services
      case "$service" in
        "postgres")
          if check_database_health; then
            return 0
          else
            return 1
          fi
          ;;
        "celebrum")
          if check_http_endpoint "$service"; then
            return 0
          else
            return 1
          fi
          ;;
        *)
          return 0
          ;;
      esac
      ;;
    "running")
      # Service is running but no health check defined
      case "$service" in
        "postgres")
          check_database_health
          ;;
        "celebrum")
          check_http_endpoint "$service"
          ;;
        *)
          return 0
          ;;
      esac
      ;;
    "starting")
      return 2 # Still starting
      ;;
    *)
      return 1 # Unhealthy or not running
      ;;
  esac
}

# Restart service
restart_service() {
  local service="$1"
  local attempt="${2:-1}"

  log_warn "Restarting service: $service (attempt $attempt/$MAX_RESTART_ATTEMPTS)"

  # Stop service gracefully
  if docker-compose stop "$service" 2>/dev/null; then
    log "Service $service stopped"
  else
    log_warn "Failed to stop service $service gracefully, forcing stop"
    docker-compose kill "$service" 2>/dev/null || true
  fi

  # Wait a moment
  sleep 5

  # Start service
  if docker-compose up -d "$service"; then
    log "Service $service restarted"

    # Wait for service to stabilize
    sleep $RESTART_COOLDOWN

    # Check if restart was successful
    if check_service_health "$service"; then
      log_success "Service $service restart successful"
      return 0
    else
      log_error "Service $service restart failed - still unhealthy"
      return 1
    fi
  else
    log_error "Failed to restart service: $service"
    return 1
  fi
}

# Restart all services using orchestrator
restart_all_services() {
  log_warn "Restarting all services using orchestrator..."

  # Stop all services
  docker-compose down 2>/dev/null || true

  # Wait a moment
  sleep 10

  # Start with orchestrator
  if bash "$SCRIPT_DIR/startup-orchestrator.sh"; then
    log_success "All services restarted successfully"
    return 0
  else
    log_error "Failed to restart all services"
    return 1
  fi
}

# Check system resources
check_system_resources() {
  local cpu_usage=$(top -l 1 -s 0 | grep "CPU usage" | awk '{print $3}' | sed 's/%//' || echo "0")
  local memory_usage=$(vm_stat | grep "Pages active" | awk '{print $3}' | sed 's/\.//' || echo "0")
  local disk_usage=$(df -h "$PROJECT_ROOT" | tail -1 | awk '{print $5}' | sed 's/%//' || echo "0")

  log_info "System resources - CPU: ${cpu_usage}%, Disk: ${disk_usage}%"

  # Check for resource issues
  if [ "${cpu_usage%.*}" -gt 90 ]; then
    log_warn "High CPU usage detected: ${cpu_usage}%"
  fi

  if [ "${disk_usage}" -gt 90 ]; then
    log_warn "High disk usage detected: ${disk_usage}%"
  fi
}

# Check Docker resources
check_docker_resources() {
  local docker_stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null || echo "")

  if [ -n "$docker_stats" ]; then
    log_info "Docker container resources:"
    echo "$docker_stats" | tee -a "$LOG_FILE"
  fi
}

# Monitor services
monitor_services() {
  local failed_services=()
  local warning_services=()
  local healthy_services=()

  log "Checking health of all services..."

  for service in "${SERVICES[@]}"; do
    local health_result
    if check_service_health "$service"; then
      health_result=$?
    else
      health_result=$?
    fi

    case $health_result in
      0)
        healthy_services+=("$service")
        ;;
      1)
        failed_services+=("$service")
        ;;
      2)
        warning_services+=("$service")
        ;;
    esac
  done

  # Report results
  if [ ${#healthy_services[@]} -gt 0 ]; then
    log_success "Healthy services: ${healthy_services[*]}"
  fi

  if [ ${#warning_services[@]} -gt 0 ]; then
    log_warn "Starting services: ${warning_services[*]}"
  fi

  if [ ${#failed_services[@]} -gt 0 ]; then
    log_error "Failed services: ${failed_services[*]}"

    # Attempt to restart failed services
    for service in "${failed_services[@]}"; do
      # Check if this is a critical service
      if [[ " ${CRITICAL_SERVICES[*]} " =~ " $service " ]]; then
        log_error "Critical service $service is down - attempting restart"
        restart_service "$service"
      else
        log_warn "Optional service $service is down"
      fi
    done
  fi

  return ${#failed_services[@]}
}

# Generate health report
generate_health_report() {
  local report_file="${PROJECT_ROOT}/logs/health-report-$(date +%Y%m%d_%H%M%S).json"

  log "Generating health report: $report_file"

  local report="{"
  report+="\"timestamp\": \"$(date -Iseconds)\","
  report+="\"services\": {"

  local first=true
  for service in "${SERVICES[@]}"; do
    if [ "$first" = false ]; then
      report+=","
    fi
    first=false

    local status=$(get_service_status "$service")
    local health="unknown"

    if check_service_health "$service"; then
      health="healthy"
    else
      health="unhealthy"
    fi

    report+="\"$service\": {"
    report+="\"status\": \"$status\","
    report+="\"health\": \"$health\""
    report+="}"
  done

  report+="}"
  report+="}"

  echo "$report" | jq '.' >"$report_file" 2>/dev/null || echo "$report" >"$report_file"

  log_success "Health report generated: $report_file"
}

# Show current status
show_status() {
  echo "\n=== Celebrum AI Health Status ==="
  echo "Timestamp: $(date)"
  echo "\n=== Service Status ==="

  for service in "${SERVICES[@]}"; do
    local status=$(get_service_status "$service")
    local health_icon="❓"

    case "$status" in
      "healthy")
        health_icon="✅"
        ;;
      "running")
        health_icon="🟡"
        ;;
      "starting")
        health_icon="🔄"
        ;;
      "unhealthy")
        health_icon="❌"
        ;;
      "exited")
        health_icon="🔴"
        ;;
      "not_found")
        health_icon="⚫"
        ;;
    esac

    printf "%-10s %s %s\n" "$service" "$health_icon" "$status"
  done

  echo "\n=== Docker Services ==="
  docker-compose ps 2>/dev/null || echo "Docker compose not available"

  echo "\n=== System Resources ==="
  check_system_resources

  echo "\n=== Recent Logs ==="
  if [ -f "$LOG_FILE" ]; then
    tail -10 "$LOG_FILE"
  else
    echo "No log file found"
  fi
}

# Continuous monitoring
continuous_monitor() {
  local duration="${1:-0}" # 0 means infinite
  local start_time=$(date +%s)

  log "Starting continuous monitoring (duration: ${duration}s, interval: ${MONITOR_INTERVAL}s)"

  while true; do
    # Check if duration limit reached
    if [ "$duration" -gt 0 ]; then
      local current_time=$(date +%s)
      local elapsed=$((current_time - start_time))

      if [ $elapsed -ge $duration ]; then
        log "Monitoring duration reached, stopping"
        break
      fi
    fi

    # Monitor services
    monitor_services

    # Check system resources periodically
    check_system_resources

    # Generate health report periodically (every 10 cycles)
    local cycle_count=$((($(date +%s) - start_time) / MONITOR_INTERVAL))
    if [ $((cycle_count % 10)) -eq 0 ]; then
      generate_health_report
    fi

    # Wait for next cycle
    sleep $MONITOR_INTERVAL
  done
}

# Main function
main() {
  local command="${1:-status}"
  local target="${2:-all}"
  local duration="${3:-0}"

  # Initialize
  init_logging

  # Check prerequisites
  if ! check_docker || ! check_docker_compose; then
    log_error "Prerequisites not met"
    exit 1
  fi

  cd "$PROJECT_ROOT"

  case "$command" in
    "monitor")
      continuous_monitor "$duration"
      ;;
    "check")
      if [ "$target" = "all" ]; then
        monitor_services
      else
        if check_service_health "$target"; then
          log_success "Service $target is healthy"
        else
          log_error "Service $target is unhealthy"
          exit 1
        fi
      fi
      ;;
    "restart")
      if [ "$target" = "all" ]; then
        restart_all_services
      else
        restart_service "$target"
      fi
      ;;
    "status")
      show_status
      ;;
    "report")
      generate_health_report
      ;;
    "verify")
      # Verification mode for deployment
      if monitor_services; then
        log_success "All services are healthy"
        exit 0
      else
        log_error "Some services are unhealthy"
        exit 1
      fi
      ;;
    "help")
      show_help
      ;;
    "")
      show_status
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
  cat <<EOF
Enhanced Health Monitoring Script for Celebrum AI

Usage: $0 [COMMAND] [TARGET] [DURATION]

Commands:
  monitor    - Start continuous monitoring
  check      - Check service health once
  restart    - Restart service(s)
  status     - Show current status (default)
  report     - Generate health report
  verify     - Verify all services (for deployment)
  help       - Show this help message

Targets:
  all        - All services (default)
  postgres   - PostgreSQL database
  celebrum   - Main application (Unified)

Duration (for monitor command):
  0          - Infinite monitoring (default)
  <seconds>  - Monitor for specified seconds

Examples:
  $0 status
  $0 check postgres
  $0 restart app
  $0 monitor all 3600
  $0 verify

Environment Variables:
  MONITOR_INTERVAL        - Monitoring interval in seconds (default: 30)
  HEALTH_CHECK_TIMEOUT    - Health check timeout in seconds (default: 60)
  MAX_RESTART_ATTEMPTS    - Maximum restart attempts (default: 3)
  RESTART_COOLDOWN       - Cooldown between restarts in seconds (default: 60)
EOF
}

# Execute main function with all arguments
main "$@"
