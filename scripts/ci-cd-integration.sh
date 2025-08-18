#!/bin/bash

# CI/CD Integration Script for Celebrum AI
# Provides seamless integration for automated deployment pipelines
# Supports GitHub Actions, Jenkins, and other CI/CD platforms

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="${PROJECT_ROOT}/logs/ci-cd.log"
CI_ARTIFACTS_DIR="${PROJECT_ROOT}/ci-artifacts"
TEST_RESULTS_DIR="${CI_ARTIFACTS_DIR}/test-results"
COVERAGE_DIR="${CI_ARTIFACTS_DIR}/coverage"
REPORTS_DIR="${CI_ARTIFACTS_DIR}/reports"

# CI/CD Environment Detection
CI_PLATFORM="unknown"
CI_BUILD_ID="${BUILD_ID:-${GITHUB_RUN_ID:-unknown}}"
CI_COMMIT_SHA="${GIT_COMMIT:-${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo 'unknown')}}"
CI_BRANCH="${GIT_BRANCH:-${GITHUB_REF_NAME:-$(git branch --show-current 2>/dev/null || echo 'unknown')}}"
CI_PR_NUMBER="${CHANGE_ID:-${GITHUB_PR_NUMBER:-}}"
CI_ENVIRONMENT="${DEPLOY_ENV:-${GITHUB_ENVIRONMENT:-staging}}"

# Deployment configuration
DEPLOYMENT_TYPE="${DEPLOYMENT_TYPE:-standard}"
RUN_TESTS="${RUN_TESTS:-true}"
RUN_SECURITY_SCAN="${RUN_SECURITY_SCAN:-true}"
RUN_PERFORMANCE_TEST="${RUN_PERFORMANCE_TEST:-false}"
SKIP_BACKUP="${SKIP_BACKUP:-false}"
AUTO_ROLLBACK="${AUTO_ROLLBACK:-true}"
NOTIFY_SLACK="${NOTIFY_SLACK:-false}"
SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
log() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}[CI/CD]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warn() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

log_info() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${MAGENTA}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_stage() {
    local stage="$1"
    local message="$2"
    echo -e "${CYAN}[STAGE: $stage]${NC} $message" | tee -a "$LOG_FILE"
}

# Initialize CI/CD environment
init_ci_environment() {
    mkdir -p "$(dirname "$LOG_FILE")"
    mkdir -p "$CI_ARTIFACTS_DIR"
    mkdir -p "$TEST_RESULTS_DIR"
    mkdir -p "$COVERAGE_DIR"
    mkdir -p "$REPORTS_DIR"
    
    # Detect CI platform
    detect_ci_platform
    
    log "CI/CD Integration initialized"
    log "Platform: $CI_PLATFORM"
    log "Build ID: $CI_BUILD_ID"
    log "Commit: $CI_COMMIT_SHA"
    log "Branch: $CI_BRANCH"
    log "Environment: $CI_ENVIRONMENT"
    
    if [ -n "$CI_PR_NUMBER" ]; then
        log "Pull Request: $CI_PR_NUMBER"
    fi
}

# Detect CI/CD platform
detect_ci_platform() {
    if [ -n "${GITHUB_ACTIONS:-}" ]; then
        CI_PLATFORM="github-actions"
    elif [ -n "${JENKINS_URL:-}" ]; then
        CI_PLATFORM="jenkins"
    elif [ -n "${CIRCLECI:-}" ]; then
        CI_PLATFORM="circleci"
    elif [ -n "${TRAVIS:-}" ]; then
        CI_PLATFORM="travis"
    elif [ -n "${BUILDKITE:-}" ]; then
        CI_PLATFORM="buildkite"
    else
        CI_PLATFORM="unknown"
    fi
}

# Set CI-specific environment variables
set_ci_environment() {
    case "$CI_PLATFORM" in
        "github-actions")
            export GITHUB_OUTPUT="${GITHUB_OUTPUT:-/dev/stdout}"
            export GITHUB_STEP_SUMMARY="${GITHUB_STEP_SUMMARY:-/dev/stdout}"
            ;;
        "jenkins")
            export BUILD_URL="${BUILD_URL:-}"
            export JOB_URL="${JOB_URL:-}"
            ;;
    esac
}

