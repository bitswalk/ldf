package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the ldfd API
type Client struct {
	BaseURL      string
	HTTPClient   *http.Client
	Token        string
	RefreshToken string
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// APIError represents a structured API error
type APIError struct {
	StatusCode int
	ErrorCode  string
	Message    string
}

func (e *APIError) Error() string {
	var base string
	if e.ErrorCode != "" {
		base = fmt.Sprintf("%s: %s (HTTP %d)", e.ErrorCode, e.Message, e.StatusCode)
	} else {
		base = fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
	}

	switch e.StatusCode {
	case 401:
		return base + "\nHint: Authentication required. Run 'ldfctl login' first."
	case 403:
		return base + "\nHint: Permission denied. You don't have access to this resource."
	case 404:
		return base + "\nHint: Resource not found. Verify the ID or name is correct."
	case 409:
		return base + "\nHint: Resource already exists with that name."
	}
	return base
}

// ListOptions holds optional query parameters for list endpoints
type ListOptions struct {
	Limit       int
	Offset      int
	Status      string
	VersionType string
	Category    string
}

// QueryString builds a URL query string from the options
func (o *ListOptions) QueryString() string {
	if o == nil {
		return ""
	}
	params := []string{}
	if o.Limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", o.Limit))
	}
	if o.Offset > 0 {
		params = append(params, fmt.Sprintf("offset=%d", o.Offset))
	}
	if o.Status != "" {
		params = append(params, "status="+o.Status)
	}
	if o.VersionType != "" {
		params = append(params, "version_type="+o.VersionType)
	}
	if o.Category != "" {
		params = append(params, "category="+o.Category)
	}
	if len(params) == 0 {
		return ""
	}
	result := "?"
	for i, p := range params {
		if i > 0 {
			result += "&"
		}
		result += p
	}
	return result
}

// New creates a new API client
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.do(ctx, http.MethodGet, path, nil, result)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, http.MethodPost, path, body, result)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body, result interface{}) error {
	return c.do(ctx, http.MethodPut, path, body, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string, result interface{}) error {
	return c.do(ctx, http.MethodDelete, path, nil, result)
}

// Do performs a raw HTTP request and decodes the response
func (c *Client) Do(req *http.Request, result interface{}) error {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("X-Subject-Token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

// RawGet performs a GET request and returns the raw response body.
// The caller is responsible for closing the body.
func (c *Client) RawGet(ctx context.Context, path string) (*http.Response, error) {
	url := c.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("X-Subject-Token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				ErrorCode:  errResp.Error,
				Message:    errResp.Message,
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return resp, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, result interface{}) error {
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("X-Subject-Token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return c.handleResponse(resp, result)
}

func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				ErrorCode:  errResp.Error,
				Message:    errResp.Message,
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
