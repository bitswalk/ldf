package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// InitramfsGenerator creates initramfs images
type InitramfsGenerator struct {
	rootfsPath string
	outputPath string
	config     *db.DistributionConfig
	targetArch db.TargetArch
}

// NewInitramfsGenerator creates a new initramfs generator
func NewInitramfsGenerator(rootfsPath, outputPath string, config *db.DistributionConfig, arch db.TargetArch) *InitramfsGenerator {
	return &InitramfsGenerator{
		rootfsPath: rootfsPath,
		outputPath: outputPath,
		config:     config,
		targetArch: arch,
	}
}

// Generate creates the initramfs image
func (g *InitramfsGenerator) Generate() error {
	// Create temporary directory for initramfs contents
	initramfsDir, err := os.MkdirTemp("", "ldf-initramfs-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(initramfsDir)

	// Create initramfs directory structure
	if err := g.createStructure(initramfsDir); err != nil {
		return fmt.Errorf("failed to create initramfs structure: %w", err)
	}

	// Copy required kernel modules
	if err := g.copyModules(initramfsDir); err != nil {
		return fmt.Errorf("failed to copy modules: %w", err)
	}

	// Copy essential binaries
	if err := g.copyBinaries(initramfsDir); err != nil {
		return fmt.Errorf("failed to copy binaries: %w", err)
	}

	// Generate init script
	if err := g.generateInit(initramfsDir); err != nil {
		return fmt.Errorf("failed to generate init: %w", err)
	}

	// Generate the cpio archive script
	if err := g.generatePackScript(initramfsDir); err != nil {
		return fmt.Errorf("failed to generate pack script: %w", err)
	}

	log.Info("Generated initramfs contents", "dir", initramfsDir)
	return nil
}

// createStructure creates the initramfs directory structure
func (g *InitramfsGenerator) createStructure(initramfsDir string) error {
	dirs := []string{
		"/bin",
		"/sbin",
		"/etc",
		"/lib",
		"/lib/modules",
		"/lib64",
		"/dev",
		"/proc",
		"/sys",
		"/run",
		"/mnt",
		"/mnt/root",
		"/usr",
		"/usr/bin",
		"/usr/sbin",
		"/usr/lib",
		"/tmp",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(initramfsDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// copyModules copies required kernel modules to initramfs
func (g *InitramfsGenerator) copyModules(initramfsDir string) error {
	// Find kernel modules directory
	modulesBase := filepath.Join(g.rootfsPath, "lib", "modules")
	entries, err := os.ReadDir(modulesBase)
	if err != nil {
		log.Warn("No kernel modules found", "error", err)
		return nil
	}

	// Find the kernel version directory
	var kernelVersion string
	for _, entry := range entries {
		if entry.IsDir() {
			kernelVersion = entry.Name()
			break
		}
	}

	if kernelVersion == "" {
		log.Warn("No kernel version directory found")
		return nil
	}

	srcModulesDir := filepath.Join(modulesBase, kernelVersion)
	dstModulesDir := filepath.Join(initramfsDir, "lib", "modules", kernelVersion)

	// Modules needed for root filesystem access
	requiredModules := g.getRequiredModules()

	// Create modules directory
	if err := os.MkdirAll(dstModulesDir, 0755); err != nil {
		return err
	}

	// Copy required modules
	for _, modPattern := range requiredModules {
		if err := g.copyModulesByPattern(srcModulesDir, dstModulesDir, modPattern); err != nil {
			log.Warn("Could not copy module pattern", "pattern", modPattern, "error", err)
		}
	}

	// Copy modules.dep and related files
	depFiles := []string{"modules.dep", "modules.dep.bin", "modules.alias", "modules.alias.bin"}
	for _, df := range depFiles {
		src := filepath.Join(srcModulesDir, df)
		dst := filepath.Join(dstModulesDir, df)
		if err := copyFile(src, dst); err != nil {
			log.Warn("Could not copy modules file", "file", df, "error", err)
		}
	}

	log.Info("Copied kernel modules to initramfs", "kernel", kernelVersion)
	return nil
}

// getRequiredModules returns module patterns required for booting
func (g *InitramfsGenerator) getRequiredModules() []string {
	modules := []string{
		// Storage drivers
		"kernel/drivers/ata/*.ko*",
		"kernel/drivers/scsi/*.ko*",
		"kernel/drivers/nvme/*.ko*",
		"kernel/drivers/virtio/*.ko*",
		"kernel/drivers/block/*.ko*",
		// Filesystem drivers
		"kernel/fs/ext4/*.ko*",
	}

	// Add filesystem-specific modules based on config
	switch strings.ToLower(g.config.System.Filesystem.Type) {
	case "xfs":
		modules = append(modules, "kernel/fs/xfs/*.ko*")
	case "btrfs":
		modules = append(modules, "kernel/fs/btrfs/*.ko*")
	case "f2fs":
		modules = append(modules, "kernel/fs/f2fs/*.ko*")
	}

	return modules
}

// copyModulesByPattern copies modules matching a pattern
func (g *InitramfsGenerator) copyModulesByPattern(srcBase, dstBase, pattern string) error {
	srcPattern := filepath.Join(srcBase, pattern)
	matches, err := filepath.Glob(srcPattern)
	if err != nil {
		return err
	}

	for _, src := range matches {
		relPath, err := filepath.Rel(srcBase, src)
		if err != nil {
			continue
		}
		dst := filepath.Join(dstBase, relPath)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}

	return nil
}

// copyBinaries copies essential binaries to initramfs
func (g *InitramfsGenerator) copyBinaries(initramfsDir string) error {
	// For a minimal initramfs, we need busybox or similar
	// In a real implementation, we would:
	// 1. Copy busybox statically linked
	// 2. Create symlinks for common commands
	// 3. Or copy specific binaries like switch_root

	// Create placeholder for busybox
	busyboxPath := filepath.Join(initramfsDir, "bin", "busybox")
	busyboxPlaceholder := `#!/bin/sh
# Busybox placeholder - replace with actual busybox binary
echo "Error: busybox not installed"
exit 1
`
	if err := os.WriteFile(busyboxPath, []byte(busyboxPlaceholder), 0755); err != nil {
		return err
	}

	// Create essential symlinks (these would point to busybox)
	essentialCommands := []string{
		"sh", "mount", "umount", "switch_root",
		"cat", "echo", "ls", "mkdir", "mknod",
		"sleep", "modprobe", "insmod",
	}

	binDir := filepath.Join(initramfsDir, "bin")
	for _, cmd := range essentialCommands {
		linkPath := filepath.Join(binDir, cmd)
		os.Remove(linkPath)
		if err := os.Symlink("busybox", linkPath); err != nil {
			log.Warn("Could not create symlink", "cmd", cmd, "error", err)
		}
	}

	return nil
}

// generateInit generates the init script
func (g *InitramfsGenerator) generateInit(initramfsDir string) error {
	// Determine root filesystem type
	fsType := g.config.System.Filesystem.Type
	if fsType == "" {
		fsType = "ext4"
	}

	initScript := fmt.Sprintf(`#!/bin/sh
# Linux Distribution Factory Initramfs Init
# Minimal init script to mount root and switch_root

# Exit on any error
set -e

# Mount essential filesystems
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev

# Create essential device nodes if devtmpfs failed
[ -e /dev/console ] || mknod -m 600 /dev/console c 5 1
[ -e /dev/null ] || mknod -m 666 /dev/null c 1 3

echo "LDF Initramfs starting..."

# Parse kernel command line
ROOT=""
ROOTFSTYPE="%s"
ROOTFLAGS="ro"

for param in $(cat /proc/cmdline); do
    case "$param" in
        root=*)
            ROOT="${param#root=}"
            ;;
        rootfstype=*)
            ROOTFSTYPE="${param#rootfstype=}"
            ;;
        rootflags=*)
            ROOTFLAGS="${param#rootflags=}"
            ;;
        ro)
            ROOTFLAGS="ro"
            ;;
        rw)
            ROOTFLAGS="rw"
            ;;
    esac
done

# Wait for root device
WAIT=0
while [ ! -e "$ROOT" ] && [ $WAIT -lt 30 ]; do
    echo "Waiting for root device $ROOT..."
    sleep 1
    WAIT=$((WAIT + 1))
done

if [ ! -e "$ROOT" ]; then
    echo "ERROR: Root device $ROOT not found!"
    echo "Dropping to shell..."
    exec /bin/sh
fi

# Load filesystem module if needed
case "$ROOTFSTYPE" in
    ext4)
        modprobe ext4 2>/dev/null || true
        ;;
    xfs)
        modprobe xfs 2>/dev/null || true
        ;;
    btrfs)
        modprobe btrfs 2>/dev/null || true
        ;;
esac

# Mount root filesystem
echo "Mounting root filesystem ($ROOT as $ROOTFSTYPE)..."
mount -t "$ROOTFSTYPE" -o "$ROOTFLAGS" "$ROOT" /mnt/root

if [ ! -x /mnt/root/sbin/init ] && [ ! -x /mnt/root/lib/systemd/systemd ]; then
    echo "ERROR: No init found on root filesystem!"
    echo "Dropping to shell..."
    exec /bin/sh
fi

# Clean up and switch root
echo "Switching to root filesystem..."
umount /proc
umount /sys

# Use switch_root to pivot to real root
exec switch_root /mnt/root /sbin/init
`, fsType)

	initPath := filepath.Join(initramfsDir, "init")
	if err := os.WriteFile(initPath, []byte(initScript), 0755); err != nil {
		return fmt.Errorf("failed to write init script: %w", err)
	}

	log.Info("Generated initramfs init script")
	return nil
}

// generatePackScript generates a script to create the cpio archive
func (g *InitramfsGenerator) generatePackScript(initramfsDir string) error {
	packScript := fmt.Sprintf(`#!/bin/bash
# Pack initramfs into cpio archive
# Run this script from within the initramfs directory

set -e

INITRAMFS_DIR="%s"
OUTPUT="%s"

cd "$INITRAMFS_DIR"

# Create cpio archive
find . -print0 | cpio --null -ov --format=newc 2>/dev/null | gzip -9 > "$OUTPUT"

echo "Created initramfs: $OUTPUT"
ls -lh "$OUTPUT"
`, initramfsDir, g.outputPath)

	scriptPath := filepath.Join(initramfsDir, "pack-initramfs.sh")
	if err := os.WriteFile(scriptPath, []byte(packScript), 0755); err != nil {
		return fmt.Errorf("failed to write pack script: %w", err)
	}

	return nil
}

// GetInitramfsScript returns a script that can be run inside a container to create the initramfs
func GetInitramfsScript(initramfsDir, outputPath, fsType string) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

INITRAMFS_DIR="%s"
OUTPUT="%s"
FSTYPE="%s"

# Ensure initramfs directory structure exists
mkdir -p "$INITRAMFS_DIR"/{bin,sbin,etc,lib,lib/modules,lib64,dev,proc,sys,run,mnt/root,usr/{bin,sbin,lib},tmp}

# Create minimal init script
cat > "$INITRAMFS_DIR/init" << 'INITEOF'
#!/bin/sh
set -e
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev
echo "LDF Initramfs starting..."
ROOT=""
for param in $(cat /proc/cmdline); do
    case "$param" in
        root=*) ROOT="${param#root=}" ;;
    esac
done
WAIT=0
while [ ! -e "$ROOT" ] && [ $WAIT -lt 30 ]; do
    sleep 1
    WAIT=$((WAIT + 1))
done
mount -t %s "$ROOT" /mnt/root
exec switch_root /mnt/root /sbin/init
INITEOF
chmod 755 "$INITRAMFS_DIR/init"

# Pack into cpio archive
cd "$INITRAMFS_DIR"
find . -print0 | cpio --null -ov --format=newc 2>/dev/null | gzip -9 > "$OUTPUT"
echo "Created initramfs: $OUTPUT"
`, initramfsDir, outputPath, fsType, fsType)
}
