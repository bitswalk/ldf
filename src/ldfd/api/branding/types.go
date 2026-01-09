package branding

import (
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// Handler handles branding-related HTTP requests
type Handler struct {
	storage storage.Backend
}

// Config contains configuration options for the Handler
type Config struct {
	Storage storage.Backend
}

// BrandingAssetInfo represents metadata about a branding asset
type BrandingAssetInfo struct {
	Asset       string `json:"asset"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Exists      bool   `json:"exists"`
}
