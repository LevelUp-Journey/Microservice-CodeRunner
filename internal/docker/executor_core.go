package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
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
	client        *client.Client
	dockerConfig  *DockerConfig
	parserFactory *ParserFactory
}

// NewDockerExecutor crea una nueva instancia de DockerExecutor
func NewDockerExecutor() (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	dockerConfig := DefaultDockerConfig()
	parserFactory := DefaultParserFactory()

	return &DockerExecutor{
		client:        cli,
		dockerConfig:  dockerConfig,
		parserFactory: parserFactory,
	}, nil
}

// Close cierra el cliente de Docker
func (e *DockerExecutor) Close() error {
	return e.client.Close()
}
