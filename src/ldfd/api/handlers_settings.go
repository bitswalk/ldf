package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// SettingDefinition represents a single server setting with metadata
type SettingDefinition struct {
	Key            string      `json:"key"`
	Value          interface{} `json:"value"`
	Type           string      `json:"type"` // string, int, bool
	Description    string      `json:"description"`
	RebootRequired bool        `json:"rebootRequired"` // True for port, bind, etc.
	Category       string      `json:"category"`       // server, log, database, storage
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
	Sensitive      bool // If true, value is masked in responses
}

// settingsRegistry defines all available settings with their metadata
var settingsRegistry = []SettingMeta{
	// Server settings
	{"server.port", "int", "Port for HTTPS server to listen on", true, "server", false},
	{"server.bind", "string", "Network address to bind to (0.0.0.0 = all interfaces)", true, "server", false},

	// Logging settings
	{"log.output", "string", "Log destination: stdout, journald, or auto", false, "log", false},
	{"log.level", "string", "Minimum log level: debug, info, warn, error", false, "log", false},

	// Database settings
	{"database.path", "string", "Path to persist in-memory SQLite database on shutdown", true, "database", false},

	// Storage settings
	{"storage.type", "string", "Storage backend: local or s3", true, "storage", false},
	{"storage.local.path", "string", "Root directory for storing distribution artifacts", true, "storage", false},

	// WebUI settings
	{"webui.devmode", "bool", "Enable developer mode in the WebUI (shows debug console and logs)", false, "webui", false},

	// S3 storage settings
	{"storage.s3.provider", "string", "S3 provider type: garage, minio, aws, or other", true, "storage", false},
	{"storage.s3.endpoint", "string", "S3 base domain (e.g., s3.example.com)", true, "storage", false},
	{"storage.s3.region", "string", "AWS/S3 region", true, "storage", false},
	{"storage.s3.bucket", "string", "S3 bucket name for storing artifacts", true, "storage", false},
	{"storage.s3.access_key", "string", "S3 access key ID", true, "storage", true},
	{"storage.s3.secret_key", "string", "S3 secret access key", true, "storage", true},
}

// GetSettingsRegistry returns the settings registry for use by core/config.go
func GetSettingsRegistry() []SettingMeta {
	return settingsRegistry
}

// getSettingValue retrieves a setting value from viper with proper type handling
// If reveal is true, sensitive values are returned unmasked (for authorized users)
func getSettingValue(key, valueType string, sensitive bool, reveal bool) interface{} {
	if sensitive && !reveal {
		// Return masked value for sensitive settings
		val := viper.GetString(key)
		if val != "" {
			return "********"
		}
		return ""
	}

	switch valueType {
	case "int":
		return viper.GetInt(key)
	case "bool":
		return viper.GetBool(key)
	default:
		return viper.GetString(key)
	}
}

// findSettingByKey looks up a setting definition by its key
func findSettingByKey(key string) *SettingMeta {
	for i := range settingsRegistry {
		if settingsRegistry[i].Key == key {
			return &settingsRegistry[i]
		}
	}
	return nil
}

// handleGetSettings returns all server settings with metadata
// GET /v1/settings
// Query params:
//   - reveal=true: Return unmasked sensitive values (root only, already enforced by middleware)
func (a *API) handleGetSettings(c *gin.Context) {
	reveal := c.Query("reveal") == "true"
	settings := make([]SettingDefinition, 0, len(settingsRegistry))

	for _, reg := range settingsRegistry {
		settings = append(settings, SettingDefinition{
			Key:            reg.Key,
			Value:          getSettingValue(reg.Key, reg.Type, reg.Sensitive, reveal),
			Type:           reg.Type,
			Description:    reg.Description,
			RebootRequired: reg.RebootRequired,
			Category:       reg.Category,
		})
	}

	c.JSON(http.StatusOK, SettingsResponse{
		Settings: settings,
	})
}

// handleGetSetting returns a single setting by key
// GET /v1/settings/:key
// Query params:
//   - reveal=true: Return unmasked sensitive values (root only, already enforced by middleware)
func (a *API) handleGetSetting(c *gin.Context) {
	reveal := c.Query("reveal") == "true"

	// The key uses dot notation but gin splits on slashes
	// So we need to handle keys like "server.port" passed as path parameter
	key := c.Param("key")

	// Handle wildcard path for nested keys like storage.s3.endpoint
	if wildcard := c.Param("path"); wildcard != "" {
		key = key + wildcard
	}

	// Clean up leading slash if present
	key = strings.TrimPrefix(key, "/")

	reg := findSettingByKey(key)
	if reg == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Setting '%s' not found", key),
		})
		return
	}

	c.JSON(http.StatusOK, SettingDefinition{
		Key:            reg.Key,
		Value:          getSettingValue(reg.Key, reg.Type, reg.Sensitive, reveal),
		Type:           reg.Type,
		Description:    reg.Description,
		RebootRequired: reg.RebootRequired,
		Category:       reg.Category,
	})
}

