package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code-runner/internal/pipeline"
)

// ExecutionResult contains the result of Docker execution
type ExecutionResult struct {
	Success         bool
	ExitCode        int
	StandardOutput  string
	StandardError   string
	ExecutionTimeMS int64
	MemoryUsedMB    int64
	ContainerID     string
	Error           error
}

// ContainerConfig holds configuration for Docker container execution
type ContainerConfig struct {
	Language        string
	ImageName       string
	WorkingDir      string
	MemoryLimitMB   int32
	TimeoutSeconds  int32
	NetworkDisabled bool
	ReadOnlyMode    bool
	TempDirectory   string
	EnvironmentVars map[string]string
}

// DockerExecutor handles execution of code in Docker containers
type DockerExecutor struct {
	imagePrefix    string
	defaultTimeout time.Duration
	maxMemoryMB    int32
	logger         pipeline.Logger
}

// NewDockerExecutor creates a new Docker executor
func NewDockerExecutor(logger pipeline.Logger) *DockerExecutor {
	return &DockerExecutor{
		imagePrefix:    "levelup/code-runner",
		defaultTimeout: 30 * time.Second,
		maxMemoryMB:    512,
		logger:         logger,
	}
}

// ExecuteCode executes code in a Docker container with the specified configuration
func (de *DockerExecutor) ExecuteCode(ctx context.Context, config *ContainerConfig, command string, files map[string]string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Log execution start
	de.logger.Info(ctx, "Starting Docker execution", map[string]interface{}{
		"language":     config.Language,
		"image":        config.ImageName,
		"timeout":      config.TimeoutSeconds,
		"memory_limit": config.MemoryLimitMB,
		"files_count":  len(files),
	})

	// Create temporary directory for execution
	tempDir, err := de.createTempDirectory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer de.cleanupTempDirectory(tempDir)

	// Write files to temp directory
	if err := de.writeFilesToDirectory(tempDir, files); err != nil {
		return nil, fmt.Errorf("failed to write files: %w", err)
	}

	// Build Docker command
	dockerCmd := de.buildDockerCommand(config, tempDir, command)

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	result, err := de.executeCommand(execCtx, dockerCmd)
	if err != nil {
		de.logger.Error(ctx, "Docker execution failed", err, map[string]interface{}{
			"command": strings.Join(dockerCmd, " "),
		})
		return result, err
	}

	// Calculate execution time
	result.ExecutionTimeMS = time.Since(startTime).Milliseconds()

	// Log execution completion
	de.logger.Info(ctx, "Docker execution completed", map[string]interface{}{
		"success":        result.Success,
		"exit_code":      result.ExitCode,
		"execution_time": result.ExecutionTimeMS,
		"memory_used":    result.MemoryUsedMB,
	})

	return result, nil
}

// createTempDirectory creates a temporary directory for execution
func (de *DockerExecutor) createTempDirectory(config *ContainerConfig) (string, error) {
	if config.TempDirectory != "" {
		// Use provided temp directory
		if err := os.MkdirAll(config.TempDirectory, 0755); err != nil {
			return "", err
		}
		return config.TempDirectory, nil
	}

	// Create system temp directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("code-runner-%s-*", config.Language))
	if err != nil {
		return "", err
	}

	return tempDir, nil
}

// writeFilesToDirectory writes all files to the specified directory
func (de *DockerExecutor) writeFilesToDirectory(dir string, files map[string]string) error {
	for filename, content := range files {
		filePath := filepath.Join(dir, filename)

		// Create subdirectories if needed
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}

		// Write file content
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	return nil
}

