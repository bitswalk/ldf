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
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new artifacts handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		distRepo:   cfg.DistRepo,
		storage:    cfg.Storage,
		jwtService: cfg.JWTService,
	}
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
	claims := common.GetTokenClaimsFromRequest(c, h.jwtService)
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
// @Summary      Upload artifact
// @Description  Uploads an artifact file for a distribution
// @Tags         Artifacts
// @Accept       multipart/form-data
// @Produce      json
// @Param        id    path      string  true   "Distribution ID"
// @Param        file  formData  file    true   "Artifact file"
// @Param        path  formData  string  false  "Storage path"
// @Success      201   {object}  ArtifactUploadResponse
// @Failure      400   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Failure      503   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/artifacts [post]
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

	_ = h.distRepo.AddLog(distID, "info", fmt.Sprintf("Artifact uploaded: %s (%d bytes)", artifactPath, header.Size))

	c.JSON(http.StatusCreated, ArtifactUploadResponse{
		Key:     key,
		Size:    header.Size,
		Message: "Artifact uploaded successfully",
	})
}

// HandleList lists all artifacts for a distribution
// @Summary      List distribution artifacts
// @Description  Lists all artifacts for a distribution
// @Tags         Artifacts
// @Produce      json
// @Param        id   path      string  true  "Distribution ID"
// @Success      200  {object}  ArtifactListResponse
// @Failure      400  {object}  common.ErrorResponse
// @Failure      404  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Failure      503  {object}  common.ErrorResponse
// @Router       /v1/distributions/{id}/artifacts [get]
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
// @Summary      Download artifact
// @Description  Downloads an artifact file from a distribution
// @Tags         Artifacts
// @Produce      octet-stream
// @Param        id    path  string  true  "Distribution ID"
// @Param        path  path  string  true  "Artifact path"
// @Success      200   {file}    string  "File download"
// @Failure      400   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      503   {object}  common.ErrorResponse
// @Router       /v1/distributions/{id}/artifacts/{path} [get]
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
	_, _ = io.Copy(c.Writer, reader)
}

// HandleDelete deletes an artifact from a distribution
// @Summary      Delete artifact
// @Description  Deletes an artifact from a distribution
// @Tags         Artifacts
// @Param        id    path      string  true  "Distribution ID"
// @Param        path  path      string  true  "Artifact path"
// @Success      204   "No Content"
// @Failure      400   {object}  common.ErrorResponse
// @Failure      404   {object}  common.ErrorResponse
// @Failure      500   {object}  common.ErrorResponse
// @Failure      503   {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /v1/distributions/{id}/artifacts/{path} [delete]
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

	_ = h.distRepo.AddLog(distID, "info", fmt.Sprintf("Artifact deleted: %s", artifactPath))

	c.Status(http.StatusNoContent)
}

// HandleGetURL generates a presigned URL for direct artifact download
// @Summary      Get presigned URL
// @Description  Generates a presigned URL for direct artifact download
// @Tags         Artifacts
// @Produce      json
// @Param        id      path   string  true   "Distribution ID"
// @Param        path    path   string  true   "Artifact path"
// @Param        expiry  query  int     false  "Expiry in seconds"
// @Success      200     {object}  ArtifactURLResponse
// @Failure      400     {object}  common.ErrorResponse
// @Failure      404     {object}  common.ErrorResponse
// @Failure      500     {object}  common.ErrorResponse
// @Failure      503     {object}  common.ErrorResponse
// @Router       /v1/distributions/{id}/artifacts-url/{path} [get]
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
// @Summary      List all artifacts
// @Description  Lists all artifacts across all accessible distributions
// @Tags         Artifacts
// @Produce      json
// @Success      200  {object}  GlobalArtifactListResponse
// @Failure      500  {object}  common.ErrorResponse
// @Failure      503  {object}  common.ErrorResponse
// @Router       /v1/artifacts [get]
func (h *Handler) HandleListAll(c *gin.Context) {
	if h.storage == nil {
		c.JSON(http.StatusServiceUnavailable, common.ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	claims := common.GetTokenClaimsFromRequest(c, h.jwtService)
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
// @Summary      Get storage status
// @Description  Returns the current status of the storage backend
// @Tags         Storage
// @Produce      json
// @Success      200  {object}  StorageStatusResponse
// @Router       /v1/storage/status [get]
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
