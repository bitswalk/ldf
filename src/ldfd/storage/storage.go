// Package storage provides storage backends for ldfd distribution artifacts.
package storage

import (
	"context"
	"io"
	"time"
)

// Backend defines the interface for storage backends
type Backend interface {
	// Upload uploads data to storage
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error

	// Download downloads an object from storage
	Download(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)

	// Delete deletes an object from storage
	Delete(ctx context.Context, key string) error

	// Exists checks if an object exists
	Exists(ctx context.Context, key string) (bool, error)

	// GetInfo retrieves metadata for an object
	GetInfo(ctx context.Context, key string) (*ObjectInfo, error)

	// Copy copies an object from srcKey to dstKey within the same backend
	Copy(ctx context.Context, srcKey, dstKey string) error

	// List lists objects with the given prefix
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)

	// GetPresignedURL generates a presigned URL for downloading (may not be supported by all backends)
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)

	// GetWebURL returns a direct web URL for accessing an artifact via the web gateway
	// For S3-compatible storage with separate web endpoints (like GarageHQ)
	GetWebURL(key string) string

	// Ping checks if the storage is accessible
	Ping(ctx context.Context) error

	// Type returns the storage backend type
	Type() string

	// Location returns a human-readable location description
	Location() string
}

// LocalPathResolver is optionally implemented by backends that store objects
// on the local filesystem (e.g., LocalBackend). It allows callers to create
// symlinks to storage objects instead of copying them.
type LocalPathResolver interface {
	// ResolvePath returns the absolute filesystem path for a storage key.
	ResolvePath(key string) string
}

// ObjectInfo holds metadata about a storage object
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified time.Time `json:"last_modified"`
}

// Config holds the storage configuration
type Config struct {
	// Type is the storage backend type: "s3" or "local"
	Type string

	// Local storage configuration
	Local LocalConfig

	// S3 storage configuration
	S3 S3Config
}

// DefaultConfig returns a default storage configuration (local filesystem)
func DefaultConfig() Config {
	return Config{
		Type: "local",
		Local: LocalConfig{
			BasePath: "~/.ldfd/artifacts",
		},
	}
}

// New creates a new storage backend based on configuration
func New(cfg Config) (Backend, error) {
	switch cfg.Type {
	case "s3":
		return NewS3(cfg.S3)
	case "local", "":
		return NewLocal(cfg.Local)
	default:
		return NewLocal(cfg.Local)
	}
}
