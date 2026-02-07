package build

import (
	"context"
	"fmt"
	"io"
)

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
