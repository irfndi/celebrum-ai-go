#!/bin/bash

# Artifact Management Script for Celebrum AI
# Handles storage, versioning, and cleanup of deployment artifacts

set -euo pipefail

# Configuration
ARTIFACTS_DIR="/opt/celebrum-ai/artifacts"
RETENTION_DAYS="${ARTIFACT_RETENTION_DAYS:-30}"
MAX_VERSIONS="${MAX_VERSIONS:-10}"
PROJECT_NAME="celebrum-ai"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR:${NC} $1"
}

warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING:${NC} $1"
}

info() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] INFO:${NC} $1"
}

# Create artifacts directory structure
setup_artifacts_dir() {
    log "Setting up artifacts directory..."
    mkdir -p "$ARTIFACTS_DIR"/{go-backend,ccxt-service,metadata}
    chmod 755 "$ARTIFACTS_DIR"
}

# Store artifact with versioning
store_artifact() {
    local artifact_type="$1"
    local source_path="$2"
    local version="$3"
    local environment="$4"
    
    if [[ ! -f "$source_path" ]]; then
        error "Artifact not found: $source_path"
        return 1
    fi
    
    local artifact_dir="$ARTIFACTS_DIR/$artifact_type/v$version"
    mkdir -p "$artifact_dir"
    
    # Copy artifact
    cp "$source_path" "$artifact_dir/"
    
    # Create metadata
    cat > "$artifact_dir/metadata.json" << EOF
{
    "project": "$PROJECT_NAME",
    "artifact_type": "$artifact_type",
    "version": "$version",
    "environment": "$environment",
    "build_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "checksum": "$(sha256sum "$source_path" | cut -d' ' -f1)",
    "size": "$(stat -c%s "$source_path") bytes"
}
EOF
    
    log "Stored $artifact_type artifact v$version ($environment)"
    
    # Create symlink for latest
    ln -sf "v$version" "$ARTIFACTS_DIR/$artifact_type/latest"
    
    return 0
}

# Store complete deployment package
store_deployment_package() {
    local package_path="$1"
    local version="$2"
    local environment="$3"
    
    if [[ ! -f "$package_path" ]]; then
        error "Deployment package not found: $package_path"
        return 1
    fi
    
    local package_dir="$ARTIFACTS_DIR/deployments/v$version"
    mkdir -p "$package_dir"
    
    # Extract package
    tar -xzf "$package_path" -C "$package_dir"
    
    # Store each artifact individually
    if [[ -f "$package_dir/go-backend/main" ]]; then
        store_artifact "go-backend" "$package_dir/go-backend/main" "$version" "$environment"
    fi
    
    if [[ -d "$package_dir/ccxt-service" ]]; then
        # Create tarball for CCXT service
        local ccxt_package="$package_dir/ccxt-service.tar.gz"
        tar -czf "$ccxt_package" -C "$package_dir" ccxt-service/
        store_artifact "ccxt-service" "$ccxt_package" "$version" "$environment"
    fi
    
    # Create deployment metadata
    cat > "$ARTIFACTS_DIR/metadata/deployment_v${version}.json" << EOF
{
    "project": "$PROJECT_NAME",
    "version": "$version",
    "environment": "$environment",
    "deployment_date": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "artifacts": {
        "go-backend": "$(find "$ARTIFACTS_DIR/go-backend/v$version" -name "main" -type f | head -1)",
        "ccxt-service": "$(find "$ARTIFACTS_DIR/ccxt-service/v$version" -name "*.tar.gz" -type f | head -1)"
    }
}
EOF
    
    log "Stored deployment package v$version ($environment)"
    return 0
}

