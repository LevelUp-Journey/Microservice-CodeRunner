# Docker-Based Code Execution System

This system provides a comprehensive solution for executing and testing code solutions in isolated Docker containers with support for multiple programming languages.

## Features

- **Multi-language support**: C++, Python, JavaScript, Java, Go
- **Docker isolation**: Secure execution in isolated containers
- **Custom validation**: Support for both input/output tests and custom validation code
- **Function parsing**: Automatic detection of main functions using regex patterns
- **Comprehensive logging**: Detailed execution metrics and logging
- **Resource management**: Memory and time limits for safe execution

## Architecture

### Core Components

1. **TestCase Types** (`internal/types/test_case.go`)
   - Standard input/output tests
   - Custom validation code support
   - Validation methods

2. **Function Parser** (`internal/utils/function_parser.go`)
   - Language-specific regex patterns
   - Function extraction from source code
   - Main function detection

3. **Code Generators** (`internal/codegen/`)
   - Factory pattern for language-specific generators
   - Test code generation combining solution + tests
   - Framework-specific test formats

4. **Docker Executor** (`internal/docker/executor.go`)
   - Secure container execution
   - Resource limits and isolation
   - Output capture and metrics

5. **Pipeline Integration** (`internal/steps/docker_execution.go`)
   - Pipeline step for Docker-based execution
   - Result processing and logging

### Docker Images

Each language has its optimized Docker image:

- **C++**: GCC 12 + doctest framework
- **Python**: Python 3.11 + pytest + testing utilities
- **JavaScript**: Node.js 18 + Jest framework
- **Java**: OpenJDK 17 + JUnit 5 + Maven
- **Go**: Go 1.21 + native testing framework

## Usage

### Building Docker Images

```bash
# Build all language images
cd docker && ./build-images.sh

# Build specific language
cd docker && ./build-images.sh python
```

### Test Case Formats

#### Standard Input/Output Test
```json
{
  "id": "test_1",
  "input": "5",
  "expected_output": "120",
  "description": "Calculate factorial of 5",
  "timeout_seconds": 30
}
```

#### Custom Validation Test
```json
{
  "id": "test_2", 
  "custom_validation_code": "assert factorial(5) == 120 and factorial(0) == 1",
  "description": "Custom validation for factorial function",
  "timeout_seconds": 30
}
```

### Integration Example

```go
// Create Docker execution step
logger := // your logger implementation
step := steps.NewDockerExecutionStep(logger)

// Prepare execution data
data := &pipeline.ExecutionData{
    Code:        "def factorial(n): return 1 if n <= 1 else n * factorial(n-1)",
    Language:    "python",
    ChallengeID: "factorial",
    Config: &pipeline.ExecutionConfig{
        TimeoutSeconds: 30,
        MemoryLimitMB:  256,
    },
}

// Execute
err := step.Execute(ctx, data)
```

## Security Features

- **Container isolation**: Each execution runs in a separate container
- **Resource limits**: Memory and CPU constraints
- **Network isolation**: Disabled network access by default
- **Read-only filesystem**: Prevents file system modifications
- **Security options**: Dropped capabilities and no new privileges

## Generated Test Code Examples

### C++ (with doctest)
```cpp
#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN
#include <doctest.h>

// Solution code here...

TEST_CASE("Test 1: Calculate factorial of 5") {
    auto result = factorial(5);
    CHECK(result == 120);
}
```

### Python (with pytest)
```python
import pytest

# Solution code here...

class TestSolution:
    def test_factorial_1(self):
        """Calculate factorial of 5"""
        result = factorial(5)
        assert result == 120
```

### JavaScript (with Jest)
```javascript
// Solution code here...

describe('factorial Tests', () => {
    test('Test 1: Calculate factorial of 5', () => {
        const result = factorial(5);
        expect(result).toBe(120);
    });
});
```

## Logging and Metrics

The system provides comprehensive logging:

- **Execution time**: Millisecond precision
- **Memory usage**: Peak memory consumption
- **Test results**: Individual test pass/fail status
- **Error details**: Compilation and runtime errors
- **Security events**: Container creation and cleanup

## Performance Considerations

- **Container reuse**: Images are built once and reused
- **Resource limits**: Configurable memory and CPU limits
- **Timeout handling**: Graceful handling of long-running code
- **Parallel execution**: Support for concurrent test execution

## Error Handling

- **Compilation errors**: Captured and reported
- **Runtime errors**: Exception handling and reporting
- **Timeout errors**: Graceful termination
- **Resource exhaustion**: Memory and CPU limit enforcement

## Future Enhancements

- **Result parsing**: Framework-specific test result parsing
- **More languages**: Additional language support (Rust, C#, etc.)
- **Advanced metrics**: CPU usage, I/O metrics
- **Caching**: Compilation result caching
- **Distributed execution**: Multi-node execution support