# Run pre-deployment checks
run_pre_deployment_checks() {
    log_stage "pre-deployment" "Running pre-deployment checks"
    
    # Check if this is a deployable branch/environment
    if ! is_deployable_ref; then
        log_info "Skipping deployment for non-deployable reference: $CI_BRANCH"
        return 1
    fi
    
    # Check for required environment variables
    check_required_env_vars
    
    # Validate deployment configuration
    validate_deployment_config
    
    log_success "Pre-deployment checks passed"
    return 0
}

# Check if current ref is deployable
is_deployable_ref() {
    local deployable_branches=("main" "master" "develop" "staging" "production")
    local deployable_patterns=("release/*" "hotfix/*")
    
    # Check exact branch matches
    for branch in "${deployable_branches[@]}"; do
        if [ "$CI_BRANCH" = "$branch" ]; then
            return 0
        fi
    done
    
    # Check pattern matches
    for pattern in "${deployable_patterns[@]}"; do
        if [[ "$CI_BRANCH" == $pattern ]]; then
            return 0
        fi
    done
    
    # Check if it's a tag
    if git describe --exact-match --tags HEAD &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Check required environment variables
check_required_env_vars() {
    local required_vars=()
    
    case "$CI_ENVIRONMENT" in
        "production")
            required_vars+=("DATABASE_URL" "REDIS_URL" "SECRET_KEY")
            ;;
        "staging")
            required_vars+=("DATABASE_URL" "REDIS_URL")
            ;;
    esac
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var:-}" ]; then
            log_error "Required environment variable not set: $var"
            return 1
        fi
    done
    
    return 0
}

# Validate deployment configuration
validate_deployment_config() {
    # Check if deployment type is valid
    case "$DEPLOYMENT_TYPE" in
        "standard"|"zero-downtime"|"rolling")
            log_info "Deployment type: $DEPLOYMENT_TYPE"
            ;;
        *)
            log_error "Invalid deployment type: $DEPLOYMENT_TYPE"
            return 1
            ;;
    esac
    
    # Validate environment-specific settings
    case "$CI_ENVIRONMENT" in
        "production")
            if [ "$SKIP_BACKUP" = "true" ]; then
                log_error "Backup cannot be skipped in production"
                return 1
            fi
            ;;
    esac
    
    return 0
}

# Run comprehensive tests
run_comprehensive_tests() {
    if [ "$RUN_TESTS" != "true" ]; then
        log_info "Tests skipped (RUN_TESTS=false)"
        return 0
    fi
    
    log_stage "testing" "Running comprehensive tests"
    
    # Unit tests
    run_unit_tests
    
    # Integration tests
    run_integration_tests
    
    # End-to-end tests
    run_e2e_tests
    
    # Generate test report
    generate_test_report
    
    log_success "All tests completed"
    return 0
}

# Run unit tests
run_unit_tests() {
    log "Running unit tests..."
    
    if bash "$SCRIPT_DIR/test-unified.sh" test unit; then
        log_success "Unit tests passed"
        
        # Copy test results
        cp "$PROJECT_ROOT/test-results/unit.xml" "$TEST_RESULTS_DIR/" 2>/dev/null || true
        
        return 0
    else
        log_error "Unit tests failed"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    log "Running integration tests..."
    
    if bash "$SCRIPT_DIR/test-unified.sh" test core; then
        log_success "Integration tests passed"
        
        # Copy test results
        cp "$PROJECT_ROOT/test-results/integration.xml" "$TEST_RESULTS_DIR/" 2>/dev/null || true
        
        return 0
    else
        log_error "Integration tests failed"
        return 1
    fi
}

# Run end-to-end tests
run_e2e_tests() {
    log "Running end-to-end tests..."
    
    if bash "$SCRIPT_DIR/test-unified.sh" test full; then
        log_success "End-to-end tests passed"
        
        # Copy test results
        cp "$PROJECT_ROOT/test-results/e2e.xml" "$TEST_RESULTS_DIR/" 2>/dev/null || true
        
        return 0
    else
        log_error "End-to-end tests failed"
        return 1
    fi
}

