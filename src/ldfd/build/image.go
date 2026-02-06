package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// ImageGenerator defines the interface for generating bootable images
type ImageGenerator interface {
	// Name returns the generator name
	Name() string

	// Format returns the output image format
	Format() db.ImageFormat

	// Generate creates the image from the assembled rootfs
	// Returns the path to the generated image
	Generate(ctx context.Context, sc *StageContext, progress ProgressFunc) (string, error)
}

// RawImageGenerator creates raw disk images
type RawImageGenerator struct {
	executor *ContainerExecutor
	sizeGB   int // Image size in GB (default: 4)
}

// NewRawImageGenerator creates a new raw image generator
func NewRawImageGenerator(executor *ContainerExecutor, sizeGB int) *RawImageGenerator {
	if sizeGB <= 0 {
		sizeGB = 4
	}
	return &RawImageGenerator{
		executor: executor,
		sizeGB:   sizeGB,
	}
}

// Name returns the generator name
func (g *RawImageGenerator) Name() string {
	return "raw"
}

// Format returns the output image format
func (g *RawImageGenerator) Format() db.ImageFormat {
	return db.ImageFormatRaw
}

// Generate creates a raw disk image
func (g *RawImageGenerator) Generate(ctx context.Context, sc *StageContext, progress ProgressFunc) (string, error) {
	imagePath := filepath.Join(sc.OutputDir, "disk.img")
	sizeMB := g.sizeGB * 1024

	progress(5, "Creating sparse disk image")

	// Create sparse image file
	if err := g.createSparseImage(imagePath, sizeMB); err != nil {
		return "", fmt.Errorf("failed to create image: %w", err)
	}

	progress(10, "Creating partition table")

	// Create GPT partition table with ESP and root partitions
	if err := g.createPartitionTable(ctx, imagePath, sc.TargetArch); err != nil {
		return "", fmt.Errorf("failed to create partitions: %w", err)
	}

	progress(20, "Setting up loop device")

	// Setup loop device
	loopDev, err := g.setupLoopDevice(ctx, imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to setup loop device: %w", err)
	}
	defer func() {
		if err := g.detachLoopDevice(ctx, loopDev); err != nil {
			log.Warn("Failed to detach loop device", "device", loopDev, "error", err)
		}
	}()

	progress(25, "Formatting partitions")

	// Format partitions
	if err := g.formatPartitions(ctx, loopDev); err != nil {
		return "", fmt.Errorf("failed to format partitions: %w", err)
	}

	progress(30, "Mounting partitions")

	// Create temporary mount point
	mountPoint, err := os.MkdirTemp("", "ldf-mount-")
	if err != nil {
		return "", fmt.Errorf("failed to create mount point: %w", err)
	}
	defer os.RemoveAll(mountPoint)

	// Mount root partition
	if err := g.mountPartitions(ctx, loopDev, mountPoint); err != nil {
		return "", fmt.Errorf("failed to mount partitions: %w", err)
	}
	defer func() {
		if err := g.unmountPartitions(ctx, mountPoint); err != nil {
			log.Warn("Failed to unmount partitions", "mount_point", mountPoint, "error", err)
		}
	}()

	progress(40, "Copying root filesystem")

	// Copy rootfs to mounted image
	if err := g.copyRootfs(ctx, sc.RootfsDir, mountPoint); err != nil {
		return "", fmt.Errorf("failed to copy rootfs: %w", err)
	}

	progress(70, "Installing bootloader to disk")

	// Install bootloader
	if err := g.installBootloader(ctx, sc, loopDev, mountPoint); err != nil {
		return "", fmt.Errorf("failed to install bootloader: %w", err)
	}

	progress(85, "Syncing and unmounting")

	// Sync before unmount
	if err := g.syncFilesystem(ctx, mountPoint); err != nil {
		log.Warn("Failed to sync filesystem", "error", err)
	}

	// Unmount (deferred, but call explicitly for progress reporting)
	if err := g.unmountPartitions(ctx, mountPoint); err != nil {
		return "", fmt.Errorf("failed to unmount: %w", err)
	}

	progress(90, "Detaching loop device")

	// Detach loop device
	if err := g.detachLoopDevice(ctx, loopDev); err != nil {
		log.Warn("Failed to detach loop device", "error", err)
	}

	progress(100, "Raw image created successfully")
	return imagePath, nil
}

