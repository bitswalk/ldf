package builds

import (
	"time"

	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// Handler handles build-related HTTP requests
type Handler struct {
	distRepo     *db.DistributionRepository
	buildManager *build.Manager
}

// Config contains configuration options for the Handler
type Config struct {
	DistRepo     *db.DistributionRepository
	BuildManager *build.Manager
}

// StartBuildRequest represents the request to start a build
type StartBuildRequest struct {
	Arch       string `json:"arch,omitempty"`
	Format     string `json:"format,omitempty"`
	ClearCache bool   `json:"clear_cache,omitempty"`
}

// BuildJobResponse represents a build job with stages
type BuildJobResponse struct {
	db.BuildJob
	Stages []db.BuildStage `json:"stages,omitempty"`
}

// BuildJobsListResponse represents a list of build jobs
type BuildJobsListResponse struct {
	Count  int                `json:"count"`
	Builds []BuildJobResponse `json:"builds"`
}

// BuildLogsResponse represents build log entries
type BuildLogsResponse struct {
	Count int           `json:"count"`
	Logs  []db.BuildLog `json:"logs"`
}

// BuildStatusEvent is sent via SSE to update build status in real-time
type BuildStatusEvent struct {
	Status          db.BuildJobStatus `json:"status"`
	CurrentStage    string            `json:"current_stage"`
	ProgressPercent int               `json:"progress_percent"`
	Stages          []db.BuildStage   `json:"stages,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	ErrorStage      string            `json:"error_stage,omitempty"`
	ArtifactSize    int64             `json:"artifact_size,omitempty"`
}
