---
applyTo: "database/**/*,internal/database/**/*"
---

# Database Instructions

## Migration Guidelines

- Place migrations in `database/migrations/`
- Use sequential naming: `YYYYMMDDHHMMSS_description.sql`
- Always include both up and down migrations
- Test migrations on a clean database before committing

## Query Safety

- Always use parameterized queries to prevent SQL injection
- Never concatenate user input into SQL strings
- Use prepared statements for frequently executed queries

## Connection Management

- Use connection pooling with `pgx`
- Configure appropriate pool sizes for your workload
- Close connections properly; use `defer` for cleanup
- Implement retry logic for transient failures

## Schema Design

- Use appropriate data types for financial values (NUMERIC, not FLOAT)
- Create indexes for frequently queried columns
- Add foreign key constraints for referential integrity
- Document schema changes in migration comments

## Testing

- Use `pgxmock` for unit testing database operations
- Write integration tests that use a real test database
- Test transaction rollback scenarios
- Verify migration up and down operations

## Commands

```bash
make migrate           # Run pending migrations
make migrate-status    # Check migration status
make migrate-list      # List available migrations
```
