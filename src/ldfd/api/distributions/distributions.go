package distributions

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

var log = logs.NewDefault()

// SetLogger sets the logger for the distributions package
func SetLogger(l *logs.Logger) {
	if l != nil {
		log = l
	}
}

// NewHandler creates a new distributions handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		distRepo:        cfg.DistRepo,
		downloadJobRepo: cfg.DownloadJobRepo,
		buildJobRepo:    cfg.BuildJobRepo,
		sourceRepo:      cfg.SourceRepo,
		jwtService:      cfg.JWTService,
		storageManager:  cfg.StorageManager,
		kernelConfigSvc: cfg.KernelConfigSvc,
	}
}

// HandleList returns a list of distributions accessible to the current user
// @Summary      List distributions
// @Description  Returns distributions accessible to the current user, optionally filtered by status
// @Tags         Distributions
// @Produce      json
// @Param        status  query     string  false  "Filter by status"
// @Success      200     {object}  DistributionListResponse
// @Failure      500     {object}  common.ErrorResponse
// @Router       /v1/distributions [get]
func (h *Handler) HandleList(c *gin.Context) {
	var statusFilter *db.DistributionStatus
	if statusParam := c.Query("status"); statusParam != "" {
		status := db.DistributionStatus(statusParam)
		statusFilter = &status
	}

	claims := common.GetTokenClaimsFromRequest(c, h.jwtService)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
		log.Debug("ListDistributions: authenticated user", "user_id", userID, "user_name", claims.UserName, "is_admin", isAdmin)
	} else {
		log.Debug("ListDistributions: anonymous user (no valid token)")
	}

	distributions, err := h.distRepo.ListAccessible(userID, isAdmin, statusFilter)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if distributions == nil {
		distributions = []db.Distribution{}
	}

	c.JSON(http.StatusOK, DistributionListResponse{
		Count:         len(distributions),
		Distributions: distributions,
	})
}

// HandleCreate creates a new distribution record
// @Summary      Create a distribution
// @Description  Creates a new distribution record
// @Tags         Distributions
// @Accept       json
// @Produce      json
// @Param        request  body      CreateDistributionRequest  true  "Distribution creation request"
// @Success      201      {object}  db.Distribution
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      409      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions [post]
func (h *Handler) HandleCreate(c *gin.Context) {
	var req CreateDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	existing, err := h.distRepo.GetByName(req.Name)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if existing != nil {
		common.Conflict(c, "Distribution with this name already exists")
		return
	}

	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	visibility := db.VisibilityPrivate
	if req.Visibility == "public" {
		visibility = db.VisibilityPublic
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}
	ownerID := claims.UserID

	dist := &db.Distribution{
		Name:       req.Name,
		Version:    version,
		Status:     db.StatusPending,
		Visibility: visibility,
		Config:     req.Config,
		SourceURL:  req.SourceURL,
		Checksum:   req.Checksum,
		OwnerID:    ownerID,
	}

	if err := h.distRepo.Create(dist); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if err := h.distRepo.AddLog(dist.ID, "info", "Distribution created"); err != nil {
		log.Warn("Failed to add distribution log", "dist_id", dist.ID, "error", err)
	}

	// Generate and store kernel config artifact
	if dist.Config != nil && h.kernelConfigSvc != nil {
		if err := h.kernelConfigSvc.GenerateAndStore(c.Request.Context(), dist); err != nil {
			log.Warn("Failed to generate kernel config artifact", "dist_id", dist.ID, "error", err)
		} else {
			if err := h.distRepo.AddLog(dist.ID, "info", "Kernel config artifact generated"); err != nil {
				log.Warn("Failed to add distribution log", "dist_id", dist.ID, "error", err)
			}
		}
	}

	common.AuditLog(c, common.AuditEvent{Action: "distribution.create", UserID: ownerID, Resource: "distribution:" + dist.ID, Success: true})

	c.JSON(http.StatusCreated, dist)
}