# List available versions
list_versions() {
    local artifact_type="${1:-}"
    
    if [[ -n "$artifact_type" ]]; then
        info "Available versions for $artifact_type:"
        if [[ -d "$ARTIFACTS_DIR/$artifact_type" ]]; then
            find "$ARTIFACTS_DIR/$artifact_type" -maxdepth 1 -name "v*" -type d | sort -V | \
            while read -r version_dir; do
                version=$(basename "$version_dir")
                if [[ -f "$version_dir/metadata.json" ]]; then
                    env=$(jq -r '.environment' "$version_dir/metadata.json")
                    date=$(jq -r '.build_date' "$version_dir/metadata.json")
                    info "  $version ($env) - $date"
                fi
            done
        else
            warn "No artifacts found for type: $artifact_type"
        fi
    else
        info "Available deployment versions:"
        find "$ARTIFACTS_DIR/metadata" -name "deployment_v*.json" | sort -V | \
        while read -r metadata_file; do
            version=$(basename "$metadata_file" | sed 's/deployment_v\(.*\)\.json/\1/')
            if [[ -f "$metadata_file" ]]; then
                env=$(jq -r '.environment' "$metadata_file")
                date=$(jq -r '.deployment_date' "$metadata_file")
                info "  v$version ($env) - $date"
            fi
        done
    fi
}

# Get artifact path
get_artifact_path() {
    local artifact_type="$1"
    local version="$2"
    
    if [[ "$version" == "latest" ]]; then
        if [[ -L "$ARTIFACTS_DIR/$artifact_type/latest" ]]; then
            version=$(readlink "$ARTIFACTS_DIR/$artifact_type/latest" | sed 's/.*v\(.*\)/\1/')
        else
            error "No latest version found for $artifact_type"
            return 1
        fi
    fi
    
    local artifact_path="$ARTIFACTS_DIR/$artifact_type/v$version"
    if [[ ! -d "$artifact_path" ]]; then
        error "Artifact version v$version not found for $artifact_type"
        return 1
    fi
    
    echo "$artifact_path"
    return 0
}

# Clean up old artifacts
cleanup_old_artifacts() {
    log "Cleaning up old artifacts..."
    
    # Clean up based on retention days
    if command -v find >/dev/null 2>&1; then
        find "$ARTIFACTS_DIR" -name "v*" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} \; 2>/dev/null || true
        find "$ARTIFACTS_DIR/metadata" -name "deployment_v*.json" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true
    fi
    
    # Keep only MAX_VERSIONS for each artifact type
    for artifact_dir in "$ARTIFACTS_DIR"/*/; do
        if [[ -d "$artifact_dir" && "$(basename "$artifact_dir")" != "metadata" ]]; then
            artifact_type=$(basename "$artifact_dir")
            versions=($(find "$artifact_dir" -maxdepth 1 -name "v*" -type d | sort -V))
            
            if [[ ${#versions[@]} -gt $MAX_VERSIONS ]]; then
                remove_count=$(( ${#versions[@]} - MAX_VERSIONS ))
                for ((i=0; i<remove_count; i++)); do
                    rm -rf "${versions[i]}"
                done
                info "Removed $remove_count old versions of $artifact_type"
            fi
        fi
    done
    
    log "Cleanup completed"
}

# Show usage
usage() {
    cat << EOF
Artifact Management Script

Usage: $0 <command> [options]

Commands:
    setup                    Setup artifacts directory structure
    store <package> <version> <env>   Store deployment package
    list [type]              List available versions (optional: go-backend, ccxt-service)
    get <type> <version>      Get artifact path (version can be 'latest')
    cleanup                  Clean up old artifacts
    usage                    Show this help message

Environment Variables:
    ARTIFACT_RETENTION_DAYS   Number of days to keep artifacts (default: 30)
    MAX_VERSIONS            Maximum versions to keep per artifact (default: 10)

Examples:
    $0 setup
    $0 store deployment.tar.gz 1.0.0 staging
    $0 list go-backend
    $0 get go-backend latest
    $0 cleanup
EOF
}

# Main script logic
main() {
    case "${1:-}" in
        "setup")
            setup_artifacts_dir
            ;;
        "store")
            if [[ $# -lt 4 ]]; then
                error "Usage: $0 store <package> <version> <environment>"
                exit 1
            fi
            store_deployment_package "$2" "$3" "$4"
            ;;
        "list")
            list_versions "$2"
            ;;
        "get")
            if [[ $# -lt 3 ]]; then
                error "Usage: $0 get <artifact_type> <version>"
                exit 1
            fi
            get_artifact_path "$2" "$3"
            ;;
        "cleanup")
            cleanup_old_artifacts
            ;;
        "usage"|"-h"|"--help")
            usage
            ;;
        "")
            error "No command specified"
            usage
            exit 1
            ;;
        *)
            error "Unknown command: $1"
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"