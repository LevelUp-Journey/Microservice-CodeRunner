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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// EnsureImagesReady verifica que todas las imágenes necesarias existen y las construye si faltan
func (e *DockerExecutor) EnsureImagesReady(ctx context.Context) error {
	log.Printf("🔍 Checking Docker images...")

	images := map[string]string{
		e.dockerConfig.CppImageName: "cpp",
	}

	for imageName, language := range images {
		_, _, err := e.client.ImageInspectWithRaw(ctx, imageName)
		if err != nil {
			if client.IsErrNotFound(err) {
				log.Printf("  ⚠️  Image %s not found, building...", imageName)
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

// ensureImage verifica que la imagen existe, si no la construye
func (e *DockerExecutor) ensureImage(ctx context.Context, imageName string) error {
	_, _, err := e.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Printf("  ⚠️  Image %s not found, building...", imageName)
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

	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("dockerfile not found at %s", dockerfilePath)
	}

	buildContext := filepath.Join("./docker/", language)
	tarBuf, err := createTarFromDirectory(buildContext)
	if err != nil {
		return fmt.Errorf("failed to create tar from directory: %w", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:        []string{imageName},
		Dockerfile:  "Dockerfile",
		Remove:      true,
		ForceRemove: true,
		PullParent:  true,
	}

	log.Printf("  📦 Building image %s from %s...", imageName, buildContext)
	log.Printf("  ⏳ This may take a few minutes on first build...")

	buildResp, err := e.client.ImageBuild(ctx, tarBuf, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer buildResp.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, buildResp.Body); err != nil {
		return fmt.Errorf("failed to read build output: %w", err)
	}

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

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

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
