package artifacts

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
	"github.com/gin-gonic/gin"
)

// Handler handles artifact-related HTTP requests
type Handler struct {
	distRepo   *db.DistributionRepository
	storage    storage.Backend
	jwtService *auth.JWTService
}

// Config contains configuration options for the Handler
type Config struct {
	DistRepo   *db.DistributionRepository
	Storage    storage.Backend
	JWTService *auth.JWTService
}

// NewHandler creates a new artifacts handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		distRepo:   cfg.DistRepo,
		storage:    cfg.Storage,
		jwtService: cfg.JWTService,
	}
}

// ArtifactUploadResponse represents the response after uploading an artifact
type ArtifactUploadResponse struct {
	Key     string `json:"key" example:"distribution/49009ac3-dddf-48d6-896d-c6900f47c072/1afe09ac3-dddf-48d6-896d-c6321f47c072/ubuntu-22.04.iso"`
	Size    int64  `json:"size" example:"2048576000"`
	Message string `json:"message" example:"Artifact uploaded successfully"`
}

// ArtifactListResponse represents a list of artifacts
type ArtifactListResponse struct {
	DistributionID string               `json:"distribution_id" example:"1afe09ac3-dddf-48d6-896d-c6321f47c072"`
	Count          int                  `json:"count" example:"3"`
	Artifacts      []storage.ObjectInfo `json:"artifacts"`
}

// ArtifactURLResponse represents a presigned URL response
type ArtifactURLResponse struct {
	URL       string    `json:"url" example:"https://s3.example.com/bucket/key?signature=..."`
	WebURL    string    `json:"web_url,omitempty" example:"https://ldf.s3.example.com/distribution/49009ac3-dddf-48d6-896d-c6900f47c072/1afe09ac3-dddf-48d6-896d-c6321f47c072/file.iso"`
	ExpiresAt time.Time `json:"expires_at"`
}

// StorageStatusResponse represents the storage status
type StorageStatusResponse struct {
	Available bool   `json:"available" example:"true"`
	Type      string `json:"type" example:"local"`
	Location  string `json:"location" example:"~/.ldfd/artifacts"`
	Message   string `json:"message" example:"Storage is operational"`
}

// GlobalArtifact represents an artifact with its distribution context
type GlobalArtifact struct {
	Key              string    `json:"key" example:"boot/vmlinuz"`
	FullKey          string    `json:"full_key" example:"distribution/49009ac3-dddf-48d6-896d-c6900f47c072/1afe09ac3-dddf-48d6-896d-c6321f47c072/boot/vmlinuz"`
	Size             int64     `json:"size" example:"10485760"`
	ContentType      string    `json:"content_type,omitempty" example:"application/octet-stream"`
	ETag             string    `json:"etag,omitempty"`
	LastModified     time.Time `json:"last_modified"`
	DistributionID   string    `json:"distribution_id" example:"1afe09ac3-dddf-48d6-896d-c6321f47c072"`
	DistributionName string    `json:"distribution_name" example:"ubuntu-server"`
	OwnerID          string    `json:"owner_id,omitempty" example:"49009ac3-dddf-48d6-896d-c6900f47c072"`
}

// GlobalArtifactListResponse represents the response for listing all artifacts
type GlobalArtifactListResponse struct {
	Count     int              `json:"count" example:"10"`
	Artifacts []GlobalArtifact `json:"artifacts"`
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

// getArtifactPrefix returns the S3 key prefix for a distribution's artifacts
func getArtifactPrefix(ownerID, distributionID string) string {
	return fmt.Sprintf("distribution/%s/%s/", ownerID, distributionID)
}

// getArtifactKey returns the full S3 key for an artifact
func getArtifactKey(ownerID, distributionID, path string) string {
	path = strings.TrimPrefix(path, "/")
	return fmt.Sprintf("distribution/%s/%s/%s", ownerID, distributionID, path)
}

// checkDistributionAccess verifies the user has access to the distribution
func (h *Handler) checkDistributionAccess(c *gin.Context, distID string) (*struct {
	OwnerID string
	ID      string
}, bool) {
	claims := h.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	canAccess, err := h.distRepo.CanUserAccess(distID, userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return nil, false
	}
	if !canAccess {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return nil, false
	}

	dist, err := h.distRepo.GetByID(distID)
	if err != nil || dist == nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to retrieve distribution details",
		})
		return nil, false
	}

	return &struct {
		OwnerID string
		ID      string
	}{OwnerID: dist.OwnerID, ID: dist.ID}, true
}

// HandleUpload uploads an artifact file for a distribution
func (h *Handler) HandleUpload(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

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

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "No file provided: " + err.Error(),
		})
		return
	}
	defer file.Close()

	artifactPath := c.PostForm("path")
	if artifactPath == "" {
		artifactPath = header.Filename
	}

	key := getArtifactKey(dist.OwnerID, dist.ID, artifactPath)

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.storage.Upload(c.Request.Context(), key, file, header.Size, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to upload artifact: " + err.Error(),
		})
		return
	}

	h.distRepo.AddLog(distID, "info", fmt.Sprintf("Artifact uploaded: %s (%d bytes)", artifactPath, header.Size))

	c.JSON(http.StatusCreated, ArtifactUploadResponse{
		Key:     key,
		Size:    header.Size,
		Message: "Artifact uploaded successfully",
	})
}

