package sources

import (
	"context"
	"net/http"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

var log = logs.NewDefault()

// SetLogger sets the logger for the sources package
func SetLogger(l *logs.Logger) {
	if l != nil {
		log = l
	}
}

// NewHandler creates a new sources handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		sourceRepo:        cfg.SourceRepo,
		sourceVersionRepo: cfg.SourceVersionRepo,
		versionDiscovery:  cfg.VersionDiscovery,
	}
}

// triggerAutoSync starts a background version sync for a newly created source
func (h *Handler) triggerAutoSync(source *db.UpstreamSource, sourceType string) {
	// Skip auto-sync if version discovery is not configured
	if h.versionDiscovery == nil {
		return
	}

	job := &db.VersionSyncJob{
		SourceID:   source.ID,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := h.sourceVersionRepo.CreateSyncJob(job); err != nil {
		log.Error("Failed to create auto-sync job", "source_id", source.ID, "error", err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		log.Info("Starting automatic version sync for new source", "source_id", source.ID, "source_name", source.Name)
		if err := h.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Automatic version sync failed", "source_id", source.ID, "error", err)
		} else {
			log.Info("Automatic version sync completed", "source_id", source.ID, "source_name", source.Name)
		}
	}()
}

// HandleList returns merged sources (defaults + user) for the authenticated user
// @Summary      List sources
// @Description  Returns merged sources (system + user) for the authenticated user
// @Tags         Sources
// @Produce      json
// @Success      200  {object}  SourceListResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources [get]
func (h *Handler) HandleList(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	sources, err := h.sourceRepo.GetMergedSources(claims.UserID)
	if err != nil {
		common.InternalError(c, err.Error())
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

// HandleListDefaults returns all default sources (root only)
func (h *Handler) HandleListDefaults(c *gin.Context) {
	sources, err := h.sourceRepo.ListDefaults()
	if err != nil {
		common.InternalError(c, err.Error())
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

// HandleCreateDefault creates a new default source (root only)
func (h *Handler) HandleCreateDefault(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
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

	forgeType := req.ForgeType
	if forgeType == "" {
		forgeType = "generic"
	}

	source := &db.UpstreamSource{
		Name:            req.Name,
		URL:             req.URL,
		ComponentIDs:    componentIDs,
		RetrievalMethod: retrievalMethod,
		URLTemplate:     req.URLTemplate,
		ForgeType:       forgeType,
		VersionFilter:   req.VersionFilter,
		Priority:        req.Priority,
		Enabled:         enabled,
	}

	if err := h.sourceRepo.CreateDefault(source); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	h.triggerAutoSync(source, "default")

	c.JSON(http.StatusCreated, source)
}

// HandleUpdateDefault updates an existing default source (root only)
func (h *Handler) HandleUpdateDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Default source not found")
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
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
	if req.ForgeType != nil {
		source.ForgeType = *req.ForgeType
	}
	if req.VersionFilter != nil {
		source.VersionFilter = *req.VersionFilter
	}
	if req.Priority != nil {
		source.Priority = *req.Priority
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}
	if req.DefaultVersion != nil {
		source.DefaultVersion = *req.DefaultVersion
	}

	if err := h.sourceRepo.UpdateDefault(source); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleDeleteDefault deletes a default source (root only)
func (h *Handler) HandleDeleteDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	if err := h.sourceRepo.DeleteDefault(id); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleCreateUserSource creates a new user source (or system source if admin)
// @Summary      Create a source
// @Description  Creates a new user source, or a system source if admin and is_system=true
// @Tags         Sources
// @Accept       json
// @Produce      json
// @Param        request  body      CreateSourceRequest  true  "Source creation request"
// @Success      201      {object}  db.UpstreamSource
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      403      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources [post]
func (h *Handler) HandleCreateUserSource(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	// Check if this should be a system source (admin only)
	isSystem := req.IsSystem != nil && *req.IsSystem
	if isSystem && !claims.HasAdminAccess() {
		common.Forbidden(c, "Only administrators can create system sources")
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

	forgeType := req.ForgeType
	if forgeType == "" {
		forgeType = "generic"
	}

	source := &db.UpstreamSource{
		Name:            req.Name,
		URL:             req.URL,
		ComponentIDs:    componentIDs,
		RetrievalMethod: retrievalMethod,
		URLTemplate:     req.URLTemplate,
		ForgeType:       forgeType,
		VersionFilter:   req.VersionFilter,
		Priority:        req.Priority,
		Enabled:         enabled,
	}

	// Create as system source (no owner) or user source
	if isSystem {
		if err := h.sourceRepo.CreateDefault(source); err != nil {
			common.InternalError(c, err.Error())
			return
		}
		h.triggerAutoSync(source, "default")
	} else {
		source.OwnerID = claims.UserID
		if err := h.sourceRepo.CreateUserSource(source); err != nil {
			common.InternalError(c, err.Error())
			return
		}
		h.triggerAutoSync(source, "user")
	}

	c.JSON(http.StatusCreated, source)
}

// HandleUpdateUserSource updates an existing user source (owner only)
func (h *Handler) HandleUpdateUserSource(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only update your own sources")
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
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
	if req.ForgeType != nil {
		source.ForgeType = *req.ForgeType
	}
	if req.VersionFilter != nil {
		source.VersionFilter = *req.VersionFilter
	}
	if req.Priority != nil {
		source.Priority = *req.Priority
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}
	if req.DefaultVersion != nil {
		source.DefaultVersion = *req.DefaultVersion
	}

	if err := h.sourceRepo.UpdateUserSource(source); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleDeleteUserSource deletes a user source (owner only)
func (h *Handler) HandleDeleteUserSource(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only delete your own sources")
		return
	}

	if err := h.sourceRepo.DeleteUserSource(id); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleListByComponent returns merged sources for a specific component
// @Summary      List sources by component
// @Description  Returns merged sources for a specific component
// @Tags         Sources
// @Produce      json
// @Param        componentId  path      string  true  "Component ID"
// @Success      200          {object}  SourceListResponse
// @Failure      400          {object}  common.ErrorResponse
// @Failure      401          {object}  common.ErrorResponse
// @Failure      500          {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/component/{componentId} [get]
func (h *Handler) HandleListByComponent(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	componentID := c.Param("componentId")
	if componentID == "" {
		common.BadRequest(c, "Component ID required")
		return
	}

	sources, err := h.sourceRepo.GetMergedSourcesByComponent(claims.UserID, componentID)
	if err != nil {
		common.InternalError(c, err.Error())
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

// HandleGetDefaultByID returns a single default source by ID
func (h *Handler) HandleGetDefaultByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Default source not found")
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleGetUserSourceByID returns a single user source by ID
func (h *Handler) HandleGetUserSourceByID(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "User source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only view your own sources")
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleListDefaultVersions lists cached versions for a default source
func (h *Handler) HandleListDefaultVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Default source not found")
		return
	}

	limit, offset := common.GetPaginationParams(c, common.MaxPaginationLimit)
	versionType := c.Query("version_type")

	versions, total, err := h.sourceVersionRepo.ListBySourcePaginated(id, "default", limit, offset, versionType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if versions == nil {
		versions = []db.SourceVersion{}
	}

	syncJob, _ := h.sourceVersionRepo.GetLatestSyncJob(id, "default")

	c.JSON(http.StatusOK, SourceVersionListResponse{
		Count:    len(versions),
		Total:    total,
		Versions: versions,
		SyncJob:  syncJob,
	})
}

// HandleListUserVersions lists cached versions for a user source
func (h *Handler) HandleListUserVersions(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "User source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only view your own source versions")
		return
	}

	limit, offset := common.GetPaginationParams(c, common.MaxPaginationLimit)
	versionType := c.Query("version_type")

	versions, total, err := h.sourceVersionRepo.ListBySourcePaginated(id, "user", limit, offset, versionType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if versions == nil {
		versions = []db.SourceVersion{}
	}

	syncJob, _ := h.sourceVersionRepo.GetLatestSyncJob(id, "user")

	c.JSON(http.StatusOK, SourceVersionListResponse{
		Count:    len(versions),
		Total:    total,
		Versions: versions,
		SyncJob:  syncJob,
	})
}

// HandleSyncDefaultVersions triggers a version sync for a default source (root only)
func (h *Handler) HandleSyncDefaultVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Default source not found")
		return
	}

	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, "default")
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if runningJob != nil {
		common.Conflict(c, "A sync job is already running for this source")
		return
	}

	sourceType := db.GetSourceType(source)

	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := h.sourceVersionRepo.CreateSyncJob(job); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := h.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Version sync failed", "source_id", id, "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, SyncTriggerResponse{
		JobID:   job.ID,
		Message: "Version sync started",
	})
}

// HandleSyncUserVersions triggers a version sync for a user source
func (h *Handler) HandleSyncUserVersions(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "User source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only sync your own sources")
		return
	}

	sourceType := db.GetSourceType(source)

	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if runningJob != nil {
		common.Conflict(c, "A sync job is already running for this source")
		return
	}

	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := h.sourceVersionRepo.CreateSyncJob(job); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := h.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Version sync failed", "source_id", id, "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, SyncTriggerResponse{
		JobID:   job.ID,
		Message: "Version sync started",
	})
}

