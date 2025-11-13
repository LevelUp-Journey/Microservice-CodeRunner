package docker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// Execute ejecuta el código en un contenedor Docker
func (e *DockerExecutor) Execute(ctx context.Context, config *ExecutionConfig) (*ExecutionResult, error) {
	startTime := time.Now()
	log.Printf("🐳 Starting Docker execution for ExecutionID: %s", config.ExecutionID)

	result := &ExecutionResult{
		ExecutionID: config.ExecutionID,
		Success:     false,
	}

	// Setup filesystem
	executionDir, err := e.setupExecutionDirectory(config)
	if err != nil {
		return nil, err
	}

	// Ensure image exists
	if err := e.ensureImage(ctx, config.ImageName); err != nil {
		return nil, fmt.Errorf("failed to ensure image: %w", err)
	}

	// Create container
	containerID, err := e.createContainer(ctx, config, executionDir)
	if err != nil {
		return nil, err
	}

	// Cleanup after execution
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Cleanup(cleanupCtx, containerID); err != nil {
			log.Printf("  ⚠️  Warning: failed to cleanup container: %v", err)
		}
	}()

	// Execute with timeout
	exitCode, timedOut, err := e.runContainer(ctx, config, containerID)
	if err != nil {
		return nil, err
	}

	if timedOut {
		result.TimedOut = true
		result.ErrorType = "timeout"
		result.ErrorMessage = fmt.Sprintf("Execution timed out after %d seconds", config.TimeoutSeconds)
		log.Printf("  ⏱️  Execution timed out")
		return result, nil
	}

	// Capture logs
	stdout, stderr, err := e.captureLogs(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Build result
	result.StdOut = stdout
	result.StdErr = stderr
	result.ExitCode = exitCode
	result.ExecutionTimeMS = time.Since(startTime).Milliseconds()
	result.Success = (exitCode == 0)

	e.logExecutionResults(result)

	parser, parserErr := e.parserFactory.GetParserForLanguage(config.Language)
	if parserErr != nil {
		log.Printf("  ⚠️  Warning: no parser strategy for language %s: %v", config.Language, parserErr)
	}

	parsed := false
	if parser != nil && parser.DetectsOutput(result.StdOut) {
		testResults, err := parser.Parse(result.StdOut, config.TestIDs)
		if err != nil {
			log.Printf("  ⚠️  Warning: Error parsing test results: %v", err)
			// Fallback: mark all as failed
			result.TotalTests = len(config.TestIDs)
			result.PassedTests = 0
			result.FailedTests = len(config.TestIDs)
			result.TestResults = make([]TestResult, 0, len(config.TestIDs))
			for _, testID := range config.TestIDs {
				result.TestResults = append(result.TestResults, TestResult{
					TestID:       testID,
					TestName:     testID,
					Passed:       false,
					ErrorMessage: "Parsing failed",
				})
			}
			parsed = true
		} else {
			result.TestResults = testResults
			result.TotalTests = len(testResults)
			result.PassedTests = 0
			result.FailedTests = 0
			for _, tr := range testResults {
				if tr.Passed {
					result.PassedTests++
				} else {
					result.FailedTests++
				}
			}
			parsed = true
		}
	}

	if parsed {
		result.Success = (exitCode == 0) && result.FailedTests == 0 && result.TotalTests > 0
	}

	if !parsed && exitCode != 0 {
		// Compilation or runtime error - detect error type
		e.detectErrorType(result)
	}

	return result, nil
}

// createContainer crea y configura un nuevo contenedor Docker
func (e *DockerExecutor) createContainer(ctx context.Context, config *ExecutionConfig, executionDir string) (string, error) {
	containerConfig := &container.Config{
		Image:        config.ImageName,
		WorkingDir:   config.WorkDir,
		Cmd:          []string{"/bin/bash", "-c", "g++ -std=c++17 solution.cpp -o solution && ./solution"},
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   config.MemoryLimitMB * 1024 * 1024,
			NanoCPUs: int64(config.CPULimit * 1e9),
		},
		NetworkMode: container.NetworkMode(e.dockerConfig.NetworkMode),
		Binds: []string{
			fmt.Sprintf("%s:%s", executionDir, config.WorkDir),
		},
		CapDrop:     e.dockerConfig.DropCapabilities,
		SecurityOpt: e.dockerConfig.SecurityOpt,
	}

	log.Printf("  🔧 Container configured: Memory=%dMB, CPU=%.1f cores, Timeout=%ds",
		config.MemoryLimitMB, config.CPULimit, config.TimeoutSeconds)

	containerName := fmt.Sprintf("coderunner-%s", config.ExecutionID.String())
	resp, err := e.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Printf("  ✅ Container created: %s", resp.ID[:12])
	return resp.ID, nil
}

