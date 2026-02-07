package build

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	urlpath "path"
	"path/filepath"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
	"github.com/ulikunitz/xz"
)

// PrepareStage creates the build workspace and extracts component sources
type PrepareStage struct {
	storage storage.Backend
}

// NewPrepareStage creates a new prepare stage
func NewPrepareStage(storage storage.Backend) *PrepareStage {
	return &PrepareStage{
		storage: storage,
	}
}

// Name returns the stage name
func (s *PrepareStage) Name() db.BuildStageName {
	return db.StagePrepare
}

// Validate checks whether this stage can run
func (s *PrepareStage) Validate(ctx context.Context, sc *StageContext) error {
	if len(sc.Components) == 0 {
		return fmt.Errorf("no components resolved")
	}
	if sc.WorkspacePath == "" {
		return fmt.Errorf("workspace path not set")
	}
	return nil
}

// Execute creates workspace directories and extracts sources
func (s *PrepareStage) Execute(ctx context.Context, sc *StageContext, progress ProgressFunc) error {
	progress(0, "Creating workspace directories")

	// Create workspace directory structure
	dirs := []string{
		sc.SourcesDir, // Downloaded source archives
		sc.RootfsDir,  // Root filesystem being assembled
		sc.OutputDir,  // Final output artifacts
		sc.ConfigDir,  // Generated configs
		filepath.Join(sc.WorkspacePath, "workspace"), // Extracted sources for building
		filepath.Join(sc.WorkspacePath, "logs"),      // Build logs
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	progress(10, "Downloading and extracting component sources")

	// Download and extract each component
	totalComponents := len(sc.Components)
	for i := range sc.Components {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rc := &sc.Components[i]
		basePct := 10 + (70 * i / totalComponents)
		progress(basePct, fmt.Sprintf("Extracting: %s v%s", rc.Component.Name, rc.Version))

		// Download artifact from storage to sources dir
		localArchive := filepath.Join(sc.SourcesDir, urlpath.Base(rc.ArtifactPath))
		if err := s.downloadArtifact(ctx, rc.ArtifactPath, localArchive); err != nil {
			return fmt.Errorf("failed to download %s: %w", rc.Component.Name, err)
		}

		// Determine extraction directory
		extractDir := filepath.Join(sc.WorkspacePath, "workspace", rc.Component.Name)
		if err := os.MkdirAll(extractDir, 0755); err != nil {
			return fmt.Errorf("failed to create extract dir for %s: %w", rc.Component.Name, err)
		}

		// Extract the archive
		if err := s.extractArchive(ctx, localArchive, extractDir); err != nil {
			return fmt.Errorf("failed to extract %s: %w", rc.Component.Name, err)
		}

		// Find the actual source directory (often archives have a top-level dir)
		sourceDir, err := s.findSourceDir(extractDir)
		if err != nil {
			return fmt.Errorf("failed to locate source dir for %s: %w", rc.Component.Name, err)
		}

		rc.LocalPath = sourceDir
		log.Info("Extracted component",
			"component", rc.Component.Name,
			"version", rc.Version,
			"path", sourceDir)
	}

	progress(85, "Generating build scripts")

	// Generate build scripts for container execution
	if err := s.generateBuildScripts(sc); err != nil {
		return fmt.Errorf("failed to generate build scripts: %w", err)
	}

	progress(100, "Workspace prepared")
	return nil
}

// downloadArtifact downloads an artifact from storage to local path
func (s *PrepareStage) downloadArtifact(ctx context.Context, artifactPath, localPath string) error {
	if s.storage == nil {
		return fmt.Errorf("storage backend not configured")
	}

	reader, _, err := s.storage.Download(ctx, artifactPath)
	if err != nil {
		return fmt.Errorf("failed to get artifact from storage: %w", err)
	}
	defer reader.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write artifact: %w", err)
	}

	return nil
}

// extractArchive extracts a tar archive (optionally compressed) to a directory
func (s *PrepareStage) extractArchive(ctx context.Context, archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle compression based on file extension
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader

	case strings.HasSuffix(archivePath, ".tar.bz2") || strings.HasSuffix(archivePath, ".tbz2"):
		reader = bzip2.NewReader(file)

	case strings.HasSuffix(archivePath, ".tar.xz") || strings.HasSuffix(archivePath, ".txz"):
		xzReader, err := xz.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create xz reader: %w", err)
		}
		reader = xzReader

	case strings.HasSuffix(archivePath, ".tar"):
		// Plain tar, no decompression needed

	default:
		return fmt.Errorf("unsupported archive format: %s", archivePath)
	}

	tarReader := tar.NewReader(reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		// Prevent path traversal attacks
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid tar path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}

		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			linkTarget := filepath.Join(destDir, header.Linkname)
			if err := os.Link(linkTarget, target); err != nil {
				return fmt.Errorf("failed to create hard link: %w", err)
			}
		}
	}

	return nil
}

// findSourceDir finds the actual source directory after extraction
// Many archives have a single top-level directory containing all files
func (s *PrepareStage) findSourceDir(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}

	// If there's exactly one directory entry, use that as the source dir
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractDir, entries[0].Name()), nil
	}

	// Otherwise, the extract dir itself is the source dir
	return extractDir, nil
}

// generateBuildScripts creates shell scripts for container execution
func (s *PrepareStage) generateBuildScripts(sc *StageContext) error {
	scriptsDir := filepath.Join(sc.WorkspacePath, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return err
	}

	// Generate kernel build script
	kernelScript := `#!/bin/bash
set -e

KERNEL_SRC="$1"
CONFIG_FILE="$2"
OUTPUT_DIR="$3"
ARCH="$4"
CROSS_COMPILE="$5"

cd "$KERNEL_SRC"

# Copy config
cp "$CONFIG_FILE" .config

# Update config for any new options
make ARCH="$ARCH" CROSS_COMPILE="$CROSS_COMPILE" olddefconfig

# Build kernel
NPROC=$(nproc)
echo "Building kernel with $NPROC parallel jobs..."

if [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "x86" ]; then
    make ARCH=x86 CROSS_COMPILE="$CROSS_COMPILE" -j$NPROC bzImage modules
else
    make ARCH="$ARCH" CROSS_COMPILE="$CROSS_COMPILE" -j$NPROC Image modules
fi

# Install modules
make ARCH="$ARCH" CROSS_COMPILE="$CROSS_COMPILE" INSTALL_MOD_PATH="$OUTPUT_DIR/modules" modules_install

# Copy kernel image
mkdir -p "$OUTPUT_DIR/boot"
if [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "x86" ]; then
    cp arch/x86/boot/bzImage "$OUTPUT_DIR/boot/vmlinuz"
else
    cp arch/"$ARCH"/boot/Image "$OUTPUT_DIR/boot/vmlinuz"
fi

# Copy System.map
cp System.map "$OUTPUT_DIR/boot/"

echo "Kernel build complete"
`

	kernelScriptPath := filepath.Join(scriptsDir, "build-kernel.sh")
	if err := os.WriteFile(kernelScriptPath, []byte(kernelScript), 0755); err != nil {
		return fmt.Errorf("failed to write kernel build script: %w", err)
	}

	return nil
}
