package build

import (
	"os/exec"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// ToolchainDeps lists the required binaries for a given build toolchain.
type ToolchainDeps struct {
	Compiler []string // compiler/linker binaries (toolchain-specific)
	Common   []string // common build utilities required by all toolchains
}

// All returns all required binaries (compiler + common).
func (d ToolchainDeps) All() []string {
	return append(d.Compiler, d.Common...)
}

// commonBuildDeps are build utilities required regardless of toolchain choice.
var commonBuildDeps = []string{"make", "bc", "flex", "bison"}

// GetToolchainDeps returns the required binaries for the given toolchain type
// and cross-compile prefix. For GCC the compiler binaries are prefixed with
// crossPrefix (empty for native builds). For LLVM the binaries are unprefixed
// since LLVM uses CROSS_COMPILE only for the assembler.
func GetToolchainDeps(toolchain db.ToolchainType, crossPrefix string) ToolchainDeps {
	switch toolchain {
	case db.ToolchainLLVM:
		return ToolchainDeps{
			Compiler: []string{"clang", "ld.lld", "llvm-ar", "llvm-nm", "llvm-strip", "llvm-objcopy", "llvm-objdump"},
			Common:   commonBuildDeps,
		}
	default: // GCC
		return ToolchainDeps{
			Compiler: []string{crossPrefix + "gcc", crossPrefix + "ld", crossPrefix + "ar"},
			Common:   commonBuildDeps,
		}
	}
}

// ValidateToolchainAvailability checks that all required toolchain binaries
// are available in the system PATH. Returns a list of missing binaries.
func ValidateToolchainAvailability(deps ToolchainDeps) []string {
	var missing []string
	for _, bin := range deps.All() {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	return missing
}

// containsCat checks if a component's categories slice contains the given category.
func containsCat(categories []string, cat string) bool {
	for _, c := range categories {
		if c == cat {
			return true
		}
	}
	return false
}

// ToolchainEnvVars returns the environment variables to pass to make for the
// given toolchain type and cross-compile prefix. These are used both in
// container and direct execution modes.
func ToolchainEnvVars(toolchain db.ToolchainType, crossPrefix string) map[string]string {
	switch toolchain {
	case db.ToolchainLLVM:
		env := map[string]string{
			"LLVM":    "1",
			"CC":      "clang",
			"LD":      "ld.lld",
			"AR":      "llvm-ar",
			"NM":      "llvm-nm",
			"STRIP":   "llvm-strip",
			"OBJCOPY": "llvm-objcopy",
			"OBJDUMP": "llvm-objdump",
			"HOSTCC":  "clang",
			"HOSTCXX": "clang++",
			"HOSTAR":  "llvm-ar",
			"HOSTLD":  "ld.lld",
		}
		if crossPrefix != "" {
			env["CROSS_COMPILE"] = crossPrefix
		}
		return env
	default: // GCC
		env := map[string]string{}
		if crossPrefix != "" {
			env["CROSS_COMPILE"] = crossPrefix
		}
		return env
	}
}
