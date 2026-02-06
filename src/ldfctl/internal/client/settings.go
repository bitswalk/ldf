package client

import (
	"context"
	"fmt"
)

// Setting represents a single setting
type Setting struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default"`
	Reboot      bool        `json:"reboot"`
}

// SettingsListResponse represents all settings
type SettingsListResponse struct {
	Settings []Setting `json:"settings"`
}

// UpdateSettingRequest represents the request to update a setting
type UpdateSettingRequest struct {
	Value interface{} `json:"value"`
}

// ListSettings returns all settings
func (c *Client) ListSettings(ctx context.Context) (*SettingsListResponse, error) {
	var resp SettingsListResponse
	if err := c.Get(ctx, "/v1/settings", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSetting returns a single setting by key
func (c *Client) GetSetting(ctx context.Context, key string) (*Setting, error) {
	var resp Setting
	if err := c.Get(ctx, fmt.Sprintf("/v1/settings/%s", key), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSetting updates a setting value
func (c *Client) UpdateSetting(ctx context.Context, key string, value interface{}) (*Setting, error) {
	req := UpdateSettingRequest{Value: value}
	var resp Setting
	if err := c.Put(ctx, fmt.Sprintf("/v1/settings/%s", key), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResetDatabase triggers a database reset
func (c *Client) ResetDatabase(ctx context.Context) error {
	return c.Post(ctx, "/v1/settings/database/reset", nil, nil)
}
