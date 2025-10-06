# 🔄 Actualización Requerida en Spring Boot

## ⚠️ IMPORTANTE: Proto Actualizado

El proto de gRPC ha sido expandido con nuevos campos. **Debes regenerar los stubs de gRPC en tu proyecto Spring Boot**.

## 📋 Cambios en el Proto

### Archivo: `code_runner.proto`

```protobuf
message ExecutionResponse {
    repeated string approved_tests = 1;      // IDs de tests aprobados (SOLO los que pasaron)
    bool completed = 2;                       // Si completó la ejecución
    int64 execution_time_ms = 3;             // ⬅️ NUEVO
    int32 total_tests = 4;                    // ⬅️ NUEVO
    int32 passed_tests = 5;                   // ⬅️ NUEVO
    int32 failed_tests = 6;                   // ⬅️ NUEVO
    bool success = 7;                         // ⬅️ NUEVO
    string message = 8;                       // ⬅️ NUEVO
    string error_message = 9;                 // ⬅️ NUEVO
    string error_type = 10;                   // ⬅️ NUEVO
}
```

## 🔧 Pasos para Actualizar Spring Boot

### 1. Copiar el Proto Actualizado

Copia el archivo actualizado desde el microservicio Go:

```bash
cp /Users/nanakusa/Desktop/LevelUpJourney/Microservice-CodeRunner/api/proto/code_runner.proto \
   /ruta/a/tu/proyecto/springboot/src/main/proto/
```

### 2. Regenerar Stubs de Java

Si usas Maven:

```bash
mvn clean compile
```

Si usas Gradle:

```bash
./gradlew clean build
```

### 3. Actualizar tu Código Java

**Antes:**
```java
ExecutionResponse response = solutionEvaluationStub.evaluateSolution(request);

List<String> approvedTests = response.getApprovedTestsList();
boolean completed = response.getCompleted();

// ⚠️ Solo tenías estos dos campos
```

**Después:**
```java
ExecutionResponse response = solutionEvaluationStub.evaluateSolution(request);

// Campos existentes
List<String> approvedTests = response.getApprovedTestsList();  // Solo tests que pasaron
boolean completed = response.getCompleted();

// ⬅️ NUEVOS CAMPOS DISPONIBLES
long executionTimeMs = response.getExecutionTimeMs();    // Tiempo de ejecución
int totalTests = response.getTotalTests();                // Total de tests enviados
int passedTests = response.getPassedTests();              // Tests que pasaron
int failedTests = response.getFailedTests();              // Tests que fallaron
boolean success = response.getSuccess();                  // true si todos pasaron
String message = response.getMessage();                   // Mensaje descriptivo
String errorMessage = response.getErrorMessage();         // Error si hubo
String errorType = response.getErrorType();               // "timeout", "test_failure", etc.
```

## 📊 Ejemplo de Uso Completo

```java
@Service
public class CodeExecutionService {
    
    @Autowired
    private SolutionEvaluationServiceBlockingStub codeRunnerStub;
    
    public ExecutionResult executeSolution(SolutionRequest solutionRequest) {
        // Construir request
        ExecutionRequest request = ExecutionRequest.newBuilder()
            .setChallengeId(solutionRequest.getChallengeId())
            .setCodeVersionId(solutionRequest.getCodeVersionId())
            .setStudentId(solutionRequest.getStudentId())
            .setCode(solutionRequest.getCode())
            .addAllTests(solutionRequest.getTests())
            .build();
        
        // Llamar a gRPC
        ExecutionResponse response = codeRunnerStub.evaluateSolution(request);
        
        // Procesar respuesta completa
        ExecutionResult result = ExecutionResult.builder()
            .approvedTestIds(response.getApprovedTestsList())
            .completed(response.getCompleted())
            .executionTimeMs(response.getExecutionTimeMs())
            .totalTests(response.getTotalTests())
            .passedTests(response.getPassedTests())
            .failedTests(response.getFailedTests())
            .success(response.getSuccess())
            .message(response.getMessage())
            .errorMessage(response.getErrorMessage())
            .errorType(response.getErrorType())
            .build();
        
        // Log para debugging
        log.info("Execution completed in {}ms: {}/{} tests passed", 
                 result.getExecutionTimeMs(), 
                 result.getPassedTests(), 
                 result.getTotalTests());
        
        if (!result.isSuccess()) {
            log.warn("Execution failed: {} - {}", 
                     result.getErrorType(), 
                     result.getErrorMessage());
        }
        
        return result;
    }
}
```

