package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// PackageStage creates the final distributable image
type PackageStage struct {
	executor Executor
	storage  storage.Backend
	sizeGB   int
}

// NewPackageStage creates a new package stage
func NewPackageStage(executor Executor, storage storage.Backend, sizeGB int) *PackageStage {
	if sizeGB <= 0 {
		sizeGB = 4
	}
	return &PackageStage{
		executor: executor,
		storage:  storage,
		sizeGB:   sizeGB,
	}
}

// Name returns the stage name
func (s *PackageStage) Name() db.BuildStageName {
	return db.StagePackage
}

// Validate checks whether this stage can run
func (s *PackageStage) Validate(ctx context.Context, sc *StageContext) error {
	if sc.RootfsDir == "" {
		return fmt.Errorf("rootfs directory not set")
	}
	if sc.OutputDir == "" {
		return fmt.Errorf("output directory not set")
	}

	// Verify rootfs exists
	if _, err := os.Stat(sc.RootfsDir); os.IsNotExist(err) {
		return fmt.Errorf("rootfs directory does not exist: %s", sc.RootfsDir)
	}

	// Verify rootfs has essential content
	essentials := []string{"boot", "etc", "usr"}
	for _, dir := range essentials {
		path := filepath.Join(sc.RootfsDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("rootfs missing essential directory: %s", dir)
		}
	}

	return nil
}

// Execute creates the final image and uploads it to storage
func (s *PackageStage) Execute(ctx context.Context, sc *StageContext, progress ProgressFunc) error {
	progress(0, "Starting image packaging")

	// Ensure output directory exists
	if err := os.MkdirAll(sc.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get the appropriate image generator
	generator := GetImageGenerator(sc.ImageFormat, s.executor, s.sizeGB)
	log.Info("Using image generator",
		"format", sc.ImageFormat,
		"generator", generator.Name(),
		"size_gb", s.sizeGB,
	)

	progress(5, fmt.Sprintf("Creating %s image", generator.Name()))

	// Generate the image (scaled progress 5-70%)
	genProgress := func(percent int, msg string) {
		scaledPercent := 5 + int(float64(percent)*0.65)
		progress(scaledPercent, msg)
	}

	imagePath, err := generator.Generate(ctx, sc, genProgress)
	if err != nil {
		return fmt.Errorf("image generation failed: %w", err)
	}

	log.Info("Image generated", "path", imagePath)

	progress(72, "Calculating checksum")

	// Calculate checksum
	checksum, err := CalculateChecksum(imagePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	progress(75, "Getting image size")

	// Get file size
	size, err := GetFileSize(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get image size: %w", err)
	}

	log.Info("Image details",
		"checksum", checksum,
		"size_bytes", size,
		"size_mb", size/(1024*1024),
	)

	progress(78, "Uploading to storage")

	// Build storage key
	// Format: distribution/{owner_id}/{dist_id}/builds/{build_id}/{filename}
	filename := filepath.Base(imagePath)
	storageKey := fmt.Sprintf("distribution/%s/%s/builds/%s/%s",
		sc.OwnerID,
		sc.DistributionID,
		sc.BuildID,
		filename,
	)

	// Upload to storage
	if err := s.uploadToStorage(ctx, imagePath, storageKey, progress); err != nil {
		return fmt.Errorf("failed to upload to storage: %w", err)
	}

	progress(95, "Writing checksum file")

	// Write checksum file alongside image
	checksumPath := imagePath + ".sha256"
	checksumContent := fmt.Sprintf("%s  %s\n", checksum, filename)
	if err := os.WriteFile(checksumPath, []byte(checksumContent), 0644); err != nil {
		log.Warn("Failed to write checksum file", "error", err)
	} else {
		// Upload checksum file too
		checksumKey := storageKey + ".sha256"
		if err := s.uploadToStorage(ctx, checksumPath, checksumKey, nil); err != nil {
			log.Warn("Failed to upload checksum file", "error", err)
		}
	}

	// Store artifact info in context for worker to update DB
	sc.ArtifactPath = storageKey
	sc.ArtifactChecksum = checksum
	sc.ArtifactSize = size

	progress(100, fmt.Sprintf("Packaging complete: %s (%d MB)", filename, size/(1024*1024)))
	return nil
}

// uploadToStorage uploads a file to the storage backend
func (s *PackageStage) uploadToStorage(ctx context.Context, localPath, storageKey string, progress ProgressFunc) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Determine content type based on extension
	contentType := "application/octet-stream"
	ext := filepath.Ext(localPath)
	switch ext {
	case ".iso":
		contentType = "application/x-iso9660-image"
	case ".qcow2":
		contentType = "application/x-qemu-disk"
	case ".img":
		contentType = "application/x-raw-disk-image"
	case ".sha256":
		contentType = "text/plain"
	}

	log.Info("Uploading artifact",
		"local_path", localPath,
		"storage_key", storageKey,
		"size", stat.Size(),
		"content_type", contentType,
	)

	return s.storage.Upload(ctx, storageKey, file, stat.Size(), contentType)
}

// ArtifactInfo holds information about the generated artifact
// These fields are added to StageContext to pass back to worker
type ArtifactInfo struct {
	Path     string // Storage key
	Checksum string // SHA256 checksum
	Size     int64  // Size in bytes
}
