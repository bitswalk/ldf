package builds

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/build"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new builds handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		distRepo:     cfg.DistRepo,
		buildManager: cfg.BuildManager,
	}
}

// HandleStartBuild starts a build for a distribution
func (h *Handler) HandleStartBuild(c *gin.Context) {
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

	if dist.Config == nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution has no configuration",
		})
		return
	}

	var req StartBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = StartBuildRequest{}
	}

	// Parse arch and format with defaults
	arch := db.ArchX86_64
	if req.Arch != "" {
		switch req.Arch {
		case "x86_64":
			arch = db.ArchX86_64
		case "aarch64":
			arch = db.ArchAARCH64
		default:
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Unsupported architecture: %s (supported: x86_64, aarch64)", req.Arch),
			})
			return
		}
	}

	format := db.ImageFormatRaw
	if req.Format != "" {
		switch req.Format {
		case "raw":
			format = db.ImageFormatRaw
		case "qcow2":
			format = db.ImageFormatQCOW2
		case "iso":
			format = db.ImageFormatISO
		default:
			c.JSON(http.StatusBadRequest, common.ErrorResponse{
				Error:   "Bad request",
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Unsupported image format: %s (supported: raw, qcow2, iso)", req.Format),
			})
			return
		}
	}

	// Pre-flight: validate build environment for the requested architecture
	runtime := build.RuntimeType(h.buildManager.GetConfig().ContainerRuntime)
	if _, err := build.ValidateBuildEnvironment(runtime, h.buildManager.GetConfig().ContainerImage, arch); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Build environment not available",
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Cannot build for %s: %v", arch, err),
		})
		return
	}

	job, err := h.buildManager.SubmitBuild(dist, claims.UserID, arch, format, req.ClearCache)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "build.start", UserID: claims.UserID, UserName: claims.UserName, Resource: "distribution:" + distID, Success: true})

	c.JSON(http.StatusAccepted, BuildJobResponse{
		BuildJob: *job,
	})
}

// HandleListDistributionBuilds lists build jobs for a distribution
func (h *Handler) HandleListDistributionBuilds(c *gin.Context) {
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

	jobs, err := h.buildManager.BuildJobRepo().ListByDistribution(distID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	response := make([]BuildJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := BuildJobResponse{BuildJob: job}
		if stages, err := h.buildManager.BuildJobRepo().GetStages(job.ID); err == nil {
			resp.Stages = stages
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, BuildJobsListResponse{
		Count:  len(response),
		Builds: response,
	})
}

// HandleGetBuild returns a single build job with stages
func (h *Handler) HandleGetBuild(c *gin.Context) {
	buildID := c.Param("buildId")
	if buildID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Build ID required",
		})
		return
	}

	job, err := h.buildManager.BuildJobRepo().GetByID(buildID)
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
			Message: "Build not found",
		})
		return
	}

	// Check access
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

	resp := BuildJobResponse{BuildJob: *job}
	if stages, err := h.buildManager.BuildJobRepo().GetStages(buildID); err == nil {
		resp.Stages = stages
	}

	c.JSON(http.StatusOK, resp)
}

