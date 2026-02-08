package engine

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

// RuntimeType represents a container/execution runtime
type RuntimeType string

const (
	RuntimePodman  RuntimeType = "podman"
	RuntimeDocker  RuntimeType = "docker"
	RuntimeNerdctl RuntimeType = "nerdctl"
	RuntimeChroot  RuntimeType = "chroot"
)

// ValidRuntimes returns all valid runtime type values
func ValidRuntimes() []RuntimeType {
	return []RuntimeType{RuntimePodman, RuntimeDocker, RuntimeNerdctl, RuntimeChroot}
}

// IsContainerRuntime returns true if the runtime uses OCI containers
func (r RuntimeType) IsContainerRuntime() bool {
	return r == RuntimePodman || r == RuntimeDocker || r == RuntimeNerdctl
}

// Executor is the interface for running isolated build commands.
// Implementations include OCI container runtimes (podman, docker, nerdctl)
// and direct host execution via chroot.
type Executor interface {
	// Run executes a command with the given options
	Run(ctx context.Context, opts ContainerRunOpts) error

	// IsAvailable checks if the runtime binary is installed and functional
	IsAvailable() bool

	// BuilderImageExists checks if the builder image exists locally.
	// For chroot mode, this checks if the sysroot directory exists.
	BuilderImageExists(ctx context.Context) bool

	// DefaultImage returns the default container image or sysroot path
	DefaultImage() string

	// RuntimeType returns the type of this executor
	RuntimeType() RuntimeType
}

// NewExecutor creates an Executor for the given runtime type
func NewExecutor(runtime RuntimeType, defaultImage string, logger io.Writer) (Executor, error) {
	switch runtime {
	case RuntimePodman, RuntimeDocker, RuntimeNerdctl:
		return NewContainerRuntime(string(runtime), defaultImage, logger), nil
	case RuntimeChroot:
		return NewChrootExecutor(defaultImage, logger), nil
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}
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