// HandleGet returns a distribution by ID if the user has access
// @Summary      Get a distribution
// @Description  Returns a distribution by ID if the user has access
// @Tags         Distributions
// @Produce      json
// @Param        id   path      string  true  "Distribution ID"
// @Success      200  {object}  db.Distribution
// @Failure      400  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/distributions/{id} [get]
func (h *Handler) HandleGet(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Distribution ID required")
		return
	}

	claims := common.GetTokenClaimsFromRequest(c, h.jwtService)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	canAccess, err := h.distRepo.CanUserAccess(id, userID, isAdmin)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if !canAccess {
		common.NotFound(c, "Distribution not found")
		return
	}

	dist, err := h.distRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if dist == nil {
		common.NotFound(c, "Distribution not found")
		return
	}

	c.JSON(http.StatusOK, dist)
}

// HandleUpdate updates an existing distribution (owner or admin only)
// @Summary      Update a distribution
// @Description  Updates an existing distribution (owner or admin only)
// @Tags         Distributions
// @Accept       json
// @Produce      json
// @Param        id       path      string                     true  "Distribution ID"
// @Param        request  body      UpdateDistributionRequest   true  "Distribution update request"
// @Success      200      {object}  db.Distribution
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      403      {object}  common.ErrorResponse
// @Failure      404      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id} [put]
func (h *Handler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Distribution ID required")
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	dist, err := h.distRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if dist == nil {
		common.NotFound(c, "Distribution not found")
		return
	}

	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only update your own distributions")
		return
	}

	var req UpdateDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.BadRequest(c, err.Error())
		return
	}

	if req.Name != "" {
		dist.Name = req.Name
	}
	if req.Version != "" {
		dist.Version = req.Version
	}
	if req.Status != "" {
		dist.Status = db.DistributionStatus(req.Status)
	}
	if req.Visibility != "" {
		if req.Visibility == "public" {
			dist.Visibility = db.VisibilityPublic
		} else {
			dist.Visibility = db.VisibilityPrivate
		}
	}
	if req.SourceURL != "" {
		dist.SourceURL = req.SourceURL
	}
	if req.Checksum != "" {
		dist.Checksum = req.Checksum
	}
	if req.SizeBytes > 0 {
		dist.SizeBytes = req.SizeBytes
	}
	if req.Config != nil {
		dist.Config = req.Config
	}

	if err := h.distRepo.Update(dist); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if err := h.distRepo.AddLog(dist.ID, "info", "Distribution updated"); err != nil {
		log.Warn("Failed to add distribution log", "dist_id", dist.ID, "error", err)
	}

	// Regenerate kernel config artifact if config was updated
	if req.Config != nil && dist.Config != nil && h.kernelConfigSvc != nil {
		if err := h.kernelConfigSvc.GenerateAndStore(c.Request.Context(), dist); err != nil {
			log.Warn("Failed to regenerate kernel config artifact", "dist_id", dist.ID, "error", err)
		} else {
			if err := h.distRepo.AddLog(dist.ID, "info", "Kernel config artifact regenerated"); err != nil {
				log.Warn("Failed to add distribution log", "dist_id", dist.ID, "error", err)
			}
		}
	}

	common.AuditLog(c, common.AuditEvent{Action: "distribution.update", UserID: claims.UserID, UserName: claims.UserName, Resource: "distribution:" + dist.ID, Success: true})

	c.JSON(http.StatusOK, dist)
}

