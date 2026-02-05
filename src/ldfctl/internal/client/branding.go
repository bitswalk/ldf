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

// BrandingAssetInfo represents branding asset metadata
type BrandingAssetInfo struct {
	Asset       string `json:"asset"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Exists      bool   `json:"exists"`
}

// GetBrandingAsset downloads a branding asset to a local file
func (c *Client) GetBrandingAsset(ctx context.Context, asset, destPath string) error {
	resp, err := c.RawGet(ctx, fmt.Sprintf("/v1/branding/%s", asset))
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

// GetBrandingAssetInfo returns metadata about a branding asset
func (c *Client) GetBrandingAssetInfo(ctx context.Context, asset string) (*BrandingAssetInfo, error) {
	var resp BrandingAssetInfo
	if err := c.Get(ctx, fmt.Sprintf("/v1/branding/%s/info", asset), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadBrandingAsset uploads a branding asset
func (c *Client) UploadBrandingAsset(ctx context.Context, asset, filePath string) error {
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

	url := fmt.Sprintf("%s/v1/branding/%s", c.BaseURL, asset)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return c.Do(req, nil)
}

// DeleteBrandingAsset deletes a branding asset
func (c *Client) DeleteBrandingAsset(ctx context.Context, asset string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/branding/%s", asset), nil)
}