# Generate test report
generate_test_report() {
    log "Generating test report..."
    
    local report_file="$REPORTS_DIR/test-report.html"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Test Report - Build $CI_BUILD_ID</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .success { color: #28a745; }
        .error { color: #dc3545; }
        .warning { color: #ffc107; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Test Report</h1>
        <p><strong>Build ID:</strong> $CI_BUILD_ID</p>
        <p><strong>Commit:</strong> $CI_COMMIT_SHA</p>
        <p><strong>Branch:</strong> $CI_BRANCH</p>
        <p><strong>Environment:</strong> $CI_ENVIRONMENT</p>
        <p><strong>Timestamp:</strong> $(date)</p>
    </div>
    
    <h2>Test Results</h2>
    <table>
        <tr><th>Test Suite</th><th>Status</th><th>Results File</th></tr>
EOF
    
    # Add test results to report
    for test_file in "$TEST_RESULTS_DIR"/*.xml; do
        if [ -f "$test_file" ]; then
            local test_name=$(basename "$test_file" .xml)
            echo "        <tr><td>$test_name</td><td class=\"success\">PASSED</td><td>$(basename "$test_file")</td></tr>" >> "$report_file"
        fi
    done
    
    cat >> "$report_file" << EOF
    </table>
</body>
</html>
EOF
    
    log_success "Test report generated: $report_file"
}

# Run security scan
run_security_scan() {
    if [ "$RUN_SECURITY_SCAN" != "true" ]; then
        log_info "Security scan skipped (RUN_SECURITY_SCAN=false)"
        return 0
    fi
    
    log_stage "security" "Running security scan"
    
    # Docker image security scan
    run_docker_security_scan
    
    # Dependency vulnerability scan
    run_dependency_scan
    
    # Configuration security check
    run_config_security_check
    
    log_success "Security scan completed"
    return 0
}

# Run Docker security scan
run_docker_security_scan() {
    log "Running Docker security scan..."
    
    # Use trivy if available
    if command -v trivy &> /dev/null; then
        trivy image --format json --output "$REPORTS_DIR/docker-security.json" celebrum-ai:latest || true
        log_success "Docker security scan completed"
    else
        log_warn "Trivy not available, skipping Docker security scan"
    fi
}

# Run dependency vulnerability scan
run_dependency_scan() {
    log "Running dependency vulnerability scan..."
    
    # Check for known vulnerabilities in dependencies
    if [ -f "$PROJECT_ROOT/go.mod" ]; then
        # Go vulnerability check
        if command -v govulncheck &> /dev/null; then
            govulncheck ./... > "$REPORTS_DIR/go-vulnerabilities.txt" 2>&1 || true
            log_success "Go vulnerability scan completed"
        fi
    fi
    
    if [ -f "$PROJECT_ROOT/package.json" ]; then
        # Node.js vulnerability check
        npm audit --json > "$REPORTS_DIR/npm-audit.json" 2>/dev/null || true
        log_success "NPM audit completed"
    fi
}

# Run configuration security check
run_config_security_check() {
    log "Running configuration security check..."
    
    local security_report="$REPORTS_DIR/security-config.txt"
    
    {
        echo "Configuration Security Check - $(date)"
        echo "==========================================="
        echo
        
        # Check for exposed secrets
        echo "Checking for exposed secrets..."
        if git log --all --full-history -- "**/*.env*" | grep -i "password\|secret\|key\|token" || true; then
            echo "WARNING: Potential secrets found in git history"
        fi
        
        # Check file permissions
        echo "\nChecking file permissions..."
        find "$PROJECT_ROOT" -name "*.env*" -exec ls -la {} \; 2>/dev/null || true
        
        # Check Docker configuration
        echo "\nChecking Docker configuration..."
        if [ -f "$PROJECT_ROOT/Dockerfile" ]; then
            if grep -q "USER root" "$PROJECT_ROOT/Dockerfile"; then
                echo "WARNING: Running as root user in Docker"
            fi
        fi
        
    } > "$security_report"
    
    log_success "Configuration security check completed"
}

# Run performance tests
run_performance_tests() {
    if [ "$RUN_PERFORMANCE_TEST" != "true" ]; then
        log_info "Performance tests skipped (RUN_PERFORMANCE_TEST=false)"
        return 0
    fi
    
    log_stage "performance" "Running performance tests"
    
    # Load testing
    run_load_tests
    
    # Resource usage monitoring
    monitor_resource_usage
    
    log_success "Performance tests completed"
    return 0
}

# Run load tests
run_load_tests() {
    log "Running load tests..."
    
    # Use k6 if available
    if command -v k6 &> /dev/null && [ -f "$PROJECT_ROOT/tests/load/script.js" ]; then
        k6 run --out json="$REPORTS_DIR/load-test.json" "$PROJECT_ROOT/tests/load/script.js" || true
        log_success "Load tests completed"
    else
        log_warn "k6 or load test script not available, skipping load tests"
    fi
}

# Monitor resource usage
monitor_resource_usage() {
    log "Monitoring resource usage..."
    
    local resource_report="$REPORTS_DIR/resource-usage.txt"
    
    {
        echo "Resource Usage Report - $(date)"
        echo "==================================="
        echo
        
        echo "Docker Container Stats:"
        docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" 2>/dev/null || echo "No running containers"
        
        echo "\nSystem Resources:"
        echo "CPU Usage: $(top -l 1 -s 0 | grep "CPU usage" | awk '{print $3}' | sed 's/%//' 2>/dev/null || echo 'N/A')"
        echo "Memory Usage: $(vm_stat | grep "Pages free" | awk '{print $3}' | sed 's/\.//' 2>/dev/null || echo 'N/A')"
        echo "Disk Usage: $(df -h / | tail -1 | awk '{print $5}' 2>/dev/null || echo 'N/A')"
        
    } > "$resource_report"
    
    log_success "Resource usage monitoring completed"
}

# Deploy application
deploy_application() {
    log_stage "deployment" "Deploying application"
    
    # Set deployment environment variables
    export DEPLOYMENT_TYPE="$DEPLOYMENT_TYPE"
    export ENVIRONMENT="$CI_ENVIRONMENT"
    export BRANCH="$CI_BRANCH"
    
    if [ "$SKIP_BACKUP" = "true" ]; then
        export SKIP_BACKUP="true"
    fi
    
    # Run deployment
    if bash "$SCRIPT_DIR/deploy-master.sh" deploy; then
        log_success "Application deployed successfully"
        
        # Generate deployment report
        generate_deployment_report
        
        return 0
    else
        log_error "Application deployment failed"
        
        # Auto-rollback if enabled
        if [ "$AUTO_ROLLBACK" = "true" ]; then
            log_warn "Attempting automatic rollback..."
            if bash "$SCRIPT_DIR/deploy-master.sh" rollback; then
                log_success "Automatic rollback completed"
            else
                log_error "Automatic rollback failed"
            fi
        fi
        
        return 1
    fi
}

# Generate deployment report
generate_deployment_report() {
    log "Generating deployment report..."
    
    local report_file="$REPORTS_DIR/deployment-report.json"
    
    cat > "$report_file" << EOF
{
    "deployment_id": "$CI_BUILD_ID",
    "timestamp": "$(date -Iseconds)",
    "environment": "$CI_ENVIRONMENT",
    "deployment_type": "$DEPLOYMENT_TYPE",
    "commit": "$CI_COMMIT_SHA",
    "branch": "$CI_BRANCH",
    "platform": "$CI_PLATFORM",
    "status": "success",
    "artifacts": {
        "test_results": "$TEST_RESULTS_DIR",
        "coverage": "$COVERAGE_DIR",
        "reports": "$REPORTS_DIR"
    }
}
EOF
    
    log_success "Deployment report generated: $report_file"
}

# Send notifications
send_notifications() {
    local status="$1"
    local message="$2"
    
    # Slack notification
    if [ "$NOTIFY_SLACK" = "true" ] && [ -n "$SLACK_WEBHOOK_URL" ]; then
        send_slack_notification "$status" "$message"
    fi
    
    # Platform-specific notifications
    case "$CI_PLATFORM" in
        "github-actions")
            send_github_notification "$status" "$message"
            ;;
    esac
}

# Send Slack notification
send_slack_notification() {
    local status="$1"
    local message="$2"
    
    local color="good"
    if [ "$status" = "failure" ]; then
        color="danger"
    elif [ "$status" = "warning" ]; then
        color="warning"
    fi
    
    local payload="{
        \"attachments\": [{
            \"color\": \"$color\",
            \"title\": \"Celebrum AI Deployment\",
            \"text\": \"$message\",
            \"fields\": [
                {\"title\": \"Environment\", \"value\": \"$CI_ENVIRONMENT\", \"short\": true},
                {\"title\": \"Branch\", \"value\": \"$CI_BRANCH\", \"short\": true},
                {\"title\": \"Commit\", \"value\": \"$CI_COMMIT_SHA\", \"short\": true},
                {\"title\": \"Build ID\", \"value\": \"$CI_BUILD_ID\", \"short\": true}
            ]
        }]
    }"
    
    curl -X POST -H 'Content-type: application/json' --data "$payload" "$SLACK_WEBHOOK_URL" &>/dev/null || true
}

# Send GitHub notification
send_github_notification() {
    local status="$1"
    local message="$2"
    
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
        cat >> "$GITHUB_STEP_SUMMARY" << EOF
## Deployment Summary

- **Status**: $status
- **Environment**: $CI_ENVIRONMENT
- **Branch**: $CI_BRANCH
- **Commit**: $CI_COMMIT_SHA
- **Message**: $message

### Artifacts
- [Test Results]($TEST_RESULTS_DIR)
- [Coverage Report]($COVERAGE_DIR)
- [Deployment Reports]($REPORTS_DIR)
EOF
    fi
}



# Cleanup CI artifacts
cleanup_ci_artifacts() {
    log "Cleaning up CI artifacts..."
    
    # Keep only last 10 builds
    find "$CI_ARTIFACTS_DIR" -maxdepth 1 -type d -name "build_*" | sort -r | tail -n +11 | xargs rm -rf 2>/dev/null || true
    
    # Compress old reports
    find "$REPORTS_DIR" -name "*.txt" -o -name "*.json" -o -name "*.html" | head -20 | tar -czf "$CI_ARTIFACTS_DIR/old-reports-$(date +%Y%m%d).tar.gz" -T - 2>/dev/null || true
    
    log_success "CI artifacts cleanup completed"
}

# Main CI/CD pipeline
main() {
    local command="${1:-pipeline}"
    
    # Initialize CI environment
    init_ci_environment
    set_ci_environment
    
    case "$command" in
        "pipeline")
            # Full CI/CD pipeline
            local pipeline_status="success"
            local pipeline_message="Deployment completed successfully"
            
            if run_pre_deployment_checks; then
                if run_comprehensive_tests && \
                   run_security_scan && \
                   run_performance_tests && \
                   deploy_application; then
                    
                    log_success "CI/CD pipeline completed successfully"
                    send_notifications "success" "$pipeline_message"
                else
                    pipeline_status="failure"
                    pipeline_message="Deployment failed"
                    log_error "CI/CD pipeline failed"
                    send_notifications "failure" "$pipeline_message"
                fi
            else
                pipeline_status="skipped"
                pipeline_message="Deployment skipped (non-deployable reference)"
                log_info "CI/CD pipeline skipped"
                send_notifications "warning" "$pipeline_message"
            fi
            
            cleanup_ci_artifacts
            
            if [ "$pipeline_status" = "success" ]; then
                exit 0
            else
                exit 1
            fi
            ;;
        "test")
            run_comprehensive_tests
            ;;
        "security")
            run_security_scan
            ;;
        "performance")
            run_performance_tests
            ;;
        "deploy")
            deploy_application
            ;;
        "notify")
            send_notifications "${2:-info}" "${3:-Manual notification}"
            ;;
        "cleanup")
            cleanup_ci_artifacts
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
CI/CD Integration Script for Celebrum AI