// HandleDeletionPreview returns a preview of what will be deleted when a distribution is removed
// @Summary      Preview distribution deletion
// @Description  Returns a preview of what will be deleted when a distribution is removed
// @Tags         Distributions
// @Produce      json
// @Param        id   path      string  true  "Distribution ID"
// @Success      200  {object}  DeletionPreviewResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/deletion-preview [get]
func (h *Handler) HandleDeletionPreview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Distribution ID required")
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	dist, err := h.distRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if dist == nil {
		common.NotFound(c, "Distribution not found")
		return
	}

	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only delete your own distributions")
		return
	}

	preview := DeletionPreviewResponse{
		Distribution: *dist,
	}

	// Count download jobs
	if h.downloadJobRepo != nil {
		jobs, err := h.downloadJobRepo.ListByDistribution(id)
		if err != nil {
			log.Error("Failed to list download jobs for deletion preview", "error", err)
		} else {
			jobNames := make([]string, 0, len(jobs))
			for _, job := range jobs {
				name := job.ComponentName
				if name == "" {
					name = job.SourceName
				}
				if name != "" {
					jobNames = append(jobNames, name+" ("+job.Version+")")
				}
			}
			preview.DownloadJobs = DeletionPreviewCount{
				Count: len(jobs),
				Items: jobNames,
			}
		}
	}

	// Count artifacts from storage
	if h.storageManager != nil {
		artifacts, err := h.storageManager.ListByDistribution(id)
		if err != nil {
			log.Error("Failed to list artifacts for deletion preview", "error", err)
		} else {
			preview.Artifacts = DeletionPreviewCount{
				Count: len(artifacts),
				Items: artifacts,
			}
		}
	}

	// Count user sources (only sources owned by the distribution owner, not system sources)
	if h.sourceRepo != nil {
		userSources, err := h.sourceRepo.ListUserSources(dist.OwnerID)
		if err != nil {
			log.Error("Failed to list user sources for deletion preview", "error", err)
		} else {
			sourceSummaries := make([]DeletionSourceSummary, 0, len(userSources))
			for _, s := range userSources {
				sourceSummaries = append(sourceSummaries, DeletionSourceSummary{
					ID:   s.ID,
					Name: s.Name,
				})
			}
			preview.UserSources = DeletionPreviewSources{
				Count:   len(userSources),
				Sources: sourceSummaries,
			}
		}
	}

	c.JSON(http.StatusOK, preview)
}

// HandleDelete deletes a distribution by ID (owner or admin only)
// This performs a cascading delete of all related resources
// @Summary      Delete a distribution
// @Description  Deletes a distribution and all related resources (cascading delete)
// @Tags         Distributions
// @Produce      json
// @Param        id   path      string  true  "Distribution ID"
// @Success      204  "No Content"
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id} [delete]
func (h *Handler) HandleDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Distribution ID required")
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	dist, err := h.distRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if dist == nil {
		common.NotFound(c, "Distribution not found")
		return
	}

	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You can only delete your own distributions")
		return
	}

	// Cascade delete: artifacts from storage
	if h.storageManager != nil {
		deletedCount, deletedBytes, err := h.storageManager.DeleteByDistribution(id)
		if err != nil {
			log.Error("Failed to delete artifacts for distribution", "distribution_id", id, "error", err)
		} else if deletedCount > 0 {
			log.Info("Deleted artifacts for distribution", "distribution_id", id, "count", deletedCount, "bytes", deletedBytes)
		}
	}

	// Cascade delete: download jobs
	if h.downloadJobRepo != nil {
		if err := h.downloadJobRepo.DeleteByDistribution(id); err != nil {
			log.Error("Failed to delete download jobs for distribution", "distribution_id", id, "error", err)
		}
	}

	// Cascade delete: build jobs (and their stages/logs via FK CASCADE)
	if h.buildJobRepo != nil {
		if err := h.buildJobRepo.DeleteByDistribution(id); err != nil {
			log.Error("Failed to delete build jobs for distribution", "distribution_id", id, "error", err)
		}
	}

	// Cascade delete: user sources (only non-system sources owned by the distribution owner)
	if h.sourceRepo != nil {
		deletedSources, err := h.sourceRepo.DeleteUserSourcesByOwner(dist.OwnerID)
		if err != nil {
			log.Error("Failed to delete user sources for distribution owner", "owner_id", dist.OwnerID, "error", err)
		} else if deletedSources > 0 {
			log.Info("Deleted user sources for distribution owner", "owner_id", dist.OwnerID, "count", deletedSources)
		}
	}

	// Finally delete the distribution itself
	if err := h.distRepo.Delete(id); err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if err := h.distRepo.AddLog(id, "info", "Distribution deleted with cascading cleanup"); err != nil {
		log.Warn("Failed to add distribution log", "dist_id", id, "error", err)
	}

	common.AuditLog(c, common.AuditEvent{Action: "distribution.delete", UserID: claims.UserID, UserName: claims.UserName, Resource: "distribution:" + id, Success: true})

	c.Status(http.StatusNoContent)
}

