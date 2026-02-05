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
	defaultImage string
	logger       io.Writer
}

// NewContainerExecutor creates a new container executor
func NewContainerExecutor(defaultImage string, logger io.Writer) *ContainerExecutor {
	return &ContainerExecutor{
		defaultImage: defaultImage,
		logger:       logger,
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
	Image      string // Container image (uses defaultImage if empty)
	Mounts     []Mount
	Env        map[string]string
	Command    []string
	WorkDir    string
	Privileged bool
	Stdout     io.Writer
	Stderr     io.Writer
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

	// Determine image to use
	image := opts.Image
	if image == "" {
		image = e.defaultImage
	}

	// Add image and command
	args = append(args, image)
	args = append(args, opts.Command...)

	cmd := exec.CommandContext(ctx, "podman", args...)

	// Set up output streams
	var stderr bytes.Buffer

	// Use provided stdout/stderr or fall back to logger
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else if e.logger != nil {
		cmd.Stdout = e.logger
	}

	if opts.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, opts.Stderr)
	} else if e.logger != nil {
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
	cmd := exec.CommandContext(ctx, "podman", "image", "exists", e.defaultImage)
	return cmd.Run() == nil
}

// DefaultImage returns the default container image
func (e *ContainerExecutor) DefaultImage() string {
	return e.defaultImage
}