// createSparseImage creates a sparse disk image file
func (g *RawImageGenerator) createSparseImage(path string, sizeMB int) error {
	// Create sparse file using truncate
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Truncate to desired size (creates sparse file)
	return f.Truncate(int64(sizeMB) * 1024 * 1024)
}

// createPartitionTable creates GPT partition table with ESP and root
func (g *RawImageGenerator) createPartitionTable(ctx context.Context, imagePath string, arch db.TargetArch) error {
	// Use sgdisk for GPT partitioning
	// Partition 1: EFI System Partition (512MB)
	// Partition 2: Root partition (rest)
	commands := []string{
		// Clear existing partition table
		fmt.Sprintf("sgdisk --zap-all %s", imagePath),
		// Create ESP (512MB, type EF00)
		fmt.Sprintf("sgdisk --new=1:2048:+512M --typecode=1:EF00 --change-name=1:ESP %s", imagePath),
		// Create root partition (rest of disk, type 8300 for Linux)
		fmt.Sprintf("sgdisk --new=2:0:0 --typecode=2:8300 --change-name=2:root %s", imagePath),
	}

	for _, cmd := range commands {
		parts := strings.Fields(cmd)
		c := exec.CommandContext(ctx, parts[0], parts[1:]...)
		if output, err := c.CombinedOutput(); err != nil {
			return fmt.Errorf("partition command failed: %s: %s", err, output)
		}
	}

	return nil
}

// setupLoopDevice attaches the image to a loop device with partition scanning
func (g *RawImageGenerator) setupLoopDevice(ctx context.Context, imagePath string) (string, error) {
	// Setup loop device with partition scanning
	cmd := exec.CommandContext(ctx, "losetup", "--find", "--show", "--partscan", imagePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("losetup failed: %w", err)
	}

	loopDev := strings.TrimSpace(string(output))
	log.Debug("Setup loop device", "device", loopDev, "image", imagePath)
	return loopDev, nil
}

// detachLoopDevice detaches a loop device
func (g *RawImageGenerator) detachLoopDevice(ctx context.Context, loopDev string) error {
	if loopDev == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "losetup", "-d", loopDev)
	return cmd.Run()
}

// formatPartitions formats the ESP and root partitions
func (g *RawImageGenerator) formatPartitions(ctx context.Context, loopDev string) error {
	espDev := loopDev + "p1"
	rootDev := loopDev + "p2"

	// Format ESP as FAT32
	cmd := exec.CommandContext(ctx, "mkfs.fat", "-F32", "-n", "ESP", espDev)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs.fat failed: %s: %s", err, output)
	}

	// Format root as ext4
	cmd = exec.CommandContext(ctx, "mkfs.ext4", "-L", "root", "-F", rootDev)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs.ext4 failed: %s: %s", err, output)
	}

	return nil
}

// mountPartitions mounts the root and ESP partitions
func (g *RawImageGenerator) mountPartitions(ctx context.Context, loopDev, mountPoint string) error {
	rootDev := loopDev + "p2"
	espDev := loopDev + "p1"

	// Mount root partition
	cmd := exec.CommandContext(ctx, "mount", rootDev, mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount root failed: %s: %s", err, output)
	}

	// Create and mount ESP
	espMount := filepath.Join(mountPoint, "boot", "efi")
	if err := os.MkdirAll(espMount, 0755); err != nil {
		return fmt.Errorf("failed to create ESP mount point: %w", err)
	}

	cmd = exec.CommandContext(ctx, "mount", espDev, espMount)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount ESP failed: %s: %s", err, output)
	}

	return nil
}

// unmountPartitions unmounts all mounted partitions
func (g *RawImageGenerator) unmountPartitions(ctx context.Context, mountPoint string) error {
	// Unmount ESP first
	espMount := filepath.Join(mountPoint, "boot", "efi")
	if err := exec.CommandContext(ctx, "umount", espMount).Run(); err != nil {
		log.Warn("Failed to unmount ESP", "mount_point", espMount, "error", err)
	}

	// Unmount root
	cmd := exec.CommandContext(ctx, "umount", mountPoint)
	return cmd.Run()
}

