package tests

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// =============================================================================
// Storage Factory Tests
// =============================================================================

func TestStorage_New_Local(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := storage.Config{
		Type: "local",
		Local: storage.LocalConfig{
			BasePath: tmpDir,
		},
	}

	backend, err := storage.New(cfg)
	if err != nil {
		t.Fatalf("failed to create local storage: %v", err)
	}

	if backend.Type() != "local" {
		t.Fatalf("expected type 'local', got '%s'", backend.Type())
	}
}

func TestStorage_New_DefaultsToLocal(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := storage.Config{
		Type: "", // Empty should default to local
		Local: storage.LocalConfig{
			BasePath: tmpDir,
		},
	}

	backend, err := storage.New(cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	if backend.Type() != "local" {
		t.Fatalf("expected type 'local', got '%s'", backend.Type())
	}
}

func TestStorage_DefaultConfig(t *testing.T) {
	cfg := storage.DefaultConfig()

	if cfg.Type != "local" {
		t.Fatalf("expected default type 'local', got '%s'", cfg.Type)
	}
	if cfg.Local.BasePath == "" {
		t.Fatal("expected default base path to be set")
	}
}

// =============================================================================
// Local Backend Tests
// =============================================================================

func setupLocalBackend(t *testing.T) (*storage.LocalBackend, string) {
	t.Helper()

	tmpDir := t.TempDir()
	backend, err := storage.NewLocal(storage.LocalConfig{
		BasePath: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create local backend: %v", err)
	}

	return backend, tmpDir
}

func TestLocalBackend_Upload(t *testing.T) {
	backend, tmpDir := setupLocalBackend(t)
	ctx := context.Background()

	content := []byte("test content")
	reader := bytes.NewReader(content)

	err := backend.Upload(ctx, "test/file.txt", reader, int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	// Verify file exists on disk
	filePath := filepath.Join(tmpDir, "test", "file.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("uploaded file should exist on disk")
	}

	// Verify content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("content mismatch: expected '%s', got '%s'", content, data)
	}
}

func TestLocalBackend_Upload_CreatesDirectories(t *testing.T) {
	backend, tmpDir := setupLocalBackend(t)
	ctx := context.Background()

	content := []byte("nested content")
	reader := bytes.NewReader(content)

	err := backend.Upload(ctx, "deep/nested/path/file.txt", reader, int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	filePath := filepath.Join(tmpDir, "deep", "nested", "path", "file.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("nested file should exist")
	}
}

func TestLocalBackend_Upload_SizeMismatch(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	content := []byte("test content")
	reader := bytes.NewReader(content)

	// Claim wrong size
	err := backend.Upload(ctx, "test.txt", reader, int64(len(content)+100), "text/plain")
	if err == nil {
		t.Fatal("expected error for size mismatch")
	}
}

func TestLocalBackend_Download(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	// Upload first
	content := []byte("download test content")
	backend.Upload(ctx, "download.txt", bytes.NewReader(content), int64(len(content)), "text/plain")

	// Download
	reader, info, err := backend.Download(ctx, "download.txt")
	if err != nil {
		t.Fatalf("failed to download: %v", err)
	}
	defer reader.Close()

	// Read content
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Fatalf("content mismatch: expected '%s', got '%s'", content, data)
	}

	// Check info
	if info.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.Size)
	}
}

func TestLocalBackend_Download_NotFound(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	_, _, err := backend.Download(ctx, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

func TestLocalBackend_Delete(t *testing.T) {
	backend, tmpDir := setupLocalBackend(t)
	ctx := context.Background()

	// Upload first
	content := []byte("delete test")
	backend.Upload(ctx, "todelete.txt", bytes.NewReader(content), int64(len(content)), "text/plain")

	// Verify exists
	if exists, _ := backend.Exists(ctx, "todelete.txt"); !exists {
		t.Fatal("file should exist before delete")
	}

	// Delete
	if err := backend.Delete(ctx, "todelete.txt"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	// Verify gone
	if exists, _ := backend.Exists(ctx, "todelete.txt"); exists {
		t.Fatal("file should not exist after delete")
	}

	// Verify file is gone from disk
	filePath := filepath.Join(tmpDir, "todelete.txt")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatal("file should not exist on disk after delete")
	}
}

func TestLocalBackend_Delete_NonExistent(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	// Deleting non-existent file should not error
	err := backend.Delete(ctx, "nonexistent.txt")
	if err != nil {
		t.Fatalf("deleting non-existent file should not error: %v", err)
	}
}

func TestLocalBackend_Delete_CleansEmptyDirs(t *testing.T) {
	backend, tmpDir := setupLocalBackend(t)
	ctx := context.Background()

	// Upload to nested path
	content := []byte("nested content")
	backend.Upload(ctx, "a/b/c/file.txt", bytes.NewReader(content), int64(len(content)), "text/plain")

	// Delete the file
	backend.Delete(ctx, "a/b/c/file.txt")

	// Empty parent directories should be cleaned up
	dirPath := filepath.Join(tmpDir, "a", "b", "c")
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Fatal("empty nested directories should be removed")
	}
}

func TestLocalBackend_Exists(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	// Check non-existent
	exists, err := backend.Exists(ctx, "nonexistent.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("file should not exist")
	}

	// Upload and check
	content := []byte("exists test")
	backend.Upload(ctx, "exists.txt", bytes.NewReader(content), int64(len(content)), "text/plain")

	exists, err = backend.Exists(ctx, "exists.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("file should exist")
	}
}

func TestLocalBackend_GetInfo(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	content := []byte("info test content")
	backend.Upload(ctx, "info.txt", bytes.NewReader(content), int64(len(content)), "text/plain")

	info, err := backend.GetInfo(ctx, "info.txt")
	if err != nil {
		t.Fatalf("failed to get info: %v", err)
	}

	if info.Key != "info.txt" {
		t.Fatalf("expected key 'info.txt', got '%s'", info.Key)
	}
	if info.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.Size)
	}
	if !strings.HasPrefix(info.ContentType, "text/plain") {
		t.Fatalf("expected content type starting with 'text/plain', got '%s'", info.ContentType)
	}
	if info.ETag == "" {
		t.Fatal("ETag should not be empty")
	}
	if info.LastModified.IsZero() {
		t.Fatal("LastModified should be set")
	}
}

