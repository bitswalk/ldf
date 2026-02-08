package build

import (
	"io"

	"github.com/bitswalk/ldf/src/ldfd/build/engine"
)

// Re-export engine types so that existing code referencing build.RuntimeType,
// build.Executor, build.ContainerRunOpts, etc. continues to compile.

// RuntimeType represents a container/execution runtime
type RuntimeType = engine.RuntimeType

const (
	RuntimePodman  = engine.RuntimePodman
	RuntimeDocker  = engine.RuntimeDocker
	RuntimeNerdctl = engine.RuntimeNerdctl
	RuntimeChroot  = engine.RuntimeChroot
)

// ValidRuntimes returns all valid runtime type values
func ValidRuntimes() []RuntimeType {
	return engine.ValidRuntimes()
}

// Executor is the interface for running isolated build commands.
type Executor = engine.Executor

// ContainerRunOpts holds options for running a container
type ContainerRunOpts = engine.ContainerRunOpts

// Mount represents a container volume mount
type Mount = engine.Mount

// NewExecutor creates an Executor for the given runtime type
func NewExecutor(runtime RuntimeType, defaultImage string, logger io.Writer) (Executor, error) {
	return engine.NewExecutor(runtime, defaultImage, logger)
}
