// Package version provides common version information structures for ldf applications.
package version

import (
	"fmt"
	"runtime"
)

// Info holds version information for an ldf application.
// These values are typically set at build time via ldflags.
type Info struct {
	// Version is the full version string: "Phoenix (2025.12) - v1.0.0-4f9f297"
	Version string

	// ReleaseName is the codename for this release (e.g., "Phoenix")
	ReleaseName string

	// ReleaseVersion is the semantic version (e.g., "1.0.0")
	ReleaseVersion string

	// BuildDate is the ISO 8601 build timestamp
	BuildDate string

	// GitCommit is the short git commit hash
	GitCommit string
}

// Default values for unset version info
var (
	DefaultVersion        = "dev"
	DefaultReleaseName    = "Phoenix"
	DefaultReleaseVersion = "0.0.0"
	DefaultBuildDate      = "unknown"
	DefaultGitCommit      = "unknown"
)

// New creates a new Info with default values
func New() *Info {
	return &Info{
		Version:        DefaultVersion,
		ReleaseName:    DefaultReleaseName,
		ReleaseVersion: DefaultReleaseVersion,
		BuildDate:      DefaultBuildDate,
		GitCommit:      DefaultGitCommit,
	}
}

// GoVersion returns the Go runtime version
func GoVersion() string {
	return runtime.Version()
}

// String returns the full version string
func (i *Info) String() string {
	return i.Version
}

// Short returns a short version string (release version + commit)
func (i *Info) Short() string {
	return fmt.Sprintf("v%s-%s", i.ReleaseVersion, i.GitCommit)
}

// Full returns a detailed multi-line version string
func (i *Info) Full() string {
	return fmt.Sprintf(`%s
  Release:    %s
  Version:    %s
  Build Date: %s
  Git Commit: %s
  Go Version: %s`,
		i.Version,
		i.ReleaseName,
		i.ReleaseVersion,
		i.BuildDate,
		i.GitCommit,
		GoVersion(),
	)
}

// Map returns version info as a map (useful for JSON responses)
func (i *Info) Map() map[string]string {
	return map[string]string{
		"version":         i.Version,
		"release_name":    i.ReleaseName,
		"release_version": i.ReleaseVersion,
		"build_date":      i.BuildDate,
		"git_commit":      i.GitCommit,
		"go_version":      GoVersion(),
	}
}
