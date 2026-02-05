package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ContainerExecutor wraps Podman operations for build isolation
type ContainerExecutor struct {
	image  string
	logger io.Writer
}

// NewContainerExecutor creates a new container executor
func NewContainerExecutor(image string, logger io.Writer) *ContainerExecutor {
	return &ContainerExecutor{
		image:  image,
		logger: logger,
	}
}

// Mount represents a container volume mount
type Mount struct {
	Source   string
	Target   string
	ReadOnly bool
}

// ContainerRunOpts holds options for running a container
type ContainerRunOpts struct {
	Mounts     []Mount
	Env        map[string]string
	Command    []string
	WorkDir    string
	Privileged bool
}

// Run executes a command inside a container with the given options
func (e *ContainerExecutor) Run(ctx context.Context, opts ContainerRunOpts) error {
	args := []string{"run", "--rm"}

	if opts.Privileged {
		args = append(args, "--privileged")
	}

	// Add mounts
	for _, m := range opts.Mounts {
		mountStr := fmt.Sprintf("%s:%s", m.Source, m.Target)
		if m.ReadOnly {
			mountStr += ":ro"
		}
		args = append(args, "-v", mountStr)
	}

	// Add environment variables
	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add working directory
	if opts.WorkDir != "" {
		args = append(args, "-w", opts.WorkDir)
	}

	// Add image and command
	args = append(args, e.image)
	args = append(args, opts.Command...)

	cmd := exec.CommandContext(ctx, "podman", args...)

	// Capture output
	var stderr bytes.Buffer
	if e.logger != nil {
		cmd.Stdout = e.logger
		cmd.Stderr = io.MultiWriter(&stderr, e.logger)
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("container execution failed: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// IsAvailable checks if podman is installed and accessible
func (e *ContainerExecutor) IsAvailable() bool {
	cmd := exec.Command("podman", "version")
	return cmd.Run() == nil
}

// BuilderImageExists checks if the builder image exists locally
func (e *ContainerExecutor) BuilderImageExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "podman", "image", "exists", e.image)
	return cmd.Run() == nil
}