// HandleGetBuildLogs returns log entries for a build
func (h *Handler) HandleGetBuildLogs(c *gin.Context) {
	buildID := c.Param("buildId")
	if buildID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Build ID required",
		})
		return
	}

	job, err := h.buildManager.BuildJobRepo().GetByID(buildID)
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
			Message: "Build not found",
		})
		return
	}

	// Parse query params
	stage := c.Query("stage")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var logs []db.BuildLog
	if stage != "" {
		logs, err = h.buildManager.BuildJobRepo().GetLogsByStage(buildID, stage)
	} else {
		logs, err = h.buildManager.BuildJobRepo().GetLogs(buildID, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if logs == nil {
		logs = []db.BuildLog{}
	}

	c.JSON(http.StatusOK, BuildLogsResponse{
		Count: len(logs),
		Logs:  logs,
	})
}

// HandleStreamBuildLogs streams build logs via SSE
func (h *Handler) HandleStreamBuildLogs(c *gin.Context) {
	buildID := c.Param("buildId")
	if buildID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Build ID required",
		})
		return
	}

	job, err := h.buildManager.BuildJobRepo().GetByID(buildID)
	if err != nil || job == nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Build not found",
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	var lastID int64
	var lastStatus db.BuildJobStatus
	var lastProgress int

	c.Stream(func(w io.Writer) bool {
		currentJob, err := h.buildManager.BuildJobRepo().GetByID(buildID)
		if err != nil || currentJob == nil {
			return false
		}

		isTerminal := currentJob.Status == db.BuildStatusCompleted ||
			currentJob.Status == db.BuildStatusFailed ||
			currentJob.Status == db.BuildStatusCancelled

		// Send status event when status or progress changed
		if currentJob.Status != lastStatus || currentJob.ProgressPercent != lastProgress {
			stages, _ := h.buildManager.BuildJobRepo().GetStages(buildID)
			statusEvent := BuildStatusEvent{
				Status:          currentJob.Status,
				CurrentStage:    currentJob.CurrentStage,
				ProgressPercent: currentJob.ProgressPercent,
				Stages:          stages,
				CompletedAt:     currentJob.CompletedAt,
				ErrorMessage:    currentJob.ErrorMessage,
				ErrorStage:      currentJob.ErrorStage,
				ArtifactSize:    currentJob.ArtifactSize,
			}
			data, _ := json.Marshal(statusEvent)
			fmt.Fprintf(w, "event: status\ndata: %s\n\n", data)
			lastStatus = currentJob.Status
			lastProgress = currentJob.ProgressPercent
		}

		// Fetch new logs
		logs, err := h.buildManager.BuildJobRepo().GetLogsSince(buildID, lastID)
		if err != nil {
			return false
		}

		for _, entry := range logs {
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if entry.ID > lastID {
				lastID = entry.ID
			}
		}

		// If terminal and no more logs, close
		if isTerminal && len(logs) == 0 {
			fmt.Fprintf(w, "event: done\ndata: {\"status\":\"%s\"}\n\n", currentJob.Status)
			return false
		}

		// Wait before polling again
		time.Sleep(1 * time.Second)
		return true
	})
}

// HandleCancelBuild cancels a running build
func (h *Handler) HandleCancelBuild(c *gin.Context) {
	buildID := c.Param("buildId")
	if buildID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Build ID required",
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

	job, err := h.buildManager.BuildJobRepo().GetByID(buildID)
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
			Message: "Build not found",
		})
		return
	}

	// Check ownership
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

	if err := h.buildManager.CancelBuild(buildID); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "build.cancel", UserID: claims.UserID, UserName: claims.UserName, Resource: "build:" + buildID, Success: true})

	c.Status(http.StatusNoContent)
}

// HandleRetryBuild retries a failed build
func (h *Handler) HandleRetryBuild(c *gin.Context) {
	buildID := c.Param("buildId")
	if buildID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Build ID required",
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

	job, err := h.buildManager.BuildJobRepo().GetByID(buildID)
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
			Message: "Build not found",
		})
		return
	}

	// Check ownership
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

	if err := h.buildManager.RetryBuild(buildID); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Re-fetch updated job
	job, _ = h.buildManager.BuildJobRepo().GetByID(buildID)
	if job != nil {
		resp := BuildJobResponse{BuildJob: *job}
		if stages, err := h.buildManager.BuildJobRepo().GetStages(buildID); err == nil {
			resp.Stages = stages
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	c.Status(http.StatusAccepted)
}

// HandleClearDistributionBuilds removes all build jobs for a distribution
func (h *Handler) HandleClearDistributionBuilds(c *gin.Context) {
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
	if claims == nil || (dist.OwnerID != claims.UserID && !claims.HasAdminAccess()) {
		c.JSON(http.StatusForbidden, common.ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "Access denied",
		})
		return
	}

	if err := h.buildManager.BuildJobRepo().DeleteByDistribution(distID); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Build history cleared"})
}

// HandleListActiveBuilds lists all active builds (admin only)
func (h *Handler) HandleListActiveBuilds(c *gin.Context) {
	jobs, err := h.buildManager.BuildJobRepo().ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	response := make([]BuildJobResponse, 0, len(jobs))
	for _, job := range jobs {
		resp := BuildJobResponse{BuildJob: job}
		if stages, err := h.buildManager.BuildJobRepo().GetStages(job.ID); err == nil {
			resp.Stages = stages
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, BuildJobsListResponse{
		Count:  len(response),
		Builds: response,
	})
}