// HandleList lists all artifacts for a distribution
func (h *Handler) HandleList(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	distInfo, ok := h.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	prefix := getArtifactPrefix(distInfo.OwnerID, distInfo.ID)
	artifacts, err := h.storage.List(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to list artifacts: " + err.Error(),
		})
		return
	}

	for i := range artifacts {
		artifacts[i].Key = strings.TrimPrefix(artifacts[i].Key, prefix)
	}

	c.JSON(http.StatusOK, ArtifactListResponse{
		DistributionID: distID,
		Count:          len(artifacts),
		Artifacts:      artifacts,
	})
}

// HandleDownload downloads an artifact file from a distribution
func (h *Handler) HandleDownload(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	distInfo, ok := h.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	artifactPath := c.Param("path")
	if artifactPath == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Artifact path required",
		})
		return
	}

	key := getArtifactKey(distInfo.OwnerID, distInfo.ID, artifactPath)

	reader, info, err := h.storage.Download(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Artifact not found: " + err.Error(),
		})
		return
	}
	defer reader.Close()

	filename := filepath.Base(artifactPath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("ETag", info.ETag)

	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}

// HandleDelete deletes an artifact from a distribution
func (h *Handler) HandleDelete(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	distInfo, ok := h.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	artifactPath := c.Param("path")
	if artifactPath == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Artifact path required",
		})
		return
	}

	key := getArtifactKey(distInfo.OwnerID, distInfo.ID, artifactPath)

	if err := h.storage.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to delete artifact: " + err.Error(),
		})
		return
	}

	h.distRepo.AddLog(distID, "info", fmt.Sprintf("Artifact deleted: %s", artifactPath))

	c.Status(http.StatusNoContent)
}

// HandleGetURL generates a presigned URL for direct artifact download
func (h *Handler) HandleGetURL(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	distInfo, ok := h.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	artifactPath := c.Param("path")
	if artifactPath == "" {
		c.JSON(http.StatusBadRequest, common.ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Artifact path required",
		})
		return
	}

	expiry := 3600
	if expiryStr := c.Query("expiry"); expiryStr != "" {
		if e, err := strconv.Atoi(expiryStr); err == nil && e > 0 {
			expiry = e
		}
	}

	key := getArtifactKey(distInfo.OwnerID, distInfo.ID, artifactPath)

	exists, err := h.storage.Exists(c.Request.Context(), key)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, common.ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Artifact not found",
		})
		return
	}

	url, err := h.storage.GetPresignedURL(c.Request.Context(), key, time.Duration(expiry)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate presigned URL: " + err.Error(),
		})
		return
	}

	webURL := h.storage.GetWebURL(key)

	c.JSON(http.StatusOK, ArtifactURLResponse{
		URL:       url,
		WebURL:    webURL,
		ExpiresAt: time.Now().Add(time.Duration(expiry) * time.Second),
	})
}

// HandleListAll lists all artifacts across all accessible distributions
func (h *Handler) HandleListAll(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
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

	distributions, err := h.distRepo.ListAccessible(userID, isAdmin, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	distMap := make(map[string]struct {
		Name    string
		OwnerID string
	})
	for _, dist := range distributions {
		distMap[dist.ID] = struct {
			Name    string
			OwnerID string
		}{
			Name:    dist.Name,
			OwnerID: dist.OwnerID,
		}
	}

	allStorageArtifacts, err := h.storage.List(c.Request.Context(), "distribution/")
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to list artifacts: " + err.Error(),
		})
		return
	}

	var artifacts []GlobalArtifact
	for _, obj := range allStorageArtifacts {
		parts := strings.SplitN(obj.Key, "/", 4)
		if len(parts) < 4 {
			continue
		}

		ownerID := parts[1]
		distID := parts[2]

		distInfo, hasAccess := distMap[distID]
		if !hasAccess {
			continue
		}

		artifactKey := parts[3]
		artifacts = append(artifacts, GlobalArtifact{
			Key:              artifactKey,
			FullKey:          obj.Key,
			Size:             obj.Size,
			ContentType:      obj.ContentType,
			ETag:             obj.ETag,
			LastModified:     obj.LastModified,
			DistributionID:   distID,
			DistributionName: distInfo.Name,
			OwnerID:          ownerID,
		})
	}

	c.JSON(http.StatusOK, GlobalArtifactListResponse{
		Count:     len(artifacts),
		Artifacts: artifacts,
	})
}

// HandleStorageStatus returns the status of the storage backend
func (h *Handler) HandleStorageStatus(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusOK, StorageStatusResponse{
			Available: false,
			Message:   "Storage service not configured",
		})
		return
	}

	err := h.storage.Ping(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, StorageStatusResponse{
			Available: false,
			Type:      h.storage.Type(),
			Location:  h.storage.Location(),
			Message:   "Storage unreachable: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, StorageStatusResponse{
		Available: true,
		Type:      h.storage.Type(),
		Location:  h.storage.Location(),
		Message:   "Storage is operational",
	})
}