// copyRootfs copies the assembled rootfs to the mounted image
func (g *RawImageGenerator) copyRootfs(ctx context.Context, srcDir, dstDir string) error {
	// Use rsync for efficient copy preserving permissions
	cmd := exec.CommandContext(ctx, "rsync", "-aAX", "--info=progress2",
		srcDir+"/", dstDir+"/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// installBootloader installs the bootloader to the disk image
func (g *RawImageGenerator) installBootloader(ctx context.Context, sc *StageContext, loopDev, mountPoint string) error {
	bootloader := GetBootloaderInstaller(sc.Config.Core.Bootloader, "LDF Linux", "1.0")
	commands := bootloader.GetInstallCommands(loopDev, sc.TargetArch)

	for _, cmdStr := range commands {
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			continue
		}

		// Run command with rootfs as the effective root
		cmd := exec.CommandContext(ctx, "chroot", append([]string{mountPoint}, parts...)...)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Warn("Bootloader install command failed", "cmd", cmdStr, "error", err, "output", string(output))
			// Continue with other commands
		}
	}

	return nil
}

// syncFilesystem syncs all pending writes
func (g *RawImageGenerator) syncFilesystem(ctx context.Context, mountPoint string) error {
	cmd := exec.CommandContext(ctx, "sync")
	return cmd.Run()
}

// QCOW2ImageGenerator creates QCOW2 virtual disk images
type QCOW2ImageGenerator struct {
	rawGenerator *RawImageGenerator
	compression  bool
}

// NewQCOW2ImageGenerator creates a new QCOW2 image generator
func NewQCOW2ImageGenerator(executor *ContainerExecutor, sizeGB int, compression bool) *QCOW2ImageGenerator {
	return &QCOW2ImageGenerator{
		rawGenerator: NewRawImageGenerator(executor, sizeGB),
		compression:  compression,
	}
}

// Name returns the generator name
func (g *QCOW2ImageGenerator) Name() string {
	return "qcow2"
}

// Format returns the output image format
func (g *QCOW2ImageGenerator) Format() db.ImageFormat {
	return db.ImageFormatQCOW2
}

// Generate creates a QCOW2 image by first generating raw, then converting
func (g *QCOW2ImageGenerator) Generate(ctx context.Context, sc *StageContext, progress ProgressFunc) (string, error) {
	// Generate raw image first (scaled progress 0-80%)
	rawProgress := func(percent int, msg string) {
		scaledPercent := int(float64(percent) * 0.8)
		progress(scaledPercent, msg)
	}

	rawPath, err := g.rawGenerator.Generate(ctx, sc, rawProgress)
	if err != nil {
		return "", fmt.Errorf("failed to generate raw image: %w", err)
	}

	progress(82, "Converting to QCOW2 format")

	// Convert to QCOW2
	qcow2Path := filepath.Join(sc.OutputDir, "disk.qcow2")
	args := []string{"convert", "-f", "raw", "-O", "qcow2"}
	if g.compression {
		args = append(args, "-c")
	}
	args = append(args, rawPath, qcow2Path)

	cmd := exec.CommandContext(ctx, "qemu-img", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("qemu-img convert failed: %s: %s", err, output)
	}

	progress(95, "Removing intermediate raw image")

	// Remove raw image
	if err := os.Remove(rawPath); err != nil {
		log.Warn("Failed to remove raw image", "path", rawPath, "error", err)
	}

	progress(100, "QCOW2 image created successfully")
	return qcow2Path, nil
}

// ISOImageGenerator creates bootable ISO images
type ISOImageGenerator struct {
	executor  *ContainerExecutor
	volumeID  string
	publisher string
}

// NewISOImageGenerator creates a new ISO image generator
func NewISOImageGenerator(executor *ContainerExecutor, volumeID, publisher string) *ISOImageGenerator {
	if volumeID == "" {
		volumeID = "LDF_LINUX"
	}
	if publisher == "" {
		publisher = "LDF Build System"
	}
	return &ISOImageGenerator{
		executor:  executor,
		volumeID:  volumeID,
		publisher: publisher,
	}
}

// Name returns the generator name
func (g *ISOImageGenerator) Name() string {
	return "iso"
}

// Format returns the output image format
func (g *ISOImageGenerator) Format() db.ImageFormat {
	return db.ImageFormatISO
}