// HandleGetDefaultSyncStatus returns sync status for a default source
func (h *Handler) HandleGetDefaultSyncStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	job, err := h.sourceVersionRepo.GetLatestSyncJob(id, "default")
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, SyncStatusResponse{
		Job: job,
	})
}

// HandleGetUserSyncStatus returns sync status for a user source
func (h *Handler) HandleGetUserSyncStatus(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "User source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only view your own source sync status")
		return
	}

	job, err := h.sourceVersionRepo.GetLatestSyncJob(id, "user")
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, SyncStatusResponse{
		Job: job,
	})
}

// HandleClearDefaultVersions clears all cached versions for a default source (root only)
func (h *Handler) HandleClearDefaultVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Default source not found")
		return
	}

	// Check for running sync job
	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, "default")
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if runningJob != nil {
		common.Conflict(c, "Cannot clear versions while sync is in progress")
		return
	}

	// Delete all versions for this source
	if err := h.sourceVersionRepo.DeleteBySource(id, "default"); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	log.Info("Cleared version cache for default source", "source_id", id, "source_name", source.Name)

	c.JSON(http.StatusOK, ClearVersionsResponse{
		Message: "Version cache cleared successfully",
	})
}

// HandleClearUserVersions clears all cached versions for a user source
func (h *Handler) HandleClearUserVersions(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "User source not found")
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only clear your own source versions")
		return
	}

	sourceType := db.GetSourceType(source)

	// Check for running sync job
	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if runningJob != nil {
		common.Conflict(c, "Cannot clear versions while sync is in progress")
		return
	}

	// Delete all versions for this source
	if err := h.sourceVersionRepo.DeleteBySource(id, sourceType); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	log.Info("Cleared version cache for user source", "source_id", id, "source_name", source.Name, "user_id", claims.UserID)

	c.JSON(http.StatusOK, ClearVersionsResponse{
		Message: "Version cache cleared successfully",
	})
}

