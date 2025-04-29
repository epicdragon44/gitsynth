package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// DockerService provides methods for interacting with Docker containers
type DockerService struct {
	client *client.Client
}

// ContainerConfig contains configuration for a Docker container
type ContainerConfig struct {
	ImageName string
	Env       []string
}

// ExecuteResult contains the result of command execution
type ExecuteResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// NewDockerService creates a new Docker service
func NewDockerService() (*DockerService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	cli.NegotiateAPIVersion(context.Background())
	return &DockerService{client: cli}, nil
}

// CreateContainer creates a new Docker container
func (s *DockerService) CreateContainer(ctx context.Context, config ContainerConfig) (string, error) {
	log.Printf("Creating container from image: %s", config.ImageName)

	// Container configuration
	containerConfig := &container.Config{
		Image: config.ImageName,
		Cmd:   []string{"/bin/sh"},
		Tty:   false,
		Env:   config.Env,
	}

	// Create the container
	containerName := fmt.Sprintf("gitsynth-worker-%d", time.Now().Unix())
	resp, err := s.client.ContainerCreate(
		ctx,
		containerConfig,
		&container.HostConfig{},
		nil,
		nil,
		containerName,
	)

	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Printf("Container created: %s (ID: %s)", containerName, resp.ID)
	return resp.ID, nil
}

// StartContainer starts a Docker container
func (s *DockerService) StartContainer(ctx context.Context, containerID string) error {
	log.Printf("Starting container: %s", containerID)

	err := s.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	log.Printf("Container started: %s", containerID)
	return nil
}

// ExecuteCommand executes a command in a running container
func (s *DockerService) ExecuteCommand(ctx context.Context, containerID string, cmd []string) (*ExecuteResult, error) {
	log.Printf("Executing command in container %s: %s", containerID, strings.Join(cmd, " "))

	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}

	// Create the exec instance
	execID, err := s.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Start the exec instance
	resp, err := s.client.ContainerExecAttach(ctx, execID.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer resp.Close()

	// Read the output
	stdout := new(strings.Builder)
	stderr := new(strings.Builder)
	_, err = stdcopy.StdCopy(stdout, stderr, resp.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	// Get the exit code
	inspect, err := s.client.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec instance: %w", err)
	}

	result := &ExecuteResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: inspect.ExitCode,
	}

	log.Printf("Command executed with exit code: %d", result.ExitCode)
	return result, nil
}

// DestroyContainer stops and removes a container
func (s *DockerService) DestroyContainer(ctx context.Context, containerID string) error {
	log.Printf("Destroying container: %s", containerID)

	// Stop the container
	timeoutSeconds := 10
	stopOptions := container.StopOptions{
		Timeout: &timeoutSeconds,  // Timeout in seconds as *int
	}
	err := s.client.ContainerStop(ctx, containerID, stopOptions)
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the container
	err = s.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Printf("Container destroyed: %s", containerID)
	return nil
}

// PullImage pulls a Docker image if it doesn't exist locally
func (s *DockerService) PullImage(ctx context.Context, imageName string) error {
	log.Printf("Pulling Docker image: %s", imageName)

	// Check if image exists locally
	_, _, err := s.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		log.Printf("Image %s already exists locally", imageName)
		return nil
	}

	// Pull the image
	pullOptions := types.ImagePullOptions{}
	pullReader, err := s.client.ImagePull(ctx, imageName, pullOptions)
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer pullReader.Close()

	// Wait for the pull to complete
	_, err = io.Copy(io.Discard, pullReader)
	if err != nil {
		return fmt.Errorf("failed during image pull: %w", err)
	}

	log.Printf("Image pulled successfully: %s", imageName)
	return nil
}