Usage: $0 [COMMAND] [ARGS...]

Commands:
  pipeline    - Run full CI/CD pipeline (default)
  test        - Run comprehensive tests only
  security    - Run security scan only
  performance - Run performance tests only
  deploy      - Deploy application only
  notify      - Send notification
  cleanup     - Clean up CI artifacts
  help        - Show this help message

Environment Variables:
  DEPLOYMENT_TYPE      - Deployment strategy (standard|zero-downtime|rolling)
  CI_ENVIRONMENT       - Target environment (production|staging|development)
  RUN_TESTS           - Run tests (true|false)
  RUN_SECURITY_SCAN   - Run security scan (true|false)
  RUN_PERFORMANCE_TEST - Run performance tests (true|false)
  SKIP_BACKUP         - Skip backup creation (true|false)
  AUTO_ROLLBACK       - Enable automatic rollback (true|false)
  NOTIFY_SLACK        - Send Slack notifications (true|false)
  SLACK_WEBHOOK_URL   - Slack webhook URL for notifications

Supported CI/CD Platforms:
  - GitHub Actions
  - Jenkins
  - CircleCI
  - Travis CI
  - Buildkite

Examples:
  $0 pipeline
  $0 test
  RUN_TESTS=false $0 deploy
  DEPLOYMENT_TYPE=zero-downtime CI_ENVIRONMENT=production $0 pipeline
EOF
}

# Execute main function with all arguments
main "$@"