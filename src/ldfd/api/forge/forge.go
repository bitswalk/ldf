package forge

import (
	"context"
	"net/http"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/forge"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new forge handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		registry: cfg.Registry,
	}
}

// HandleDetect detects the forge type for a given URL and returns defaults
//
// @Summary      Detect forge type
// @Description  Detects the forge type for a given URL and returns repository info, defaults, and available forge types
// @Tags         Forge
// @Accept       json
// @Produce      json
// @Param        body  body      DetectRequest   true  "URL to detect"
// @Success      200   {object}  DetectResponse
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forge/detect [post]
func (h *Handler) HandleDetect(c *gin.Context) {
	var req DetectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
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
//
// @Summary      Preview version filter
// @Description  Fetches versions from upstream and previews which ones match the given filter expression
// @Tags         Forge
// @Accept       json
// @Produce      json
// @Param        body  body      PreviewFilterRequest   true  "Filter preview request"
// @Success      200   {object}  PreviewFilterResponse
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      502   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forge/preview-filter [post]
func (h *Handler) HandlePreviewFilter(c *gin.Context) {
	var req PreviewFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
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
		common.BadGateway(c, err.Error())
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
//
// @Summary      List forge types
// @Description  Returns all available forge types with their metadata
// @Tags         Forge
// @Produce      json
// @Success      200   {object}  object
// @Failure      401   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forge/types [get]
func (h *Handler) HandleListForgeTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"forge_types": forge.GetForgeTypeInfo(),
	})
}

// HandleCommonFilters returns common filter presets
//
// @Summary      List common filters
// @Description  Returns common version filter presets that can be applied to sources
// @Tags         Forge
// @Produce      json
// @Success      200   {object}  CommonFiltersResponse
// @Failure      401   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/forge/common-filters [get]
func (h *Handler) HandleCommonFilters(c *gin.Context) {
	c.JSON(http.StatusOK, CommonFiltersResponse{
		Filters: forge.CommonFilters,
	})
}
