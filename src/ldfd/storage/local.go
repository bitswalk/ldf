package storage

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitswalk/ldf/src/common/paths"
)

// LocalConfig holds the local filesystem storage configuration
type LocalConfig struct {
	// BasePath is the root directory for storing artifacts
	BasePath string
}

// LocalBackend implements storage on the local filesystem
type LocalBackend struct {
	basePath string
}

// NewLocal creates a new local filesystem storage backend
func NewLocal(cfg LocalConfig) (*LocalBackend, error) {
	// Expand path (handle ~ and env vars)
	basePath := paths.Expand(cfg.BasePath)

	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory %s: %w", basePath, err)
	}

	return &LocalBackend{
		basePath: basePath,
	}, nil
}

// fullPath returns the full filesystem path for a key
func (b *LocalBackend) fullPath(key string) string {
	// Clean the key to prevent directory traversal
	cleanKey := filepath.Clean(key)

	// Remove all leading path separators and parent directory references
	for strings.HasPrefix(cleanKey, "/") || strings.HasPrefix(cleanKey, "../") || strings.HasPrefix(cleanKey, "..\\") {
		cleanKey = strings.TrimPrefix(cleanKey, "/")
		cleanKey = strings.TrimPrefix(cleanKey, "../")
		cleanKey = strings.TrimPrefix(cleanKey, "..\\")
	}

	// Join with base path and verify the result is within basePath
	fullPath := filepath.Join(b.basePath, cleanKey)

	// Final safety check: ensure the path is within basePath
	absBase, _ := filepath.Abs(b.basePath)
	absFull, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFull, absBase) {
		// If somehow we escaped, return base path with sanitized filename only
		return filepath.Join(b.basePath, filepath.Base(cleanKey))
	}

	return fullPath
}

// ResolvePath returns the absolute filesystem path for a storage key.
// This implements the LocalPathResolver interface.
func (b *LocalBackend) ResolvePath(key string) string {
	return b.fullPath(key)
}

// Upload uploads data to local filesystem
func (b *LocalBackend) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	fullPath := b.fullPath(key)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fullPath, err)
	}
	defer file.Close()

	// Copy data
	written, err := io.Copy(file, reader)
	if err != nil {
		os.Remove(fullPath) // Clean up on error
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	if size > 0 && written != size {
		os.Remove(fullPath) // Clean up on error
		return fmt.Errorf("size mismatch: expected %d bytes, wrote %d bytes", size, written)
	}

	return nil
}

// Copy copies a file from srcKey to dstKey using hard link when possible,
// falling back to file copy for cross-device scenarios.
func (b *LocalBackend) Copy(ctx context.Context, srcKey, dstKey string) error {
	srcPath := b.fullPath(srcKey)
	dstPath := b.fullPath(dstKey)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dstKey, err)
	}

	// Try hard link first (zero-copy on same filesystem)
	if err := os.Link(srcPath, dstPath); err == nil {
		return nil
	}

	// Fallback to file copy
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source %s: %w", srcKey, err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination %s: %w", dstKey, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dstPath)
		return fmt.Errorf("failed to copy %s to %s: %w", srcKey, dstKey, err)
	}

	return nil
}

// Download downloads a file from local filesystem
func (b *LocalBackend) Download(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	fullPath := b.fullPath(key)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("object not found: %s", key)
		}
		return nil, nil, fmt.Errorf("failed to open file %s: %w", fullPath, err)
	}

	info, err := b.GetInfo(ctx, key)
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return file, info, nil
}

// Delete deletes a file from local filesystem
func (b *LocalBackend) Delete(ctx context.Context, key string) error {
	fullPath := b.fullPath(key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted, not an error
		}
		return fmt.Errorf("failed to delete file %s: %w", fullPath, err)
	}

	// Try to remove empty parent directories
	b.cleanEmptyDirs(filepath.Dir(fullPath))

	return nil
}

// cleanEmptyDirs removes empty parent directories up to basePath
func (b *LocalBackend) cleanEmptyDirs(dir string) {
	for dir != b.basePath && strings.HasPrefix(dir, b.basePath) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// Exists checks if a file exists
func (b *LocalBackend) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := b.fullPath(key)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat file %s: %w", fullPath, err)
	}
	return true, nil
}

// GetInfo retrieves metadata for a file
func (b *LocalBackend) GetInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	fullPath := b.fullPath(key)

	stat, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat file %s: %w", fullPath, err)
	}

	// Detect content type from extension
	contentType := mime.TypeByExtension(filepath.Ext(key))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Generate ETag from modification time and size
	etag := b.generateETag(stat)

	return &ObjectInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  contentType,
		ETag:         etag,
		LastModified: stat.ModTime(),
	}, nil
}

// generateETag generates an ETag from file stats
func (b *LocalBackend) generateETag(stat os.FileInfo) string {
	data := fmt.Sprintf("%s-%d-%d", stat.Name(), stat.Size(), stat.ModTime().UnixNano())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("\"%s\"", hex.EncodeToString(hash[:]))
}

// List lists files with the given prefix
func (b *LocalBackend) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	searchPath := b.fullPath(prefix)

	// If prefix is a directory, list its contents
	// If prefix is a partial path, find matching files
	err := filepath.Walk(b.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil // Skip directories
		}

		// Get relative key
		relPath, err := filepath.Rel(b.basePath, path)
		if err != nil {
			return nil
		}

		// Check if matches prefix
		if prefix != "" && !strings.HasPrefix(relPath, strings.TrimPrefix(prefix, "/")) {
			return nil
		}

		// Detect content type
		contentType := mime.TypeByExtension(filepath.Ext(relPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		objects = append(objects, ObjectInfo{
			Key:          relPath,
			Size:         info.Size(),
			ContentType:  contentType,
			ETag:         b.generateETag(info),
			LastModified: info.ModTime(),
		})

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list files in %s: %w", searchPath, err)
	}

	return objects, nil
}

// GetPresignedURL is not supported for local filesystem
func (b *LocalBackend) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", fmt.Errorf("presigned URLs not supported for local filesystem storage")
}

// GetWebURL is not supported for local filesystem (no web gateway)
func (b *LocalBackend) GetWebURL(key string) string {
	// Local storage doesn't have a web endpoint
	return ""
}

// Ping checks if the storage directory is accessible
func (b *LocalBackend) Ping(ctx context.Context) error {
	_, err := os.Stat(b.basePath)
	if err != nil {
		return fmt.Errorf("storage directory not accessible: %w", err)
	}
	return nil
}

// Type returns the storage backend type
func (b *LocalBackend) Type() string {
	return "local"
}

// Location returns the base path
func (b *LocalBackend) Location() string {
	return b.basePath
}