func TestLocalBackend_GetInfo_NotFound(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	_, err := backend.GetInfo(ctx, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLocalBackend_List(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	// Upload multiple files
	files := []string{
		"file1.txt",
		"file2.txt",
		"subdir/file3.txt",
		"subdir/file4.txt",
		"other/file5.txt",
	}

	for _, f := range files {
		content := []byte("content of " + f)
		backend.Upload(ctx, f, bytes.NewReader(content), int64(len(content)), "text/plain")
	}

	// List all
	objects, err := backend.List(ctx, "")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(objects) != 5 {
		t.Fatalf("expected 5 objects, got %d", len(objects))
	}

	// List with prefix
	objects, err = backend.List(ctx, "subdir/")
	if err != nil {
		t.Fatalf("failed to list with prefix: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects in subdir, got %d", len(objects))
	}
}

func TestLocalBackend_GetPresignedURL(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	_, err := backend.GetPresignedURL(ctx, "file.txt", time.Hour)
	if err == nil {
		t.Fatal("local backend should not support presigned URLs")
	}
}

func TestLocalBackend_GetWebURL(t *testing.T) {
	backend, _ := setupLocalBackend(t)

	url := backend.GetWebURL("file.txt")
	if url != "" {
		t.Fatal("local backend should return empty web URL")
	}
}

func TestLocalBackend_Ping(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	if err := backend.Ping(ctx); err != nil {
		t.Fatalf("ping should succeed: %v", err)
	}
}

func TestLocalBackend_Type(t *testing.T) {
	backend, _ := setupLocalBackend(t)

	if backend.Type() != "local" {
		t.Fatalf("expected type 'local', got '%s'", backend.Type())
	}
}

func TestLocalBackend_Location(t *testing.T) {
	backend, tmpDir := setupLocalBackend(t)

	if backend.Location() != tmpDir {
		t.Fatalf("expected location '%s', got '%s'", tmpDir, backend.Location())
	}
}

// =============================================================================
// Security Tests
// =============================================================================

func TestLocalBackend_PathTraversal(t *testing.T) {
	backend, tmpDir := setupLocalBackend(t)
	ctx := context.Background()

	// Create a file that would exist if path traversal succeeded
	parentDir := filepath.Dir(tmpDir)
	escapeTarget := filepath.Join(parentDir, "escape.txt")

	// Ensure the escape target doesn't exist before the test
	os.Remove(escapeTarget)
	defer os.Remove(escapeTarget) // Clean up just in case

	// Try to escape base directory
	maliciousKeys := []string{
		"../escape.txt",
		"../../escape.txt",
		"subdir/../../escape.txt",
	}

	for _, key := range maliciousKeys {
		content := []byte("malicious content")
		err := backend.Upload(ctx, key, bytes.NewReader(content), int64(len(content)), "text/plain")

		// Regardless of whether upload returns error, verify no file escaped
		if _, statErr := os.Stat(escapeTarget); statErr == nil {
			t.Fatalf("path traversal succeeded with key '%s': file created at %s", key, escapeTarget)
		}

		// If upload succeeded, the file should be safely contained in tmpDir
		if err == nil {
			// Verify file exists somewhere in tmpDir (sanitized path)
			exists, _ := backend.Exists(ctx, key)
			if !exists {
				// Try the base filename as fallback check
				exists, _ = backend.Exists(ctx, filepath.Base(key))
			}
			// It's okay if exists is false - the backend might reject the key
		}
	}
}

// =============================================================================
// ObjectInfo Tests
// =============================================================================

func TestObjectInfo_Fields(t *testing.T) {
	info := storage.ObjectInfo{
		Key:          "test/file.txt",
		Size:         1024,
		ContentType:  "text/plain",
		ETag:         "\"abc123\"",
		LastModified: time.Now(),
	}

	if info.Key != "test/file.txt" {
		t.Fatalf("expected key 'test/file.txt', got '%s'", info.Key)
	}
	if info.Size != 1024 {
		t.Fatalf("expected size 1024, got %d", info.Size)
	}
	if info.ContentType != "text/plain" {
		t.Fatalf("expected content type 'text/plain', got '%s'", info.ContentType)
	}
}

// =============================================================================
// Content Type Detection Tests
// =============================================================================

func TestLocalBackend_ContentTypeDetection(t *testing.T) {
	backend, _ := setupLocalBackend(t)
	ctx := context.Background()

	tests := []struct {
		filename    string
		expected    string
		shouldMatch bool
	}{
		{"file.txt", "text/plain", true},
		{"file.html", "text/html", true},
		{"file.json", "application/json", true},
		{"file.unknown", "application/octet-stream", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			content := []byte("test")
			backend.Upload(ctx, tt.filename, bytes.NewReader(content), int64(len(content)), "")

			info, err := backend.GetInfo(ctx, tt.filename)
			if err != nil {
				t.Fatalf("failed to get info: %v", err)
			}

			if tt.shouldMatch && !strings.HasPrefix(info.ContentType, strings.Split(tt.expected, ";")[0]) {
				t.Fatalf("expected content type starting with '%s', got '%s'", tt.expected, info.ContentType)
			}
		})
	}
}
