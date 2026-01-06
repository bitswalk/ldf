package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// SourceVersionListResponse represents a list of source versions
type SourceVersionListResponse struct {
	Count    int                `json:"count"`
	Total    int                `json:"total,omitempty"`
	Versions []db.SourceVersion `json:"versions"`
	SyncJob  *db.VersionSyncJob `json:"sync_job,omitempty"`
}

// SyncTriggerResponse represents the response when triggering a sync
type SyncTriggerResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// SyncStatusResponse represents the status of a sync job
type SyncStatusResponse struct {
	Job *db.VersionSyncJob `json:"job"`
}

// handleListDefaultSourceVersions lists cached versions for a default source
func (a *API) handleListDefaultSourceVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	// Verify source exists
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

	// Get pagination params
	limit, offset := getPaginationParams(c)
	versionType := c.Query("version_type") // "mainline", "stable", "longterm", "linux-next", or empty for all

	// Get versions
	versions, total, err := a.sourceVersionRepo.ListBySourcePaginated(id, "default", limit, offset, versionType)
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

	// Get latest sync job
	syncJob, _ := a.sourceVersionRepo.GetLatestSyncJob(id, "default")

	c.JSON(http.StatusOK, SourceVersionListResponse{
		Count:    len(versions),
		Total:    total,
		Versions: versions,
		SyncJob:  syncJob,
	})
}

// handleListUserSourceVersions lists cached versions for a user source
func (a *API) handleListUserSourceVersions(c *gin.Context) {
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

	// Verify source exists and user has access
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
			Message: "User source not found",
		})
		return
	}

	// Check ownership
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only view your own source versions",
		})
		return
	}

	// Get pagination params
	limit, offset := getPaginationParams(c)
	versionType := c.Query("version_type") // "mainline", "stable", "longterm", "linux-next", or empty for all

	// Get versions
	versions, total, err := a.sourceVersionRepo.ListBySourcePaginated(id, "user", limit, offset, versionType)
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

	// Get latest sync job
	syncJob, _ := a.sourceVersionRepo.GetLatestSyncJob(id, "user")

	c.JSON(http.StatusOK, SourceVersionListResponse{
		Count:    len(versions),
		Total:    total,
		Versions: versions,
		SyncJob:  syncJob,
	})
}

// handleSyncDefaultSourceVersions triggers a version sync for a default source (root only)
func (a *API) handleSyncDefaultSourceVersions(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	// Verify source exists
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

	// Check if a sync is already running
	runningJob, err := a.sourceVersionRepo.GetRunningSyncJob(id, "default")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if runningJob != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "Conflict",
			Code:    http.StatusConflict,
			Message: "A sync job is already running for this source",
		})
		return
	}

	// Get source type from the source itself
	sourceType := db.GetSourceType(source)

	// Create a new sync job
	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := a.sourceVersionRepo.CreateSyncJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Start sync in background (source is already the right type)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := a.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Version sync failed", "source_id", id, "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, SyncTriggerResponse{
		JobID:   job.ID,
		Message: "Version sync started",
	})
}

// handleSyncUserSourceVersions triggers a version sync for a user source
func (a *API) handleSyncUserSourceVersions(c *gin.Context) {
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

	// Verify source exists
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
			Message: "User source not found",
		})
		return
	}

	// Check ownership
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only sync your own sources",
		})
		return
	}

	// Get source type from the source itself
	sourceType := db.GetSourceType(source)

	// Check if a sync is already running
	runningJob, err := a.sourceVersionRepo.GetRunningSyncJob(id, sourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if runningJob != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "Conflict",
			Code:    http.StatusConflict,
			Message: "A sync job is already running for this source",
		})
		return
	}

	// Create a new sync job
	job := &db.VersionSyncJob{
		SourceID:   id,
		SourceType: sourceType,
		Status:     db.SyncStatusPending,
	}

	if err := a.sourceVersionRepo.CreateSyncJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Start sync in background (source is already the right type)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := a.versionDiscovery.SyncVersions(ctx, source, sourceType, job); err != nil {
			log.Error("Version sync failed", "source_id", id, "error", err)
		}
	}()

	c.JSON(http.StatusAccepted, SyncTriggerResponse{
		JobID:   job.ID,
		Message: "Version sync started",
	})
}

// handleGetDefaultSourceSyncStatus returns sync status for a default source
func (a *API) handleGetDefaultSourceSyncStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Source ID required",
		})
		return
	}

	// Get latest sync job
	job, err := a.sourceVersionRepo.GetLatestSyncJob(id, "default")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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

// handleGetUserSourceSyncStatus returns sync status for a user source
func (a *API) handleGetUserSourceSyncStatus(c *gin.Context) {
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

	// Verify source exists and user has access
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
			Message: "User source not found",
		})
		return
	}

	// Check ownership
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only view your own source sync status",
		})
		return
	}

	// Get latest sync job
	job, err := a.sourceVersionRepo.GetLatestSyncJob(id, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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

// handleGetDefaultSourceByID returns a single default source by ID
func (a *API) handleGetDefaultSourceByID(c *gin.Context) {
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

	c.JSON(http.StatusOK, source)
}

// handleGetUserSourceByID returns a single user source by ID
func (a *API) handleGetUserSourceByID(c *gin.Context) {
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
			Message: "User source not found",
		})
		return
	}

	// Check ownership
	if source.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only view your own sources",
		})
		return
	}

	c.JSON(http.StatusOK, source)
}

// getPaginationParams extracts limit and offset from query parameters
func getPaginationParams(c *gin.Context) (int, int) {
	limit := 50 // default
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}
