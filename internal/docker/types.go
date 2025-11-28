package docker

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExecutionConfig representa la configuración para ejecutar código en Docker
type ExecutionConfig struct {
	// Script information
	Language    string
	SourceCode  string // El código C++ completo generado
	ExecutionID uuid.UUID
	TestIDs     []string // IDs de los tests en orden de aparición

	// Resource limits
	MemoryLimitMB  int64   // Límite de memoria en MB
	CPULimit       float64 // Límite de CPU (0.5 = 50% de un core)
	TimeoutSeconds int     // Timeout de ejecución en segundos

	// Docker configuration
	ImageName     string // Nombre de la imagen Docker a usar
	ContainerName string // Nombre del contenedor (opcional)
	WorkDir       string // Directorio de trabajo dentro del contenedor
}

// ExecutionResult representa el resultado de ejecutar código en Docker
type ExecutionResult struct {
	// Execution info
	ExecutionID uuid.UUID
	Success     bool
	ExitCode    int

	// Output
	StdOut         string
	StdErr         string
	CompilationLog string

	// Test results
	TotalTests  int
	PassedTests int
	FailedTests int
	TestResults []TestResult

	// Performance metrics
	ExecutionTimeMS int64
	MemoryUsageMB   float64

	// Error information
	ErrorType    string
	ErrorMessage string
	TimedOut     bool
}

// TestResult representa el resultado de un test individual
type TestResult struct {
	TestID          string
	TestName        string
	Passed          bool
	Input           string
	ExpectedOutput  string
	ActualOutput    string
	ErrorMessage    string
	ExecutionTimeMS int64
}

// DockerConfig representa la configuración general de Docker
type DockerConfig struct {
	// Default limits
	DefaultMemoryMB int64
	DefaultCPULimit float64
	DefaultTimeout  time.Duration

	// Image names
	CppImageName    string
	PythonImageName string
	JavaImageName   string

	// Network settings
	NetworkMode      string
	EnableNetworking bool

	// Security settings
	ReadOnlyRootFS   bool
	DropCapabilities []string
	SecurityOpt      []string
}

// DefaultDockerConfig retorna la configuración por defecto
func DefaultDockerConfig() *DockerConfig {
	return &DockerConfig{
		DefaultMemoryMB: 256, // 256 MB
		DefaultCPULimit: 0.5, // 50% de un core
		DefaultTimeout:  30 * time.Second,

		CppImageName:    "coderunner-cpp:latest",
		PythonImageName: "coderunner-python:latest",
		JavaImageName:   "coderunner-java:latest",

		NetworkMode:      "none",
		EnableNetworking: false,

		ReadOnlyRootFS:   true,
		DropCapabilities: []string{"ALL"},
		SecurityOpt:      []string{"no-new-privileges"},
	}
}

// DefaultExecutionConfig crea una configuración por defecto para C++
func DefaultExecutionConfig(executionID uuid.UUID, sourceCode string) *ExecutionConfig {
	dockerConfig := DefaultDockerConfig()

	// Extraer los test IDs del código fuente en orden
	testIDs := extractTestIDsFromSource(sourceCode)

	return &ExecutionConfig{
		Language:       "cpp",
		SourceCode:     sourceCode,
		ExecutionID:    executionID,
		TestIDs:        testIDs,
		MemoryLimitMB:  dockerConfig.DefaultMemoryMB,
		CPULimit:       dockerConfig.DefaultCPULimit,
		TimeoutSeconds: int(dockerConfig.DefaultTimeout.Seconds()),
		ImageName:      dockerConfig.CppImageName,
		WorkDir:        "/workspace",
	}
}

// extractTestIDsFromSource extrae los IDs de los TEST_CASE en orden de aparición
func extractTestIDsFromSource(sourceCode string) []string {
	var testIDs []string
	lines := strings.Split(sourceCode, "\n")

	// Buscar líneas que contengan TEST_CASE("uuid")
	// Formato: TEST_CASE("3a03b67c-9e38-4a3f-ba0e-a5825e41f2bb") {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TEST_CASE(\"") {
			// Extraer el ID entre las comillas
			start := strings.Index(line, "\"")
			if start != -1 {
				end := strings.Index(line[start+1:], "\"")
				if end != -1 {
					testID := line[start+1 : start+1+end]
					testIDs = append(testIDs, testID)
				}
			}
		}
	}

	return testIDs
}
