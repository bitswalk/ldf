package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// BrandingAsset represents valid branding asset types
type BrandingAsset string

const (
	BrandingAssetLogo    BrandingAsset = "logo"
	BrandingAssetFavicon BrandingAsset = "favicon"
)

// BrandingAssetInfo represents metadata about a branding asset
type BrandingAssetInfo struct {
	Asset       string `json:"asset"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Exists      bool   `json:"exists"`
}

// validBrandingAssets defines the allowed asset types
var validBrandingAssets = map[string]bool{
	"logo":    true,
	"favicon": true,
}

// allowedImageTypes defines allowed MIME types for branding assets
var allowedImageTypes = map[string][]string{
	"logo": {
		"image/png",
		"image/jpeg",
		"image/webp",
		"image/svg+xml",
	},
	"favicon": {
		"image/png",
		"image/x-icon",
		"image/vnd.microsoft.icon",
		"image/ico",
		"image/svg+xml",
	},
}

// getBrandingStorageKey returns the storage key for a branding asset
func getBrandingStorageKey(asset string, ext string) string {
	return fmt.Sprintf("system/%s%s", asset, ext)
}

// findBrandingAsset searches for an existing branding asset with any supported extension
func (a *API) findBrandingAsset(ctx context.Context, asset string) (string, error) {
	// Check for common extensions
	extensions := []string{".png", ".jpg", ".jpeg", ".webp", ".svg", ".ico"}

	for _, ext := range extensions {
		key := getBrandingStorageKey(asset, ext)
		exists, err := a.storage.Exists(ctx, key)
		if err != nil {
			continue
		}
		if exists {
			return key, nil
		}
	}

	return "", nil
}

// handleGetBrandingAsset retrieves a branding asset (logo or favicon)
// GET /v1/branding/:asset
func (a *API) handleGetBrandingAsset(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid asset type. Must be 'logo' or 'favicon'",
		})
		return
	}

	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage backend not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Find the asset with any extension
	key, err := a.findBrandingAsset(ctx, asset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to check asset: %v", err),
		})
		return
	}

	if key == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Branding asset '%s' not found", asset),
		})
		return
	}

	// Download and stream the asset
	reader, info, err := a.storage.Download(ctx, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to download asset: %v", err),
		})
		return
	}
	defer reader.Close()

	contentType := info.ContentType
	if contentType == "" {
		// Infer from extension
		ext := filepath.Ext(key)
		switch ext {
		case ".png":
			contentType = "image/png"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".webp":
			contentType = "image/webp"
		case ".svg":
			contentType = "image/svg+xml"
		case ".ico":
			contentType = "image/x-icon"
		default:
			contentType = "application/octet-stream"
		}
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}

// handleGetBrandingAssetInfo returns metadata about a branding asset
// GET /v1/branding/:asset/info
func (a *API) handleGetBrandingAssetInfo(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid asset type. Must be 'logo' or 'favicon'",
		})
		return
	}

	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage backend not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Find the asset
	key, err := a.findBrandingAsset(ctx, asset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to check asset: %v", err),
		})
		return
	}

	if key == "" {
		c.JSON(http.StatusOK, BrandingAssetInfo{
			Asset:  asset,
			Exists: false,
		})
		return
	}

	// Get asset info
	info, err := a.storage.GetInfo(ctx, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to get asset info: %v", err),
		})
		return
	}

	// Get web URL
	url := a.storage.GetWebURL(key)

	c.JSON(http.StatusOK, BrandingAssetInfo{
		Asset:       asset,
		URL:         url,
		ContentType: info.ContentType,
		Size:        info.Size,
		Exists:      true,
	})
}

// handleUploadBrandingAsset uploads a branding asset (logo or favicon)
// POST /v1/branding/:asset
func (a *API) handleUploadBrandingAsset(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid asset type. Must be 'logo' or 'favicon'",
		})
		return
	}

	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage backend not configured",
		})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "No file provided. Please upload an image file.",
		})
		return
	}
	defer file.Close()

	// Detect content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Failed to read file",
		})
		return
	}
	contentType := http.DetectContentType(buffer[:n])

	// Reset file reader
	file.Seek(0, io.SeekStart)

	// Validate content type
	allowed := allowedImageTypes[asset]
	isAllowed := false
	for _, t := range allowed {
		if contentType == t {
			isAllowed = true
			break
		}
	}

	// Also check by extension if content type detection fails
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isAllowed {
		switch ext {
		case ".png":
			contentType = "image/png"
			isAllowed = true
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
			isAllowed = true
		case ".webp":
			contentType = "image/webp"
			isAllowed = true
		case ".svg":
			contentType = "image/svg+xml"
			isAllowed = true
		case ".ico":
			if asset == "favicon" {
				contentType = "image/x-icon"
				isAllowed = true
			}
		}
	}

	if !isAllowed {
		allowedStr := strings.Join(allowed, ", ")
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Invalid file type '%s'. Allowed types: %s", contentType, allowedStr),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Delete any existing asset with different extension
	existingKey, _ := a.findBrandingAsset(ctx, asset)
	if existingKey != "" {
		a.storage.Delete(ctx, existingKey)
	}

	// Determine extension from content type
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	case "image/svg+xml":
		ext = ".svg"
	case "image/x-icon", "image/vnd.microsoft.icon", "image/ico":
		ext = ".ico"
	}

	// Upload the asset
	key := getBrandingStorageKey(asset, ext)
	if err := a.storage.Upload(ctx, key, file, header.Size, contentType); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to upload asset: %v", err),
		})
		return
	}

	// Get web URL
	url := a.storage.GetWebURL(key)

	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("Branding asset '%s' uploaded successfully", asset),
		"asset":        asset,
		"url":          url,
		"content_type": contentType,
		"size":         header.Size,
	})
}

// handleDeleteBrandingAsset deletes a branding asset
// DELETE /v1/branding/:asset
func (a *API) handleDeleteBrandingAsset(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Bad request",
			Code:    http.StatusBadRequest,
			Message: "Invalid asset type. Must be 'logo' or 'favicon'",
		})
		return
	}

	if a.storage == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service unavailable",
			Code:    http.StatusServiceUnavailable,
			Message: "Storage backend not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Find the asset
	key, err := a.findBrandingAsset(ctx, asset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to check asset: %v", err),
		})
		return
	}

	if key == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Not found",
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Branding asset '%s' not found", asset),
		})
		return
	}

	// Delete the asset
	if err := a.storage.Delete(ctx, key); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to delete asset: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Branding asset '%s' deleted successfully", asset),
		"asset":   asset,
	})
}