// Generate creates a bootable ISO image
func (g *ISOImageGenerator) Generate(ctx context.Context, sc *StageContext, progress ProgressFunc) (string, error) {
	isoPath := filepath.Join(sc.OutputDir, "ldf-linux.iso")

	progress(5, "Preparing ISO filesystem structure")

	// Create ISO staging directory
	isoStaging := filepath.Join(sc.WorkspacePath, "iso-staging")
	if err := os.MkdirAll(isoStaging, 0755); err != nil {
		return "", fmt.Errorf("failed to create ISO staging: %w", err)
	}

	// Create standard ISO structure
	dirs := []string{
		"boot/grub",
		"EFI/BOOT",
		"isolinux",
		"LiveOS",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(isoStaging, dir), 0755); err != nil {
			return "", fmt.Errorf("failed to create ISO dir %s: %w", dir, err)
		}
	}

	progress(15, "Copying kernel and initramfs")

	// Copy kernel and initramfs to boot directory
	bootDir := filepath.Join(isoStaging, "boot")
	if err := g.copyBootFiles(sc.RootfsDir, bootDir); err != nil {
		return "", fmt.Errorf("failed to copy boot files: %w", err)
	}

	progress(25, "Creating squashfs image")

	// Create squashfs of rootfs for LiveOS
	squashfsPath := filepath.Join(isoStaging, "LiveOS", "squashfs.img")
	if err := g.createSquashfs(ctx, sc.RootfsDir, squashfsPath); err != nil {
		return "", fmt.Errorf("failed to create squashfs: %w", err)
	}

	progress(50, "Setting up GRUB for ISO boot")

	// Create GRUB config for ISO
	if err := g.createISOGrubConfig(isoStaging, sc.TargetArch); err != nil {
		return "", fmt.Errorf("failed to create GRUB config: %w", err)
	}

	progress(60, "Creating EFI boot image")

	// Create EFI boot image
	efiImagePath := filepath.Join(isoStaging, "boot", "efi.img")
	if err := g.createEFIImage(ctx, sc, efiImagePath); err != nil {
		return "", fmt.Errorf("failed to create EFI image: %w", err)
	}

	progress(75, "Generating ISO image")

	// Generate ISO using xorriso
	if err := g.generateISO(ctx, isoStaging, isoPath, sc.TargetArch); err != nil {
		return "", fmt.Errorf("failed to generate ISO: %w", err)
	}

	progress(95, "Cleaning up staging directory")

	// Clean up staging
	os.RemoveAll(isoStaging)

	progress(100, "ISO image created successfully")
	return isoPath, nil
}

// copyBootFiles copies kernel and initramfs to ISO boot directory
func (g *ISOImageGenerator) copyBootFiles(rootfsDir, bootDir string) error {
	files := map[string]string{
		"boot/vmlinuz":       "vmlinuz",
		"boot/initramfs.img": "initramfs.img",
	}

	for src, dst := range files {
		srcPath := filepath.Join(rootfsDir, src)
		dstPath := filepath.Join(bootDir, dst)

		if err := copyFile(srcPath, dstPath); err != nil {
			// Try alternate names
			if strings.Contains(src, "vmlinuz") {
				// Try vmlinuz-* pattern
				matches, _ := filepath.Glob(filepath.Join(rootfsDir, "boot", "vmlinuz-*"))
				if len(matches) > 0 {
					srcPath = matches[0]
					err = copyFile(srcPath, dstPath)
				}
			}
			if err != nil {
				return fmt.Errorf("failed to copy %s: %w", src, err)
			}
		}
	}

	return nil
}

