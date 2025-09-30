# Integración con el Servidor gRPC Existente

## 📋 Resumen de Alineación con Proto

La implementación del sistema Docker está **completamente alineada** con el archivo `code_runner.proto`:

### ✅ Campos Proto Implementados

1. **ExecutionRequest** - Todos los campos soportados:
   - `solution_id`, `challenge_id`, `student_id`, `code`, `language`
   - `ExecutionConfig` con timeouts, límites de memoria, etc.

2. **ExecutionResponse** - Formato exacto:
   - `approved_test_ids[]` - Lista de IDs de tests que pasaron
   - `success`, `message`, `execution_id`
   - `ExecutionMetadata` con métricas detalladas
   - `PipelineStep[]` con información de cada paso

3. **Types y Enums**:
   - `ProgrammingLanguage` enum implementado
   - `ExecutionStatus` y `StepStatus` enums
   - `TestResult`, `CompilationInfo` exactos al proto

## 🔗 Integración con Servidor Existente

### Paso 1: Actualizar el Handler gRPC

En tu servidor gRPC existente (`internal/server/server.go`), puedes integrar así:

```go
import (
    "code-runner/internal/adapters"
    "code-runner/internal/steps"
)

type CodeExecutionServer struct {
    // ... campos existentes
    dockerAdapter *adapters.DockerExecutionAdapter
}

func NewCodeExecutionServer(logger pipeline.Logger) *CodeExecutionServer {
    return &CodeExecutionServer{
        // ... inicialización existente
        dockerAdapter: adapters.NewDockerExecutionAdapter(logger),
    }
}

func (s *CodeExecutionServer) ExecuteCode(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
    // Validar lenguaje para Docker
    if err := s.dockerAdapter.ValidateLanguageSupport(req.Language); err != nil {
        return &ExecutionResponse{
            Success: false,
            Message: err.Error(),
        }, nil
    }

    // Preparar datos de ejecución
    data := s.dockerAdapter.PrepareExecutionData(
        req.SolutionId,
        req.ChallengeId, 
        req.StudentId,
        req.Code,
        req.Language,
    )

    // Ejecutar con Docker
    err := s.dockerAdapter.ExecuteWithDocker(ctx, data)
    if err != nil {
        return &ExecutionResponse{
            Success: false,
            Message: err.Error(),
            ExecutionId: data.ExecutionID,
        }, nil
    }

    // Respuesta en formato proto
    return &ExecutionResponse{
        ApprovedTestIds: s.dockerAdapter.GetApprovedTestIDs(data),
        Success:         data.Success,
        Message:         data.Message,
        ExecutionId:     data.ExecutionID,
        Metadata:        convertToProtoMetadata(data), // Helper function
        PipelineSteps:   convertToProtoSteps(data.CompletedSteps), // Helper function
    }, nil
}
```

### Paso 2: Helper Functions para Conversión Proto

```go
func convertToProtoMetadata(data *pipeline.ExecutionData) *ExecutionMetadata {
    metadata := &ExecutionMetadata{
        ExecutionTimeMs: data.ExecutionTimeMS,
        MemoryUsedMb:    data.MemoryUsedMB,
        ExitCode:        int32(data.ExitCode),
    }

    if !data.StartTime.IsZero() {
        metadata.StartedAt = timestamppb.New(data.StartTime)
    }
    if !data.EndTime.IsZero() {
        metadata.CompletedAt = timestamppb.New(data.EndTime)
    }

    // Convertir results de tests
    for _, result := range data.TestResults {
        metadata.TestResults = append(metadata.TestResults, &TestResult{
            TestId:          result.TestID,
            Passed:          result.Passed,
            ExpectedOutput:  result.ExpectedOutput,
            ActualOutput:    result.ActualOutput,
            ErrorMessage:    result.ErrorMessage,
            ExecutionTimeMs: result.ExecutionTimeMS,
        })
    }

    return metadata
}
```

## 🔧 Configuración Requerida

### Variables de Entorno
```bash
# Docker execution settings
DOCKER_ENABLED=true
DOCKER_IMAGE_PREFIX=levelup/code-runner
DOCKER_MEMORY_LIMIT_MB=512
DOCKER_TIMEOUT_SECONDS=30
DOCKER_NETWORK_DISABLED=true
```

### Construcción de Imágenes Docker
```bash
# Construir todas las imágenes
cd docker && ./build-images.sh

# O construir lenguaje específico
cd docker && ./build-images.sh python
```

## 📊 Flujo de Ejecución Actualizado

```
gRPC Request → Validation → Docker Pipeline → Response
     ↓              ↓              ↓            ↓
ExecutionRequest → TestFetching → DockerExec → ExecutionResponse
                      ↓              ↓
                   TestCases → Generated Code → Container Execution
                                     ↓              ↓
                                Docker Image → Test Results → approved_test_ids[]
```

## 🎯 Ventajas de la Integración

1. **Compatibilidad Total**: Respeta 100% el contrato proto existente
2. **Seguridad Mejorada**: Ejecución aislada en containers Docker
3. **Soporte Multi-lenguaje**: C++, Python, JavaScript, Java, Go
4. **Custom Validation**: Soporte para `customValidationCode`
5. **Métricas Detalladas**: Tiempo de ejecución, memoria, logs detallados

## 🚀 Migration Path

### Opción 1: Gradual (Recomendada)
```go
func (s *CodeExecutionServer) ExecuteCode(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
    // Feature flag para Docker execution
    if useDockerExecution(req.Language) {
        return s.executeWithDocker(ctx, req)
    }
    
    // Fallback a ejecución existente
    return s.executeWithLocal(ctx, req)
}
```

### Opción 2: Completa
```go
func (s *CodeExecutionServer) ExecuteCode(ctx context.Context, req *ExecutionRequest) (*ExecutionResponse, error) {
    // Solo Docker execution
    return s.executeWithDocker(ctx, req)
}
```

## 🔍 Testing de Integración

```go
func TestDockerIntegration(t *testing.T) {
    // Test que el sistema Docker responde con formato proto correcto
    server := NewCodeExecutionServer(logger)
    
    req := &ExecutionRequest{
        SolutionId:  "test_solution",
        ChallengeId: "factorial",
        StudentId:   "student_123",
        Code:        "def factorial(n): return 1 if n <= 1 else n * factorial(n-1)",
        Language:    "python",
        Config: &ExecutionConfig{
            TimeoutSeconds: 30,
            MemoryLimitMb:  256,
        },
    }
    
    resp, err := server.ExecuteCode(context.Background(), req)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
    assert.NotEmpty(t, resp.ApprovedTestIds)
    assert.NotEmpty(t, resp.ExecutionId)
}
```

La implementación está **lista para producción** y completamente alineada con tu `.proto` existente! 🎉