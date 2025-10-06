# Microservice CodeRunner - gRPC

A gRPC microservice for executing code solutions in multiple programming languages with a pipeline architecture and PostgreSQL database integration.

## Features

- **Pure gRPC**: High-performance RPC communication
- **Pipeline Architecture**: Modular execution steps (validation, compilation, test fetching, execution, cleanup)
- **Multi-Language Support**: JavaScript, Python, Java, C++, C#, Go, Rust, TypeScript
- **PostgreSQL Database**: Complete execution tracking and analytics with UUID-based records
- **Execution Monitoring**: Detailed logging, step tracking, and performance metrics
- **Docker Support**: Ready-to-use Docker Compose setup with PostgreSQL
- **Mock Test Cases**: Built-in test case generation for development

## Quick Start

### Development with Docker Compose (Recommended)

```bash
# Clone and setup
git clone <repository-url>
cd Microservice-CodeRunner

# Copy environment configuration
cp .env.example .env

# Start PostgreSQL and pgAdmin
docker-compose up -d postgres pgadmin

# Install dependencies
go mod tidy

# Build and run
go build -o bin/coderunner ./cmd/server
./bin/coderunner
```

### Manual Setup

```bash
# Install PostgreSQL
# Set up database (see Database Setup section)

# Build
go build -o bin/coderunner ./cmd/server

# Run
./bin/coderunner
```

## Database Setup

### Using Docker Compose (Recommended)
```bash
# Start PostgreSQL container
docker-compose up -d postgres

# The database will be automatically initialized with migrations
```

### Manual PostgreSQL Setup
```bash
# Create database
createdb coderunner

# Run migrations
psql -d coderunner -f migrations/001_create_execution_tables.sql
```

### pgAdmin Access
- URL: http://localhost:5050
- Email: admin@coderunner.local
- Password: admin123

## Configuration

### Environment Variables

#### Application
- `APP_NAME` - Application name (default: "Microservice CodeRunner gRPC API")
- `API_VERSION` - API version (default: "v1")
- `GRPC_PORT` - gRPC server port (default: 9084)
- `PORT` - HTTP port for health checks (default: 8084)

#### Database (PostgreSQL)
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database user (default: postgres)
- `DB_PASSWORD` - Database password (default: postgres)
- `DB_NAME` - Database name (default: coderunner)
- `DB_SSLMODE` - SSL mode (default: disable)
- `DB_TIMEZONE` - Database timezone (default: UTC)

#### Database Connection Pool
- `DB_MAX_OPEN_CONNS` - Max open connections (default: 25)
- `DB_MAX_IDLE_CONNS` - Max idle connections (default: 10)
- `DB_CONN_MAX_LIFETIME` - Connection max lifetime in seconds (default: 3600)

## gRPC Service

```protobuf
service CodeExecutionService {
    rpc ExecuteCode (ExecutionRequest) returns (ExecutionResponse);
    rpc GetExecutionStatus (ExecutionStatusRequest) returns (ExecutionStatusResponse);
    rpc HealthCheck (HealthCheckRequest) returns (HealthCheckResponse);
    rpc StreamExecutionLogs (StreamLogsRequest) returns (stream LogEntry);
}
```

### ExecuteCode Request

```json
{
    "solution_id": "sol_123",
    "challenge_id": "factorial",
    "student_id": "student_789",
    "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
    "language": "javascript"
}
```

### ExecuteCode Response

```json
{
    "approved_test_ids": ["test_1", "test_2"],
    "success": true,
    "message": "All 2 tests passed",
    "execution_id": "exec_1234567890"
}
```

## Supported Challenge IDs

- `factorial` - Factorial calculation
- `sum` / `addition` - Number addition
- `reverse` / `string_reverse` - String reversal
- `hello_world` / `print` - Hello World output
- Default: Echo input tests

## Architecture

### Pipeline Architecture
```
ExecutionRequest → Database Record Creation → Pipeline Execution → Database Update
                                          ↓
            Validation → Compilation → Test Fetching → Execution → Cleanup
                                          ↓
                                    Database Logging & Step Tracking
```

### Database Schema

The system uses PostgreSQL with the following main tables:

#### `executions`
- Stores complete execution records with UUID primary keys
- Tracks request information, results, performance metrics
- Includes error tracking and metadata

#### `execution_steps`
- Records each pipeline step execution
- Tracks timing, status, and error information
- Links to parent execution via foreign key

#### `execution_logs`
- Stores detailed execution logs
- Multiple log levels (debug, info, warn, error)
- Timestamped with source tracking

#### `test_results`
- Individual test case results
- Input/output comparison data
- Performance metrics per test

Each step in the pipeline can be rolled back on failure and provides detailed logging stored in PostgreSQL.

## Database Operations

### Viewing Execution Data

```sql
-- View recent executions
SELECT id, solution_id, challenge_id, student_id, language, status, success, created_at 
FROM executions 
ORDER BY created_at DESC 
LIMIT 10;

-- View execution with steps and logs
SELECT e.id, e.status, e.success, e.message,
       es.step_name, es.status as step_status, es.duration_ms,
       el.level, el.message as log_message, el.timestamp
FROM executions e
LEFT JOIN execution_steps es ON e.id = es.execution_id
LEFT JOIN execution_logs el ON e.id = el.execution_id
WHERE e.id = 'your-execution-uuid'
ORDER BY es.step_order, el.timestamp;

-- Get execution statistics
SELECT 
    status,
    COUNT(*) as count,
    AVG(execution_time_ms) as avg_duration,
    AVG(passed_tests::float / NULLIF(total_tests, 0) * 100) as avg_success_rate
FROM executions 
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY status;
```

## Testing with grpcurl

```bash
# Health check
grpcurl -plaintext localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/HealthCheck

# Execute code (creates database records)
grpcurl -plaintext -d '{
    "solution_id": "test_sol",
    "challenge_id": "factorial",
    "student_id": "test_student",
    "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
    "language": "javascript"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode

# Check execution status
grpcurl -plaintext -d '{
    "execution_id": "returned-execution-id"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/GetExecutionStatus
```

## Docker Commands

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f coderunner
docker-compose logs -f postgres

# Stop services
docker-compose down

# Reset database (removes all data)
docker-compose down -v
docker-compose up -d postgres

# Rebuild application
docker-compose build coderunner
docker-compose up -d coderunner
```

## Development

### Adding New Pipeline Steps

1. Create step in `internal/steps/`
2. Steps are automatically tracked in database
3. Event handler logs all step execution details

### Database Migrations

1. Add new migration files to `migrations/` folder
2. Follow naming convention: `XXX_description.sql`
3. Migrations run automatically on startup

### Monitoring and Analytics

The database stores comprehensive execution data for:
- Performance analysis
- Error tracking
- Student progress monitoring
- Challenge difficulty assessment
- System resource usage