// =============================================================================
// Unified Handlers (replace dual default/user handlers)
// =============================================================================

// checkSourceAccess verifies user has access to the source
// Returns the source if access is granted, or nil with appropriate error response
func (h *Handler) checkSourceAccess(c *gin.Context, claims *auth.TokenClaims, source *db.UpstreamSource, requireWrite bool) bool {
	// System sources require admin access for write operations
	if source.IsSystem {
		if requireWrite && !claims.HasAdminAccess() {
			common.Forbidden(c, "Only administrators can modify system sources")
			return false
		}
		return true
	}

	// User sources: owner or admin can access
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only access your own sources")
		return false
	}

	return true
}

// HandleGetByID returns a single source by ID (unified)
// @Summary      Get a source
// @Description  Returns a single source by ID
// @Tags         Sources
// @Produce      json
// @Param        id   path      string  true  "Source ID"
// @Success      200  {object}  db.UpstreamSource
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id} [get]
func (h *Handler) HandleGetByID(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, false) {
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleUpdate updates an existing source (unified)
// @Summary      Update a source
// @Description  Updates an existing source
// @Tags         Sources
// @Accept       json
// @Produce      json
// @Param        id       path      string               true  "Source ID"
// @Param        request  body      UpdateSourceRequest   true  "Source update request"
// @Success      200      {object}  db.UpstreamSource
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      403      {object}  common.ErrorResponse
// @Failure      404      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id} [put]
func (h *Handler) HandleUpdate(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, true) {
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
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
	if req.ForgeType != nil {
		source.ForgeType = *req.ForgeType
	}
	if req.VersionFilter != nil {
		source.VersionFilter = *req.VersionFilter
	}
	if req.Priority != nil {
		source.Priority = *req.Priority
	}
	if req.Enabled != nil {
		source.Enabled = *req.Enabled
	}
	if req.DefaultVersion != nil {
		source.DefaultVersion = *req.DefaultVersion
	}

	if err := h.sourceRepo.Update(source); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleDelete deletes a source (unified)
// @Summary      Delete a source
// @Description  Deletes a source
// @Tags         Sources
// @Param        id   path      string  true  "Source ID"
// @Success      204  "No Content"
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id} [delete]
func (h *Handler) HandleDelete(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, true) {
		return
	}

	if err := h.sourceRepo.Delete(id); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleListVersions lists cached versions for a source (unified)
// @Summary      List source versions
// @Description  Lists cached versions for a source
// @Tags         Sources
// @Produce      json
// @Param        id            path      string  true   "Source ID"
// @Param        limit         query     int     false  "Maximum results"
// @Param        offset        query     int     false  "Offset for pagination"
// @Param        version_type  query     string  false  "Filter by version type"
// @Success      200           {object}  SourceVersionListResponse
// @Failure      400           {object}  common.ErrorResponse
// @Failure      401           {object}  common.ErrorResponse
// @Failure      403           {object}  common.ErrorResponse
// @Failure      404           {object}  common.ErrorResponse
// @Failure      500           {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id}/versions [get]
func (h *Handler) HandleListVersions(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, false) {
		return
	}

	sourceType := db.GetSourceType(source)
	limit, offset := common.GetPaginationParams(c, common.MaxPaginationLimit)
	versionType := c.Query("version_type")

	versions, total, err := h.sourceVersionRepo.ListBySourcePaginated(id, sourceType, limit, offset, versionType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if versions == nil {
		versions = []db.SourceVersion{}
	}

	syncJob, _ := h.sourceVersionRepo.GetLatestSyncJob(id, sourceType)

	c.JSON(http.StatusOK, SourceVersionListResponse{
		Count:    len(versions),
		Total:    total,
		Versions: versions,
		SyncJob:  syncJob,
	})
}

// HandleSync triggers a version sync for a source (unified)
// @Summary      Trigger version sync
// @Description  Triggers a version sync for a source
// @Tags         Sources
// @Produce      json
// @Param        id   path      string  true  "Source ID"
// @Success      202  {object}  SyncTriggerResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      409  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id}/sync [post]
func (h *Handler) HandleSync(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, true) {
		return
	}

	sourceType := db.GetSourceType(source)

	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if runningJob != nil {
		common.Conflict(c, "A sync job is already running for this source")
		return
	}

	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := h.sourceVersionRepo.CreateSyncJob(job); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := h.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Version sync failed", "source_id", id, "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, SyncTriggerResponse{
		JobID:   job.ID,
		Message: "Version sync started",
	})
}

