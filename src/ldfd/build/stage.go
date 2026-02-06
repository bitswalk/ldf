// Package build provides a build pipeline for creating Linux distribution images.
package build

import (
	"context"
	"io"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Stage defines the interface for a single build pipeline stage
type Stage interface {
	// Name returns the stage name
	Name() db.BuildStageName

	// Validate checks whether this stage can run given the current context
	Validate(ctx context.Context, sc *StageContext) error

	// Execute runs the stage, updating progress via the callback
	Execute(ctx context.Context, sc *StageContext, progress ProgressFunc) error
}

// ProgressFunc reports stage progress (0-100) with an optional message
type ProgressFunc func(percent int, message string)

// StageContext holds shared state passed through the pipeline
type StageContext struct {
	BuildID        string
	DistributionID string
	OwnerID        string
	Config         *db.DistributionConfig
	TargetArch     db.TargetArch
	ImageFormat    db.ImageFormat
	WorkspacePath  string // Root workspace directory for this build
	SourcesDir     string // Where downloaded sources are extracted
	RootfsDir      string // Root filesystem being assembled
	OutputDir      string // Final output artifacts
	ConfigDir      string // Generated configs (kernel .config, fstab, etc.)
	LogWriter      io.Writer
	Components     []ResolvedComponent // Populated by resolve stage
	BoardProfile   *db.BoardProfile    // Populated by resolve stage when board_profile_id is set
	BuildEnv       *BuildEnvironment   // Populated by worker before pipeline starts

	// Artifact info populated by package stage
	ArtifactPath     string // Storage key of final artifact
	ArtifactChecksum string // SHA256 checksum
	ArtifactSize     int64  // Size in bytes
}

// ResolvedComponent holds a resolved component with its source artifact
type ResolvedComponent struct {
	Component    db.Component
	Version      string
	ArtifactPath string // Storage key of downloaded source
	LocalPath    string // Extracted path in workspace
}
