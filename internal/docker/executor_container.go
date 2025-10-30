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

	// Parse test results
	e.parseTestResults(result, config.TestIDs)

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
