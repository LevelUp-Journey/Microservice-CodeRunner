package docker

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionConfig representa la configuración para ejecutar código en Docker
type ExecutionConfig struct {
	// Script information
	Language    string
	SourceCode  string // El código C++ completo generado
	ExecutionID uuid.UUID

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

	return &ExecutionConfig{
		Language:       "cpp",
		SourceCode:     sourceCode,
		ExecutionID:    executionID,
		MemoryLimitMB:  dockerConfig.DefaultMemoryMB,
		CPULimit:       dockerConfig.DefaultCPULimit,
		TimeoutSeconds: int(dockerConfig.DefaultTimeout.Seconds()),
		ImageName:      dockerConfig.CppImageName,
		WorkDir:        "/workspace",
	}
}
