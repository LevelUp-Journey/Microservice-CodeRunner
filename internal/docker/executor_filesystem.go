package docker

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// setupExecutionDirectory crea y configura el directorio de ejecución
func (e *DockerExecutor) setupExecutionDirectory(config *ExecutionConfig) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	baseDir := filepath.Join(cwd, "compiled_test_codes")
	executionDir := filepath.Join(baseDir, config.ExecutionID.String())

	// Create directory with write permissions
	if err := os.MkdirAll(executionDir, 0777); err != nil {
		return "", fmt.Errorf("failed to create execution directory: %w", err)
	}

	// Change ownership to coderunner user (UID 1000) on Linux/macOS
	if runtime.GOOS != "windows" {
		if err := os.Chown(executionDir, 1000, 1000); err != nil {
			log.Printf("  ⚠️  Warning: failed to chown directory: %v", err)
		}
	}

	log.Printf("  📁 Execution directory created: %s", executionDir)

	// Save source code
	sourceFile := filepath.Join(executionDir, "solution.cpp")
	if err := os.WriteFile(sourceFile, []byte(config.SourceCode), 0666); err != nil {
		return "", fmt.Errorf("failed to write source file: %w", err)
	}

	// Change ownership of source file
	if runtime.GOOS != "windows" {
		if err := os.Chown(sourceFile, 1000, 1000); err != nil {
			log.Printf("  ⚠️  Warning: failed to chown source file: %v", err)
		}
	}

	log.Printf("  💾 Source code saved: %s (%d bytes)", sourceFile, len(config.SourceCode))

	return executionDir, nil
}