## 🎯 Casos de Uso

### Caso 1: Todos los Tests Pasan

```java
ExecutionResponse response = codeRunnerStub.evaluateSolution(request);

// response.getApprovedTestsList() = ["test-1", "test-2", "test-3"]
// response.getSuccess() = true
// response.getPassedTests() = 3
// response.getFailedTests() = 0
// response.getMessage() = "Execution successful: 3/3 tests passed"

// Actualizar en base de datos
updateSolutionStatus(
    solutionId, 
    SolutionStatus.APPROVED, 
    response.getApprovedTestsList(),
    response.getExecutionTimeMs()
);
```

### Caso 2: Algunos Tests Fallan

```java
ExecutionResponse response = codeRunnerStub.evaluateSolution(request);

// response.getApprovedTestsList() = ["test-1", "test-2"]  // Solo los que pasaron
// response.getSuccess() = false
// response.getPassedTests() = 2
// response.getFailedTests() = 1
// response.getErrorType() = "test_failure"
// response.getMessage() = "Execution successful: 2/3 tests passed"

// Actualizar solo con tests aprobados
updateSolutionStatus(
    solutionId, 
    SolutionStatus.PARTIAL, 
    response.getApprovedTestsList(),  // Solo los que pasaron
    response.getExecutionTimeMs()
);

// Notificar al estudiante qué tests fallaron
int failedCount = response.getTotalTests() - response.getPassedTests();
notifyStudent("Tu solución pasó " + response.getPassedTests() + " de " + 
              response.getTotalTests() + " tests. " + failedCount + " tests fallaron.");
```

### Caso 3: Timeout

```java
ExecutionResponse response = codeRunnerStub.evaluateSolution(request);

// response.getApprovedTestsList() = []  // Vacío
// response.getSuccess() = false
// response.getErrorType() = "timeout"
// response.getErrorMessage() = "Execution timed out after 30 seconds"
// response.getExecutionTimeMs() = 30000

// Manejo especial para timeout
if ("timeout".equals(response.getErrorType())) {
    updateSolutionStatus(
        solutionId, 
        SolutionStatus.TIMEOUT, 
        Collections.emptyList(),
        response.getExecutionTimeMs()
    );
    
    notifyStudent("Tu código tomó demasiado tiempo (más de 30 segundos). " +
                 "Verifica que no tengas loops infinitos.");
}
```

### Caso 4: Error de Compilación

```java
ExecutionResponse response = codeRunnerStub.evaluateSolution(request);

// response.getSuccess() = false
// response.getErrorType() = "compilation_error"
// response.getErrorMessage() = "solution.cpp:5:10: error: ..."

if ("compilation_error".equals(response.getErrorType())) {
    updateSolutionStatus(
        solutionId, 
        SolutionStatus.COMPILATION_ERROR,
        Collections.emptyList(),
        response.getExecutionTimeMs()
    );
    
    notifyStudent("Error de compilación:\n" + response.getErrorMessage());
}
```

## 📊 Estructura de Datos Recomendada

```java
@Data
@Builder
public class ExecutionResult {
    private List<String> approvedTestIds;
    private boolean completed;
    private long executionTimeMs;
    private int totalTests;
    private int passedTests;
    private int failedTests;
    private boolean success;
    private String message;
    private String errorMessage;
    private String errorType;
    
    // Métodos útiles
    public boolean hasErrors() {
        return errorMessage != null && !errorMessage.isEmpty();
    }
    
    public boolean isPartialSuccess() {
        return passedTests > 0 && passedTests < totalTests;
    }
    
    public double getSuccessRate() {
        return totalTests > 0 ? (double) passedTests / totalTests * 100 : 0;
    }
}
```

