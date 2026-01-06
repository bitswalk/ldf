package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/storage"
	"github.com/gin-gonic/gin"
)

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

// getArtifactPrefix returns the S3 key prefix for a distribution's artifacts
// Path structure: distribution/<ownerID>/<distributionID>/
func getArtifactPrefix(ownerID, distributionID string) string {
	return fmt.Sprintf("distribution/%s/%s/", ownerID, distributionID)
}

// getArtifactKey returns the full S3 key for an artifact
// Path structure: distribution/<ownerID>/<distributionID>/<path>
func getArtifactKey(ownerID, distributionID, path string) string {
	// Clean the path and remove leading slashes
	path = strings.TrimPrefix(path, "/")
	return fmt.Sprintf("distribution/%s/%s/%s", ownerID, distributionID, path)
}

// checkDistributionAccess verifies the user has access to the distribution
// Returns the distribution info if access is granted, nil and error response if not
func (a *API) checkDistributionAccess(c *gin.Context, distID string) (*struct {
	OwnerID string
	ID      string
}, bool) {
	// Get user context (may be nil for anonymous)
	claims := a.getTokenClaims(c)
	var userID string
	var isAdmin bool
	if claims != nil {
		userID = claims.UserID
		isAdmin = claims.HasAdminAccess()
	}

	// Check if user can access this distribution
	canAccess, err := a.distRepo.CanUserAccess(distID, userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return nil, false
	}
	if !canAccess {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Distribution not found",
		})
		return nil, false
	}

	// Fetch the distribution to get OwnerID for artifact path construction
	dist, err := a.distRepo.GetByID(distID)
	if err != nil || dist == nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
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

// handleUploadArtifact uploads an artifact file for a distribution
func (a *API) handleUploadArtifact(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	// Get distribution ID from URL parameter
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Verify distribution exists
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

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "No file provided: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Determine artifact path
	artifactPath := c.PostForm("path")
	if artifactPath == "" {
		artifactPath = header.Filename
	}

	// Generate S3 key using owner ID and distribution ID
	key := getArtifactKey(dist.OwnerID, dist.ID, artifactPath)

	// Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload to S3
	if err := a.storage.Upload(c.Request.Context(), key, file, header.Size, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to upload artifact: " + err.Error(),
		})
		return
	}

	// Log the upload
	a.distRepo.AddLog(distID, "info", fmt.Sprintf("Artifact uploaded: %s (%d bytes)", artifactPath, header.Size))

	c.JSON(http.StatusCreated, ArtifactUploadResponse{
		Key:     key,
		Size:    header.Size,
		Message: "Artifact uploaded successfully",
	})
}

// handleListArtifacts lists all artifacts for a distribution (requires access)
func (a *API) handleListArtifacts(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	// Get distribution ID from URL parameter
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Verify user has access to distribution
	distInfo, ok := a.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	// List artifacts using owner ID and distribution ID
	prefix := getArtifactPrefix(distInfo.OwnerID, distInfo.ID)
	artifacts, err := a.storage.List(c.Request.Context(), prefix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to list artifacts: " + err.Error(),
		})
		return
	}

	// Strip prefix from keys for cleaner response
	for i := range artifacts {
		artifacts[i].Key = strings.TrimPrefix(artifacts[i].Key, prefix)
	}

	c.JSON(http.StatusOK, ArtifactListResponse{
		DistributionID: distID,
		Count:          len(artifacts),
		Artifacts:      artifacts,
	})
}

// handleDownloadArtifact downloads an artifact file from a distribution (requires access)
func (a *API) handleDownloadArtifact(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	// Get distribution ID from URL parameter
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Verify user has access to distribution
	distInfo, ok := a.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	// Get artifact path
	artifactPath := c.Param("path")
	if artifactPath == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Artifact path required",
		})
		return
	}

	// Generate S3 key using owner ID and distribution ID
	key := getArtifactKey(distInfo.OwnerID, distInfo.ID, artifactPath)

	// Download from S3
	reader, info, err := a.storage.Download(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Artifact not found: " + err.Error(),
		})
		return
	}
	defer reader.Close()

	// Set headers
	filename := filepath.Base(artifactPath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("ETag", info.ETag)

	// Stream the file
	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}

