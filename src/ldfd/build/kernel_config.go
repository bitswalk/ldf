package build

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/storage"
)

// KernelConfigGenerator generates kernel .config files based on distribution configuration
type KernelConfigGenerator struct {
	storage storage.Backend
}

// NewKernelConfigGenerator creates a new kernel config generator
func NewKernelConfigGenerator(storage storage.Backend) *KernelConfigGenerator {
	return &KernelConfigGenerator{storage: storage}
}

// Generate creates a kernel .config file based on the distribution's kernel configuration mode
func (g *KernelConfigGenerator) Generate(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, outputPath string) error {
	kernelConfig := sc.Config.Core.Kernel
	mode := kernelConfig.ConfigMode

	// Default to defconfig if mode is empty
	if mode == "" {
		mode = db.KernelConfigModeDefconfig
	}

	log.Info("Generating kernel config",
		"mode", mode,
		"arch", sc.TargetArch,
		"kernel_version", kernel.Version)

	switch mode {
	case db.KernelConfigModeDefconfig:
		return g.generateDefconfig(ctx, sc, kernel, outputPath)

	case db.KernelConfigModeOptions:
		return g.generateWithOptions(ctx, sc, kernel, kernelConfig.ConfigOptions, outputPath)

	case db.KernelConfigModeCustom:
		return g.generateFromCustom(ctx, sc, kernelConfig.CustomConfigPath, outputPath)

	default:
		return fmt.Errorf("unknown kernel config mode: %s", mode)
	}
}

// generateDefconfig generates a config using the kernel's default architecture config
// This creates a minimal config file that tells the build to use defconfig
func (g *KernelConfigGenerator) generateDefconfig(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, outputPath string) error {
	// For defconfig mode, we generate a marker file that the build script will use
	// to run `make defconfig` before building
	content := fmt.Sprintf(`# LDF Kernel Configuration
# Mode: defconfig
# Architecture: %s
# Generated for kernel %s
#
# This is a marker file. The actual config will be generated
# by running 'make %s_defconfig' during the build.

LDF_CONFIG_MODE=defconfig
LDF_TARGET_ARCH=%s
`, sc.TargetArch, kernel.Version, g.getDefconfigName(sc.TargetArch), sc.TargetArch)

	// Also generate recommended options that will be applied after defconfig
	recommendedOptions := g.getRecommendedOptions(sc)
	if len(recommendedOptions) > 0 {
		content += "\n# Recommended options (applied after defconfig):\n"
		for key, value := range recommendedOptions {
			content += fmt.Sprintf("# %s=%s\n", key, value)
		}
	}

	return os.WriteFile(outputPath, []byte(content), 0644)
}

// generateWithOptions generates a config starting from defconfig and applying custom options
func (g *KernelConfigGenerator) generateWithOptions(ctx context.Context, sc *StageContext, kernel *ResolvedComponent, options map[string]string, outputPath string) error {
	// Start with recommended options
	allOptions := g.getRecommendedOptions(sc)

	// Apply user-specified options (these override recommended ones)
	for key, value := range options {
		// Normalize key to ensure CONFIG_ prefix
		if !strings.HasPrefix(key, "CONFIG_") {
			key = "CONFIG_" + key
		}
		allOptions[key] = value
	}

	// Generate the config file
	var content strings.Builder
	content.WriteString(fmt.Sprintf(`# LDF Kernel Configuration
# Mode: options (defconfig + custom options)
# Architecture: %s
# Generated for kernel %s
#
# The build will:
# 1. Run 'make %s_defconfig'
# 2. Apply the options below using scripts/config
# 3. Run 'make olddefconfig' to resolve dependencies

LDF_CONFIG_MODE=options
LDF_TARGET_ARCH=%s

# Custom options to apply:
`, sc.TargetArch, kernel.Version, g.getDefconfigName(sc.TargetArch), sc.TargetArch))

	// Sort keys for deterministic output
	keys := make([]string, 0, len(allOptions))
	for k := range allOptions {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := allOptions[key]
		switch value {
		case "y":
			content.WriteString(fmt.Sprintf("%s=y\n", key))
		case "m":
			content.WriteString(fmt.Sprintf("%s=m\n", key))
		case "n":
			content.WriteString(fmt.Sprintf("# %s is not set\n", key))
		default:
			// String or numeric value
			if strings.HasPrefix(value, "\"") {
				content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			} else {
				content.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
			}
		}
	}

	return os.WriteFile(outputPath, []byte(content.String()), 0644)
}

