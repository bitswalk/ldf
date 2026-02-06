package downloads

import (
	"net/http"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

var log = logs.NewDefault()

// NewHandler creates a new downloads handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		distRepo:        cfg.DistRepo,
		componentRepo:   cfg.ComponentRepo,
		downloadManager: cfg.DownloadManager,
	}
}

// calculateProgress calculates download progress as a percentage
func calculateProgress(progressBytes, totalBytes int64) float64 {
	if totalBytes <= 0 {
		return 0
	}
	return float64(progressBytes) / float64(totalBytes) * 100
}

// HandleStartDistributionDownloads starts downloads for a distribution
// @Summary      Start distribution downloads
// @Description  Starts downloads for all or selected components of a distribution
// @Tags         Downloads
// @Accept       json
// @Produce      json
// @Param        id       path      string                true  "Distribution ID"
// @Param        request  body      StartDownloadsRequest  true  "Download request"
// @Success      202      {object}  StartDownloadsResponse
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      403      {object}  common.ErrorResponse
// @Failure      404      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/downloads [post]
func (h *Handler) HandleStartDistributionDownloads(c *gin.Context) {
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	dist, err := h.distRepo.GetByID(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if dist == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	var req StartDownloadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = StartDownloadsRequest{}
	}

	jobs, err := h.downloadManager.CreateJobsForDistribution(dist, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if len(req.Components) > 0 {
		componentSet := make(map[string]bool)
		for _, id := range req.Components {
			componentSet[id] = true
		}

		var filteredJobs []db.DownloadJob
		for _, job := range jobs {
			if componentSet[job.ComponentID] {
				filteredJobs = append(filteredJobs, job)
			}
		}
		jobs = filteredJobs
	}

	response := make([]DownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := DownloadJobResponse{
			DownloadJob: job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := h.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		response = append(response, resp)
	}

	common.AuditLog(c, common.AuditEvent{Action: "downloads.start", UserID: claims.UserID, UserName: claims.UserName, Resource: "distribution:" + distID, Success: true})

	c.JSON(http.StatusAccepted, StartDownloadsResponse{
		Count: len(response),
		Jobs:  response,
	})
}

// HandleListDistributionDownloads lists download jobs for a distribution
// @Summary      List distribution downloads
// @Description  Lists download jobs for a distribution
// @Tags         Downloads
// @Produce      json
// @Param        id   path      string  true  "Distribution ID"
// @Success      200  {object}  DownloadJobsListResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/downloads [get]
func (h *Handler) HandleListDistributionDownloads(c *gin.Context) {
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	dist, err := h.distRepo.GetByID(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if dist == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	claims := common.GetClaimsFromContext(c)
	if dist.Visibility == db.VisibilityPrivate {
		if claims == nil || (dist.OwnerID != claims.UserID && !claims.HasAdminAccess()) {
			c.JSON(http.StatusForbidden, common.ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Access denied to private distribution",
			})
			return
		}
	}

	jobs, err := h.downloadManager.JobRepo().ListByDistribution(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	response := make([]DownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := DownloadJobResponse{
			DownloadJob: job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := h.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		response = append(response, resp)
	}

	c.JSON(http.StatusOK, DownloadJobsListResponse{
		Count: len(response),
		Jobs:  response,
	})
}

// HandleGetDownloadJob returns a single download job
// @Summary      Get download job
// @Description  Returns a single download job by ID
// @Tags         Downloads
// @Produce      json
// @Param        jobId  path      string  true  "Download job ID"
// @Success      200    {object}  DownloadJobResponse
// @Failure      400    {object}  common.ErrorResponse
// @Failure      403    {object}  common.ErrorResponse
// @Failure      404    {object}  common.ErrorResponse
// @Failure      500    {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/downloads/{jobId} [get]
func (h *Handler) HandleGetDownloadJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Job ID required",
		})
		return
	}

	job, err := h.downloadManager.JobRepo().GetByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Download job not found",
		})
		return
	}

	dist, err := h.distRepo.GetByID(job.DistributionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	claims := common.GetClaimsFromContext(c)
	if dist != nil && dist.Visibility == db.VisibilityPrivate {
		if claims == nil || (dist.OwnerID != claims.UserID && !claims.HasAdminAccess()) {
			c.JSON(http.StatusForbidden, common.ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Access denied",
			})
			return
		}
	}

	resp := DownloadJobResponse{
		DownloadJob: *job,
		Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
	}

	if component, err := h.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
		resp.ComponentName = component.DisplayName
	}

	c.JSON(http.StatusOK, resp)
}