// HandleGetSyncStatus returns sync status for a source (unified)
// @Summary      Get sync status
// @Description  Returns sync status for a source
// @Tags         Sources
// @Produce      json
// @Param        id   path      string  true  "Source ID"
// @Success      200  {object}  SyncStatusResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id}/sync/status [get]
func (h *Handler) HandleGetSyncStatus(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, false) {
		return
	}

	sourceType := db.GetSourceType(source)

	job, err := h.sourceVersionRepo.GetLatestSyncJob(id, sourceType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, SyncStatusResponse{
		Job: job,
	})
}

// HandleGetVersionTypes returns the distinct version types for a source (unified)
// @Summary      Get version types
// @Description  Returns the distinct version types for a source
// @Tags         Sources
// @Produce      json
// @Param        id   path      string  true  "Source ID"
// @Success      200  {object}  VersionTypesResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id}/versions/types [get]
func (h *Handler) HandleGetVersionTypes(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, false) {
		return
	}

	sourceType := db.GetSourceType(source)

	types, err := h.sourceVersionRepo.GetDistinctVersionTypes(id, sourceType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if types == nil {
		types = []string{}
	}

	c.JSON(http.StatusOK, VersionTypesResponse{
		Types: types,
	})
}

// HandleClearVersions clears all cached versions for a source (unified)
// @Summary      Clear version cache
// @Description  Clears all cached versions for a source
// @Tags         Sources
// @Produce      json
// @Param        id   path      string  true  "Source ID"
// @Success      200  {object}  ClearVersionsResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      409  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/sources/{id}/versions [delete]
func (h *Handler) HandleClearVersions(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Source ID required")
		return
	}

	source, err := h.sourceRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if source == nil {
		common.NotFound(c, "Source not found")
		return
	}

	if !h.checkSourceAccess(c, claims, source, true) {
		return
	}

	sourceType := db.GetSourceType(source)

	// Check for running sync job
	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if runningJob != nil {
		common.Conflict(c, "Cannot clear versions while sync is in progress")
		return
	}

	// Delete all versions for this source
	if err := h.sourceVersionRepo.DeleteBySource(id, sourceType); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	log.Info("Cleared version cache for source", "source_id", id, "source_name", source.Name, "is_system", source.IsSystem)

	c.JSON(http.StatusOK, ClearVersionsResponse{
		Message: "Version cache cleared successfully",
	})
}