// handleDeleteArtifact deletes an artifact from a distribution
func (a *API) handleDeleteArtifact(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	// Get distribution ID from URL parameter
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Verify user has access to distribution and get distribution info
	distInfo, ok := a.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	// Get artifact path
	artifactPath := c.Param("path")
	if artifactPath == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Artifact path required",
		})
		return
	}

	// Generate S3 key using owner ID and distribution ID
	key := getArtifactKey(distInfo.OwnerID, distInfo.ID, artifactPath)

	// Delete from S3
	if err := a.storage.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to delete artifact: " + err.Error(),
		})
		return
	}

	// Log the deletion
	a.distRepo.AddLog(distID, "info", fmt.Sprintf("Artifact deleted: %s", artifactPath))

	c.Status(http.StatusNoContent)
}

// handleGetArtifactURL generates a presigned URL for direct artifact download (requires access)
func (a *API) handleGetArtifactURL(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
		})
		return
	}

	// Get distribution ID from URL parameter
	distID := c.Param("id")
	if distID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Distribution ID required",
		})
		return
	}

	// Verify user has access to distribution
	distInfo, ok := a.checkDistributionAccess(c, distID)
	if !ok {
		return
	}

	// Get artifact path
	artifactPath := c.Param("path")
	if artifactPath == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Artifact path required",
		})
		return
	}

	// Parse expiry
	expiry := 3600 // Default 1 hour
	if expiryStr := c.Query("expiry"); expiryStr != "" {
		if e, err := strconv.Atoi(expiryStr); err == nil && e > 0 {
			expiry = e
		}
	}

	// Generate S3 key using owner ID and distribution ID
	key := getArtifactKey(distInfo.OwnerID, distInfo.ID, artifactPath)

	// Check if artifact exists
	exists, err := a.storage.Exists(c.Request.Context(), key)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: "Artifact not found",
		})
		return
	}

	// Generate presigned URL
	url, err := a.storage.GetPresignedURL(c.Request.Context(), key, time.Duration(expiry)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to generate presigned URL: " + err.Error(),
		})
		return
	}

	// Get web URL for direct access via web gateway (e.g., GarageHQ)
	webURL := a.storage.GetWebURL(key)

	c.JSON(http.StatusOK, ArtifactURLResponse{
		URL:       url,
		WebURL:    webURL,
		ExpiresAt: time.Now().Add(time.Duration(expiry) * time.Second),
	})
}

// handleListAllArtifacts lists all artifacts across all accessible distributions
// GET /v1/artifacts
func (a *API) handleListAllArtifacts(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage service not configured",
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

	// Get all distributions the user can access (any status)
	distributions, err := a.distRepo.ListAccessible(userID, isAdmin, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Build a map of distribution ID to distribution info for quick lookup
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

	// List all artifacts from storage with "distribution/" prefix
	allStorageArtifacts, err := a.storage.List(c.Request.Context(), "distribution/")
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: "Failed to list artifacts: " + err.Error(),
		})
		return
	}

	// Filter artifacts to only include those from accessible distributions
	var artifacts []GlobalArtifact
	for _, obj := range allStorageArtifacts {
		// Parse distribution info from key (format: distribution/{ownerID}/{distributionID}/...)
		parts := strings.SplitN(obj.Key, "/", 4)
		if len(parts) < 4 {
			continue // Invalid key format
		}

		ownerID := parts[1]
		distID := parts[2]

		// Check if user has access to this distribution
		distInfo, hasAccess := distMap[distID]
		if !hasAccess {
			continue
		}

		// Build the artifact entry
		artifactKey := parts[3] // The path within the distribution
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

// handleStorageStatus returns the status of the storage backend
func (a *API) handleStorageStatus(c *gin.Context) {
	if a.storage == nil {
		c.JSON(http.StatusOK, StorageStatusResponse{
			Available: false,
			Message:   "Storage service not configured",
		})
		return
	}

	// Ping storage
	err := a.storage.Ping(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusOK, StorageStatusResponse{
			Available: false,
			Type:      a.storage.Type(),
			Location:  a.storage.Location(),
			Message:   "Storage unreachable: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, StorageStatusResponse{
		Available: true,
		Type:      a.storage.Type(),
		Location:  a.storage.Location(),
		Message:   "Storage is operational",
	})
}
