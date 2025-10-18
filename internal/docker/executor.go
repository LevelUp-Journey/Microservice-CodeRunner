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
	"runtime"
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

	// Crear directorio local permanente para guardar códigos compilados
	// Get current working directory to build absolute path
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	baseDir := filepath.Join(cwd, "compiled_test_codes")
	executionDir := filepath.Join(baseDir, config.ExecutionID.String())

	// Crear directorio si no existe con permisos de escritura para todos
	if err := os.MkdirAll(executionDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to create execution directory: %w", err)
	}

	// Cambiar propietario del directorio a UID 1000 (usuario coderunner en contenedor)
	// Solo en Linux/macOS - Windows no soporta chown
	if runtime.GOOS != "windows" {
		if err := os.Chown(executionDir, 1000, 1000); err != nil {
			log.Printf("  ⚠️  Warning: failed to chown directory: %v", err)
		}
	}

	log.Printf("  📁 Execution directory created: %s", executionDir)

	// Guardar el código fuente
	sourceFile := filepath.Join(executionDir, "solution.cpp")
	if err := os.WriteFile(sourceFile, []byte(config.SourceCode), 0666); err != nil {
		return nil, fmt.Errorf("failed to write source file: %w", err)
	}

	// Cambiar propietario del archivo también
	// Solo en Linux/macOS - Windows no soporta chown
	if runtime.GOOS != "windows" {
		if err := os.Chown(sourceFile, 1000, 1000); err != nil {
			log.Printf("  ⚠️  Warning: failed to chown source file: %v", err)
		}
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
		NetworkMode: container.NetworkMode(e.dockerConfig.NetworkMode),
		// ReadonlyRootfs disabled to allow compilation temporary files
		Binds: []string{
			fmt.Sprintf("%s:%s", executionDir, config.WorkDir), // Remove :ro to allow compilation
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

	log.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📊 EXECUTION COMPLETED")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("   ⏱️  Duration: %dms", result.ExecutionTimeMS)
	log.Printf("   � Exit code: %d", exitCode)
	log.Printf("   �📝 Stdout length: %d bytes", len(result.StdOut))
	log.Printf("   📝 Stderr length: %d bytes", len(result.StdErr))

	// Mostrar stdout si hay contenido
	if len(result.StdOut) > 0 {
		log.Printf("\n📤 STDOUT OUTPUT:")
		log.Printf("─────────────────────────────────────────────────────")
		// Mostrar las primeras 1000 caracteres o todo si es más corto
		stdoutPreview := result.StdOut
		if len(stdoutPreview) > 1000 {
			stdoutPreview = stdoutPreview[:1000] + "\n... (truncated, total: " + fmt.Sprintf("%d", len(result.StdOut)) + " bytes)"
		}
		log.Printf("%s", stdoutPreview)
		log.Printf("─────────────────────────────────────────────────────")
	}

	// Mostrar stderr si hay contenido (siempre importante)
	if len(result.StdErr) > 0 {
		log.Printf("\n⚠️  STDERR OUTPUT:")
		log.Printf("─────────────────────────────────────────────────────")
		log.Printf("%s", result.StdErr)
		log.Printf("─────────────────────────────────────────────────────")
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// Parsear resultados de los tests con la lista ordenada de IDs
	e.parseTestResults(result, config.TestIDs)

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
// Estrategia final:
// 1. Doctest solo muestra en el output los tests que FALLAN
// 2. Los que pasan no aparecen en el output
// 3. Extraer los UUIDs de los tests que aparecen en el output (fallidos)
// 4. Los tests que NO aparecen = PASARON
func (e *DockerExecutor) parseTestResults(result *ExecutionResult, testIDs []string) {
	output := result.StdOut
	lines := strings.Split(output, "\n")

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🔍 PARSING TEST RESULTS")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📋 Expected tests: %d", len(testIDs))
	log.Printf("📝 Test IDs to validate:")
	for i, id := range testIDs {
		log.Printf("   %d. %s", i+1, id)
	}
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Set de test IDs que aparecen en el output (los que fallaron)
	failedTestIDs := make(map[string]bool)

	// Variables para parsear estadísticas generales
	var totalTestCases, passedTestCases, failedTestCases int

	log.Printf("\n🔎 ANALYZING DOCTEST OUTPUT:")
	log.Printf("─────────────────────────────────────────────────────")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detectar línea con TEST CASE: <uuid>
		// Solo aparecen los tests que FALLAN
		if strings.Contains(line, "TEST CASE:") {
			// Extraer el UUID
			parts := strings.Split(line, "TEST CASE:")
			if len(parts) == 2 {
				testID := strings.TrimSpace(parts[1])
				// Verificar que parece un UUID
				if len(testID) >= 36 && strings.Count(testID, "-") >= 4 {
					failedTestIDs[testID] = true
					log.Printf("   ❌ FAILED TEST DETECTED: %s", testID)
				}
			}
		}

		// Parsear estadísticas generales al final
		// Formato: "[doctest] test cases:  3 |  3 passed | 0 failed | 0 skipped"
		if strings.Contains(line, "test cases:") {
			log.Printf("\n📊 DOCTEST SUMMARY LINE: %s", line)
			parts := strings.Split(line, "|")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.Contains(part, "passed") {
					fmt.Sscanf(part, "%d passed", &passedTestCases)
				} else if strings.Contains(part, "failed") {
					fmt.Sscanf(part, "%d failed", &failedTestCases)
				}
			}
			// El primer número antes del "|" es el total
			if len(parts) > 0 {
				firstPart := strings.TrimSpace(parts[0])
				if idx := strings.LastIndex(firstPart, ":"); idx != -1 {
					fmt.Sscanf(firstPart[idx+1:], "%d", &totalTestCases)
				}
			}
		}
	}

	log.Printf("─────────────────────────────────────────────────────")

	// Asignar estadísticas generales
	result.TotalTests = totalTestCases
	result.PassedTests = passedTestCases
	result.FailedTests = failedTestCases

	log.Printf("\n📈 EXECUTION STATISTICS:")
	log.Printf("   Total test cases: %d", totalTestCases)
	log.Printf("   ✅ Passed: %d", passedTestCases)
	log.Printf("   ❌ Failed: %d", failedTestCases)
	log.Printf("   🔍 Failed test IDs found in output: %d", len(failedTestIDs))

	log.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ MATCHING TEST IDs WITH RESULTS:")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Construir TestResults basándose en si el ID aparece en failedTestIDs
	// Si NO aparece en el output = PASÓ
	// Si aparece en el output = FALLÓ
	passedIDs := []string{}
	failedIDs := []string{}

	for i, testID := range testIDs {
		passed := !failedTestIDs[testID] // Si NO está en failed, entonces pasó
		result.TestResults = append(result.TestResults, TestResult{
			TestID:   testID,
			TestName: testID,
			Passed:   passed,
		})

		if passed {
			passedIDs = append(passedIDs, testID)
			log.Printf("   ✅ Test #%d: PASSED", i+1)
			log.Printf("      ID: %s", testID)
		} else {
			failedIDs = append(failedIDs, testID)
			log.Printf("   ❌ Test #%d: FAILED", i+1)
			log.Printf("      ID: %s", testID)
		}
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Calcular estadísticas
	passedCount := len(passedIDs)
	failedCount := len(failedIDs)

	log.Printf("\n🎯 FINAL RESULTS SUMMARY:")
	log.Printf("   Total: %d tests", len(testIDs))
	log.Printf("   ✅ Passed: %d tests (%d%%)", passedCount, (passedCount*100)/max(len(testIDs), 1))
	log.Printf("   ❌ Failed: %d tests (%d%%)", failedCount, (failedCount*100)/max(len(testIDs), 1))

	if passedCount > 0 {
		log.Printf("\n✅ PASSED TEST IDs:")
		for i, id := range passedIDs {
			log.Printf("   %d. %s", i+1, id)
		}
	}

	if failedCount > 0 {
		log.Printf("\n❌ FAILED TEST IDs:")
		for i, id := range failedIDs {
			log.Printf("   %d. %s", i+1, id)
		}
	}

	// Verificar consistencia con doctest
	if passedCount != passedTestCases {
		log.Printf("\n⚠️  WARNING: Mismatch detected!")
		log.Printf("   Expected passed (from doctest): %d", passedTestCases)
		log.Printf("   Actual passed (from parsing): %d", passedCount)
		log.Printf("   Difference: %d", passedTestCases-passedCount)
	} else {
		log.Printf("\n✅ Consistency check: PASSED")
		log.Printf("   Parsed results match doctest summary")
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// max retorna el máximo entre dos enteros
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
} // Close cierra el cliente de Docker
func (e *DockerExecutor) Close() error {
	return e.client.Close()
}