// HandleGetLogs returns logs for a distribution if the user has access
// @Summary      Get distribution logs
// @Description  Returns logs for a distribution if the user has access
// @Tags         Distributions
// @Produce      json
// @Param        id     path      string  true   "Distribution ID"
// @Param        limit  query     int     false  "Maximum number of logs to return"
// @Success      200    {array}   db.DistributionLog
// @Failure      400    {object}  common.ErrorResponse
// @Failure      404    {object}  common.ErrorResponse
// @Failure      500    {object}  common.ErrorResponse
// @Router       /v1/distributions/{id}/logs [get]
func (h *Handler) HandleGetLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Distribution ID required")
		return
	}

	claims := common.GetTokenClaimsFromRequest(c, h.jwtService)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	canAccess, err := h.distRepo.CanUserAccess(id, userID, isAdmin)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if !canAccess {
		common.NotFound(c, "Distribution not found")
		return
	}

	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.distRepo.GetLogs(id, limit)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	if logs == nil {
		logs = []db.DistributionLog{}
	}

	c.JSON(http.StatusOK, logs)
}

// HandleGetStats returns statistics about distributions grouped by status
// @Summary      Get distribution statistics
// @Description  Returns statistics about distributions grouped by status
// @Tags         Distributions
// @Produce      json
// @Success      200  {object}  DistributionStatsResponse
// @Failure      500  {object}  common.ErrorResponse
// @Router       /v1/distributions/stats [get]
func (h *Handler) HandleGetStats(c *gin.Context) {
	stats, err := h.distRepo.GetStats()
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}

	var total int64
	for _, count := range stats {
		total += count
	}

	c.JSON(http.StatusOK, DistributionStatsResponse{
		Total: total,
		Stats: stats,
	})
}

// HandleUploadKernelConfig replaces the kernel .config artifact with a user-provided file
// @Summary      Upload kernel config
// @Description  Replaces the generated kernel .config with a user-provided configuration file
// @Tags         Distributions
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      string  true  "Distribution ID"
// @Param        file  formData  file    true  "Kernel .config file"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  common.ErrorResponse
// @Failure      401   {object}  common.ErrorResponse
// @Failure      403   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/kernel-config [post]
func (h *Handler) HandleUploadKernelConfig(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		common.BadRequest(c, "Distribution ID required")
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		common.Unauthorized(c, "Authentication required")
		return
	}

	dist, err := h.distRepo.GetByID(id)
	if err != nil {
		common.InternalError(c, err.Error())
		return
	}
	if dist == nil {
		common.NotFound(c, "Distribution not found")
		return
	}

	// Check ownership or admin access
	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		common.Forbidden(c, "You don't have permission to modify this distribution")
		return
	}

	// Read uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		common.BadRequest(c, "File upload required (field: file)")
		return
	}
	defer file.Close()

	configData, err := io.ReadAll(file)
	if err != nil {
		common.InternalError(c, "Failed to read uploaded file")
		return
	}

	// Basic validation: file should contain at least one CONFIG_ line
	if !strings.Contains(string(configData), "CONFIG_") {
		common.BadRequest(c, "File does not appear to be a valid kernel .config (no CONFIG_ entries found)")
		return
	}

	if h.kernelConfigSvc == nil {
		common.InternalError(c, "Kernel config service not available")
		return
	}

	// Store the custom config
	if err := h.kernelConfigSvc.StoreCustomConfig(c.Request.Context(), dist, configData); err != nil {
		common.InternalError(c, "Failed to store kernel config: "+err.Error())
		return
	}

	// Update distribution config mode to custom
	if dist.Config != nil {
		dist.Config.Core.Kernel.ConfigMode = db.KernelConfigModeCustom
		if err := h.distRepo.Update(dist); err != nil {
			log.Warn("Failed to update distribution config mode", "dist_id", dist.ID, "error", err)
		}
	}

	if err := h.distRepo.AddLog(dist.ID, "info", "Custom kernel config uploaded"); err != nil {
		log.Warn("Failed to add distribution log", "dist_id", dist.ID, "error", err)
	}

	common.AuditLog(c, common.AuditEvent{
		Action:   "distribution.kernel_config.upload",
		UserID:   claims.UserID,
		UserName: claims.UserName,
		Resource: "distribution:" + dist.ID,
		Success:  true,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Kernel configuration uploaded successfully",
		"key":     build.KernelConfigArtifactPath(dist.OwnerID, dist.ID),
	})
}
