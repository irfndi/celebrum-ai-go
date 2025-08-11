# Database Migrations

This directory contains organized database migrations for the Celebrum AI platform.

## Migration Structure

Migrations are numbered sequentially and organized by feature area:

### Current Migrations

1. **001_initial_schema.sql** - Basic initial schema (existing)
2. **002_initial_data.sql** - Initial data seeding (existing)  
3. **003_add_market_data_columns.sql** - Additional market data columns (existing)
4. **004_enhanced_initial_schema.sql** - Comprehensive enhanced schema
5. **005_add_funding_rate_support.sql** - Funding rate arbitrage features
6. **006_minimal_schema.sql** - Minimal schema for CI/CD environments

### Migration Files Source

| Migration File | Based On | Description |
|----------------|----------|-------------|
| 004_enhanced_initial_schema.sql | scripts/init.sql | Complete production schema |
| 005_add_funding_rate_support.sql | scripts/add_funding_rate_support.sql | Funding rate arbitrage |
| 006_minimal_schema.sql | scripts/init-minimal.sql | CI/CD testing |

## Usage

### Running Migrations

Use the provided migration script:

```bash
# Run all pending migrations
./migrate.sh

# Check migration status
./migrate.sh status

# List available migrations
./migrate.sh list

# Run specific migration
./migrate.sh 004

# Rollback specific migration (requires manual implementation)
./migrate.sh rollback 005
```

### Environment Variables

Set the following environment variables for database connection:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=celebrum_ai
export DB_USER=celebrum_ai_user
export DB_PASSWORD=your_password
```

### Docker Usage

To run migrations in Docker environment:

```bash
# Using docker-compose
docker-compose exec postgres psql -U celebrum_ai -d celebrum_ai -f /database/migrations/004_enhanced_initial_schema.sql

# Using direct docker
docker exec celebrum-postgres psql -U celebrum_ai -d celebrum_ai -f /database/migrations/004_enhanced_initial_schema.sql
```

## Creating New Migrations

1. **Naming Convention**: Use sequential numbering with descriptive names
   - Format: `NNN_descriptive_name.sql`
   - Examples: `007_add_user_preferences.sql`, `008_enhanced_analytics.sql`

2. **Structure Template**:
   ```sql
   -- Brief description
   -- Migration NNN: What this migration does
   -- Created: YYYY-MM-DD
   -- Based on: source file (if applicable)

   BEGIN;

   -- Your migration SQL here

   -- Migration completed
   INSERT INTO system_config (config_key, config_value, description) VALUES
   ('migration_NNN_completed', 'true', 'Migration NNN completed')
   ON CONFLICT (config_key) DO UPDATE SET
       config_value = EXCLUDED.config_value,
       updated_at = CURRENT_TIMESTAMP;

   COMMIT;
   ```

3. **Best Practices**:
   - Always use `IF NOT EXISTS` for CREATE statements
   - Include rollback statements in comments
   - Add appropriate indexes
   - Update system_config with migration completion
   - Test in both fresh and upgrade scenarios

## Migration Strategy

### Production Deployment
1. **Fresh Install**: Run migrations 001-006 in sequence
2. **Existing Database**: Run only new migrations (004-006)

### CI/CD Environments
- Use `006_minimal_schema.sql` for faster test execution
- Skip complex features like funding rates in CI

### Development
- Use `004_enhanced_initial_schema.sql` for full feature development
- Use `005_add_funding_rate_support.sql` for funding rate features

## Historical Preservation

Original SQL files in `/scripts/` are preserved for reference:
- `init.sql` → `004_enhanced_initial_schema.sql`
- `init-minimal.sql` → `006_minimal_schema.sql`  
- `add_funding_rate_support.sql` → `005_add_funding_rate_support.sql`

## Troubleshooting

### Common Issues

1. **Migration Already Applied**: Check `system_config` table for migration completion status
2. **Permission Errors**: Ensure database user has appropriate permissions
3. **Connection Issues**: Verify environment variables and network connectivity

### Debug Commands

```bash
# Check current schema version
psql -h localhost -U celebrum_ai -d celebrum_ai -c "SELECT * FROM system_config WHERE config_key LIKE 'migration_%'"

# List applied migrations
psql -h localhost -U celebrum_ai -d celebrum_ai -c "SELECT * FROM migrations ORDER BY applied_at DESC"

# Check for missing migrations
ls database/migrations/*.sql | xargs -I {} basename {} | grep -v $(psql -h localhost -U celebrum_ai -d celebrum_ai -c "SELECT filename FROM migrations" -t -A)
```

## Integration with Makefile

Add migration targets to your Makefile:

```makefile
.PHONY: migrate migrate-status migrate-list

migrate:
	cd database && ./migrate.sh

migrate-status:
	cd database && ./migrate.sh status

migrate-list:
	cd database && ./migrate.sh list
```