// runContainer ejecuta el contenedor y espera su finalización
func (e *DockerExecutor) runContainer(ctx context.Context, config *ExecutionConfig, containerID string) (int, bool, error) {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	if err := e.client.ContainerStart(execCtx, containerID, container.StartOptions{}); err != nil {
		return 0, false, fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("  🚀 Container started")

	statusCh, errCh := e.client.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return 0, true, nil // Timeout occurred
		}
	case status := <-statusCh:
		log.Printf("  ✅ Container finished with exit code: %d", status.StatusCode)
		return int(status.StatusCode), false, nil
	}

	return 0, false, nil
}

// captureLogs captura stdout y stderr del contenedor
func (e *DockerExecutor) captureLogs(ctx context.Context, containerID string) (string, string, error) {
	out, err := e.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer out.Close()

	var stdout, stderr strings.Builder
	if _, err := stdcopy.StdCopy(&stdout, &stderr, out); err != nil {
		log.Printf("  ⚠️  Warning: error reading logs: %v", err)
	}

	return stdout.String(), stderr.String(), nil
}

// detectErrorType detecta el tipo de error basándose en stderr con mejor clasificación
func (e *DockerExecutor) detectErrorType(result *ExecutionResult) {
	stderr := result.StdErr
	lines := strings.Split(stderr, "\n")

	// Detectar errores de compilación con clasificación más detallada
	if strings.Contains(stderr, "error:") || strings.Contains(stderr, "fatal error:") {
		result.ErrorType = "compilation_error"

		// Buscar el primer error específico
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "error:") {
				// Clasificar el tipo de error de compilación
				if strings.Contains(line, "expected") || strings.Contains(line, "syntax error") {
					result.ErrorType = "syntax_error"
					result.ErrorMessage = fmt.Sprintf("Syntax error: %s", line)
				} else if strings.Contains(line, "undefined reference") {
					result.ErrorType = "linking_error"
					result.ErrorMessage = fmt.Sprintf("Linking error: %s", line)
				} else if strings.Contains(line, "no matching function") || strings.Contains(line, "cannot convert") {
					result.ErrorType = "type_error"
					result.ErrorMessage = fmt.Sprintf("Type error: %s", line)
				} else if strings.Contains(line, "redeclared") || strings.Contains(line, "redefinition") {
					result.ErrorType = "redeclaration_error"
					result.ErrorMessage = fmt.Sprintf("Redeclaration error: %s", line)
				} else {
					result.ErrorMessage = fmt.Sprintf("Compilation error: %s", line)
				}
				break
			}
		}

		if result.ErrorMessage == "" {
			result.ErrorMessage = "Compilation failed - see stderr for details"
		}

		log.Printf("  🔴 %s DETECTED", strings.ToUpper(result.ErrorType))
		log.Printf("  📝 Error: %s", result.ErrorMessage)

		// No tests passed if compilation failed
		result.PassedTests = 0
		result.FailedTests = 0
		result.TotalTests = 0
		return
	}

	// Detectar errores de runtime con más casos
	if strings.Contains(stderr, "Segmentation fault") || strings.Contains(stderr, "core dumped") {
		result.ErrorType = "runtime_error"
		result.ErrorMessage = "Runtime error: Segmentation fault"
		log.Printf("  🔴 RUNTIME ERROR: Segmentation fault")
		return
	}

	if strings.Contains(stderr, "Floating point exception") {
		result.ErrorType = "runtime_error"
		result.ErrorMessage = "Runtime error: Floating point exception"
		log.Printf("  🔴 RUNTIME ERROR: Floating point exception")
		return
	}

	if strings.Contains(stderr, "Aborted") || strings.Contains(stderr, "abort") {
		result.ErrorType = "runtime_error"
		result.ErrorMessage = "Runtime error: Program aborted"
		log.Printf("  🔴 RUNTIME ERROR: Program aborted")
		return
	}

	// Detectar errores de tiempo de ejecución en tests (si no hay compilación)
	if result.TotalTests > 0 && result.FailedTests > 0 && result.ErrorType == "" {
		result.ErrorType = "test_failure"
		result.ErrorMessage = "Some tests failed - check test results"
		log.Printf("  🔴 TEST FAILURE: %d/%d tests failed", result.FailedTests, result.TotalTests)
		return
	}

	// Error genérico si no se clasificó
	if result.ErrorType == "" {
		result.ErrorType = "execution_error"
		result.ErrorMessage = "Execution failed - see stderr for details"
		log.Printf("  🔴 EXECUTION ERROR")
	}
}

// Cleanup limpia recursos del contenedor
func (e *DockerExecutor) Cleanup(ctx context.Context, containerID string) error {
	log.Printf("  🧹 Cleaning up container: %s", containerID[:12])

	timeout := 5
	if err := e.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		log.Printf("  ⚠️  Warning: failed to stop container: %v", err)
	}

	if err := e.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Printf("  ✅ Container cleaned up")
	return nil
}