// generateFromCustom copies a user-provided custom config from storage
func (g *KernelConfigGenerator) generateFromCustom(ctx context.Context, sc *StageContext, customConfigPath, outputPath string) error {
	if customConfigPath == "" {
		return fmt.Errorf("custom config path is empty")
	}

	if g.storage == nil {
		return fmt.Errorf("storage backend not configured")
	}

	// Download custom config from storage
	reader, _, err := g.storage.Download(ctx, customConfigPath)
	if err != nil {
		return fmt.Errorf("failed to retrieve custom config from storage: %w", err)
	}
	defer reader.Close()

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Write header comment
	header := fmt.Sprintf(`# LDF Kernel Configuration
# Mode: custom (user-provided)
# Architecture: %s
# Source: %s
#
# This config was provided by the user and will be used as-is.
# Run 'make olddefconfig' to resolve any missing options.

LDF_CONFIG_MODE=custom
LDF_TARGET_ARCH=%s

`, sc.TargetArch, customConfigPath, sc.TargetArch)

	if _, err := outFile.WriteString(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Copy the rest of the config
	if _, err := io.Copy(outFile, reader); err != nil {
		return fmt.Errorf("failed to copy custom config: %w", err)
	}

	log.Info("Copied custom kernel config",
		"source", customConfigPath,
		"dest", outputPath)

	return nil
}

// getDefconfigName returns the defconfig target name for an architecture
func (g *KernelConfigGenerator) getDefconfigName(arch db.TargetArch) string {
	switch arch {
	case db.ArchX86_64:
		return "x86_64"
	case db.ArchAARCH64:
		return "defconfig" // ARM64 uses generic defconfig
	default:
		return "defconfig"
	}
}

// getRecommendedOptions returns recommended kernel options based on distribution config
func (g *KernelConfigGenerator) getRecommendedOptions(sc *StageContext) map[string]string {
	options := make(map[string]string)

	// Basic system options
	options["CONFIG_PRINTK"] = "y"
	options["CONFIG_BUG"] = "y"
	options["CONFIG_ELF_CORE"] = "y"
	options["CONFIG_PROC_FS"] = "y"
	options["CONFIG_SYSFS"] = "y"
	options["CONFIG_TMPFS"] = "y"
	options["CONFIG_DEVTMPFS"] = "y"
	options["CONFIG_DEVTMPFS_MOUNT"] = "y"

	// Filesystem options based on config
	fsType := sc.Config.System.Filesystem.Type
	switch strings.ToLower(fsType) {
	case "ext4":
		options["CONFIG_EXT4_FS"] = "y"
		options["CONFIG_EXT4_USE_FOR_EXT2"] = "y"
	case "xfs":
		options["CONFIG_XFS_FS"] = "y"
	case "btrfs":
		options["CONFIG_BTRFS_FS"] = "y"
	case "f2fs":
		options["CONFIG_F2FS_FS"] = "y"
	}

	// Always enable common filesystems
	options["CONFIG_VFAT_FS"] = "y"
	options["CONFIG_FAT_FS"] = "y"
	options["CONFIG_MSDOS_FS"] = "y"
	options["CONFIG_ISO9660_FS"] = "m"

	// Init system options
	initSystem := sc.Config.System.Init
	switch strings.ToLower(initSystem) {
	case "systemd":
		options["CONFIG_CGROUPS"] = "y"
		options["CONFIG_CGROUP_SCHED"] = "y"
		options["CONFIG_CGROUP_PIDS"] = "y"
		options["CONFIG_CGROUP_FREEZER"] = "y"
		options["CONFIG_CPUSETS"] = "y"
		options["CONFIG_MEMCG"] = "y"
		options["CONFIG_NAMESPACES"] = "y"
		options["CONFIG_USER_NS"] = "y"
		options["CONFIG_NET_NS"] = "y"
		options["CONFIG_PID_NS"] = "y"
		options["CONFIG_IPC_NS"] = "y"
		options["CONFIG_UTS_NS"] = "y"
		options["CONFIG_INOTIFY_USER"] = "y"
		options["CONFIG_SIGNALFD"] = "y"
		options["CONFIG_TIMERFD"] = "y"
		options["CONFIG_EPOLL"] = "y"
		options["CONFIG_FHANDLE"] = "y"
		options["CONFIG_DMIID"] = "y"
		options["CONFIG_AUTOFS_FS"] = "y"
	case "openrc":
		options["CONFIG_SYSVIPC"] = "y"
	}

	// Security options
	secSystem := sc.Config.Security.System
	switch strings.ToLower(secSystem) {
	case "selinux":
		options["CONFIG_SECURITY"] = "y"
		options["CONFIG_SECURITY_NETWORK"] = "y"
		options["CONFIG_SECURITY_SELINUX"] = "y"
		options["CONFIG_SECURITY_SELINUX_BOOTPARAM"] = "y"
		options["CONFIG_AUDIT"] = "y"
		options["CONFIG_AUDITSYSCALL"] = "y"
	case "apparmor":
		options["CONFIG_SECURITY"] = "y"
		options["CONFIG_SECURITY_NETWORK"] = "y"
		options["CONFIG_SECURITY_APPARMOR"] = "y"
		options["CONFIG_SECURITY_APPARMOR_BOOTPARAM_VALUE"] = "1"
		options["CONFIG_AUDIT"] = "y"
	}

	// Virtualization options
	virtSystem := sc.Config.Runtime.Virtualization
	switch strings.ToLower(virtSystem) {
	case "kvm":
		options["CONFIG_VIRTUALIZATION"] = "y"
		options["CONFIG_KVM"] = "m"
		if sc.TargetArch == db.ArchX86_64 {
			options["CONFIG_KVM_INTEL"] = "m"
			options["CONFIG_KVM_AMD"] = "m"
		}
	}

	// Container options
	containerSystem := sc.Config.Runtime.Container
	if containerSystem != "" {
		options["CONFIG_NAMESPACES"] = "y"
		options["CONFIG_USER_NS"] = "y"
		options["CONFIG_NET_NS"] = "y"
		options["CONFIG_PID_NS"] = "y"
		options["CONFIG_IPC_NS"] = "y"
		options["CONFIG_UTS_NS"] = "y"
		options["CONFIG_CGROUPS"] = "y"
		options["CONFIG_CGROUP_DEVICE"] = "y"
		options["CONFIG_CGROUP_SCHED"] = "y"
		options["CONFIG_CGROUP_PIDS"] = "y"
		options["CONFIG_CGROUP_FREEZER"] = "y"
		options["CONFIG_CPUSETS"] = "y"
		options["CONFIG_MEMCG"] = "y"
		options["CONFIG_VETH"] = "y"
		options["CONFIG_BRIDGE"] = "y"
		options["CONFIG_NETFILTER_ADVANCED"] = "y"
		options["CONFIG_NF_NAT"] = "y"
		options["CONFIG_NETFILTER_XT_MATCH_CONNTRACK"] = "y"
		options["CONFIG_OVERLAY_FS"] = "y"
	}

	// Networking basics
	options["CONFIG_NET"] = "y"
	options["CONFIG_INET"] = "y"
	options["CONFIG_IPV6"] = "y"
	options["CONFIG_NETDEVICES"] = "y"

	// Block device support
	options["CONFIG_BLOCK"] = "y"
	options["CONFIG_BLK_DEV"] = "y"
	options["CONFIG_BLK_DEV_LOOP"] = "y"

	return options
}

// ParseConfigFile parses a kernel .config file into a map
func ParseConfigFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	options := make(map[string]string)
	scanner := bufio.NewScanner(file)

	// Regex for CONFIG_FOO=value
	setRegex := regexp.MustCompile(`^(CONFIG_[A-Z0-9_]+)=(.*)$`)
	// Regex for # CONFIG_FOO is not set
	unsetRegex := regexp.MustCompile(`^# (CONFIG_[A-Z0-9_]+) is not set$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if matches := setRegex.FindStringSubmatch(line); matches != nil {
			key := matches[1]
			value := matches[2]
			// Remove quotes from string values
			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				value = value[1 : len(value)-1]
			}
			options[key] = value
		} else if matches := unsetRegex.FindStringSubmatch(line); matches != nil {
			options[matches[1]] = "n"
		}
	}

	return options, scanner.Err()
}

// MergeConfigOptions merges two config option maps, with override taking precedence
func MergeConfigOptions(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// GenerateConfigFragment generates a kernel config fragment file
// This can be used with scripts/kconfig/merge_config.sh
func GenerateConfigFragment(options map[string]string, outputPath string) error {
	var content strings.Builder

	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := options[key]
		switch value {
		case "y", "m":
			content.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		case "n":
			content.WriteString(fmt.Sprintf("# %s is not set\n", key))
		default:
			content.WriteString(fmt.Sprintf("%s=\"%s\"\n", key, value))
		}
	}

	return os.WriteFile(outputPath, []byte(content.String()), 0644)
}

// GetKernelSourceConfigPath returns the path to the generated .config in the kernel source
func GetKernelSourceConfigPath(kernelSourceDir string) string {
	return filepath.Join(kernelSourceDir, ".config")
}
