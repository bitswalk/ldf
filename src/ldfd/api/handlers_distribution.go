package api

import (
	"net/http"
	"strconv"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

// Note: Distribution IDs are now UUIDs (strings), not integers

// CreateDistributionRequest represents the request to create a distribution
type CreateDistributionRequest struct {
	Name       string                 `json:"name" binding:"required" example:"ubuntu-22.04"`
	Version    string                 `json:"version" example:"22.04.3"`
	Visibility string                 `json:"visibility" example:"private"`
	Config     *db.DistributionConfig `json:"config"`
	SourceURL  string                 `json:"source_url" example:"https://releases.ubuntu.com/22.04/ubuntu-22.04.3-live-server-amd64.iso"`
	Checksum   string                 `json:"checksum" example:"sha256:a4acfda10b18da50e2ec50ccaf860d7f20b389df8765611142305c0e911d16fd"`
}

// UpdateDistributionRequest represents the request to update a distribution
type UpdateDistributionRequest struct {
	Name       string                 `json:"name" example:"ubuntu-22.04"`
	Version    string                 `json:"version" example:"22.04.3"`
	Status     string                 `json:"status" example:"ready"`
	Visibility string                 `json:"visibility" example:"public"`
	SourceURL  string                 `json:"source_url" example:"https://releases.ubuntu.com/22.04/ubuntu-22.04.3-live-server-amd64.iso"`
	Checksum   string                 `json:"checksum" example:"sha256:a4acfda10b18da50e2ec50ccaf860d7f20b389df8765611142305c0e911d16fd"`
	SizeBytes  int64                  `json:"size_bytes" example:"2048576000"`
	Config     *db.DistributionConfig `json:"config,omitempty"`
}

// DistributionListResponse represents a list of distributions
type DistributionListResponse struct {
	Count         int               `json:"count" example:"5"`
	Distributions []db.Distribution `json:"distributions"`
}

// DistributionStatsResponse represents distribution statistics
type DistributionStatsResponse struct {
	Total int64            `json:"total" example:"10"`
	Stats map[string]int64 `json:"stats"`
}

// handleListDistributions returns a list of distributions accessible to the current user
// - Anonymous users: only public distributions
// - Authenticated users: public + their own private distributions
// - Admins: all distributions
func (a *API) handleListDistributions(c *gin.Context) {
	var statusFilter *db.DistributionStatus
	if statusParam := c.Query("status"); statusParam != "" {
		status := db.DistributionStatus(statusParam)
		statusFilter = &status
	}

	// Get user context (may be nil for anonymous)
	claims := a.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
		log.Printf("[DEBUG] ListDistributions: authenticated user=%s (name=%s), isAdmin=%v", userID, claims.UserName, isAdmin)
	} else {
		log.Printf("[DEBUG] ListDistributions: anonymous user (no valid token)")
	}

	distributions, err := a.distRepo.ListAccessible(userID, isAdmin, statusFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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

// handleCreateDistribution creates a new distribution record
func (a *API) handleCreateDistribution(c *gin.Context) {
	var req CreateDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Check if distribution with same name exists
	existing, err := a.distRepo.GetByName(req.Name)
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
			Message: "Distribution with this name already exists",
		})
		return
	}

	// Set distribution version (not kernel version - that's stored in config)
	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	// Parse visibility (default to private)
	visibility := db.VisibilityPrivate
	if req.Visibility == "public" {
		visibility = db.VisibilityPublic
	}

	// Get owner from authenticated user (set by writeAccessRequired middleware)
	claims := getClaimsFromContext(c)
	if claims == nil {
		// This shouldn't happen as writeAccessRequired middleware should have rejected
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
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

	if err := a.distRepo.Create(dist); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Add creation log
	a.distRepo.AddLog(dist.ID, "info", "Distribution created")

	c.JSON(http.StatusCreated, dist)
}

// handleGetDistribution returns a distribution by ID if the user has access
func (a *API) handleGetDistribution(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Get user context (may be nil for anonymous)
	claims := a.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	// Check if user can access this distribution
	canAccess, err := a.distRepo.CanUserAccess(id, userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if !canAccess {
		// Return 404 instead of 403 to not reveal existence of private distributions
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	dist, err := a.distRepo.GetByID(id)
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

	c.JSON(http.StatusOK, dist)
}

// handleUpdateDistribution updates an existing distribution (owner or admin only)
func (a *API) handleUpdateDistribution(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Get authenticated user
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	dist, err := a.distRepo.GetByID(id)
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

	// Check ownership or admin access
	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only update your own distributions",
		})
		return
	}

	var req UpdateDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// Update fields if provided
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

	if err := a.distRepo.Update(dist); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	a.distRepo.AddLog(dist.ID, "info", "Distribution updated")

	c.JSON(http.StatusOK, dist)
}

// handleDeleteDistribution deletes a distribution by ID (owner or admin only)
func (a *API) handleDeleteDistribution(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Get authenticated user
	claims := getClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Code:    http.StatusUnauthorized,
			Message: "Authentication required",
		})
		return
	}

	// Check if distribution exists and user has access
	dist, err := a.distRepo.GetByID(id)
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

	// Check ownership or admin access
	if dist.OwnerID != claims.UserID && !claims.HasAdminAccess() {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Code:    http.StatusForbidden,
			Message: "You can only delete your own distributions",
		})
		return
	}

	if err := a.distRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleGetDistributionLogs returns logs for a distribution if the user has access
func (a *API) handleGetDistributionLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Get user context (may be nil for anonymous)
	claims := a.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	// Check if user can access this distribution
	canAccess, err := a.distRepo.CanUserAccess(id, userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if !canAccess {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := a.distRepo.GetLogs(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	if logs == nil {
		logs = []db.DistributionLog{}
	}

	c.JSON(http.StatusOK, logs)
}

// handleGetDistributionStats returns statistics about distributions grouped by status
func (a *API) handleGetDistributionStats(c *gin.Context) {
	stats, err := a.distRepo.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
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
