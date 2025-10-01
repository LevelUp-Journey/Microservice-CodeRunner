package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Executor define la interfaz para ejecutar código en Docker
type Executor interface {
	Execute(ctx context.Context, config *ExecutionConfig) (*ExecutionResult, error)
	BuildImage(ctx context.Context, language string) error
	Cleanup(ctx context.Context, containerID string) error
	EnsureImagesReady(ctx context.Context) error
}

// DockerExecutor implementa Executor usando Docker
type DockerExecutor struct {
	client       *client.Client
	dockerConfig *DockerConfig
}

// NewDockerExecutor crea una nueva instancia de DockerExecutor
func NewDockerExecutor() (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerExecutor{
		client:       cli,
		dockerConfig: DefaultDockerConfig(),
	}, nil
}

// EnsureImagesReady verifica que todas las imágenes necesarias existen y las construye si faltan
func (e *DockerExecutor) EnsureImagesReady(ctx context.Context) error {
	log.Printf("🔍 Checking Docker images...")

	// Lista de imágenes necesarias con sus lenguajes
	images := map[string]string{
		e.dockerConfig.CppImageName: "cpp",
	}

	for imageName, language := range images {
		// Verificar si la imagen existe
		_, _, err := e.client.ImageInspectWithRaw(ctx, imageName)
		if err != nil {
			if client.IsErrNotFound(err) {
				log.Printf("  ⚠️  Image %s not found, building...", imageName)
				// Construir la imagen
				if err := e.BuildImage(ctx, language); err != nil {
					return fmt.Errorf("failed to build image %s: %w", imageName, err)
				}
				log.Printf("  ✅ Image %s ready", imageName)
			} else {
				return fmt.Errorf("failed to inspect image %s: %w", imageName, err)
			}
		} else {
			log.Printf("  ✅ Image %s found", imageName)
		}
	}

	log.Printf("✅ All Docker images are ready")
	return nil
}

