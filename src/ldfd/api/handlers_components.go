package api

import (
	"net/http"
	"strconv"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// ComponentListResponse represents a list of components
type ComponentListResponse struct {
	Count      int            `json:"count" example:"16"`
	Components []db.Component `json:"components"`
}

// CreateComponentRequest represents the request to create a component
type CreateComponentRequest struct {
	Name                     string `json:"name" binding:"required" example:"kernel"`
	Category                 string `json:"category" binding:"required" example:"core"`
	DisplayName              string `json:"display_name" binding:"required" example:"Linux Kernel"`
	Description              string `json:"description" example:"The Linux kernel source code"`
	ArtifactPattern          string `json:"artifact_pattern" example:"linux-{version}.tar.xz"`
	DefaultURLTemplate       string `json:"default_url_template" example:"{base_url}/linux-{version}.tar.xz"`
	GitHubNormalizedTemplate string `json:"github_normalized_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	IsOptional               *bool  `json:"is_optional" example:"false"`
	DefaultVersion           string `json:"default_version" example:"6.12.1"`
	DefaultVersionRule       string `json:"default_version_rule" example:"latest-stable"`
}

// UpdateComponentRequest represents the request to update a component
type UpdateComponentRequest struct {
	Name                     string `json:"name" example:"kernel"`
	Category                 string `json:"category" example:"core"`
	DisplayName              string `json:"display_name" example:"Linux Kernel"`
	Description              string `json:"description" example:"The Linux kernel source code"`
	ArtifactPattern          string `json:"artifact_pattern" example:"linux-{version}.tar.xz"`
	DefaultURLTemplate       string `json:"default_url_template" example:"{base_url}/linux-{version}.tar.xz"`
	GitHubNormalizedTemplate string `json:"github_normalized_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	IsOptional               *bool  `json:"is_optional" example:"false"`
	DefaultVersion           string `json:"default_version" example:"6.12.1"`
	DefaultVersionRule       string `json:"default_version_rule" example:"latest-stable"`
}

// ComponentVersionsResponse represents a paginated list of versions for a component
type ComponentVersionsResponse struct {
	Versions []db.SourceVersion `json:"versions"`
	Total    int                `json:"total"`
	Limit    int                `json:"limit"`
	Offset   int                `json:"offset"`
}

// ResolvedVersionResponse represents a resolved version from a rule
type ResolvedVersionResponse struct {
	Rule            string            `json:"rule"`
	ResolvedVersion string            `json:"resolved_version"`
	Version         *db.SourceVersion `json:"version,omitempty"`
}

// handleListComponents returns all components
func (a *API) handleListComponents(c *gin.Context) {
	components, err := a.componentRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if components == nil {
		components = []db.Component{}
	}

	c.JSON(http.StatusOK, ComponentListResponse{
		Count:      len(components),
		Components: components,
	})
}

// handleGetComponent returns a single component by ID
func (a *API) handleGetComponent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	component, err := a.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if component == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	c.JSON(http.StatusOK, component)
}

// handleListComponentsByCategory returns components in a specific category
func (a *API) handleListComponentsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Category required",
		})
		return
	}

	components, err := a.componentRepo.GetByCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if components == nil {
		components = []db.Component{}
	}

	c.JSON(http.StatusOK, ComponentListResponse{
		Count:      len(components),
		Components: components,
	})
}

// handleGetComponentCategories returns all distinct component categories
func (a *API) handleGetComponentCategories(c *gin.Context) {
	categories, err := a.componentRepo.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if categories == nil {
		categories = []string{}
	}

	c.JSON(http.StatusOK, gin.H{
		"count":      len(categories),
		"categories": categories,
	})
}

// handleCreateComponent creates a new component (root only)
func (a *API) handleCreateComponent(c *gin.Context) {
	var req CreateComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	isOptional := false
	if req.IsOptional != nil {
		isOptional = *req.IsOptional
	}

	// Validate version rule if provided
	versionRule := db.VersionRule(req.DefaultVersionRule)
	if req.DefaultVersionRule != "" {
		if !isValidVersionRule(versionRule) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: "Invalid version rule. Must be one of: pinned, latest-stable, latest-lts",
			})
			return
		}
	}

	component := &db.Component{
		Name:                     req.Name,
		Category:                 req.Category,
		DisplayName:              req.DisplayName,
		Description:              req.Description,
		ArtifactPattern:          req.ArtifactPattern,
		DefaultURLTemplate:       req.DefaultURLTemplate,
		GitHubNormalizedTemplate: req.GitHubNormalizedTemplate,
		IsOptional:               isOptional,
		DefaultVersion:           req.DefaultVersion,
		DefaultVersionRule:       versionRule,
	}

	if err := a.componentRepo.Create(component); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, component)
}