// handleUpdateSetting updates a setting value
// PUT /v1/settings/:key
func (a *API) handleUpdateSetting(c *gin.Context) {
	// Handle the key parameter
	key := c.Param("key")
	if wildcard := c.Param("path"); wildcard != "" {
		key = key + wildcard
	}
	key = strings.TrimPrefix(key, "/")

	reg := findSettingByKey(key)
	if reg == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Setting '%s' not found", key),
		})
		return
	}

	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Validate and convert the value based on expected type
	var typedValue interface{}
	var stringValue string

	switch reg.Type {
	case "int":
		switch v := req.Value.(type) {
		case float64:
			typedValue = int(v)
			stringValue = fmt.Sprintf("%d", int(v))
		case int:
			typedValue = v
			stringValue = fmt.Sprintf("%d", v)
		case json.Number:
			intVal, err := v.Int64()
			if err != nil {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Error:   "Bad request",
					Code:    http.StatusBadRequest,
					Message: fmt.Sprintf("Invalid integer value for '%s'", key),
				})
				return
			}
			typedValue = int(intVal)
			stringValue = fmt.Sprintf("%d", intVal)
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Expected integer value for '%s', got %T", key, req.Value),
			})
			return
		}
	case "bool":
		switch v := req.Value.(type) {
		case bool:
			typedValue = v
			stringValue = fmt.Sprintf("%t", v)
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Expected boolean value for '%s', got %T", key, req.Value),
			})
			return
		}
	default: // string
		switch v := req.Value.(type) {
		case string:
			typedValue = v
			stringValue = v
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Expected string value for '%s', got %T", key, req.Value),
			})
			return
		}
	}

	// Update viper in-memory configuration
	viper.Set(key, typedValue)

	// Persist to database for restart persistence
	if err := a.database.SetSetting(key, stringValue); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to persist setting: %v", err),
		})
		return
	}

	// Apply hot-reload for supported settings
	if !reg.RebootRequired {
		a.applySettingChange(key, typedValue)
	}

	// Prepare response message
	message := "Setting updated successfully"
	if reg.RebootRequired {
		message = "Setting updated. Server reboot required for changes to take effect."
	}

	// Mask sensitive values in response
	responseValue := typedValue
	if reg.Sensitive {
		responseValue = "********"
	}

	c.JSON(http.StatusOK, UpdateSettingResponse{
		Key:            key,
		Value:          responseValue,
		RebootRequired: reg.RebootRequired,
		Message:        message,
	})
}

// applySettingChange applies hot-reloadable settings immediately
func (a *API) applySettingChange(key string, value interface{}) {
	switch key {
	case "log.level":
		if level, ok := value.(string); ok {
			// Update the global log level
			// The log package from common/logs should support SetLevel
			log.SetLevel(level)
		}
	case "log.output":
		// Log output change would require recreating the logger
		// For now, this will take effect on restart
	}
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

// handleResetDatabase resets the database to its default state
// POST /v1/settings/database/reset
// This is a destructive operation that requires confirmation
func (a *API) handleResetDatabase(c *gin.Context) {
	var req ResetDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Require explicit confirmation string to prevent accidental resets
	if req.Confirmation != "RESET_DATABASE" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid confirmation. Send confirmation: \"RESET_DATABASE\" to proceed.",
		})
		return
	}

	// Perform the reset
	if err := a.database.ResetToDefaults(); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to reset database: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, ResetDatabaseResponse{
		Success: true,
		Message: "Database has been reset to defaults. All user data has been deleted.",
	})
}

// LoadConfigFromDatabase loads settings from the database into viper.
// Database settings have the highest priority and override CLI/config file values.
// This should be called at server startup after the database is initialized.
func LoadConfigFromDatabase(database *db.Database) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings from database: %w", err)
	}

	// If no settings in database, nothing to load
	if len(settings) == 0 {
		return nil
	}

	for _, meta := range settingsRegistry {
		value, exists := settings[meta.Key]
		if !exists {
			// Setting not in database, keep current viper value
			continue
		}

		// Convert string value to appropriate type and set in viper
		switch meta.Type {
		case "int":
			var intVal int
			if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
				viper.Set(meta.Key, intVal)
			}
		case "bool":
			var boolVal bool
			if _, err := fmt.Sscanf(value, "%t", &boolVal); err == nil {
				viper.Set(meta.Key, boolVal)
			}
		default:
			viper.Set(meta.Key, value)
		}
	}

	return nil
}

// SyncConfigToDatabase syncs viper configuration values to the database settings table.
// Only writes settings that don't already exist in the database (preserves DB values).
// This ensures new settings are persisted while respecting user-configured values.
func SyncConfigToDatabase(database *db.Database) error {
	existingSettings, err := database.GetAllSettings()
	if err != nil {
		return fmt.Errorf("failed to get existing settings: %w", err)
	}

	for _, meta := range settingsRegistry {
		// Skip if setting already exists in database
		if _, exists := existingSettings[meta.Key]; exists {
			continue
		}

		// Setting doesn't exist in DB, write current viper value
		var stringValue string

		switch meta.Type {
		case "int":
			stringValue = fmt.Sprintf("%d", viper.GetInt(meta.Key))
		case "bool":
			stringValue = fmt.Sprintf("%t", viper.GetBool(meta.Key))
		default:
			stringValue = viper.GetString(meta.Key)
		}

		if err := database.SetSetting(meta.Key, stringValue); err != nil {
			return fmt.Errorf("failed to sync setting %s: %w", meta.Key, err)
		}
	}

	return nil
}
