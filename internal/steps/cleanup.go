package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"code-runner/internal/pipeline"
)

// CleanupStep handles cleanup of temporary files and resources after execution
type CleanupStep struct {
	*BaseStep
}

// NewCleanupStep creates a new cleanup step
func NewCleanupStep() pipeline.PipelineStep {
	return &CleanupStep{
		BaseStep: NewBaseStep("cleanup", 5),
	}
}

// Execute performs cleanup of temporary resources
func (c *CleanupStep) Execute(ctx context.Context, data *pipeline.ExecutionData) error {
	c.AddLog(data, pipeline.LogLevelInfo, "Starting cleanup step")

	// Track cleanup statistics
	cleanupStart := time.Now()
	filesRemoved := 0
	foldersRemoved := 0
	errors := 0

	// Clean up temporary files
	if err := c.cleanupTempFiles(data, &filesRemoved, &errors); err != nil {
		c.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Failed to cleanup temp files: %v", err))
	}

	// Clean up working directory
	if err := c.cleanupWorkingDirectory(data, &foldersRemoved, &errors); err != nil {
		c.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Failed to cleanup working directory: %v", err))
	}

	// Clean up process-related resources
	if err := c.cleanupProcessResources(data); err != nil {
		c.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Failed to cleanup process resources: %v", err))
	}

	// Clear sensitive data from memory
	c.clearSensitiveData(data)

	cleanupDuration := time.Since(cleanupStart)

	// Log cleanup summary
	c.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf(
		"Cleanup completed: %d files removed, %d folders removed, %d errors, duration: %dms",
		filesRemoved, foldersRemoved, errors, cleanupDuration.Milliseconds(),
	))

	// Store cleanup metadata
	c.SetMetadata("cleanup_duration_ms", fmt.Sprintf("%d", cleanupDuration.Milliseconds()))
	c.SetMetadata("files_removed", fmt.Sprintf("%d", filesRemoved))
	c.SetMetadata("folders_removed", fmt.Sprintf("%d", foldersRemoved))
	c.SetMetadata("cleanup_errors", fmt.Sprintf("%d", errors))

	// Add cleanup summary to execution data
	if data.Metadata == nil {
		data.Metadata = make(map[string]string)
	}
	data.Metadata["cleanup_files_removed"] = fmt.Sprintf("%d", filesRemoved)
	data.Metadata["cleanup_duration_ms"] = fmt.Sprintf("%d", cleanupDuration.Milliseconds())

	// Cleanup should not fail the entire pipeline, so we don't return errors
	if errors > 0 {
		c.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Cleanup completed with %d errors", errors))
	} else {
		c.AddLog(data, pipeline.LogLevelInfo, "Cleanup completed successfully")
	}

	return nil
}

// cleanupTempFiles removes all temporary files created during execution
func (c *CleanupStep) cleanupTempFiles(data *pipeline.ExecutionData, filesRemoved, errors *int) error {
	if len(data.TempFiles) == 0 {
		c.AddLog(data, pipeline.LogLevelDebug, "No temporary files to cleanup")
		return nil
	}

	c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Cleaning up %d temporary files", len(data.TempFiles)))

	for _, filePath := range data.TempFiles {
		if err := c.removeFileOrDirectory(filePath); err != nil {
			c.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Failed to remove temp file %s: %v", filePath, err))
			*errors++
		} else {
			c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Removed temp file: %s", filePath))
			*filesRemoved++
		}
	}

	// Clear the temp files list
	data.TempFiles = make([]string, 0)

	return nil
}

// cleanupWorkingDirectory removes the working directory and its contents
func (c *CleanupStep) cleanupWorkingDirectory(data *pipeline.ExecutionData, foldersRemoved, errors *int) error {
	if data.WorkingDirectory == "" {
		c.AddLog(data, pipeline.LogLevelDebug, "No working directory to cleanup")
		return nil
	}

	// Check if directory exists
	if _, err := os.Stat(data.WorkingDirectory); os.IsNotExist(err) {
		c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Working directory already removed: %s", data.WorkingDirectory))
		data.WorkingDirectory = ""
		return nil
	}

	c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Cleaning up working directory: %s", data.WorkingDirectory))

	// Remove the entire directory tree
	if err := os.RemoveAll(data.WorkingDirectory); err != nil {
		c.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Failed to remove working directory %s: %v", data.WorkingDirectory, err))
		*errors++
		return err
	}

	c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Removed working directory: %s", data.WorkingDirectory))
	*foldersRemoved++

	// Clear the working directory reference
	data.WorkingDirectory = ""

	return nil
}

// cleanupProcessResources cleans up any remaining process-related resources
func (c *CleanupStep) cleanupProcessResources(data *pipeline.ExecutionData) error {
	// This method can be extended to clean up:
	// - Kill any remaining processes
	// - Close file handles
	// - Release network connections
	// - Clear environment variables

	c.AddLog(data, pipeline.LogLevelDebug, "Cleaning up process resources")

	// For now, we just ensure no sensitive environment variables are left
	if data.Config != nil && data.Config.EnvironmentVariables != nil {
		// Clear any sensitive environment variables that might have been set
		for key := range data.Config.EnvironmentVariables {
			if c.isSensitiveEnvVar(key) {
				os.Unsetenv(key)
				c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Cleared sensitive environment variable: %s", key))
			}
		}
	}

	return nil
}

