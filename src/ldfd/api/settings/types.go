package settings

import (
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/security"
)

// Handler handles settings-related HTTP requests
type Handler struct {
	database      *db.Database
	secretManager *security.SecretManager
}

// Config contains configuration options for the Handler
type Config struct {
	Database      *db.Database
	SecretManager *security.SecretManager
}

// SettingDefinition represents a single server setting with metadata
type SettingDefinition struct {
	Key            string      `json:"key"`
	Value          interface{} `json:"value"`
	Type           string      `json:"type"`
	Description    string      `json:"description"`
	RebootRequired bool        `json:"rebootRequired"`
	Category       string      `json:"category"`
}

// SettingsResponse represents the response for GET /v1/settings
type SettingsResponse struct {
	Settings []SettingDefinition `json:"settings"`
}

// UpdateSettingRequest represents the request body for PUT /v1/settings/:key
type UpdateSettingRequest struct {
	Value interface{} `json:"value"`
}

// UpdateSettingResponse represents the response for PUT /v1/settings/:key
type UpdateSettingResponse struct {
	Key            string      `json:"key"`
	Value          interface{} `json:"value"`
	RebootRequired bool        `json:"rebootRequired"`
	Message        string      `json:"message"`
}

// SettingMeta contains metadata about a setting
type SettingMeta struct {
	Key            string
	Type           string
	Description    string
	RebootRequired bool
	Category       string
	Sensitive      bool
}

// ResetDatabaseRequest represents the request body for POST /v1/settings/database/reset
type ResetDatabaseRequest struct {
	Confirmation string `json:"confirmation" binding:"required"`
}

// ResetDatabaseResponse represents the response for POST /v1/settings/database/reset
type ResetDatabaseResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
