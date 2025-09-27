# Microservice CodeRunner - gRPC

A gRPC microservice for executing code solutions in multiple programming languages with a pipeline architecture.

## Features

- **Pure gRPC**: High-performance RPC communication
- **Pipeline Architecture**: Modular execution steps (validation, compilation, test fetching, execution, cleanup)
- **Multi-Language Support**: JavaScript, Python, Java, C++, C#, Go, Rust, TypeScript
- **Mock Test Cases**: Built-in test case generation for development

## Quick Start

```bash
# Build
go build -o bin/coderunner.exe ./cmd/server

# Run
./bin/coderunner.exe
```

## Configuration

Set environment variables:
- `GRPC_PORT` - gRPC server port (default: 9084)
- `PORT` - HTTP port for health checks (default: 8084)

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

```
ExecutionRequest → Validation → Compilation → Test Fetching → Execution → Cleanup → ExecutionResponse
```

Each step in the pipeline can be rolled back on failure and provides detailed logging.

## Testing with grpcurl

```bash
# Health check
grpcurl -plaintext localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/HealthCheck

# Execute code
grpcurl -plaintext -d '{
    "solution_id": "test_sol",
    "challenge_id": "factorial",
    "student_id": "test_student",
    "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
    "language": "javascript"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode
```