// buildDockerCommand builds the Docker command with all necessary flags
func (de *DockerExecutor) buildDockerCommand(config *ContainerConfig, tempDir, command string) []string {
	var cmd []string

	// Base docker run command
	cmd = append(cmd, "docker", "run")

	// Remove container after execution
	cmd = append(cmd, "--rm")

	// Memory limit
	if config.MemoryLimitMB > 0 {
		cmd = append(cmd, "--memory", fmt.Sprintf("%dm", config.MemoryLimitMB))
	}

	// CPU limit (optional - can be added later)
	cmd = append(cmd, "--cpus", "1")

	// Network settings
	if config.NetworkDisabled {
		cmd = append(cmd, "--network", "none")
	}

	// Read-only mode
	if config.ReadOnlyMode {
		cmd = append(cmd, "--read-only")
		cmd = append(cmd, "--tmpfs", "/tmp:rw,noexec,nosuid,size=100m")
	}

	// Security options
	cmd = append(cmd, "--security-opt", "no-new-privileges")
	cmd = append(cmd, "--cap-drop", "ALL")

	// Volume mount
	cmd = append(cmd, "-v", fmt.Sprintf("%s:/workspace", tempDir))

	// Working directory
	workingDir := "/workspace"
	if config.WorkingDir != "" {
		workingDir = config.WorkingDir
	}
	cmd = append(cmd, "-w", workingDir)

	// Environment variables
	for key, value := range config.EnvironmentVars {
		cmd = append(cmd, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Image name
	imageName := config.ImageName
	if imageName == "" {
		imageName = fmt.Sprintf("%s:%s", de.imagePrefix, config.Language)
	}
	cmd = append(cmd, imageName)

	// Command to execute
	cmd = append(cmd, "bash", "-c", command)

	return cmd
}

// executeCommand executes the Docker command and captures output
func (de *DockerExecutor) executeCommand(ctx context.Context, dockerCmd []string) (*ExecutionResult, error) {
	result := &ExecutionResult{}

	// Create command
	cmd := exec.CommandContext(ctx, dockerCmd[0], dockerCmd[1:]...)

	// Capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create stdout pipe: %w", err)
		return result, result.Error
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
		return result, result.Error
	}

	// Start command
	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start Docker command: %w", err)
		return result, result.Error
	}

	// Read output
	stdoutBytes, _ := de.readPipeWithTimeout(stdout, 10*time.Second)
	stderrBytes, _ := de.readPipeWithTimeout(stderr, 10*time.Second)

	result.StandardOutput = string(stdoutBytes)
	result.StandardError = string(stderrBytes)

	// Wait for command completion
	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = fmt.Errorf("Docker execution failed: %w", err)
			return result, result.Error
		}
	}

	// Command succeeded if exit code is 0
	result.Success = result.ExitCode == 0

	return result, nil
}

// readPipeWithTimeout reads from a pipe with timeout
func (de *DockerExecutor) readPipeWithTimeout(pipe interface{}, timeout time.Duration) ([]byte, error) {
	// Implementation would depend on the specific pipe interface
	// This is a simplified version
	switch p := pipe.(type) {
	case *os.File:
		// Read with timeout logic
		return de.readFileWithTimeout(p, timeout)
	default:
		return nil, fmt.Errorf("unsupported pipe type")
	}
}

// readFileWithTimeout reads from file with timeout
func (de *DockerExecutor) readFileWithTimeout(file *os.File, timeout time.Duration) ([]byte, error) {
	// Simplified implementation - in production, use proper timeout logic
	buffer := make([]byte, 64*1024) // 64KB buffer
	n, err := file.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// cleanupTempDirectory removes the temporary directory
func (de *DockerExecutor) cleanupTempDirectory(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		// Log error but don't fail the execution
		// de.logger.Warn would be called here if available
	}
}

// GetImageName returns the full image name for a language
func (de *DockerExecutor) GetImageName(language string) string {
	return fmt.Sprintf("%s:%s", de.imagePrefix, strings.ToLower(language))
}

// IsImageAvailable checks if a Docker image is available locally
func (de *DockerExecutor) IsImageAvailable(ctx context.Context, imageName string) bool {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName)
	err := cmd.Run()
	return err == nil
}

// PullImage pulls a Docker image if not available locally
func (de *DockerExecutor) PullImage(ctx context.Context, imageName string) error {
	if de.IsImageAvailable(ctx, imageName) {
		return nil // Image already available
	}

	de.logger.Info(ctx, "Pulling Docker image", map[string]interface{}{
		"image": imageName,
	})

	cmd := exec.CommandContext(ctx, "docker", "pull", imageName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	return nil
}