// clearSensitiveData clears sensitive data from execution data
func (c *CleanupStep) clearSensitiveData(data *pipeline.ExecutionData) {
	// Clear sensitive code content from memory
	if len(data.Code) > 0 {
		// Overwrite the code string with zeros (best effort)
		codeBytes := []byte(data.Code)
		for i := range codeBytes {
			codeBytes[i] = 0
		}
		data.Code = "" // Clear the reference
	}

	// Clear sensitive outputs that might contain private information
	if len(data.StandardOutput) > 1000 {
		// Keep only first 1000 characters for debugging, clear the rest
		data.StandardOutput = data.StandardOutput[:1000] + "... (truncated for security)"
	}

	if len(data.StandardError) > 1000 {
		data.StandardError = data.StandardError[:1000] + "... (truncated for security)"
	}

	// Clear any sensitive metadata
	if data.Metadata != nil {
		sensitiveKeys := []string{
			"api_key", "token", "password", "secret",
			"private_key", "credentials", "auth",
		}

		for _, key := range sensitiveKeys {
			for metaKey := range data.Metadata {
				if c.containsIgnoreCase(metaKey, key) {
					data.Metadata[metaKey] = "[REDACTED]"
				}
			}
		}
	}

	c.AddLog(data, pipeline.LogLevelDebug, "Cleared sensitive data from memory")
}

// removeFileOrDirectory safely removes a file or directory
func (c *CleanupStep) removeFileOrDirectory(path string) error {
	if path == "" {
		return nil
	}

	// Security check: ensure we're not trying to remove system directories
	if c.isSystemPath(path) {
		return fmt.Errorf("attempted to remove system path: %s", path)
	}

	// Check if it's a file or directory
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil // Already removed
	}
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	} else {
		return os.Remove(path)
	}
}

// isSystemPath checks if a path is a system path that shouldn't be removed
func (c *CleanupStep) isSystemPath(path string) bool {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return true // If we can't resolve it, don't risk it
	}

	// Common system paths to protect (Windows and Unix)
	systemPaths := []string{
		"/", "/bin", "/usr", "/etc", "/var", "/home", "/root",
		"C:", "C:\\", "C:\\Windows", "C:\\Program Files", "C:\\Users",
		"/System", "/Library", "/Applications", // macOS
	}

	for _, sysPath := range systemPaths {
		sysSysPath, _ := filepath.Abs(sysPath)
		if absPath == sysSysPath || absPath+"/" == sysSysPath || absPath+"\\" == sysSysPath {
			return true
		}
	}

	// Check if it's within /tmp or similar temp directories
	tempDirs := []string{"/tmp", "/var/tmp", os.TempDir()}
	for _, tempDir := range tempDirs {
		if tempDir != "" {
			absTempDir, _ := filepath.Abs(tempDir)
			if c.isPathWithin(absPath, absTempDir) {
				return false // Safe to remove if within temp directory
			}
		}
	}

	// If path is not in temp directory and looks like a system path, protect it
	if len(absPath) < 10 || filepath.Dir(absPath) == "/" || filepath.Dir(absPath) == "C:\\" {
		return true
	}

	return false
}

// isPathWithin checks if path is within parentPath
func (c *CleanupStep) isPathWithin(path, parentPath string) bool {
	rel, err := filepath.Rel(parentPath, path)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && !c.containsString(rel, "..")
}

// isSensitiveEnvVar checks if an environment variable name is sensitive
func (c *CleanupStep) isSensitiveEnvVar(name string) bool {
	sensitiveVars := []string{
		"API_KEY", "TOKEN", "PASSWORD", "SECRET",
		"PRIVATE_KEY", "CREDENTIALS", "AUTH",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY",
		"GOOGLE_APPLICATION_CREDENTIALS", "GITHUB_TOKEN",
	}

	upperName := c.toUpperCase(name)
	for _, sensitive := range sensitiveVars {
		if c.containsIgnoreCase(upperName, sensitive) {
			return true
		}
	}

	return false
}

// Helper functions for string operations (to avoid external dependencies)

func (c *CleanupStep) containsIgnoreCase(s, substr string) bool {
	return c.containsString(c.toUpperCase(s), c.toUpperCase(substr))
}

func (c *CleanupStep) containsString(s, substr string) bool {
	return c.indexOfString(s, substr) >= 0
}

func (c *CleanupStep) indexOfString(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func (c *CleanupStep) toUpperCase(s string) string {
	result := make([]rune, len(s))
	for i, r := range s {
		if r >= 'a' && r <= 'z' {
			result[i] = r - 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

// CanSkip determines if cleanup can be skipped (never skip cleanup)
func (c *CleanupStep) CanSkip(data *pipeline.ExecutionData) bool {
	return false
}

// Rollback performs cleanup for cleanup step (nothing to rollback)
func (c *CleanupStep) Rollback(ctx context.Context, data *pipeline.ExecutionData) error {
	// Cleanup step doesn't need rollback since it's the final step
	// and its purpose is to clean up resources
	c.AddLog(data, pipeline.LogLevelInfo, "Cleanup step rollback completed (no action required)")
	return nil
}

// GetCleanupSummary returns a summary of what was cleaned up
func (c *CleanupStep) GetCleanupSummary() map[string]string {
	summary := make(map[string]string)

	if duration, exists := c.GetMetadata("cleanup_duration_ms"); exists {
		summary["duration_ms"] = duration
	}

	if files, exists := c.GetMetadata("files_removed"); exists {
		summary["files_removed"] = files
	}

	if folders, exists := c.GetMetadata("folders_removed"); exists {
		summary["folders_removed"] = folders
	}

	if errors, exists := c.GetMetadata("cleanup_errors"); exists {
		summary["errors"] = errors
	}

	return summary
}
