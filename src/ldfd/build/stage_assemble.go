package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// AssembleStage assembles the root filesystem from compiled components
type AssembleStage struct{}

// NewAssembleStage creates a new assemble stage
func NewAssembleStage() *AssembleStage {
	return &AssembleStage{}
}

// Name returns the stage name
func (s *AssembleStage) Name() db.BuildStageName {
	return db.StageAssemble
}

// Validate checks whether this stage can run
func (s *AssembleStage) Validate(ctx context.Context, sc *StageContext) error {
	if len(sc.Components) == 0 {
		return fmt.Errorf("no components resolved")
	}
	if sc.RootfsDir == "" {
		return fmt.Errorf("rootfs directory not set")
	}
	if sc.Config == nil {
		return fmt.Errorf("distribution config not set")
	}
	return nil
}

// Execute assembles the root filesystem
func (s *AssembleStage) Execute(ctx context.Context, sc *StageContext, progress ProgressFunc) error {
	progress(0, "Starting root filesystem assembly")

	// Get distribution info for os-release
	distName := "LDF Linux"
	distVersion := "1.0"
	// These would normally come from the distribution record

	// Create rootfs builder
	builder := NewRootfsBuilder(sc.RootfsDir, distName, distVersion, sc.Config)

	// Step 1: Create FHS directory skeleton (10%)
	progress(5, "Creating directory skeleton")
	if err := builder.CreateSkeleton(); err != nil {
		return fmt.Errorf("failed to create skeleton: %w", err)
	}
	progress(10, "Directory skeleton created")

	// Step 2: Install kernel and modules (20%)
	progress(12, "Installing kernel")
	kernelOutputDir := filepath.Join(sc.WorkspacePath, "kernel-output")
	if err := builder.InstallKernel(kernelOutputDir); err != nil {
		return fmt.Errorf("failed to install kernel: %w", err)
	}
	progress(15, "Installing kernel modules")
	if err := builder.InstallModules(kernelOutputDir); err != nil {
		return fmt.Errorf("failed to install modules: %w", err)
	}
	progress(20, "Kernel installed")

	// Step 3: Install init system (35%)
	progress(22, "Installing init system")
	initInstaller := GetInitInstaller(sc.Config.System.Init)
	initComponent := s.findComponentByType(sc.Components, "init")
	if err := initInstaller.Install(sc.RootfsDir, initComponent); err != nil {
		return fmt.Errorf("failed to install init system: %w", err)
	}
	progress(28, "Configuring init system")
	if err := initInstaller.Configure(sc.RootfsDir); err != nil {
		return fmt.Errorf("failed to configure init system: %w", err)
	}
	progress(35, fmt.Sprintf("Init system (%s) installed", initInstaller.Name()))

	// Step 4: Install bootloader (50%)
	progress(37, "Installing bootloader")
	bootloaderInstaller := GetBootloaderInstaller(sc.Config.Core.Bootloader, distName, distVersion)
	bootloaderComponent := s.findComponentByType(sc.Components, "bootloader")
	if err := bootloaderInstaller.Install(sc.RootfsDir, bootloaderComponent); err != nil {
		return fmt.Errorf("failed to install bootloader: %w", err)
	}
	progress(45, "Configuring bootloader")
	kernelVersion := s.getKernelVersion(sc.Components)
	if err := bootloaderInstaller.Configure(sc.RootfsDir, kernelVersion, sc.TargetArch, true); err != nil {
		return fmt.Errorf("failed to configure bootloader: %w", err)
	}
	progress(50, fmt.Sprintf("Bootloader (%s) installed", bootloaderInstaller.Name()))

	// Step 5: Install filesystem tools (60%)
	progress(52, "Installing filesystem tools")
	// Filesystem tools are optional (userspace)
	if sc.Config.System.FilesystemUserspace {
		fsComponent := s.findComponentByType(sc.Components, "filesystem")
		if fsComponent != nil {
			log.Info("Filesystem userspace component available", "path", fsComponent.LocalPath)
		}
	}
	progress(60, "Filesystem configuration complete")

	// Step 6: Configure security framework (70%)
	progress(62, "Configuring security framework")
	securitySetup := GetSecuritySetup(sc.Config.Security.System)
	securityComponent := s.findComponentByType(sc.Components, "security")
	if err := securitySetup.Install(sc.RootfsDir, securityComponent); err != nil {
		return fmt.Errorf("failed to install security framework: %w", err)
	}
	if err := securitySetup.Configure(sc.RootfsDir); err != nil {
		return fmt.Errorf("failed to configure security framework: %w", err)
	}
	progress(70, fmt.Sprintf("Security framework (%s) configured", securitySetup.Name()))

	// Step 7: Generate initramfs (80%)
	progress(72, "Generating initramfs")
	initramfsPath := filepath.Join(sc.RootfsDir, "boot", "initramfs.img")
	initramfsGen := NewInitramfsGenerator(sc.RootfsDir, initramfsPath, sc.Config, sc.TargetArch)
	if err := initramfsGen.Generate(); err != nil {
		return fmt.Errorf("failed to generate initramfs: %w", err)
	}
	progress(80, "Initramfs generated")

	// Step 8: Configure system files (90%)
	progress(82, "Generating fstab")
	if err := builder.GenerateFstab(); err != nil {
		return fmt.Errorf("failed to generate fstab: %w", err)
	}

	progress(84, "Generating os-release")
	if err := builder.GenerateOSRelease(); err != nil {
		return fmt.Errorf("failed to generate os-release: %w", err)
	}

	progress(86, "Configuring hostname")
	if err := builder.GenerateHostname(""); err != nil {
		return fmt.Errorf("failed to configure hostname: %w", err)
	}

	progress(88, "Configuring networking")
	if err := builder.ConfigureNetworking(); err != nil {
		return fmt.Errorf("failed to configure networking: %w", err)
	}

	progress(89, "Configuring root account")
	if err := builder.ConfigureRootAccount(); err != nil {
		return fmt.Errorf("failed to configure root account: %w", err)
	}
	progress(90, "System configuration complete")

	// Step 9: Final validation (100%)
	progress(92, "Validating rootfs")
	if err := s.validateRootfs(sc.RootfsDir); err != nil {
		return fmt.Errorf("rootfs validation failed: %w", err)
	}

	progress(100, "Root filesystem assembly complete")
	return nil
}

