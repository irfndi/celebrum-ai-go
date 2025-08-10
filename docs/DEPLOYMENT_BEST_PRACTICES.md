# Deployment Best Practices & CI/CD Improvements

## Current Deployment Strategy Analysis

### Manual Rsync Deployment (Current)
**Pros:**
- Direct control over deployment process
- Immediate feedback on deployment issues
- No GitHub Actions complexity
- Flexible for hotfixes

**Cons:**
- No automated testing before deployment
- Manual intervention required for every deployment
- Risk of human error
- No rollback mechanism
- No deployment history/tracking

### Recommended CI/CD Improvements

## 1. Hybrid Approach: GitHub Actions + Manual Control

### Strategy
Use GitHub Actions for automated testing and building, but maintain manual deployment control for production.

### Implementation

#### GitHub Actions Workflow (`.github/workflows/deploy-manual.yml`)
```yaml
name: Build and Prepare Deployment

on:
  push:
    branches: [main, development]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'
          cache: 'npm'
          cache-dependency-path: ccxt-service/package-lock.json
          
      - name: Run tests
        run: |
          make test
          cd ccxt-service && npm ci && npm test
          
      - name: Build Docker images
        run: |
          docker-compose -f docker-compose.single-droplet.yml build
          
      - name: Create deployment artifacts
        run: |
          make create-deployment-package
          
      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: deployment-package-${{ github.sha }}
          path: deployment-package.tar.gz
          retention-days: 30
```

#### Manual Deployment Script (`scripts/deploy.sh`)
```bash
#!/bin/bash
set -e

# Usage: ./scripts/deploy.sh [environment] [commit-sha]
ENVIRONMENT=${1:-production}
COMMIT_SHA=${2:-$(git rev-parse HEAD)}

echo "🚀 Starting deployment for commit: $COMMIT_SHA"

# Download artifacts from GitHub Actions
gh run download --name deployment-package-$COMMIT_SHA

# Pre-deployment checks
./scripts/pre-deploy-checks.sh

# Backup current deployment
./scripts/backup-current.sh

# Deploy new version
rsync -avz --exclude='.git' --exclude='node_modules' \
  deployment-package/ root@$SERVER_IP:/root/celebrum-ai-go/

# Post-deployment validation
./scripts/post-deploy-checks.sh

echo "✅ Deployment completed successfully!"
```

## 2. Enhanced Service Health Monitoring

### Health Check Endpoints
Add to your Go application:

```go
// internal/api/handlers/health.go
package handlers

import (
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]string)
	
	// Check database
	if err := h.db.Ping(); err != nil {
		services["database"] = "unhealthy"
	} else {
		services["database"] = "healthy"
	}
	
	// Check Redis
	if _, err := h.redis.Ping().Result(); err != nil {
		services["redis"] = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}
	
	// Check CCXT service
	if err := h.checkCCXTService(); err != nil {
		services["ccxt"] = "unhealthy"
	} else {
		services["ccxt"] = "healthy"
	}
	
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Services:  services,
		Version:   os.Getenv("APP_VERSION"),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
```

### Service Restart Script (`scripts/health-restart.sh`)
```bash
#!/bin/bash
# Automatic service recovery script

SERVICES=("celebrum-app" "celebrum-ccxt" "celebrum-redis" "celebrum-postgres")
HEALTH_CHECK_URL="http://localhost/api/v1/health"

for service in "${SERVICES[@]}"; do
    if ! docker-compose ps | grep -q "$service.*Up"; then
        echo "🔴 $service is down, restarting..."
        docker-compose restart $service
    fi
done

# Check application health
if ! curl -f $HEALTH_CHECK_URL > /dev/null 2>&1; then
    echo "🔴 Application health check failed, restarting all services..."
    docker-compose restart
fi
```

## 3. Zero-Downtime Deployment Strategy

