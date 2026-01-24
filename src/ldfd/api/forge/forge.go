package forge

import (
	"context"
	"net/http"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/forge"
	"github.com/gin-gonic/gin"
)

var log *logs.Logger

// SetLogger sets the logger for the forge API package
func SetLogger(l *logs.Logger) {
	log = l
}

// NewHandler creates a new forge handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		registry: cfg.Registry,
	}
}

// HandleDetect detects the forge type for a given URL and returns defaults
func (h *Handler) HandleDetect(c *gin.Context) {
	var req DetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Create context with timeout for API calls
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Detect forge type
	forgeType := h.registry.DetectForge(req.URL)
	provider := h.registry.GetProvider(forgeType)

	// Parse repo info
	repoInfo, err := provider.ParseRepoInfo(req.URL)
	if err != nil {
		// Non-fatal - we can still return forge type
		repoInfo = nil
	}

	// Get defaults (URL template and version filter)
	defaults, err := provider.GetDefaults(ctx, req.URL)
	if err != nil {
		// Non-fatal - we can still return forge type
		defaults = nil
	}

	c.JSON(http.StatusOK, DetectResponse{
		ForgeType:  forgeType,
		RepoInfo:   repoInfo,
		Defaults:   defaults,
		ForgeTypes: forge.GetForgeTypeInfo(),
	})
}

// HandlePreviewFilter previews the effect of a version filter on actual versions
func (h *Handler) HandlePreviewFilter(c *gin.Context) {
	var req PreviewFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Determine forge type
	var forgeType forge.ForgeType
	if req.ForgeType != "" {
		forgeType = forge.ForgeType(req.ForgeType)
	} else {
		forgeType = h.registry.DetectForge(req.URL)
	}

	provider := h.registry.GetProvider(forgeType)

	// Discover versions from upstream
	versions, err := provider.DiscoverVersions(ctx, req.URL)
	if err != nil {
		c.JSON(http.StatusBadGateway, common.ErrorResponse{
			Error:   "Failed to fetch versions",
			Code:    http.StatusBadGateway,
			Message: err.Error(),
		})
		return
	}

	// Determine which filter to use
	filterStr := req.VersionFilter
	filterSource := "custom"

	if filterStr == "" {
		// Get default filter from provider
		defaults, err := provider.GetDefaults(ctx, req.URL)
		if err == nil && defaults != nil {
			filterStr = defaults.VersionFilter
			filterSource = defaults.FilterSource
		}
	}

	// Parse and apply filter
	versionFilter := forge.ParseVersionFilter(filterStr)

	// Build preview results (limit to first 50 versions for UI)
	maxVersions := 50
	if len(versions) < maxVersions {
		maxVersions = len(versions)
	}

	previews := make([]VersionPreview, 0, maxVersions)
	includedCount := 0
	excludedCount := 0

	// Convert versions to strings for filter with reasons
	versionStrings := make([]string, len(versions))
	for i, v := range versions {
		versionStrings[i] = v.Version
	}

	filterResults := versionFilter.FilterWithReasons(versionStrings)

	for i := 0; i < len(versions) && i < maxVersions; i++ {
		v := versions[i]
		result := filterResults[i]

		previews = append(previews, VersionPreview{
			Version:      v.Version,
			Included:     result.Included,
			Reason:       result.Reason,
			IsPrerelease: v.IsPrerelease,
		})
	}

	// Count total included/excluded
	for _, result := range filterResults {
		if result.Included {
			includedCount++
		} else {
			excludedCount++
		}
	}

	c.JSON(http.StatusOK, PreviewFilterResponse{
		TotalVersions:    len(versions),
		IncludedVersions: includedCount,
		ExcludedVersions: excludedCount,
		Versions:         previews,
		AppliedFilter:    filterStr,
		FilterSource:     filterSource,
	})
}

// HandleListForgeTypes returns all available forge types
func (h *Handler) HandleListForgeTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"forge_types": forge.GetForgeTypeInfo(),
	})
}

// HandleCommonFilters returns common filter presets
func (h *Handler) HandleCommonFilters(c *gin.Context) {
	c.JSON(http.StatusOK, CommonFiltersResponse{
		Filters: forge.CommonFilters,
	})
}
