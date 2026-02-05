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

// NewHandler creates a new base handler
func NewHandler() *Handler {
	return &Handler{}
}

// HandleRoot returns API discovery information
// @Summary      Get API discovery information
// @Description  Returns API endpoints and general server information
// @Tags         Base
// @Produce      json
// @Success      200  {object}  APIInfo
// @Router       / [get]
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
			Docs:    "/swagger/index.html",
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
// @Summary      Get health status
// @Description  Returns the current health status of the server
// @Tags         Base
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /v1/health [get]
func (h *Handler) HandleHealth(c *gin.Context) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// HandleVersion returns version and build information for the server
// @Summary      Get version information
// @Description  Returns version and build information for the server
// @Tags         Base
// @Produce      json
// @Success      200  {object}  VersionResponse
// @Router       /v1/version [get]
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