// Execute ejecuta el código en un contenedor Docker
func (e *DockerExecutor) Execute(ctx context.Context, config *ExecutionConfig) (*ExecutionResult, error) {
	startTime := time.Now()

	log.Printf("🐳 Starting Docker execution for ExecutionID: %s", config.ExecutionID)

	result := &ExecutionResult{
		ExecutionID: config.ExecutionID,
		Success:     false,
	}

	// Crear directorio temporal para el código
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("coderunner-%s", config.ExecutionID.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("  📁 Temp directory created: %s", tempDir)

	// Guardar el código fuente
	sourceFile := filepath.Join(tempDir, "solution.cpp")
	if err := os.WriteFile(sourceFile, []byte(config.SourceCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	log.Printf("  💾 Source code saved: %s (%d bytes)", sourceFile, len(config.SourceCode))

	// Verificar que la imagen existe
	if err := e.ensureImage(ctx, config.ImageName); err != nil {
		return nil, fmt.Errorf("failed to ensure image: %w", err)
	}

	// Configurar el contenedor
	containerConfig := &container.Config{
		Image:        config.ImageName,
		WorkingDir:   config.WorkDir,
		Cmd:          []string{"/bin/bash", "-c", "g++ -std=c++17 solution.cpp -o solution && ./solution"},
		Tty:          false,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Configurar límites de recursos
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   config.MemoryLimitMB * 1024 * 1024, // Convert MB to bytes
			NanoCPUs: int64(config.CPULimit * 1e9),       // Convert to nano CPUs
		},
		NetworkMode:    container.NetworkMode(e.dockerConfig.NetworkMode),
		ReadonlyRootfs: e.dockerConfig.ReadOnlyRootFS,
		Binds: []string{
			fmt.Sprintf("%s:%s:ro", tempDir, config.WorkDir),
		},
		CapDrop:     e.dockerConfig.DropCapabilities,
		SecurityOpt: e.dockerConfig.SecurityOpt,
	}

	log.Printf("  🔧 Container configured: Memory=%dMB, CPU=%.1f cores, Timeout=%ds",
		config.MemoryLimitMB, config.CPULimit, config.TimeoutSeconds)

	// Crear el contenedor
	containerName := fmt.Sprintf("coderunner-%s", config.ExecutionID.String())
	resp, err := e.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	containerID := resp.ID

	log.Printf("  ✅ Container created: %s", containerID[:12])

	// Asegurar limpieza del contenedor
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Cleanup(cleanupCtx, containerID); err != nil {
			log.Printf("  ⚠️  Warning: failed to cleanup container: %v", err)
		}
	}()

	// Crear contexto con timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Iniciar el contenedor
	if err := e.client.ContainerStart(execCtx, containerID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("  🚀 Container started")

	// Esperar a que termine o timeout
	statusCh, errCh := e.client.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)

	var exitCode int64
	select {
	case err := <-errCh:
		if err != nil {
			result.TimedOut = true
			result.ErrorType = "timeout"
			result.ErrorMessage = fmt.Sprintf("Execution timed out after %d seconds", config.TimeoutSeconds)
			log.Printf("  ⏱️  Execution timed out")
			return result, nil
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
		log.Printf("  ✅ Container finished with exit code: %d", exitCode)
	}

	// Capturar logs del contenedor
	out, err := e.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer out.Close()

	// Separar stdout y stderr
	var stdout, stderr strings.Builder
	if _, err := stdcopy.StdCopy(&stdout, &stderr, out); err != nil {
		log.Printf("  ⚠️  Warning: error reading logs: %v", err)
	}

	result.StdOut = stdout.String()
	result.StdErr = stderr.String()
	result.ExitCode = int(exitCode)
	result.ExecutionTimeMS = time.Since(startTime).Milliseconds()
	result.Success = (exitCode == 0)

	log.Printf("  📊 Execution completed in %dms", result.ExecutionTimeMS)
	log.Printf("  📝 Output length: stdout=%d, stderr=%d", len(result.StdOut), len(result.StdErr))

	// Parsear resultados de los tests
	e.parseTestResults(result)

	return result, nil
}

// ensureImage verifica que la imagen existe, si no la construye
func (e *DockerExecutor) ensureImage(ctx context.Context, imageName string) error {
	_, _, err := e.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Printf("  🔨 Image %s not found, building...", imageName)
			return e.BuildImage(ctx, "cpp")
		}
		return fmt.Errorf("failed to inspect image: %w", err)
	}
	log.Printf("  ✅ Image %s found", imageName)
	return nil
}

// BuildImage construye la imagen Docker para un lenguaje específico
func (e *DockerExecutor) BuildImage(ctx context.Context, language string) error {
	log.Printf("🔨 Building Docker image for %s...", language)

	imageName := e.dockerConfig.CppImageName
	dockerfilePath := fmt.Sprintf("./docker/%s/Dockerfile", language)

	// Verificar si el Dockerfile existe
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found at %s", dockerfilePath)
	}

	// Crear tar del contexto de build
	buildContext := filepath.Join("./docker/", language)
	tarBuf, err := createTarFromDirectory(buildContext)
	if err != nil {
		return fmt.Errorf("failed to create tar from directory: %w", err)
	}

	// Opciones de build
	buildOptions := types.ImageBuildOptions{
		Tags:        []string{imageName},
		Dockerfile:  "Dockerfile",
		Remove:      true,
		ForceRemove: true,
		PullParent:  true,
	}

	// Construir la imagen
	log.Printf("  📦 Building image %s from %s...", imageName, buildContext)
	log.Printf("  ⏳ This may take a few minutes on first build...")
	buildResp, err := e.client.ImageBuild(ctx, tarBuf, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResp.Body.Close()

	// Leer la salida del build
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, buildResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read build output: %w", err)
	}

	// Verificar si hubo errores en el build
	output := buf.String()
	if strings.Contains(strings.ToLower(output), "error") {
		log.Printf("  ❌ Build output:\n%s", output)
		return fmt.Errorf("build failed, check output above")
	}

	log.Printf("  ✅ Image %s built successfully", imageName)
	return nil
}

