package sources

import (
	"context"
	"net/http"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

var log *logs.Logger

// SetLogger sets the logger for the sources package
func SetLogger(l *logs.Logger) {
	log = l
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
func (h *Handler) HandleList(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	sources, err := h.sourceRepo.GetMergedSources(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleListDefaults returns all default sources (root only)
func (h *Handler) HandleListDefaults(c *gin.Context) {
	sources, err := h.sourceRepo.ListDefaults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleCreateDefault creates a new default source (root only)
func (h *Handler) HandleCreateDefault(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	h.triggerAutoSync(source, "default")

	c.JSON(http.StatusCreated, source)
}

// HandleUpdateDefault updates an existing default source (root only)
func (h *Handler) HandleUpdateDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Default source not found",
		})
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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

	if err := h.sourceRepo.UpdateDefault(source); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleDeleteDefault deletes a default source (root only)
func (h *Handler) HandleDeleteDefault(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	if err := h.sourceRepo.DeleteDefault(id); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleCreateUserSource creates a new user source
func (h *Handler) HandleCreateUserSource(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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

	forgeType := req.ForgeType
	if forgeType == "" {
		forgeType = "generic"
	}

	source := &db.UpstreamSource{
		OwnerID:         claims.UserID,
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

	if err := h.sourceRepo.CreateUserSource(source); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	h.triggerAutoSync(source, "user")

	c.JSON(http.StatusCreated, source)
}

// HandleUpdateUserSource updates an existing user source (owner only)
func (h *Handler) HandleUpdateUserSource(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only update your own sources",
		})
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
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

	if err := h.sourceRepo.UpdateUserSource(source); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleDeleteUserSource deletes a user source (owner only)
func (h *Handler) HandleDeleteUserSource(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only delete your own sources",
		})
		return
	}

	if err := h.sourceRepo.DeleteUserSource(id); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleListByComponent returns merged sources for a specific component
func (h *Handler) HandleListByComponent(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	componentID := c.Param("componentId")
	if componentID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Component ID required",
		})
		return
	}

	sources, err := h.sourceRepo.GetMergedSourcesByComponent(claims.UserID, componentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleGetDefaultByID returns a single default source by ID
func (h *Handler) HandleGetDefaultByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Default source not found",
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleGetUserSourceByID returns a single user source by ID
func (h *Handler) HandleGetUserSourceByID(c *gin.Context) {
	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "User source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only view your own sources",
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// HandleListDefaultVersions lists cached versions for a default source
func (h *Handler) HandleListDefaultVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Default source not found",
		})
		return
	}

	limit, offset := common.GetPaginationParams(c, 500)
	versionType := c.Query("version_type")

	versions, total, err := h.sourceVersionRepo.ListBySourcePaginated(id, "default", limit, offset, versionType)
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
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "User source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only view your own source versions",
		})
		return
	}

	limit, offset := common.GetPaginationParams(c, 500)
	versionType := c.Query("version_type")

	versions, total, err := h.sourceVersionRepo.ListBySourcePaginated(id, "user", limit, offset, versionType)
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
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Default source not found",
		})
		return
	}

	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, "default")
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if runningJob != nil {
		c.JSON(http.StatusConflict, common.ErrorResponse{
			Error:   "Conflict",
			Code:    http.StatusConflict,
			Message: "A sync job is already running for this source",
		})
		return
	}

	sourceType := db.GetSourceType(source)

	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := h.sourceVersionRepo.CreateSyncJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "User source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only sync your own sources",
		})
		return
	}

	sourceType := db.GetSourceType(source)

	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if runningJob != nil {
		c.JSON(http.StatusConflict, common.ErrorResponse{
			Error:   "Conflict",
			Code:    http.StatusConflict,
			Message: "A sync job is already running for this source",
		})
		return
	}

	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := h.sourceVersionRepo.CreateSyncJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	job, err := h.sourceVersionRepo.GetLatestSyncJob(id, "default")
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "User source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only view your own source sync status",
		})
		return
	}

	job, err := h.sourceVersionRepo.GetLatestSyncJob(id, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetDefaultByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Default source not found",
		})
		return
	}

	// Check for running sync job
	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, "default")
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if runningJob != nil {
		c.JSON(http.StatusConflict, common.ErrorResponse{
			Error:   "Conflict",
			Code:    http.StatusConflict,
			Message: "Cannot clear versions while sync is in progress",
		})
		return
	}

	// Delete all versions for this source
	if err := h.sourceVersionRepo.DeleteBySource(id, "default"); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	source, err := h.sourceRepo.GetUserSourceByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if source == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "User source not found",
		})
		return
	}

	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only clear your own source versions",
		})
		return
	}

	sourceType := db.GetSourceType(source)

	// Check for running sync job
	runningJob, err := h.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if runningJob != nil {
		c.JSON(http.StatusConflict, common.ErrorResponse{
			Error:   "Conflict",
			Code:    http.StatusConflict,
			Message: "Cannot clear versions while sync is in progress",
		})
		return
	}

	// Delete all versions for this source
	if err := h.sourceVersionRepo.DeleteBySource(id, sourceType); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	log.Info("Cleared version cache for user source", "source_id", id, "source_name", source.Name, "user_id", claims.UserID)

	c.JSON(http.StatusOK, ClearVersionsResponse{
		Message: "Version cache cleared successfully",
	})
}
