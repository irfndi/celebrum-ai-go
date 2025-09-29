# Celebrum AI Deployment Scripts

This directory contains the optimized deployment and operations scripts for Celebrum AI. The scripts have been consolidated and enhanced to provide a streamlined, reliable deployment experience.

## Core Scripts

### 🚀 Deployment
- **`deploy-master.sh`** - Master deployment orchestrator
  - Coordinates the entire deployment process
  - Supports multiple deployment strategies (standard, zero-downtime, rolling)
  - Includes backup, migration, build, and verification phases
  - Usage: `./deploy-master.sh [deploy|rollback|status|verify|backup|cleanup|help]`

### 🏗️ Service Management
- **`startup-orchestrator.sh`** - 4-phase service startup orchestrator
  - Phase 1: Infrastructure (postgres, redis)
  - Phase 2: Core services (app)
  - Phase 3: Supporting services (ccxt)
  - Phase 4: Proxy services (nginx)
  - Usage: `./startup-orchestrator.sh [start|stop|restart|status|help]`

### 🔍 Health Monitoring
- **`health-monitor-enhanced.sh`** - Comprehensive health monitoring
  - Real-time service health checks
  - Auto-recovery capabilities
  - Resource monitoring
  - Health reporting
  - Usage: `./health-monitor-enhanced.sh [monitor|check|restart|status|report|verify|help]`

### 🧪 Testing
- **`test-unified.sh`** - Unified testing framework
  - Consolidates all testing scenarios
  - Supports full, core, and CI testing modes
  - Automated cleanup options
  - Usage: `./test-unified.sh [test|build|start|health|status|cleanup|help]`

### 🗄️ Database Management
- **`migrate-optimized.sh`** - Enhanced database migration
  - Transaction safety with rollback capabilities
  - Automatic backup creation
  - Migration validation
  - Schema verification
  - Usage: `./migrate-optimized.sh [migrate|status|verify|rollback|backup|create|help]`

### 🔄 CI/CD Integration
- **`ci-cd-integration.sh`** - CI/CD pipeline integration
  - Supports GitHub Actions, Jenkins, CircleCI, Travis CI, Buildkite
  - Comprehensive testing, security scanning, and deployment
  - Automated notifications and cleanup
  - Usage: `./ci-cd-integration.sh [pipeline|test|security|performance|deploy|notify|cleanup|help]`

## Quick Start

### Local Development Deployment
```bash
# Deploy with standard strategy
./scripts/deploy-master.sh deploy

# Check deployment status
./scripts/deploy-master.sh status

# Monitor services
./scripts/health-monitor-enhanced.sh monitor
```

### Production Deployment
```bash
# Deploy with zero-downtime strategy
DEPLOYMENT_TYPE=zero-downtime ENVIRONMENT=production ./scripts/deploy-master.sh deploy

# Verify deployment
./scripts/deploy-master.sh verify

# Monitor continuously
./scripts/health-monitor-enhanced.sh monitor
```

### CI/CD Pipeline
```bash
# Run full CI/CD pipeline
./scripts/ci-cd-integration.sh pipeline

# Run tests only
./scripts/ci-cd-integration.sh test

# Deploy only
CI_ENVIRONMENT=production DEPLOYMENT_TYPE=zero-downtime ./scripts/ci-cd-integration.sh deploy
```

## Environment Variables

### Deployment Configuration
- `DEPLOYMENT_TYPE` - Deployment strategy (standard|zero-downtime|rolling)
- `ENVIRONMENT` - Target environment (production|staging|development)
- `BRANCH` - Git branch to deploy (default: main)
- `TAG` - Git tag to deploy (overrides branch)

### Database Configuration
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_NAME` - Database name (default: celebrum_ai)
- `DB_USER` - Database user (default: postgres)
- `DB_PASSWORD` - Database password
- `DB_CONTAINER` - Docker container name (default: postgres)

### Monitoring Configuration
- `MONITOR_INTERVAL` - Monitoring interval in seconds (default: 30)
- `HEALTH_CHECK_TIMEOUT` - Health check timeout in seconds (default: 60)
- `MAX_RESTART_ATTEMPTS` - Maximum restart attempts (default: 3)

### CI/CD Configuration
- `RUN_TESTS` - Run tests (true|false)
- `RUN_SECURITY_SCAN` - Run security scan (true|false)
- `RUN_PERFORMANCE_TEST` - Run performance tests (true|false)
- `AUTO_ROLLBACK` - Enable automatic rollback (true|false)
- `NOTIFY_SLACK` - Send Slack notifications (true|false)
- `SLACK_WEBHOOK_URL` - Slack webhook URL for notifications

## Deployment Strategies

### Standard Deployment
- Stops all services, then starts them
- Fastest deployment but with downtime
- Suitable for development and staging environments

### Zero-Downtime Deployment
- Deploys without service interruption
- Uses blue-green deployment strategy
- Recommended for production environments

### Rolling Deployment
- Updates services one by one
- Maintains partial service availability
- Good balance between speed and availability

## Backup and Recovery

All deployment operations automatically create comprehensive backups:
- Database backup (SQL dump)
- Code backup (Git state)
- Configuration backup
- Docker state backup

Rollback to any backup:
```bash
./scripts/deploy-master.sh rollback backup_20240115_143022
```

## Monitoring and Health Checks

The health monitoring system provides:
- Real-time service status
- HTTP endpoint health checks
- Database connectivity verification
- Redis connectivity verification
- System resource monitoring
- Automatic service restart on failure
- Health reporting and alerting

## Integration with GitHub Actions

The CI/CD workflow is configured in `.github/workflows/ci-cd.yml` and provides:
- Automated testing on pull requests
- Security scanning
- Automated deployment to staging/production
- Post-deployment verification
- Slack notifications
- Automatic rollback on failure

## Troubleshooting

### Common Issues

1. **Service startup failures**
   ```bash
   ./scripts/health-monitor-enhanced.sh status
   ./scripts/startup-orchestrator.sh restart
   ```

2. **Database connection issues**
   ```bash
   ./scripts/migrate-optimized.sh status
   ./scripts/health-monitor-enhanced.sh check postgres
   ```

3. **Deployment failures**
   ```bash
   ./scripts/deploy-master.sh status
   ./scripts/deploy-master.sh rollback [backup_name]
   ```

### Logs

All scripts generate detailed logs:
- Deployment logs: `logs/deploy-master.log`
- Health monitoring logs: `logs/health-monitor.log`
- Migration logs: `logs/migrate-optimized.log`
- CI/CD logs: `ci-artifacts/logs/`

## Support

For issues or questions:
1. Check the logs for detailed error information
2. Use the `help` command for any script to see usage information
3. Verify environment variables are correctly set
4. Ensure Docker and docker-compose are properly installed and running