# Analytics Persistence Guide

## Overview

This guide explains the current state of data cleanup in the Celebrum AI system and how to configure analytics persistence for arbitrage opportunities.

## Current Cleanup Status

### ✅ **Market Data Cleanup - ACTIVE**
- **Retention Period**: 24 hours (default)
- **Cleanup Frequency**: Every hour
- **Current Records**: 164,455 market data entries
- **Status**: Working correctly, cleaning up data older than 24 hours

### ✅ **Arbitrage Opportunities - CONFIGURABLE**
- **Previous Setting**: 72 hours (3 days) retention
- **New Analytics Setting**: 1 year (8760 hours) retention
- **Current Records**: 0 opportunities (none generated yet)
- **Status**: Now preserved for long-term analytics

## Configuration Files

### Analytics Persistence Configuration
Location: `configs/analytics-persistence-config.yaml`

```yaml
cleanup:
  market_data_retention_hours: 168    # 7 days
  funding_rate_retention_hours: 168   # 7 days  
  arbitrage_retention_hours: 8760     # 1 year
  funding_arbitrage_retention_hours: 8760  # 1 year
  cleanup_interval_minutes: 360       # Every 6 hours
```

### Standard Configuration
Location: Original `config.yaml` (with 24-72 hour retention)

## Usage Instructions

### Enable Analytics Persistence
```bash
# Run the analytics persistence configuration
./scripts/enable-analytics-persistence.sh

# This will:
# 1. Backup current configuration
# 2. Apply extended retention settings
# 3. Restart services with new configuration
# 4. Verify the setup
```

### Disable Analytics Persistence (Rollback)
```bash
# Revert to standard cleanup configuration
./scripts/disable-analytics-persistence.sh

# This will:
# 1. Restore from backup or create standard config
# 2. Restart services
# 3. Verify standard configuration
```

### Check Current Status
```bash
# Verify current configuration
./scripts/enable-analytics-persistence.sh --status

# Check database record counts
./scripts/enable-analytics-persistence.sh --verify
```

## Database Queries for Verification

### Check Current Record Counts
```sql
SELECT 
    'market_data' as table_name, 
    COUNT(*) as record_count,
    MIN(created_at) as oldest_record,
    MAX(created_at) as newest_record
FROM market_data
UNION ALL
SELECT 
    'arbitrage_opportunities', 
    COUNT(*) as record_count,
    MIN(created_at) as oldest_record,
    MAX(created_at) as newest_record
FROM arbitrage_opportunities
UNION ALL
SELECT 
    'funding_arbitrage_opportunities', 
    COUNT(*) as record_count,
    MIN(created_at) as oldest_record,
    MAX(created_at) as newest_record
FROM funding_arbitrage_opportunities;
```

### Monitor Cleanup Activity
```sql
-- Check recent cleanup activity (requires logging)
SELECT * FROM logs WHERE message LIKE '%cleanup%' ORDER BY created_at DESC LIMIT 10;
```

## Storage Impact Analysis

With analytics persistence enabled:

| Data Type | Retention | Est. Daily Growth | Annual Storage |
|-----------|-----------|------------------|----------------|
| Market Data | 7 days | ~16 MB/day | ~112 MB |
| Arbitrage Opportunities | 1 year | ~1 MB/1000 ops | ~365 MB |
| Funding Arbitrage | 1 year | ~1 MB/1000 ops | ~365 MB |

**Total Estimated Annual Storage**: ~850 MB - 1.5 GB (depending on activity)

## Monitoring Recommendations

### Storage Monitoring
```bash
# Set up storage alerts (add to monitoring)
docker-compose exec postgres psql -U postgres -d celebrum_ai -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
```

### Automated Cleanup Verification
```bash
# Create monitoring script
cat > scripts/monitor-cleanup.sh << 'EOF'
#!/bin/bash
# Monitor cleanup service health
docker-compose logs app | grep -i cleanup | tail -5
echo "Current record counts:"
docker-compose exec postgres psql -U postgres -d celebrum_ai -c "
SELECT 'market_data', COUNT(*) FROM market_data;
SELECT 'arbitrage_opportunities', COUNT(*) FROM arbitrage_opportunities;
"
EOF
chmod +x scripts/monitor-cleanup.sh
```

## Troubleshooting

### Common Issues

1. **High Storage Usage**
   - Solution: Reduce retention periods in config
   - Command: `./scripts/disable-analytics-persistence.sh`

2. **Cleanup Service Not Running**
   - Check: `docker-compose logs app | grep cleanup`
   - Restart: `docker-compose restart app`

3. **Configuration Not Applied**
   - Verify: Check `config.yaml` contents
   - Restart: `docker-compose restart app`

### Emergency Rollback
```bash
# Quick rollback to standard cleanup
./scripts/disable-analytics-persistence.sh
```

## API Endpoints for Analytics

### Manual Cleanup Trigger
```bash
# Trigger cleanup with custom retention
curl -X POST "http://localhost:8080/api/cleanup/trigger?arbitrage_hours=8760"
```

### Get Data Statistics
```bash
# Check current data counts
curl "http://localhost:8080/api/cleanup/stats"
```

## Summary

- ✅ **Market data cleanup is working correctly** with 24-hour retention
- ✅ **Arbitrage opportunities are now preserved** for 1 year instead of being cleaned up after 72 hours
- ✅ **Configuration scripts provided** for easy enable/disable of analytics persistence
- ✅ **Comprehensive monitoring and rollback capabilities** included
- ✅ **Storage impact documented** for capacity planning

The system now supports both real-time arbitrage detection AND long-term analytics by preserving generated opportunities indefinitely.