package api

import (
	"context"
	"net/http"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// CreateSourceRequest represents the request to create a source
type CreateSourceRequest struct {
	Name            string   `json:"name" binding:"required" example:"Ubuntu Releases"`
	URL             string   `json:"url" binding:"required,url" example:"https://releases.ubuntu.com"`
	ComponentIDs    []string `json:"component_ids" example:"[\"uuid-of-kernel-component\"]"`
	RetrievalMethod string   `json:"retrieval_method" example:"release"`
	URLTemplate     string   `json:"url_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	Priority        int      `json:"priority" example:"10"`
	Enabled         *bool    `json:"enabled" example:"true"`
}

// UpdateSourceRequest represents the request to update a source
type UpdateSourceRequest struct {
	Name            string   `json:"name" example:"Ubuntu Releases"`
	URL             string   `json:"url" example:"https://releases.ubuntu.com"`
	ComponentIDs    []string `json:"component_ids" example:"[\"uuid-of-kernel-component\"]"`
	RetrievalMethod string   `json:"retrieval_method" example:"release"`
	URLTemplate     string   `json:"url_template" example:"{base_url}/archive/refs/tags/v{version}.tar.gz"`
	Priority        *int     `json:"priority" example:"10"`
	Enabled         *bool    `json:"enabled" example:"true"`
}

// SourceListResponse represents a list of sources
type SourceListResponse struct {
	Count   int                 `json:"count" example:"5"`
	Sources []db.UpstreamSource `json:"sources"`
}

// triggerAutoSync starts a background version sync for a newly created source
func (a *API) triggerAutoSync(source *db.UpstreamSource, sourceType string) {
	// Create a new sync job
	job := &db.VersionSyncJob{
		SourceID:   source.ID,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := a.sourceVersionRepo.CreateSyncJob(job); err != nil {
		log.Error("Failed to create auto-sync job", "source_id", source.ID, "error", err)
		return
	}

	// Start sync in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		log.Info("Starting automatic version sync for new source", "source_id", source.ID, "source_name", source.Name)
		if err := a.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Automatic version sync failed", "source_id", source.ID, "error", err)
		} else {
			log.Info("Automatic version sync completed", "source_id", source.ID, "source_name", source.Name)
		}
	}()
}

// DefaultSourceListResponse represents a list of default/system sources
type DefaultSourceListResponse struct {
	Count   int                 `json:"count" example:"3"`
	Sources []db.UpstreamSource `json:"sources"`
}

// handleListSources returns merged sources (defaults + user) for the authenticated user
func (a *API) handleListSources(c *gin.Context) {
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	sources, err := a.sourceRepo.GetMergedSources(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if sources == nil {
		sources = []db.UpstreamSource{}
	}

	c.JSON(http.StatusOK, SourceListResponse{
		Count:   len(sources),
		Sources: sources,
	})
}

// handleListDefaultSources returns all default sources (root only)
func (a *API) handleListDefaultSources(c *gin.Context) {
	sources, err := a.sourceRepo.ListDefaults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if sources == nil {
		sources = []db.UpstreamSource{}
	}

	c.JSON(http.StatusOK, DefaultSourceListResponse{
		Count:   len(sources),
		Sources: sources,
	})
}

// handleCreateDefaultSource creates a new default source (root only)
func (a *API) handleCreateDefaultSource(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	retrievalMethod := "release"
	if req.RetrievalMethod != "" {
		retrievalMethod = req.RetrievalMethod
	}

	componentIDs := req.ComponentIDs
	if componentIDs == nil {
		componentIDs = []string{}
	}

	source := &db.SourceDefault{
		Name:            req.Name,
		URL:             req.URL,
		ComponentIDs:    componentIDs,
		RetrievalMethod: retrievalMethod,
		URLTemplate:     req.URLTemplate,
		Priority:        req.Priority,
		Enabled:         enabled,
	}

	if err := a.sourceRepo.CreateDefault(source); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Trigger automatic version sync in background
	a.triggerAutoSync(source, "default")

	c.JSON(http.StatusCreated, source)
}

// handleUpdateDefaultSource updates an existing default source (root only)
func (a *API) handleUpdateDefaultSource(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := a.sourceRepo.GetDefaultByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Default source not found",
		})
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if req.Name != "" {
		source.Name = req.Name
	}
	if req.URL != "" {
		source.URL = req.URL
	}
	if req.ComponentIDs != nil {
		source.ComponentIDs = req.ComponentIDs
	}
	if req.RetrievalMethod != "" {
		source.RetrievalMethod = req.RetrievalMethod
	}
	if req.URLTemplate != "" {
		source.URLTemplate = req.URLTemplate
	}
	if req.Priority != nil {
		source.Priority = *req.Priority
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}

	if err := a.sourceRepo.UpdateDefault(source); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// handleDeleteDefaultSource deletes a default source (root only)
func (a *API) handleDeleteDefaultSource(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	if err := a.sourceRepo.DeleteDefault(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleCreateUserSource creates a new user source
func (a *API) handleCreateUserSource(c *gin.Context) {
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	retrievalMethod := "release"
	if req.RetrievalMethod != "" {
		retrievalMethod = req.RetrievalMethod
	}

	componentIDs := req.ComponentIDs
	if componentIDs == nil {
		componentIDs = []string{}
	}

	source := &db.UserSource{
		OwnerID:         claims.UserID,
		Name:            req.Name,
		URL:             req.URL,
		ComponentIDs:    componentIDs,
		RetrievalMethod: retrievalMethod,
		URLTemplate:     req.URLTemplate,
		Priority:        req.Priority,
		Enabled:         enabled,
	}

	if err := a.sourceRepo.CreateUserSource(source); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Trigger automatic version sync in background
	a.triggerAutoSync(source, "user")

	c.JSON(http.StatusCreated, source)
}

// handleUpdateUserSource updates an existing user source (owner only)
func (a *API) handleUpdateUserSource(c *gin.Context) {
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := a.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Source not found",
		})
		return
	}

	// Check ownership (admins can also update any user source)
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only update your own sources",
		})
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	if req.Name != "" {
		source.Name = req.Name
	}
	if req.URL != "" {
		source.URL = req.URL
	}
	if req.ComponentIDs != nil {
		source.ComponentIDs = req.ComponentIDs
	}
	if req.RetrievalMethod != "" {
		source.RetrievalMethod = req.RetrievalMethod
	}
	if req.URLTemplate != "" {
		source.URLTemplate = req.URLTemplate
	}
	if req.Priority != nil {
		source.Priority = *req.Priority
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}

	if err := a.sourceRepo.UpdateUserSource(source); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// handleDeleteUserSource deletes a user source (owner only)
func (a *API) handleDeleteUserSource(c *gin.Context) {
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := a.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Source not found",
		})
		return
	}

	// Check ownership (admins can also delete any user source)
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only delete your own sources",
		})
		return
	}

	if err := a.sourceRepo.DeleteUserSource(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleListSourcesByComponent returns merged sources for a specific component
func (a *API) handleListSourcesByComponent(c *gin.Context) {
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	componentID := c.Param("componentId")
	if componentID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	sources, err := a.sourceRepo.GetMergedSourcesByComponent(claims.UserID, componentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if sources == nil {
		sources = []db.UpstreamSource{}
	}

	c.JSON(http.StatusOK, SourceListResponse{
		Count:   len(sources),
		Sources: sources,
	})
}
