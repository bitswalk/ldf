package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ChrootExecutor runs build commands directly on the host, either via
// chroot into a sysroot directory or as plain shell commands.
type ChrootExecutor struct {
	sysrootPath string // Path to sysroot directory (empty = direct host execution)
	logger      io.Writer
}

// NewChrootExecutor creates a new chroot executor.
// If sysrootPath is empty, commands run directly on the host.
// If sysrootPath is set, commands run inside a chroot.
func NewChrootExecutor(sysrootPath string, logger io.Writer) *ChrootExecutor {
	return &ChrootExecutor{
		sysrootPath: sysrootPath,
		logger:      logger,
	}
}

// Run executes a command on the host or inside a chroot.
// Mounts are translated to bind-mounts when using chroot, or used to set
// up the environment for direct execution.
func (e *ChrootExecutor) Run(ctx context.Context, opts ContainerRunOpts) error {
	if e.sysrootPath != "" {
		return e.runInChroot(ctx, opts)
	}
	return e.runDirect(ctx, opts)
}

// runDirect executes commands directly on the host
func (e *ChrootExecutor) runDirect(ctx context.Context, opts ContainerRunOpts) error {
	if len(opts.Command) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmd := exec.CommandContext(ctx, opts.Command[0], opts.Command[1:]...)

	// Set working directory
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}

	// Set environment variables (inherit host env + overrides)
	cmd.Env = os.Environ()
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up output streams
	var stderr bytes.Buffer

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
		return fmt.Errorf("direct execution failed: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// runInChroot executes commands inside a chroot environment
func (e *ChrootExecutor) runInChroot(ctx context.Context, opts ContainerRunOpts) error {
	if len(opts.Command) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Bind-mount sources into the sysroot
	var mountedPaths []string
	for _, m := range opts.Mounts {
		targetPath := e.sysrootPath + m.Target
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			e.unmountAll(mountedPaths)
			return fmt.Errorf("failed to create mount target %s: %w", targetPath, err)
		}

		mountArgs := []string{"--bind"}
		if m.ReadOnly {
			mountArgs = []string{"--bind", "-o", "ro"}
		}
		mountArgs = append(mountArgs, m.Source, targetPath)

		if err := exec.CommandContext(ctx, "mount", mountArgs...).Run(); err != nil {
			e.unmountAll(mountedPaths)
			return fmt.Errorf("failed to bind-mount %s to %s: %w", m.Source, targetPath, err)
		}
		mountedPaths = append(mountedPaths, targetPath)
	}
	defer e.unmountAll(mountedPaths)

	// Build the chroot command
	args := []string{e.sysrootPath}
	args = append(args, opts.Command...)

	cmd := exec.CommandContext(ctx, "chroot", args...)

	// Set environment variables
	cmd.Env = []string{}
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	// Add PATH for the chroot environment
	cmd.Env = append(cmd.Env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")

	// Set up output streams
	var stderr bytes.Buffer

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
		return fmt.Errorf("chroot execution failed: %w\nstderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// unmountAll cleans up bind mounts in reverse order
func (e *ChrootExecutor) unmountAll(paths []string) {
	for i := len(paths) - 1; i >= 0; i-- {
		if err := exec.Command("umount", paths[i]).Run(); err != nil {
			log.Warn("Failed to unmount bind-mount", "path", paths[i], "error", err)
		}
	}
}

// IsAvailable checks if chroot execution is possible
func (e *ChrootExecutor) IsAvailable() bool {
	if e.sysrootPath == "" {
		// Direct mode: always available
		return true
	}
	// Chroot mode: check that chroot binary exists and sysroot is accessible
	if _, err := exec.LookPath("chroot"); err != nil {
		return false
	}
	info, err := os.Stat(e.sysrootPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// BuilderImageExists checks if the sysroot directory exists.
// For direct mode (empty sysrootPath), always returns true.
func (e *ChrootExecutor) BuilderImageExists(ctx context.Context) bool {
	if e.sysrootPath == "" {
		return true
	}
	info, err := os.Stat(e.sysrootPath)
	return err == nil && info.IsDir()
}

// DefaultImage returns the sysroot path
func (e *ChrootExecutor) DefaultImage() string {
	return e.sysrootPath
}

// RuntimeType returns RuntimeChroot
func (e *ChrootExecutor) RuntimeType() RuntimeType {
	return RuntimeChroot
}
