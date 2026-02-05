package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// LanguagePackMeta represents language pack metadata
type LanguagePackMeta struct {
	Locale  string `json:"locale"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Author  string `json:"author,omitempty"`
}

// LanguagePackListResponse represents a list of language packs
type LanguagePackListResponse struct {
	LanguagePacks []LanguagePackMeta `json:"language_packs"`
}

// LanguagePackResponse represents a single language pack with dictionary
type LanguagePackResponse struct {
	Locale     string          `json:"locale"`
	Name       string          `json:"name"`
	Version    string          `json:"version"`
	Author     string          `json:"author,omitempty"`
	Dictionary json.RawMessage `json:"dictionary"`
}

// ListLangPacks returns all language packs
func (c *Client) ListLangPacks(ctx context.Context) (*LanguagePackListResponse, error) {
	var resp LanguagePackListResponse
	if err := c.Get(ctx, "/v1/language-packs", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetLangPack returns a single language pack by locale
func (c *Client) GetLangPack(ctx context.Context, locale string) (*LanguagePackResponse, error) {
	var resp LanguagePackResponse
	if err := c.Get(ctx, fmt.Sprintf("/v1/language-packs/%s", locale), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadLangPack uploads a language pack archive
func (c *Client) UploadLangPack(ctx context.Context, filePath string) error {
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

	url := fmt.Sprintf("%s/v1/language-packs", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return c.Do(req, nil)
}

// DeleteLangPack deletes a language pack by locale
func (c *Client) DeleteLangPack(ctx context.Context, locale string) error {
	return c.Delete(ctx, fmt.Sprintf("/v1/language-packs/%s", locale), nil)
}