## 🗄️ Actualización en Base de Datos

```java
@Transactional
public void updateSolutionWithExecutionResults(Long solutionId, ExecutionResponse response) {
    Solution solution = solutionRepository.findById(solutionId)
        .orElseThrow(() -> new NotFoundException("Solution not found"));
    
    // Actualizar con resultados
    solution.setStatus(response.getSuccess() ? SolutionStatus.APPROVED : SolutionStatus.FAILED);
    solution.setExecutionTimeMs(response.getExecutionTimeMs());
    solution.setTotalTests(response.getTotalTests());
    solution.setPassedTests(response.getPassedTests());
    solution.setFailedTests(response.getFailedTests());
    solution.setErrorType(response.getErrorType());
    solution.setErrorMessage(response.getErrorMessage());
    solution.setExecutedAt(LocalDateTime.now());
    
    // Actualizar tests aprobados
    List<String> approvedTestIds = response.getApprovedTestsList();
    for (String testId : approvedTestIds) {
        TestResult testResult = new TestResult();
        testResult.setSolution(solution);
        testResult.setTestId(testId);
        testResult.setPassed(true);
        testResultRepository.save(testResult);
    }
    
    solutionRepository.save(solution);
    
    log.info("Solution {} updated: {} tests passed in {}ms", 
             solutionId, response.getPassedTests(), response.getExecutionTimeMs());
}
```

## ⚠️ Cambio Importante

### ⚠️ `approved_tests` ahora contiene SOLO los tests que pasaron

**Antes (comportamiento antiguo):**
```java
// Devolvía TODOS los test IDs sin importar si pasaron o no
response.getApprovedTestsList(); // ["test-1", "test-2", "test-3"]
```

**Ahora (comportamiento correcto):**
```java
// Devuelve SOLO los que realmente pasaron
response.getApprovedTestsList(); // ["test-1", "test-2"]  // test-3 falló

// Para saber cuáles fallaron, usa:
int failed = response.getFailedTests();  // 1
```

## 🧪 Testing

```java
@Test
public void testExecutionWithPartialSuccess() {
    // Arrange
    ExecutionRequest request = createMockRequest();
    
    // Act
    ExecutionResponse response = codeRunnerStub.evaluateSolution(request);
    
    // Assert
    assertThat(response.getCompleted()).isTrue();
    assertThat(response.getTotalTests()).isEqualTo(3);
    assertThat(response.getPassedTests()).isEqualTo(2);
    assertThat(response.getFailedTests()).isEqualTo(1);
    assertThat(response.getSuccess()).isFalse();  // No todos pasaron
    assertThat(response.getApprovedTestsList()).hasSize(2);  // Solo los que pasaron
    assertThat(response.getExecutionTimeMs()).isGreaterThan(0);
}
```

## 📝 Checklist de Migración

- [ ] Copiar proto actualizado
- [ ] Regenerar stubs de Java (mvn/gradle)
- [ ] Actualizar código para usar nuevos campos
- [ ] Actualizar modelo de base de datos si es necesario
- [ ] Actualizar tests unitarios
- [ ] Probar con el microservicio Go
- [ ] Verificar logs y métricas
- [ ] Actualizar documentación API

## 🚀 Testing de Integración

```bash
# 1. Asegúrate que el servidor Go esté corriendo
cd /Users/nanakusa/Desktop/LevelUpJourney/Microservice-CodeRunner
go run ./cmd/server/main.go

# 2. Desde Spring Boot, ejecuta tus tests
mvn test

# 3. Verifica logs en ambos lados
```

¡Listo para actualizar Spring Boot! 🎉
