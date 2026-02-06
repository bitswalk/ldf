package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// HostArch represents the detected host machine architecture
type HostArch string

const (
	HostArchX86_64  HostArch = "x86_64"
	HostArchAARCH64 HostArch = "aarch64"
)

// ToolchainInfo describes the cross-compilation toolchain for a
// specific host-to-target combination. Toolchains are assumed to
// exist inside the container image; this struct tells the build
// pipeline WHICH toolchain to reference.
type ToolchainInfo struct {
	CrossCompilePrefix string // e.g. "aarch64-linux-gnu-"
	MakeArch           string // e.g. "arm64"
	ToolchainPkg       string // informational: package name, e.g. "gcc-aarch64-linux-gnu"
}

// ArchPair is the key for toolchain lookup: host -> target
type ArchPair struct {
	Host   HostArch
	Target db.TargetArch
}

// QEMUSupport holds the result of QEMU binfmt availability detection
type QEMUSupport struct {
	Available        bool   // Is qemu-user-static or equivalent installed?
	BinfmtRegistered bool   // Is binfmt_misc registered for the target arch?
	QEMUBinary       string // Path to the qemu-<arch>-static binary, if found
}

// BuildEnvironment is the validated result of pre-flight checks,
// carried forward into StageContext for use by all stages
type BuildEnvironment struct {
	HostArch           HostArch
	TargetArch         db.TargetArch
	IsNative           bool
	Toolchain          ToolchainInfo
	ContainerImage     string // resolved image name (arch-specific or default)
	UseQEMUEmulation   bool   // true if running foreign-arch container via --platform
	QEMUSupport        QEMUSupport
	PodmanPlatformFlag string // e.g. "linux/arm64" for --platform, empty if native
}

// toolchainRegistry maps host→target pairs to their toolchain configuration.
// All 4 combinations in the 2x2 matrix are covered.
var toolchainRegistry = map[ArchPair]ToolchainInfo{
	// Native builds (empty CrossCompilePrefix)
	{HostArchX86_64, db.ArchX86_64}:   {MakeArch: "x86"},
	{HostArchAARCH64, db.ArchAARCH64}: {MakeArch: "arm64"},
	// Cross-compilation builds
	{HostArchX86_64, db.ArchAARCH64}: {
		CrossCompilePrefix: "aarch64-linux-gnu-",
		MakeArch:           "arm64",
		ToolchainPkg:       "gcc-aarch64-linux-gnu",
	},
	{HostArchAARCH64, db.ArchX86_64}: {
		CrossCompilePrefix: "x86_64-linux-gnu-",
		MakeArch:           "x86",
		ToolchainPkg:       "gcc-x86-64-linux-gnu",
	},
}

// podmanPlatforms maps target architectures to Podman --platform values
var podmanPlatforms = map[db.TargetArch]string{
	db.ArchX86_64:  "linux/amd64",
	db.ArchAARCH64: "linux/arm64",
}

// qemuBinaryNames maps target architectures to QEMU static binary names
var qemuBinaryNames = map[db.TargetArch]string{
	db.ArchX86_64:  "qemu-x86_64-static",
	db.ArchAARCH64: "qemu-aarch64-static",
}

// qemuBinfmtNames maps target architectures to binfmt_misc registration names
var qemuBinfmtNames = map[db.TargetArch]string{
	db.ArchX86_64:  "qemu-x86_64",
	db.ArchAARCH64: "qemu-aarch64",
}

// DetectHostArch returns the architecture of the machine running ldfd
func DetectHostArch() HostArch {
	switch runtime.GOARCH {
	case "amd64":
		return HostArchX86_64
	case "arm64":
		return HostArchAARCH64
	default:
		// Best-effort fallback
		return HostArchX86_64
	}
}

// IsNativeBuild returns true when host and target architectures match
func IsNativeBuild(host HostArch, target db.TargetArch) bool {
	switch {
	case host == HostArchX86_64 && target == db.ArchX86_64:
		return true
	case host == HostArchAARCH64 && target == db.ArchAARCH64:
		return true
	default:
		return false
	}
}

// GetToolchain returns the ToolchainInfo for a given host→target pair.
// Returns a zero-value ToolchainInfo with empty CrossCompilePrefix for native builds.
// Returns an error for unsupported combinations.
func GetToolchain(host HostArch, target db.TargetArch) (ToolchainInfo, error) {
	pair := ArchPair{Host: host, Target: target}
	tc, ok := toolchainRegistry[pair]
	if !ok {
		return ToolchainInfo{}, fmt.Errorf("unsupported architecture pair: host=%s target=%s", host, target)
	}
	return tc, nil
}

// DetectQEMUSupport checks whether QEMU user-mode emulation is available
// for the given target architecture on the current host
func DetectQEMUSupport(target db.TargetArch) QEMUSupport {
	result := QEMUSupport{}

	binaryName, ok := qemuBinaryNames[target]
	if !ok {
		return result
	}

	// Check if QEMU binary exists in PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		result.Available = true
		result.QEMUBinary = path
	}

	// Check binfmt_misc registration
	binfmtName, ok := qemuBinfmtNames[target]
	if !ok {
		return result
	}
	binfmtPath := fmt.Sprintf("/proc/sys/fs/binfmt_misc/%s", binfmtName)
	if _, err := os.Stat(binfmtPath); err == nil {
		result.BinfmtRegistered = true
	}

	return result
}

// ContainerImageForArch returns an architecture-tagged container image name
// if it exists locally, otherwise falls back to the base image with :latest tag.
func ContainerImageForArch(baseImage string, target db.TargetArch) string {
	// Strip any existing tag from the base image
	base := baseImage
	if idx := strings.LastIndex(baseImage, ":"); idx > 0 {
		base = baseImage[:idx]
	}

	// Try architecture-specific image
	archImage := fmt.Sprintf("%s:%s", base, string(target))
	cmd := exec.CommandContext(context.Background(), "podman", "image", "exists", archImage)
	if cmd.Run() == nil {
		return archImage
	}

	// Fall back to the original image (or :latest)
	if strings.Contains(baseImage, ":") {
		return baseImage
	}
	return baseImage + ":latest"
}

// ValidateBuildEnvironment performs pre-flight checks for a build.
// Strategy priority: native > cross-compile > QEMU emulation.
// Returns a BuildEnvironment summary or an error explaining what is missing.
func ValidateBuildEnvironment(baseImage string, target db.TargetArch) (*BuildEnvironment, error) {
	host := DetectHostArch()

	env := &BuildEnvironment{
		HostArch:   host,
		TargetArch: target,
		IsNative:   IsNativeBuild(host, target),
	}

	// Resolve toolchain
	tc, err := GetToolchain(host, target)
	if err != nil {
		return nil, fmt.Errorf("no toolchain available: %w", err)
	}
	env.Toolchain = tc

	// Resolve container image
	env.ContainerImage = ContainerImageForArch(baseImage, target)

	// For native builds, we're done
	if env.IsNative {
		return env, nil
	}

	// Cross-compilation: toolchain is available (from registry).
	// The container image should contain the cross-compiler.
	// If the resolved image is the generic one, that's fine --
	// the toolchain is expected to be in the image.

	// Also check QEMU support as fallback metadata
	env.QEMUSupport = DetectQEMUSupport(target)

	// If QEMU binfmt is registered, we can use --platform for
	// running foreign-arch containers as an alternative strategy
	if env.QEMUSupport.BinfmtRegistered {
		if platform, ok := podmanPlatforms[target]; ok {
			env.PodmanPlatformFlag = platform
			env.UseQEMUEmulation = true
		}
	}

	return env, nil
}
