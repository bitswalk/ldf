package branding

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/gin-gonic/gin"
)

// NewHandler creates a new branding handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		storage: cfg.Storage,
	}
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
func (h *Handler) findBrandingAsset(ctx context.Context, asset string) (string, error) {
	extensions := []string{".png", ".jpg", ".jpeg", ".webp", ".svg", ".ico"}

	for _, ext := range extensions {
		key := getBrandingStorageKey(asset, ext)
		exists, err := h.storage.Exists(ctx, key)
		if err != nil {
			continue
		}
		if exists {
			return key, nil
		}
	}

	return "", nil
}

// HandleGetAsset retrieves a branding asset (logo or favicon)
func (h *Handler) HandleGetAsset(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		common.BadRequest(c, "Invalid asset type. Must be 'logo' or 'favicon'")
		return
	}

	if h.storage == nil {
		common.ServiceUnavailable(c, "Storage backend not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	key, err := h.findBrandingAsset(ctx, asset)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to check asset: %v", err))
		return
	}

	if key == "" {
		common.NotFound(c, fmt.Sprintf("Branding asset '%s' not found", asset))
		return
	}

	reader, info, err := h.storage.Download(ctx, key)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to download asset: %v", err))
		return
	}
	defer reader.Close()

	contentType := info.ContentType
	if contentType == "" {
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

// HandleGetAssetInfo returns metadata about a branding asset
func (h *Handler) HandleGetAssetInfo(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		common.BadRequest(c, "Invalid asset type. Must be 'logo' or 'favicon'")
		return
	}

	if h.storage == nil {
		common.ServiceUnavailable(c, "Storage backend not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	key, err := h.findBrandingAsset(ctx, asset)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to check asset: %v", err))
		return
	}

	if key == "" {
		c.JSON(http.StatusOK, BrandingAssetInfo{
			Asset:  asset,
			Exists: false,
		})
		return
	}

	info, err := h.storage.GetInfo(ctx, key)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to get asset info: %v", err))
		return
	}

	url := h.storage.GetWebURL(key)

	c.JSON(http.StatusOK, BrandingAssetInfo{
		Asset:       asset,
		URL:         url,
		ContentType: info.ContentType,
		Size:        info.Size,
		Exists:      true,
	})
}

// HandleUploadAsset uploads a branding asset (logo or favicon)
func (h *Handler) HandleUploadAsset(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		common.BadRequest(c, "Invalid asset type. Must be 'logo' or 'favicon'")
		return
	}

	if h.storage == nil {
		common.ServiceUnavailable(c, "Storage backend not configured")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		common.BadRequest(c, "No file provided. Please upload an image file.")
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		common.BadRequest(c, "Failed to read file")
		return
	}
	contentType := http.DetectContentType(buffer[:n])

	file.Seek(0, io.SeekStart)

	allowed := allowedImageTypes[asset]
	isAllowed := false
	for _, t := range allowed {
		if contentType == t {
			isAllowed = true
			break
		}
	}

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
		common.BadRequest(c, fmt.Sprintf("Invalid file type '%s'. Allowed types: %s", contentType, allowedStr))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	existingKey, _ := h.findBrandingAsset(ctx, asset)
	if existingKey != "" {
		h.storage.Delete(ctx, existingKey)
	}

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

	key := getBrandingStorageKey(asset, ext)
	if err := h.storage.Upload(ctx, key, file, header.Size, contentType); err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to upload asset: %v", err))
		return
	}

	url := h.storage.GetWebURL(key)

	c.JSON(http.StatusOK, gin.H{
		"message":      fmt.Sprintf("Branding asset '%s' uploaded successfully", asset),
		"asset":        asset,
		"url":          url,
		"content_type": contentType,
		"size":         header.Size,
	})
}

// HandleDeleteAsset deletes a branding asset
func (h *Handler) HandleDeleteAsset(c *gin.Context) {
	asset := c.Param("asset")

	if !validBrandingAssets[asset] {
		common.BadRequest(c, "Invalid asset type. Must be 'logo' or 'favicon'")
		return
	}

	if h.storage == nil {
		common.ServiceUnavailable(c, "Storage backend not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	key, err := h.findBrandingAsset(ctx, asset)
	if err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to check asset: %v", err))
		return
	}

	if key == "" {
		common.NotFound(c, fmt.Sprintf("Branding asset '%s' not found", asset))
		return
	}

	if err := h.storage.Delete(ctx, key); err != nil {
		common.InternalError(c, fmt.Sprintf("Failed to delete asset: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Branding asset '%s' deleted successfully", asset),
		"asset":   asset,
	})
}
