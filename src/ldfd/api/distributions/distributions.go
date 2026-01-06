package distributions

import (
	"net/http"
	"strconv"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/gin-gonic/gin"
)

var log *logs.Logger

// SetLogger sets the logger for the distributions package
func SetLogger(l *logs.Logger) {
	log = l
}

// Handler handles distribution-related HTTP requests
type Handler struct {
	distRepo   *db.DistributionRepository
	jwtService *auth.JWTService
}

// Config contains configuration options for the Handler
type Config struct {
	DistRepo   *db.DistributionRepository
	JWTService *auth.JWTService
}

// NewHandler creates a new distributions handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		distRepo:   cfg.DistRepo,
		jwtService: cfg.JWTService,
	}
}

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

// getTokenClaims extracts and validates JWT claims from the request
func (h *Handler) getTokenClaims(c *gin.Context) *auth.TokenClaims {
	token := c.GetHeader("X-Subject-Token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		return nil
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return nil
	}

	return claims
}

// HandleList returns a list of distributions accessible to the current user
func (h *Handler) HandleList(c *gin.Context) {
	var statusFilter *db.DistributionStatus
	if statusParam := c.Query("status"); statusParam != "" {
		status := db.DistributionStatus(statusParam)
		statusFilter = &status
	}

	claims := h.getTokenClaims(c)
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
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleCreate creates a new distribution record
func (h *Handler) HandleCreate(c *gin.Context) {
	var req CreateDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	existing, err := h.distRepo.GetByName(req.Name)
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
			Message: "Distribution with this name already exists",
		})
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
		c.JSON(http.StatusUnauthorized, common.ErrorResponse{
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

	if err := h.distRepo.Create(dist); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	h.distRepo.AddLog(dist.ID, "info", "Distribution created")

	c.JSON(http.StatusCreated, dist)
}

// HandleGet returns a distribution by ID if the user has access
func (h *Handler) HandleGet(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	claims := h.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	canAccess, err := h.distRepo.CanUserAccess(id, userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if !canAccess {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return
	}

	dist, err := h.distRepo.GetByID(id)
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

	c.JSON(http.StatusOK, dist)
}

// HandleUpdate updates an existing distribution (owner or admin only)
func (h *Handler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
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

	dist, err := h.distRepo.GetByID(id)
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
			Message: "You can only update your own distributions",
		})
		return
	}

	var req UpdateDistributionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	h.distRepo.AddLog(dist.ID, "info", "Distribution updated")

	c.JSON(http.StatusOK, dist)
}

// HandleDelete deletes a distribution by ID (owner or admin only)
func (h *Handler) HandleDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
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

	dist, err := h.distRepo.GetByID(id)
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
			Message: "You can only delete your own distributions",
		})
		return
	}

	if err := h.distRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// HandleGetLogs returns logs for a distribution if the user has access
func (h *Handler) HandleGetLogs(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	claims := h.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	canAccess, err := h.distRepo.CanUserAccess(id, userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}
	if !canAccess {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
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

	logs, err := h.distRepo.GetLogs(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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

// HandleGetStats returns statistics about distributions grouped by status
func (h *Handler) HandleGetStats(c *gin.Context) {
	stats, err := h.distRepo.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
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
