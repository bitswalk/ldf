package client

import (
	"context"
	"fmt"
)

// BuildJob represents a build job
type BuildJob struct {
	ID               string       `json:"id"`
	DistributionID   string       `json:"distribution_id"`
	OwnerID          string       `json:"owner_id"`
	Status           string       `json:"status"`
	CurrentStage     string       `json:"current_stage"`
	TargetArch       string       `json:"target_arch"`
	ImageFormat      string       `json:"image_format"`
	ProgressPercent  int          `json:"progress_percent"`
	ArtifactPath     string       `json:"artifact_path,omitempty"`
	ArtifactChecksum string       `json:"artifact_checksum,omitempty"`
	ArtifactSize     int64        `json:"artifact_size"`
	ErrorMessage     string       `json:"error_message,omitempty"`
	ErrorStage       string       `json:"error_stage,omitempty"`
	RetryCount       int          `json:"retry_count"`
	MaxRetries       int          `json:"max_retries"`
	CreatedAt        string       `json:"created_at"`
	StartedAt        string       `json:"started_at,omitempty"`
	CompletedAt      string       `json:"completed_at,omitempty"`
	Stages           []BuildStage `json:"stages,omitempty"`
}

// BuildStage represents a single build pipeline stage
type BuildStage struct {
	ID              int64  `json:"id"`
	BuildID         string `json:"build_id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	ProgressPercent int    `json:"progress_percent"`
	DurationMs      int64  `json:"duration_ms"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// BuildLogEntry represents a build log entry
type BuildLogEntry struct {
	ID        int64  `json:"id"`
	BuildID   string `json:"build_id"`
	Stage     string `json:"stage"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// BuildJobsListResponse represents a list of build jobs
type BuildJobsListResponse struct {
	Count  int        `json:"count"`
	Builds []BuildJob `json:"builds"`
}

// BuildLogsResponse represents build log entries
type BuildLogsResponse struct {
	Count int             `json:"count"`
	Logs  []BuildLogEntry `json:"logs"`
}

// StartBuildRequest represents the request to start a build
type StartBuildRequest struct {
	Arch   string `json:"arch,omitempty"`
	Format string `json:"format,omitempty"`
}

// StartBuild triggers a build for a distribution
func (c *Client) StartBuild(ctx context.Context, distID string, req *StartBuildRequest) (*BuildJob, error) {
	var resp BuildJob
	if err := c.Post(ctx, fmt.Sprintf("/v1/distributions/%s/build", distID), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBuild returns a single build job with stages
func (c *Client) GetBuild(ctx context.Context, buildID string) (*BuildJob, error) {
	var resp BuildJob
	if err := c.Get(ctx, fmt.Sprintf("/v1/builds/%s", buildID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListDistributionBuilds returns builds for a distribution
func (c *Client) ListDistributionBuilds(ctx context.Context, distID string, opts *ListOptions) (*BuildJobsListResponse, error) {
	var resp BuildJobsListResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s/builds", distID)+opts.QueryString(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBuildLogs returns log entries for a build
func (c *Client) GetBuildLogs(ctx context.Context, buildID string) (*BuildLogsResponse, error) {
	var resp BuildLogsResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/builds/%s/logs", buildID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelBuild cancels a running build
func (c *Client) CancelBuild(ctx context.Context, buildID string) error {
	return c.Post(ctx, fmt.Sprintf("/v1/builds/%s/cancel", buildID), nil, nil)
}

// RetryBuild retries a failed build
func (c *Client) RetryBuild(ctx context.Context, buildID string) error {
	return c.Post(ctx, fmt.Sprintf("/v1/builds/%s/retry", buildID), nil, nil)
}

// ListActiveBuilds returns all active builds
func (c *Client) ListActiveBuilds(ctx context.Context) (*BuildJobsListResponse, error) {
	var resp BuildJobsListResponse
	if err := c.Get(ctx, "/v1/builds/active", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