### Blue-Green Deployment Script
```bash
#!/bin/bash
# scripts/zero-downtime-deploy.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
NEW_CONTAINER="celebrum-app-$TIMESTAMP"

# Build new image
docker build -t celebrum-app:$TIMESTAMP .

# Start new container
docker-compose -f docker-compose.blue-green.yml up -d celebrum-app-$TIMESTAMP

# Health check new container
./scripts/wait-for-healthy.sh celebrum-app-$TIMESTAMP

# Switch traffic
./scripts/switch-traffic.sh celebrum-app-$TIMESTAMP

# Stop old container
docker-compose stop celebrum-app
```

## 4. Monitoring & Alerting Setup

### Docker Health Checks
Add to `docker-compose.single-droplet.yml`:

```yaml
services:
  api-server:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    restart: unless-stopped
    
  ccxt-service:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://ccxt-service:3001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
    restart: unless-stopped
```

### Monitoring Dashboard
Create `scripts/monitor.sh`:
```bash
#!/bin/bash
# Simple monitoring dashboard

while true; do
    clear
    echo "=== Celebrum AI Service Status ==="
    echo "$(date)"
    echo
    
    docker-compose ps
    echo
    
    echo "=== Service Health ==="
    curl -s http://localhost/api/v1/health | jq .
    
    echo "=== Recent Logs ==="
    docker-compose logs --tail=10
    
    sleep 30
done
```

## 5. Rollback Strategy

### Quick Rollback Script (`scripts/rollback.sh`)
```bash
#!/bin/bash
set -e

LAST_GOOD_VERSION=${1:-$(cat .last-good-version)}

echo "🔄 Rolling back to version: $LAST_GOOD_VERSION"

# Stop current services
docker-compose down

# Restore from backup
rsync -avz --delete \
  backups/$LAST_GOOD_VERSION/ /root/celebrum-ai-go/

# Restart services
docker-compose -f docker-compose.single-droplet.yml up -d

echo "✅ Rollback completed!"
```

## 6. Environment Management

### Environment-Specific Configurations
Create `config/environments/`:

```
config/
├── environments/
│   ├── development.yml
│   ├── staging.yml
│   └── production.yml
├── nginx/
│   ├── development.conf
│   ├── staging.conf
│   └── production.conf
└── scripts/
    ├── deploy-dev.sh
    ├── deploy-staging.sh
    └── deploy-production.sh
```

### Environment Variables Validation
Create `scripts/validate-env.sh`:
```bash
#!/bin/bash
# Validate required environment variables

REQUIRED_VARS=(
    "TELEGRAM_BOT_TOKEN"
    "JWT_SECRET"
    "DATABASE_URL"
    "REDIS_URL"
    "CCXT_SERVICE_URL"
)

for var in "${REQUIRED_VARS[@]}"; do
    if [[ -z "${!var}" ]]; then
        echo "❌ Missing required environment variable: $var"
        exit 1
    fi
done

echo "✅ All required environment variables are set"
```

## 7. Recommended Next Steps

### Immediate (Week 1)
1. ✅ Add health check endpoints to the application
2. ✅ Create service restart scripts
3. ✅ Implement environment validation

### Short-term (Week 2-3)
1. ✅ Set up GitHub Actions for automated testing
2. ✅ Create deployment artifacts
3. ✅ Implement monitoring dashboard

### Long-term (Month 1-2)
1. ✅ Set up blue-green deployment
2. ✅ Add automated rollback
3. ✅ Implement alerting (Slack/Discord notifications)
4. ✅ Add performance monitoring

## 8. Deployment Checklist

### Pre-deployment
- [ ] All tests passing
- [ ] Environment variables validated
- [ ] Database migrations tested
- [ ] Health checks implemented

### During deployment
- [ ] Services started in correct order
- [ ] Health checks passing
- [ ] Telegram webhook responding
- [ ] All endpoints accessible

### Post-deployment
- [ ] Monitor logs for 30 minutes
- [ ] Verify Telegram bot functionality
- [ ] Check service resource usage
- [ ] Update deployment documentation