// createTarFromDirectory crea un archivo tar desde un directorio
func createTarFromDirectory(dir string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Crear header del tar
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// Actualizar el nombre para ser relativo al directorio
		relPath, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Escribir header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Si es un archivo regular, escribir contenido
		if !fi.IsDir() {
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return buf, nil
}

// Cleanup limpia recursos del contenedor
func (e *DockerExecutor) Cleanup(ctx context.Context, containerID string) error {
	log.Printf("  🧹 Cleaning up container: %s", containerID[:12])

	// Detener el contenedor si está corriendo
	timeout := 5
	if err := e.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		log.Printf("  ⚠️  Warning: failed to stop container: %v", err)
	}

	// Remover el contenedor
	if err := e.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Printf("  ✅ Container cleaned up")
	return nil
}

// parseTestResults parsea la salida de doctest para extraer resultados de tests individuales
func (e *DockerExecutor) parseTestResults(result *ExecutionResult) {
	output := result.StdOut

	// Parsear resultados de doctest
	// Ejemplo de salida de doctest:
	// [doctest] doctest version is "2.4.9"
	// [doctest] run with "--help" for options
	// ===============================================================================
	// test_id_123
	// ===============================================================================
	// /workspace/solution.cpp:20: PASSED:
	//   CHECK( fibonacci(1) == 1 )
	// with expansion:
	//   1 == 1
	// ===============================================================================
	// [doctest] test cases:  3 |  3 passed | 0 failed | 0 skipped
	// [doctest] assertions: 10 | 10 passed | 0 failed |

	lines := strings.Split(output, "\n")
	passedTestIDs := make(map[string]bool)
	failedTestIDs := make(map[string]bool)
	currentTestID := ""

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Detectar inicio de un test case (el ID aparece después de una línea de "===")
		// Buscar patrón: línea con solo "===" seguida de un UUID/ID
		if strings.HasPrefix(line, "===") && len(line) > 10 {
			// La siguiente línea debería ser el test ID
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				// Verificar si parece un UUID o test ID (formato: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
				if len(nextLine) > 0 && !strings.HasPrefix(nextLine, "===") && !strings.HasPrefix(nextLine, "[doctest]") {
					currentTestID = nextLine
				}
			}
		}

		// Detectar si el test pasó o falló
		if currentTestID != "" {
			if strings.Contains(line, "PASSED:") || strings.Contains(line, "passed") {
				passedTestIDs[currentTestID] = true
			} else if strings.Contains(line, "FAILED:") || strings.Contains(line, "ERROR:") {
				failedTestIDs[currentTestID] = true
			}
		}

		// Buscar línea de test cases para estadísticas generales
		if strings.Contains(line, "test cases:") {
			// Parsear: [doctest] test cases:  3 |  3 passed | 0 failed | 0 skipped
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.Contains(part, "passed") {
						fmt.Sscanf(part, "%d passed", &result.PassedTests)
					} else if strings.Contains(part, "failed") {
						fmt.Sscanf(part, "%d failed", &result.FailedTests)
					}
				}
			}
		}

		// Buscar línea de assertions para total
		if strings.Contains(line, "assertions:") {
			parts := strings.Split(line, "|")
			if len(parts) >= 1 {
				firstPart := strings.TrimSpace(parts[0])
				if idx := strings.Index(firstPart, "assertions:"); idx != -1 {
					fmt.Sscanf(firstPart[idx:], "assertions: %d", &result.TotalTests)
				}
			}
		}
	}

	// Construir TestResults basado en los IDs encontrados
	for testID := range passedTestIDs {
		result.TestResults = append(result.TestResults, TestResult{
			TestID:   testID,
			TestName: testID,
			Passed:   true,
		})
	}

	for testID := range failedTestIDs {
		// Solo agregar si no está ya en passed (por si hay conflicto)
		if !passedTestIDs[testID] {
			result.TestResults = append(result.TestResults, TestResult{
				TestID:   testID,
				TestName: testID,
				Passed:   false,
			})
		}
	}

	log.Printf("  🧪 Test results: %d/%d passed", result.PassedTests, result.TotalTests)
	log.Printf("  📋 Individual test results: %d passed IDs, %d failed IDs", len(passedTestIDs), len(failedTestIDs))
} // Close cierra el cliente de Docker
func (e *DockerExecutor) Close() error {
	return e.client.Close()
}
