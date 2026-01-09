package components

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new components handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		componentRepo:     cfg.ComponentRepo,
		sourceVersionRepo: cfg.SourceVersionRepo,
	}
}

// HandleList returns all components
func (h *Handler) HandleList(c *gin.Context) {
	components, err := h.componentRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleGet returns a single component by ID
func (h *Handler) HandleGet(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	component, err := h.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if component == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	c.JSON(http.StatusOK, component)
}

// HandleListByCategory returns components in a specific category
func (h *Handler) HandleListByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Category required",
		})
		return
	}

	components, err := h.componentRepo.GetByCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleGetCategories returns all distinct component categories
func (h *Handler) HandleGetCategories(c *gin.Context) {
	categories, err := h.componentRepo.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleCreate creates a new component (root only)
func (h *Handler) HandleCreate(c *gin.Context) {
	var req CreateComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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

	versionRule := db.VersionRule(req.DefaultVersionRule)
	if req.DefaultVersionRule != "" {
		if !isValidVersionRule(versionRule) {
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
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

	if err := h.componentRepo.Create(component); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, component)
}

// HandleUpdate updates an existing component (root only)
func (h *Handler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	component, err := h.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if component == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	var req UpdateComponentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if req.Name != "" && req.Name != component.Name {
		existing, err := h.componentRepo.GetByName(req.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, common.ErrorResponse{
				Error:   "Internal server error",
				Code:    http.StatusInternalServerError,
				Message: err.Error(),
			})
			return
		}
		if existing != nil {
			c.JSON(http.StatusConflict, common.ErrorResponse{
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
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: "Invalid version rule. Must be one of: pinned, latest-stable, latest-lts",
			})
			return
		}
		component.DefaultVersionRule = versionRule
	}

	if err := h.componentRepo.Update(component); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, component)
}

// HandleDelete deletes a component (root only)
func (h *Handler) HandleDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	if err := h.componentRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleGetVersions returns paginated versions for a component
func (h *Handler) HandleGetVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	component, err := h.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if component == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	limit, offset := common.GetPaginationParams(c, 100)
	versionType := c.Query("version_type")

	versions, total, err := h.sourceVersionRepo.ListByComponentPaginated(id, limit, offset, versionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleResolveVersion resolves a version rule to an actual version
func (h *Handler) HandleResolveVersion(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	rule := c.Query("rule")
	if rule == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Version rule required (e.g., latest-stable, latest-lts)",
		})
		return
	}

	component, err := h.componentRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if component == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Component not found",
		})
		return
	}

	var version *db.SourceVersion

	switch db.VersionRule(rule) {
	case db.VersionRuleLatestStable:
		version, err = h.sourceVersionRepo.GetLatestStableByComponent(id)
	case db.VersionRuleLatestLTS:
		version, err = h.sourceVersionRepo.GetLatestLongtermByComponent(id)
	case db.VersionRulePinned:
		c.JSON(http.StatusOK, ResolvedVersionResponse{
			Rule:            rule,
			ResolvedVersion: component.DefaultVersion,
		})
		return
	default:
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid version rule. Must be one of: pinned, latest-stable, latest-lts",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if version == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
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
