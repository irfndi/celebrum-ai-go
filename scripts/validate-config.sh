#!/bin/bash

# Configuration Validation Script
# Validates that the configuration system is set up correctly

set -euo pipefail
trap 'error "Script failed at line $LINENO"' ERR

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

check_file() {
    local file="$1"
    local description="$2"
    
    if [[ -f "$file" ]]; then
        success "$description exists: $file"
        return 0
    else
        error "$description missing: $file"
        return 1
    fi
}

check_directory() {
    local dir="$1"
    local description="$2"
    
    if [[ -d "$dir" ]]; then
        success "$description exists: $dir"
        return 0
    else
        error "$description missing: $dir"
        return 1
    fi
}

validate_yaml() {
    local file="$1"
    local description="$2"
    
    if command -v yq &> /dev/null; then
        if yq eval '.' "$file" > /dev/null 2>&1; then
            success "$description YAML is valid"
            return 0
        else
            error "$description YAML is invalid"
            return 1
        fi
    else
        warn "yq not found, skipping YAML validation for $description"
        return 0
    fi
}

validate_docker_compose() {
    local description="$1"
    shift
    local files=("$@")
    
    if command -v docker-compose &> /dev/null; then
        if docker-compose "${files[@]}" config > /dev/null 2>&1; then
            success "$description Docker Compose is valid"
            return 0
        else
            error "$description Docker Compose is invalid"
            return 1
        fi
    else
        warn "docker-compose not found, skipping validation for $description"
        return 0
    fi
}

# Main validation
main() {
    log "Starting configuration validation..."
    
    cd "$PROJECT_ROOT"
    
    local errors=0
    
    # Check required directories
    check_directory "configs" "Configuration directory" || ((errors++))
    check_directory "scripts" "Scripts directory" || ((errors++))
    
    # Check configuration templates
    check_file "configs/config.template.yml" "Configuration template" || ((errors++))
    check_file ".env.template" "Environment template" || ((errors++))
    
    # Check environment-specific configurations
    check_file "configs/config.local.yml" "Development configuration" || ((errors++))
    check_file "configs/config.ci.yml" "CI configuration" || ((errors++))
    check_file "configs/config.prod.yml" "Production configuration" || ((errors++))
    check_file "configs/config.yml" "Base configuration" || ((errors++))
    
    # Check Docker Compose files
    check_file "docker-compose.yml" "Base Docker Compose" || ((errors++))
    check_file "docker-compose.override.yml" "Development Docker Compose" || ((errors++))
    check_file "docker-compose.staging.yml" "Staging Docker Compose" || ((errors++))
    check_file "docker-compose.prod.yml" "Production Docker Compose" || ((errors++))
    check_file "docker-compose.single-droplet.yml" "Single Droplet Docker Compose" || ((errors++))
    
    # Validate YAML files
    validate_yaml "configs/config.local.yml" "Development config" || ((errors++))
    validate_yaml "configs/config.ci.yml" "CI config" || ((errors++))
    validate_yaml "configs/config.prod.yml" "Production config" || ((errors++))
    
    # Validate Docker Compose files
    validate_docker_compose "Base" "docker-compose.yml" || ((errors++))
    validate_docker_compose "Development" "-f" "docker-compose.yml" "-f" "docker-compose.override.yml" || ((errors++))
    validate_docker_compose "Staging" "-f" "docker-compose.yml" "-f" "docker-compose.staging.yml" || ((errors++))
    validate_docker_compose "Production" "-f" "docker-compose.yml" "-f" "docker-compose.prod.yml" || ((errors++))
    
    # Check scripts
    check_file "scripts/setup-environment.sh" "Setup script" || ((errors++))
    check_file "scripts/deploy-enhanced.sh" "Deployment script" || ((errors++))
    check_file "scripts/deploy-manual.sh" "Manual deployment script" || ((errors++))
    
    # Check if setup script is executable
    if [[ -x "scripts/setup-environment.sh" ]]; then
        success "Setup script is executable"
    else
        error "Setup script is not executable"
        ((errors++))
    fi
    
    # Check Makefile targets
    log "Checking Makefile targets..."
    if grep -q "setup-dev:" Makefile; then
        success "Development setup target exists"
    else
        error "Development setup target missing"
        ((errors++))
    fi
    
    if grep -q "setup-prod:" Makefile; then
        success "Production setup target exists"
    else
        error "Production setup target missing"
        ((errors++))
    fi
    
    # Summary
    log "Validation complete!"
    
    if [[ $errors -eq 0 ]]; then
        success "All configuration checks passed!"
        log "Next steps:"
        log "1. Run: ./scripts/setup-environment.sh -e development"
        log "2. Run: make dev-up"
        log "3. Check: curl http://localhost:8080/health"
        exit 0
    else
        error "Found $errors configuration issues. Please fix them before proceeding."
        exit 1
    fi
}

# Run main function
main "$@"