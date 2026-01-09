package settings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var log *logs.Logger

// SetLogger sets the logger for the settings package
func SetLogger(l *logs.Logger) {
	log = l
}

// NewHandler creates a new settings handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		database: cfg.Database,
	}
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
func getSettingValue(key, valueType string, sensitive bool, reveal bool) interface{} {
	if sensitive && !reveal {
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

// HandleGetAll returns all server settings with metadata
func (h *Handler) HandleGetAll(c *gin.Context) {
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

// HandleGet returns a single setting by key
func (h *Handler) HandleGet(c *gin.Context) {
	reveal := c.Query("reveal") == "true"

	key := c.Param("key")
	if wildcard := c.Param("path"); wildcard != "" {
		key = key + wildcard
	}
	key = strings.TrimPrefix(key, "/")

	reg := findSettingByKey(key)
	if reg == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
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

// HandleUpdate updates a setting value
func (h *Handler) HandleUpdate(c *gin.Context) {
	key := c.Param("key")
	if wildcard := c.Param("path"); wildcard != "" {
		key = key + wildcard
	}
	key = strings.TrimPrefix(key, "/")

	reg := findSettingByKey(key)
	if reg == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Setting '%s' not found", key),
		})
		return
	}

	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

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
				c.JSON(http.StatusBadRequest, common.ErrorResponse{
					Error:   "Bad request",
					Code:    http.StatusBadRequest,
					Message: fmt.Sprintf("Invalid integer value for '%s'", key),
				})
				return
			}
			typedValue = int(intVal)
			stringValue = fmt.Sprintf("%d", intVal)
		default:
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
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
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Expected boolean value for '%s', got %T", key, req.Value),
			})
			return
		}
	default:
		switch v := req.Value.(type) {
		case string:
			typedValue = v
			stringValue = v
		default:
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Expected string value for '%s', got %T", key, req.Value),
			})
			return
		}
	}

	viper.Set(key, typedValue)

	if err := h.database.SetSetting(key, stringValue); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to persist setting: %v", err),
		})
		return
	}

	if !reg.RebootRequired {
		h.applySettingChange(key, typedValue)
	}

	message := "Setting updated successfully"
	if reg.RebootRequired {
		message = "Setting updated. Server reboot required for changes to take effect."
	}

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
func (h *Handler) applySettingChange(key string, value interface{}) {
	switch key {
	case "log.level":
		if level, ok := value.(string); ok {
			log.SetLevel(level)
		}
	}
}

// HandleResetDatabase resets the database to its default state
func (h *Handler) HandleResetDatabase(c *gin.Context) {
	var req ResetDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if req.Confirmation != "RESET_DATABASE" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid confirmation. Send confirmation: \"RESET_DATABASE\" to proceed.",
		})
		return
	}

	if err := h.database.ResetToDefaults(); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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
func LoadConfigFromDatabase(database *db.Database) error {
	settings, err := database.GetAllSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings from database: %w", err)
	}

	if len(settings) == 0 {
		return nil
	}

	for _, meta := range settingsRegistry {
		value, exists := settings[meta.Key]
		if !exists {
			continue
		}

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
func SyncConfigToDatabase(database *db.Database) error {
	existingSettings, err := database.GetAllSettings()
	if err != nil {
		return fmt.Errorf("failed to get existing settings: %w", err)
	}

	for _, meta := range settingsRegistry {
		if _, exists := existingSettings[meta.Key]; exists {
			continue
		}

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