// createSquashfs creates a squashfs image of the rootfs
func (g *ISOImageGenerator) createSquashfs(ctx context.Context, rootfsDir, outputPath string) error {
	cmd := exec.CommandContext(ctx, "mksquashfs", rootfsDir, outputPath,
		"-comp", "xz", "-Xbcj", "x86", "-b", "1M", "-no-recovery")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// createISOGrubConfig creates GRUB configuration for ISO boot
func (g *ISOImageGenerator) createISOGrubConfig(isoStaging string, arch db.TargetArch) error {
	grubCfg := `# GRUB configuration for LDF Linux Live ISO

set timeout=10
set default=0

menuentry "LDF Linux (Live)" {
    linux /boot/vmlinuz root=live:CDLABEL=%s rd.live.image quiet
    initrd /boot/initramfs.img
}

menuentry "LDF Linux (Live, Debug)" {
    linux /boot/vmlinuz root=live:CDLABEL=%s rd.live.image rd.debug
    initrd /boot/initramfs.img
}
`
	grubCfg = fmt.Sprintf(grubCfg, g.volumeID, g.volumeID)

	grubCfgPath := filepath.Join(isoStaging, "boot", "grub", "grub.cfg")
	return os.WriteFile(grubCfgPath, []byte(grubCfg), 0644)
}

// createEFIImage creates an EFI boot image for the ISO
func (g *ISOImageGenerator) createEFIImage(ctx context.Context, sc *StageContext, outputPath string) error {
	// Create a small FAT image for EFI boot
	sizeMB := 64

	// Create image file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	if err := f.Truncate(int64(sizeMB) * 1024 * 1024); err != nil {
		f.Close()
		return fmt.Errorf("truncate EFI image: %w", err)
	}
	f.Close()

	// Format as FAT
	cmd := exec.CommandContext(ctx, "mkfs.fat", "-F12", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs.fat for EFI failed: %s: %s", err, output)
	}

	// Mount and copy EFI files
	mountPoint, err := os.MkdirTemp("", "ldf-efi-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mountPoint)

	cmd = exec.CommandContext(ctx, "mount", "-o", "loop", outputPath, mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount EFI image failed: %s: %s", err, output)
	}
	defer func() {
		if err := exec.CommandContext(ctx, "umount", mountPoint).Run(); err != nil {
			log.Warn("Failed to unmount EFI image", "mount_point", mountPoint, "error", err)
		}
	}()

	// Create EFI directory structure
	efiBootDir := filepath.Join(mountPoint, "EFI", "BOOT")
	if err := os.MkdirAll(efiBootDir, 0755); err != nil {
		return err
	}

	// Copy or create EFI bootloader
	// Try to find existing EFI bootloader in rootfs
	var efiSrc string
	switch sc.TargetArch {
	case db.ArchX86_64:
		candidates := []string{
			filepath.Join(sc.RootfsDir, "boot/efi/EFI/BOOT/BOOTX64.EFI"),
			filepath.Join(sc.RootfsDir, "usr/lib/grub/x86_64-efi/monolithic/grubx64.efi"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				efiSrc = c
				break
			}
		}
	case db.ArchAARCH64:
		candidates := []string{
			filepath.Join(sc.RootfsDir, "boot/efi/EFI/BOOT/BOOTAA64.EFI"),
			filepath.Join(sc.RootfsDir, "usr/lib/grub/arm64-efi/monolithic/grubaa64.efi"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				efiSrc = c
				break
			}
		}
	}

	if efiSrc != "" {
		var efiDst string
		switch sc.TargetArch {
		case db.ArchX86_64:
			efiDst = filepath.Join(efiBootDir, "BOOTX64.EFI")
		case db.ArchAARCH64:
			efiDst = filepath.Join(efiBootDir, "BOOTAA64.EFI")
		}
		if err := copyFile(efiSrc, efiDst); err != nil {
			log.Warn("Failed to copy EFI bootloader", "error", err)
		}
	}

	return nil
}

// generateISO generates the final ISO image using xorriso
func (g *ISOImageGenerator) generateISO(ctx context.Context, isoStaging, outputPath string, arch db.TargetArch) error {
	args := []string{
		"-as", "mkisofs",
		"-o", outputPath,
		"-V", g.volumeID,
		"-publisher", g.publisher,
		"-J", "-R", "-l",
		"-b", "boot/efi.img",
		"-no-emul-boot",
		"-boot-load-size", "4",
		"-boot-info-table",
		"-eltorito-alt-boot",
		"-e", "boot/efi.img",
		"-no-emul-boot",
		"-isohybrid-gpt-basdat",
		isoStaging,
	}

	cmd := exec.CommandContext(ctx, "xorriso", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetImageGenerator returns the appropriate image generator for the format
func GetImageGenerator(format db.ImageFormat, executor *ContainerExecutor, sizeGB int) ImageGenerator {
	switch format {
	case db.ImageFormatQCOW2:
		return NewQCOW2ImageGenerator(executor, sizeGB, true)
	case db.ImageFormatISO:
		return NewISOImageGenerator(executor, "", "")
	default:
		return NewRawImageGenerator(executor, sizeGB)
	}
}

// CalculateChecksum calculates SHA256 checksum of a file
func CalculateChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