// findComponentByType finds a component by its category/type
func (s *AssembleStage) findComponentByType(components []ResolvedComponent, compType string) *ResolvedComponent {
	compType = strings.ToLower(compType)
	for i := range components {
		category := strings.ToLower(components[i].Component.Category)
		name := strings.ToLower(components[i].Component.Name)

		if category == compType || strings.Contains(name, compType) {
			return &components[i]
		}
	}
	return nil
}

// getKernelVersion extracts the kernel version from components
func (s *AssembleStage) getKernelVersion(components []ResolvedComponent) string {
	for _, c := range components {
		if strings.Contains(strings.ToLower(c.Component.Name), "kernel") {
			return c.Version
		}
	}
	return "unknown"
}

// validateRootfs performs basic validation of the assembled rootfs
func (s *AssembleStage) validateRootfs(rootfsPath string) error {
	// Check for essential files/directories
	essentials := []string{
		"bin",
		"sbin",
		"etc",
		"lib",
		"usr",
		"var",
		"boot/vmlinuz",
		"etc/fstab",
		"etc/passwd",
		"etc/group",
		"etc/os-release",
	}

	for _, path := range essentials {
		fullPath := filepath.Join(rootfsPath, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("essential path missing: %s", path)
		}
	}

	// Check for init
	initPaths := []string{
		"sbin/init",
		"lib/systemd/systemd",
		"sbin/openrc-init",
	}

	initFound := false
	for _, path := range initPaths {
		fullPath := filepath.Join(rootfsPath, path)
		if _, err := os.Stat(fullPath); err == nil {
			initFound = true
			break
		}
		// Also check if it's a symlink
		if target, err := os.Readlink(fullPath); err == nil && target != "" {
			initFound = true
			break
		}
	}

	if !initFound {
		log.Warn("No init system binary found - image may not boot properly")
	}

	log.Info("Rootfs validation passed")
	return nil
}
