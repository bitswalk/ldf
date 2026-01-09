package artifacts

import (
	"time"

	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
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