// HandleCancelDownload cancels a download job
// @Summary      Cancel download
// @Description  Cancels an active download job
// @Tags         Downloads
// @Param        jobId  path      string  true  "Download job ID"
// @Success      204    "No Content"
// @Failure      400    {object}  common.ErrorResponse
// @Failure      401    {object}  common.ErrorResponse
// @Failure      403    {object}  common.ErrorResponse
// @Failure      404    {object}  common.ErrorResponse
// @Failure      500    {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/downloads/{jobId}/cancel [post]
func (h *Handler) HandleCancelDownload(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Job ID required",
		})
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	job, err := h.downloadManager.JobRepo().GetByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Download job not found",
		})
		return
	}

	dist, err := h.distRepo.GetByID(job.DistributionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if dist != nil && dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	if err := h.downloadManager.CancelJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "downloads.cancel", UserID: claims.UserID, UserName: claims.UserName, Resource: "job:" + jobID, Success: true})

	c.Status(http.StatusNoContent)
}

// HandleRetryDownload retries a failed download job
// @Summary      Retry download
// @Description  Retries a failed download job
// @Tags         Downloads
// @Produce      json
// @Param        jobId  path      string  true  "Download job ID"
// @Success      200    {object}  DownloadJobResponse
// @Failure      400    {object}  common.ErrorResponse
// @Failure      401    {object}  common.ErrorResponse
// @Failure      403    {object}  common.ErrorResponse
// @Failure      404    {object}  common.ErrorResponse
// @Failure      500    {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/downloads/{jobId}/retry [post]
func (h *Handler) HandleRetryDownload(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Job ID required",
		})
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	job, err := h.downloadManager.JobRepo().GetByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Download job not found",
		})
		return
	}

	dist, err := h.distRepo.GetByID(job.DistributionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if dist != nil && dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	if err := h.downloadManager.RetryJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	job, _ = h.downloadManager.JobRepo().GetByID(jobID)
	if job != nil {
		resp := DownloadJobResponse{
			DownloadJob: *job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := h.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	c.Status(http.StatusAccepted)
}

// HandleListActiveDownloads lists all active downloads (admin only)
// @Summary      List active downloads
// @Description  Lists all active downloads (admin only)
// @Tags         Downloads
// @Produce      json
// @Success      200  {object}  DownloadJobsListResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/downloads/active [get]
func (h *Handler) HandleListActiveDownloads(c *gin.Context) {
	jobs, err := h.downloadManager.JobRepo().ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	response := make([]DownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := DownloadJobResponse{
			DownloadJob: job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := h.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		response = append(response, resp)
	}

	c.JSON(http.StatusOK, DownloadJobsListResponse{
		Count: len(response),
		Jobs:  response,
	})
}

// HandleFlushDistributionDownloads deletes all download jobs for a distribution
// @Summary      Flush distribution downloads
// @Description  Deletes all download jobs for a distribution
// @Tags         Downloads
// @Param        id   path      string  true  "Distribution ID"
// @Success      204  "No Content"
// @Failure      400  {object}  common.ErrorResponse
// @Failure      401  {object}  common.ErrorResponse
// @Failure      403  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/downloads [delete]
func (h *Handler) HandleFlushDistributionDownloads(c *gin.Context) {
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	claims := common.GetClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	dist, err := h.distRepo.GetByID(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if dist == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	activeJobs, err := h.downloadManager.JobRepo().ListByDistribution(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	for _, job := range activeJobs {
		if job.Status == "pending" || job.Status == "verifying" || job.Status == "downloading" {
			if err := h.downloadManager.CancelJob(job.ID); err != nil {
				log.Warn("Failed to cancel download job during flush", "job_id", job.ID, "error", err)
			}
		}
	}

	if err := h.downloadManager.JobRepo().DeleteByDistribution(distID); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "downloads.flush", UserID: claims.UserID, UserName: claims.UserName, Resource: "distribution:" + distID, Success: true})

	c.Status(http.StatusNoContent)
}
