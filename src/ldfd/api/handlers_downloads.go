package api

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// DownloadJobResponse represents a download job with additional info
type DownloadJobResponse struct {
	db.DownloadJob
	ComponentName string  `json:"component_name,omitempty"`
	Progress      float64 `json:"progress"` // 0-100
}

// DownloadJobsListResponse represents a list of download jobs
type DownloadJobsListResponse struct {
	Count int                   `json:"count"`
	Jobs  []DownloadJobResponse `json:"jobs"`
}

// StartDownloadsRequest represents the request to start downloads
type StartDownloadsRequest struct {
	Components []string `json:"components"` // Component IDs to download, empty = all required
}

// StartDownloadsResponse represents the response after starting downloads
type StartDownloadsResponse struct {
	Count int                   `json:"count"`
	Jobs  []DownloadJobResponse `json:"jobs"`
}

// handleStartDistributionDownloads starts downloads for a distribution
func (a *API) handleStartDistributionDownloads(c *gin.Context) {
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	// Check distribution exists and user has write access
	dist, err := a.distRepo.GetByID(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if dist == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	// Check write access
	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	var req StartDownloadsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK, means download all required components
		req = StartDownloadsRequest{}
	}

	// Create jobs for distribution
	jobs, err := a.downloadManager.CreateJobsForDistribution(dist, claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Filter by requested components if specified
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

	// Build response with component names
	response := make([]DownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := DownloadJobResponse{
			DownloadJob: job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := a.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		response = append(response, resp)
	}

	c.JSON(http.StatusAccepted, StartDownloadsResponse{
		Count: len(response),
		Jobs:  response,
	})
}

// handleListDistributionDownloads lists download jobs for a distribution
func (a *API) handleListDistributionDownloads(c *gin.Context) {
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Check distribution exists
	dist, err := a.distRepo.GetByID(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if dist == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	// Check access
	claims := getClaimsFromContext(c)
	if dist.Visibility == db.VisibilityPrivate {
		if claims == nil || (dist.OwnerID != claims.UserID && !claims.HasAdminAccess()) {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Access denied to private distribution",
			})
			return
		}
	}

	jobs, err := a.downloadManager.JobRepo().ListByDistribution(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Build response with component names
	response := make([]DownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := DownloadJobResponse{
			DownloadJob: job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := a.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		response = append(response, resp)
	}

	c.JSON(http.StatusOK, DownloadJobsListResponse{
		Count: len(response),
		Jobs:  response,
	})
}

// handleGetDownloadJob returns a single download job
func (a *API) handleGetDownloadJob(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Job ID required",
		})
		return
	}

	job, err := a.downloadManager.JobRepo().GetByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Download job not found",
		})
		return
	}

	// Check access via distribution
	dist, err := a.distRepo.GetByID(job.DistributionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	claims := getClaimsFromContext(c)
	if dist != nil && dist.Visibility == db.VisibilityPrivate {
		if claims == nil || (dist.OwnerID != claims.UserID && !claims.HasAdminAccess()) {
			c.JSON(http.StatusForbidden, ErrorResponse{
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

	if component, err := a.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
		resp.ComponentName = component.DisplayName
	}

	c.JSON(http.StatusOK, resp)
}

// handleCancelDownload cancels a download job
func (a *API) handleCancelDownload(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Job ID required",
		})
		return
	}

	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	job, err := a.downloadManager.JobRepo().GetByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Download job not found",
		})
		return
	}

	// Check write access via distribution
	dist, err := a.distRepo.GetByID(job.DistributionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if dist != nil && dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	if err := a.downloadManager.CancelJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleRetryDownload retries a failed download job
func (a *API) handleRetryDownload(c *gin.Context) {
	jobID := c.Param("jobId")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Job ID required",
		})
		return
	}

	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	job, err := a.downloadManager.JobRepo().GetByID(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if job == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Download job not found",
		})
		return
	}

	// Check write access via distribution
	dist, err := a.distRepo.GetByID(job.DistributionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if dist != nil && dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	if err := a.downloadManager.RetryJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Return updated job
	job, _ = a.downloadManager.JobRepo().GetByID(jobID)
	if job != nil {
		resp := DownloadJobResponse{
			DownloadJob: *job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := a.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		c.JSON(http.StatusOK, resp)
		return
	}

	c.Status(http.StatusAccepted)
}

// handleListActiveDownloads lists all active downloads (admin only)
func (a *API) handleListActiveDownloads(c *gin.Context) {
	jobs, err := a.downloadManager.JobRepo().ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Build response with component names
	response := make([]DownloadJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := DownloadJobResponse{
			DownloadJob: job,
			Progress:    calculateProgress(job.ProgressBytes, job.TotalBytes),
		}

		if component, err := a.componentRepo.GetByID(job.ComponentID); err == nil && component != nil {
			resp.ComponentName = component.DisplayName
		}

		response = append(response, resp)
	}

	c.JSON(http.StatusOK, DownloadJobsListResponse{
		Count: len(response),
		Jobs:  response,
	})
}

// handleFlushDistributionDownloads deletes all download jobs for a distribution
func (a *API) handleFlushDistributionDownloads(c *gin.Context) {
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	// Check distribution exists and user has write access
	dist, err := a.distRepo.GetByID(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if dist == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	// Check write access
	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Write access required",
		})
		return
	}

	// Cancel any active jobs first
	activeJobs, err := a.downloadManager.JobRepo().ListByDistribution(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	for _, job := range activeJobs {
		if job.Status == "pending" || job.Status == "verifying" || job.Status == "downloading" {
			_ = a.downloadManager.CancelJob(job.ID)
		}
	}

	// Delete all download jobs for this distribution
	if err := a.downloadManager.JobRepo().DeleteByDistribution(distID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// calculateProgress calculates download progress as a percentage
func calculateProgress(progressBytes, totalBytes int64) float64 {
	if totalBytes <= 0 {
		return 0
	}
	return float64(progressBytes) / float64(totalBytes) * 100
}
