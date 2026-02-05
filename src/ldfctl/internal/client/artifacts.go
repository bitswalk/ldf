package client

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// Artifact represents an artifact path entry
type Artifact struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// ArtifactListResponse represents a list of artifacts
type ArtifactListResponse struct {
	DistributionID string     `json:"distribution_id,omitempty"`
	Artifacts      []Artifact `json:"artifacts"`
	Count          int        `json:"count"`
}

// StorageStatusResponse represents storage backend status
type StorageStatusResponse struct {
	Type      string `json:"type"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}

// ListArtifacts returns artifacts for a distribution
func (c *Client) ListArtifacts(ctx context.Context, distID string) (*ArtifactListResponse, error) {
	var resp ArtifactListResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s/artifacts", distID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAllArtifacts returns all artifacts across all distributions
func (c *Client) ListAllArtifacts(ctx context.Context) (*ArtifactListResponse, error) {
	var resp ArtifactListResponse
	if err := c.Get(ctx, "/v1/artifacts", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DownloadArtifact downloads an artifact to a local file
func (c *Client) DownloadArtifact(ctx context.Context, distID, artifactPath, destPath string) error {
	resp, err := c.RawGet(ctx, fmt.Sprintf("/v1/distributions/%s/artifacts/%s", distID, artifactPath))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// UploadArtifact uploads a file as an artifact
func (c *Client) UploadArtifact(ctx context.Context, distID, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			pw.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	url := fmt.Sprintf("%s/v1/distributions/%s/artifacts", c.BaseURL, distID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return c.Do(req, nil)
}

// DeleteArtifact deletes an artifact
func (c *Client) DeleteArtifact(ctx context.Context, distID, artifactPath string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/distributions/%s/artifacts/%s", distID, artifactPath), nil)
}

// GetArtifactURL returns a direct download URL for an artifact
func (c *Client) GetArtifactURL(ctx context.Context, distID, artifactPath string) (interface{}, error) {
	var resp interface{}
	if err := c.Get(ctx, fmt.Sprintf("/v1/distributions/%s/artifacts-url/%s", distID, artifactPath), &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetStorageStatus returns the storage backend status
func (c *Client) GetStorageStatus(ctx context.Context) (*StorageStatusResponse, error) {
	var resp StorageStatusResponse
	if err := c.Get(ctx, "/v1/storage/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