// handleUpdateComponent updates an existing component (root only)
func (a *API) handleUpdateComponent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	component, err := a.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if component == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	var req UpdateComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Check if new name conflicts with existing component (if name is being changed)
	if req.Name != "" && req.Name != component.Name {
		existing, err := a.componentRepo.GetByName(req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Internal server error",
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			})
			return
		}
		if existing != nil {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "Conflict",
				Code:    http.StatusConflict,
				Message: "A component with this name already exists",
			})
			return
		}
		component.Name = req.Name
	}
	if req.Category != "" {
		component.Category = req.Category
	}
	if req.DisplayName != "" {
		component.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		component.Description = req.Description
	}
	if req.ArtifactPattern != "" {
		component.ArtifactPattern = req.ArtifactPattern
	}
	if req.DefaultURLTemplate != "" {
		component.DefaultURLTemplate = req.DefaultURLTemplate
	}
	if req.GitHubNormalizedTemplate != "" {
		component.GitHubNormalizedTemplate = req.GitHubNormalizedTemplate
	}
	if req.IsOptional != nil {
		component.IsOptional = *req.IsOptional
	}
	if req.DefaultVersion != "" {
		component.DefaultVersion = req.DefaultVersion
	}
	if req.DefaultVersionRule != "" {
		versionRule := db.VersionRule(req.DefaultVersionRule)
		if !isValidVersionRule(versionRule) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: "Invalid version rule. Must be one of: pinned, latest-stable, latest-lts",
			})
			return
		}
		component.DefaultVersionRule = versionRule
	}

	if err := a.componentRepo.Update(component); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, component)
}

// handleDeleteComponent deletes a component (root only)
func (a *API) handleDeleteComponent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	if err := a.componentRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleGetComponentVersions returns paginated versions for a component
func (a *API) handleGetComponentVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	// Verify component exists
	component, err := a.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if component == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0
	versionType := c.Query("version_type") // "all", "stable", "longterm", "mainline"

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	versions, total, err := a.sourceVersionRepo.ListByComponentPaginated(id, limit, offset, versionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if versions == nil {
		versions = []db.SourceVersion{}
	}

	c.JSON(http.StatusOK, ComponentVersionsResponse{
		Versions: versions,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	})
}

// handleResolveComponentVersion resolves a version rule to an actual version
func (a *API) handleResolveComponentVersion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	rule := c.Query("rule")
	if rule == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Version rule required (e.g., latest-stable, latest-lts)",
		})
		return
	}

	// Verify component exists
	component, err := a.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if component == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	var version *db.SourceVersion

	switch db.VersionRule(rule) {
	case db.VersionRuleLatestStable:
		version, err = a.sourceVersionRepo.GetLatestStableByComponent(id)
	case db.VersionRuleLatestLTS:
		version, err = a.sourceVersionRepo.GetLatestLongtermByComponent(id)
	case db.VersionRulePinned:
		// For pinned, return the component's default version
		c.JSON(http.StatusOK, ResolvedVersionResponse{
			Rule:            rule,
			ResolvedVersion: component.DefaultVersion,
		})
		return
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid version rule. Must be one of: pinned, latest-stable, latest-lts",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if version == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "No version found matching the rule",
		})
		return
	}

	c.JSON(http.StatusOK, ResolvedVersionResponse{
		Rule:            rule,
		ResolvedVersion: version.Version,
		Version:         version,
	})
}

// isValidVersionRule validates a version rule string
func isValidVersionRule(rule db.VersionRule) bool {
	switch rule {
	case db.VersionRulePinned, db.VersionRuleLatestStable, db.VersionRuleLatestLTS:
		return true
	default:
		return false
	}
}
