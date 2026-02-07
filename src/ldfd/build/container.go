package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ContainerRuntime wraps OCI container runtime operations (podman, docker, nerdctl)
// for build isolation
type ContainerRuntime struct {
	binaryName   string // "podman", "docker", or "nerdctl"
	defaultImage string
	logger       io.Writer
}

// NewContainerRuntime creates a new container runtime executor
func NewContainerRuntime(binaryName, defaultImage string, logger io.Writer) *ContainerRuntime {
	return &ContainerRuntime{
		binaryName:   binaryName,
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
	Platform   string // --platform flag, e.g. "linux/arm64" (empty = native)
	Mounts     []Mount
	Env        map[string]string
	Command    []string
	WorkDir    string
	Privileged bool
	Stdout     io.Writer
	Stderr     io.Writer
}

// Run executes a command inside a container with the given options
func (e *ContainerRuntime) Run(ctx context.Context, opts ContainerRunOpts) error {
	args := []string{"run", "--rm"}

	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}

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

	cmd := exec.CommandContext(ctx, e.binaryName, args...)

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

// IsAvailable checks if the container runtime is installed and accessible
func (e *ContainerRuntime) IsAvailable() bool {
	cmd := exec.Command(e.binaryName, "version")
	return cmd.Run() == nil
}

// BuilderImageExists checks if the builder image exists locally
func (e *ContainerRuntime) BuilderImageExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, e.binaryName, "image", "exists", e.defaultImage)
	return cmd.Run() == nil
}

// DefaultImage returns the default container image
func (e *ContainerRuntime) DefaultImage() string {
	return e.defaultImage
}

// RuntimeType returns the type of this executor
func (e *ContainerRuntime) RuntimeType() RuntimeType {
	return RuntimeType(e.binaryName)
}
