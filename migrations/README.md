# Database Migrations

This directory contains database migration files for the CodeRunner microservice.

## Overview

Migrations are SQL scripts that define the database schema and any schema changes over time.

## Usage

### Automatic Migrations

When using Docker Compose, migrations in this directory are automatically executed when the PostgreSQL container starts for the first time.

```bash
# Migrations are mounted as:
# ./migrations:/docker-entrypoint-initdb.d:ro
```

### Manual Migrations

To run migrations manually:

```bash
# Using docker-compose
docker exec -it coderunner-postgres psql -U postgres -d code_runner_db -f /docker-entrypoint-initdb.d/001_create_tables.sql

# Using psql directly
psql -h localhost -U postgres -d code_runner_db -f migrations/001_create_tables.sql
```

## Naming Convention

Migration files should follow this naming pattern:

```
XXX_description.sql

Where:
- XXX = Sequential number (001, 002, 003, etc.)
- description = Brief description of the migration (use underscores)

Examples:
- 001_create_execution_tables.sql
- 002_add_performance_indexes.sql
- 003_add_user_table.sql
```

## Migration Order

Migrations are executed in alphabetical order by filename. This is why we use sequential numbers at the beginning of each filename.

## Creating a New Migration

1. Create a new file with the next sequential number
2. Write your SQL statements
3. Test locally before deploying to production
4. Commit to version control

## Example Migration

```sql
-- 001_create_execution_tables.sql
BEGIN;

CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id VARCHAR(255) NOT NULL,
    challenge_id VARCHAR(255) NOT NULL,
    student_id VARCHAR(255) NOT NULL,
    language VARCHAR(50) NOT NULL,
    code TEXT NOT NULL,
    status VARCHAR(50) NOT NULL,
    success BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_executions_student ON executions(student_id);
CREATE INDEX idx_executions_challenge ON executions(challenge_id);
CREATE INDEX idx_executions_created ON executions(created_at);

COMMIT;
```

## Best Practices

1. **Always use transactions** - Wrap DDL statements in BEGIN/COMMIT
2. **Make migrations idempotent** - Use `IF NOT EXISTS` where applicable
3. **Test thoroughly** - Test migrations on a copy of production data
4. **Keep migrations small** - One logical change per migration
5. **Don't modify existing migrations** - Create new migrations for changes
6. **Include rollback plan** - Document how to undo the migration if needed

## Rollback

If you need to rollback a migration:

1. Create a new migration file with the rollback SQL
2. Or manually execute the rollback commands:

```bash
# Connect to database
docker exec -it coderunner-postgres psql -U postgres -d code_runner_db

# Execute rollback commands
DROP TABLE IF EXISTS table_name;
```

## Troubleshooting

### Migration Failed

If a migration fails:

1. Check PostgreSQL logs:
   ```bash
   docker-compose logs postgres
   ```

2. Connect to database and check state:
   ```bash
   docker exec -it coderunner-postgres psql -U postgres -d code_runner_db
   \dt  -- List tables
   ```

3. Fix the migration file and restart database:
   ```bash
   docker-compose down -v postgres  # WARNING: This deletes data
   docker-compose up -d postgres
   ```

### Migration Not Running

If migrations don't run automatically:

- Ensure files are in `migrations/` directory
- Check file permissions (should be readable)
- Verify docker-compose.yml mounts the directory correctly
- Check if database already has the schema (migrations only run on first start)

## Notes

- Migrations in `/docker-entrypoint-initdb.d` only run when the database is initialized (first startup with empty data directory)
- If you need to run migrations on an existing database, use manual migration approach
- For production, consider using migration tools like Flyway or Liquibase