package base

import (
	"net/http"
	"time"

	"github.com/bitswalk/ldf/src/common/version"
	"github.com/gin-gonic/gin"
)

var VersionInfo *version.Info

// SetVersionInfo sets the version info for the base package
func SetVersionInfo(v *version.Info) {
	VersionInfo = v
}

// Handler handles base HTTP requests (root, health, version)
type Handler struct{}

// NewHandler creates a new base handler
func NewHandler() *Handler {
	return &Handler{}
}

// APIInfo represents the root API discovery response
type APIInfo struct {
	Name        string           `json:"name" example:"ldfd"`
	Description string           `json:"description" example:"LDF Platform API Server"`
	Version     string           `json:"version" example:"1.0.0"`
	APIVersions []string         `json:"api_versions" example:"v1"`
	Endpoints   APIInfoEndpoints `json:"endpoints"`
}

// APIInfoEndpoints contains the available API endpoints
type APIInfoEndpoints struct {
	Health  string        `json:"health" example:"/v1/health"`
	Version string        `json:"version" example:"/v1/version"`
	APIv1   string        `json:"api_v1" example:"/v1/"`
	Auth    AuthEndpoints `json:"auth"`
}

// AuthEndpoints contains the authentication endpoints
type AuthEndpoints struct {
	Create   string `json:"create" example:"/auth/create"`
	Login    string `json:"login" example:"/auth/login"`
	Logout   string `json:"logout" example:"/auth/logout"`
	Refresh  string `json:"refresh" example:"/auth/refresh"`
	Validate string `json:"validate" example:"/auth/validate"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"healthy"`
	Timestamp string `json:"timestamp" example:"2024-01-15T10:30:00Z"`
}

// VersionResponse represents the version information response
type VersionResponse struct {
	Version        string `json:"version" example:"Phoenix (2025.12) - v1.0.0-4f9f297"`
	ReleaseName    string `json:"release_name" example:"Phoenix"`
	ReleaseVersion string `json:"release_version" example:"1.0.0"`
	BuildDate      string `json:"build_date" example:"2024-01-15T10:30:00Z"`
	GitCommit      string `json:"git_commit" example:"4f9f297"`
	GoVersion      string `json:"go_version" example:"go1.24"`
}

// HandleRoot returns API discovery information
func (h *Handler) HandleRoot(c *gin.Context) {
	info := APIInfo{
		Name:        "ldfd",
		Description: "LDF Platform API Server",
		Version:     VersionInfo.Version,
		APIVersions: []string{"v1"},
		Endpoints: APIInfoEndpoints{
			Health:  "/v1/health",
			Version: "/v1/version",
			APIv1:   "/v1/",
			Auth: AuthEndpoints{
				Create:   "/auth/create",
				Login:    "/auth/login",
				Logout:   "/auth/logout",
				Refresh:  "/auth/refresh",
				Validate: "/auth/validate",
			},
		},
	}

	c.JSON(http.StatusOK, info)
}

// HandleHealth returns the current health status of the server
func (h *Handler) HandleHealth(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// HandleVersion returns version and build information for the server
func (h *Handler) HandleVersion(c *gin.Context) {
	response := VersionResponse{
		Version:        VersionInfo.Version,
		ReleaseName:    VersionInfo.ReleaseName,
		ReleaseVersion: VersionInfo.ReleaseVersion,
		BuildDate:      VersionInfo.BuildDate,
		GitCommit:      VersionInfo.GitCommit,
		GoVersion:      version.GoVersion(),
	}

	c.JSON(http.StatusOK, response)
}
