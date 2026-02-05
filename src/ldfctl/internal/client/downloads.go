package client

import (
	"context"
	"fmt"
)

// DownloadJob represents a download job
type DownloadJob struct {
	ID              string `json:"id"`
	DistributionID  string `json:"distribution_id"`
	ComponentID     string `json:"component_id"`
	SourceURL       string `json:"source_url"`
	Status          string `json:"status"`
	Progress        int    `json:"progress"`
	TotalBytes      int64  `json:"total_bytes"`
	DownloadedBytes int64  `json:"downloaded_bytes"`
	Error           string `json:"error,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// DownloadJobsListResponse represents a list of download jobs
type DownloadJobsListResponse struct {
	Count int           `json:"count"`
	Jobs  []DownloadJob `json:"jobs"`
}

// StartDownloadsRequest represents the request to start downloads
type StartDownloadsRequest struct {
	Components []string `json:"components,omitempty"`
}

// ListDistributionDownloads returns downloads for a distribution
func (c *Client) ListDistributionDownloads(ctx context.Context, distID string, opts *ListOptions) (*DownloadJobsListResponse, error) {
	var resp DownloadJobsListResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s/downloads", distID)+opts.QueryString(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDownloadJob returns a single download job
func (c *Client) GetDownloadJob(ctx context.Context, jobID string) (*DownloadJob, error) {
	var resp DownloadJob
	if err := c.Get(ctx, fmt.Sprintf("/v1/downloads/%s", jobID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StartDistributionDownloads starts downloads for a distribution
func (c *Client) StartDistributionDownloads(ctx context.Context, distID string, req *StartDownloadsRequest) (interface{}, error) {
	var resp interface{}
	if err := c.Post(ctx, fmt.Sprintf("/v1/distributions/%s/downloads", distID), req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelDownload cancels a running download
func (c *Client) CancelDownload(ctx context.Context, jobID string) error {
	return c.Post(ctx, fmt.Sprintf("/v1/downloads/%s/cancel", jobID), nil, nil)
}

// RetryDownload retries a failed download
func (c *Client) RetryDownload(ctx context.Context, jobID string) error {
	return c.Post(ctx, fmt.Sprintf("/v1/downloads/%s/retry", jobID), nil, nil)
}

// ListActiveDownloads returns all active downloads
func (c *Client) ListActiveDownloads(ctx context.Context) (*DownloadJobsListResponse, error) {
	var resp DownloadJobsListResponse
	if err := c.Get(ctx, "/v1/downloads/active", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
