package build

import (
	"testing"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

func TestGetToolchainDeps_GCC_Native(t *testing.T) {
	deps := GetToolchainDeps(db.ToolchainGCC, "")
	if len(deps.Compiler) != 3 {
		t.Fatalf("expected 3 GCC compiler deps, got %d", len(deps.Compiler))
	}
	// Native GCC: unprefixed binaries
	expected := []string{"gcc", "ld", "ar"}
	for i, bin := range expected {
		if deps.Compiler[i] != bin {
			t.Errorf("compiler[%d]: expected %q, got %q", i, bin, deps.Compiler[i])
		}
	}
	if len(deps.Common) != 4 {
		t.Errorf("expected 4 common deps, got %d", len(deps.Common))
	}
}

func TestGetToolchainDeps_GCC_Cross(t *testing.T) {
	deps := GetToolchainDeps(db.ToolchainGCC, "aarch64-linux-gnu-")
	expected := []string{"aarch64-linux-gnu-gcc", "aarch64-linux-gnu-ld", "aarch64-linux-gnu-ar"}
	for i, bin := range expected {
		if deps.Compiler[i] != bin {
			t.Errorf("compiler[%d]: expected %q, got %q", i, bin, deps.Compiler[i])
		}
	}
}

func TestGetToolchainDeps_LLVM(t *testing.T) {
	deps := GetToolchainDeps(db.ToolchainLLVM, "")
	if len(deps.Compiler) != 7 {
		t.Fatalf("expected 7 LLVM compiler deps, got %d", len(deps.Compiler))
	}
	// LLVM binaries are never prefixed
	if deps.Compiler[0] != "clang" {
		t.Errorf("expected first LLVM dep to be clang, got %q", deps.Compiler[0])
	}
	if deps.Compiler[1] != "ld.lld" {
		t.Errorf("expected second LLVM dep to be ld.lld, got %q", deps.Compiler[1])
	}
}

func TestGetToolchainDeps_All(t *testing.T) {
	deps := GetToolchainDeps(db.ToolchainGCC, "")
	all := deps.All()
	if len(all) != len(deps.Compiler)+len(deps.Common) {
		t.Errorf("All() length mismatch: got %d, expected %d", len(all), len(deps.Compiler)+len(deps.Common))
	}
}

func TestToolchainEnvVars_GCC_Native(t *testing.T) {
	env := ToolchainEnvVars(db.ToolchainGCC, "")
	if len(env) != 0 {
		t.Errorf("expected empty env for native GCC, got %d entries", len(env))
	}
}

func TestToolchainEnvVars_GCC_Cross(t *testing.T) {
	env := ToolchainEnvVars(db.ToolchainGCC, "aarch64-linux-gnu-")
	if env["CROSS_COMPILE"] != "aarch64-linux-gnu-" {
		t.Errorf("expected CROSS_COMPILE=aarch64-linux-gnu-, got %q", env["CROSS_COMPILE"])
	}
	if len(env) != 1 {
		t.Errorf("expected 1 env var for cross GCC, got %d", len(env))
	}
}

func TestToolchainEnvVars_LLVM(t *testing.T) {
	env := ToolchainEnvVars(db.ToolchainLLVM, "")
	if env["LLVM"] != "1" {
		t.Errorf("expected LLVM=1, got %q", env["LLVM"])
	}
	if env["CC"] != "clang" {
		t.Errorf("expected CC=clang, got %q", env["CC"])
	}
	if env["LD"] != "ld.lld" {
		t.Errorf("expected LD=ld.lld, got %q", env["LD"])
	}
	if env["HOSTCC"] != "clang" {
		t.Errorf("expected HOSTCC=clang, got %q", env["HOSTCC"])
	}
	if _, ok := env["CROSS_COMPILE"]; ok {
		t.Error("expected no CROSS_COMPILE for native LLVM build")
	}
}

func TestToolchainEnvVars_LLVM_Cross(t *testing.T) {
	env := ToolchainEnvVars(db.ToolchainLLVM, "aarch64-linux-gnu-")
	if env["LLVM"] != "1" {
		t.Errorf("expected LLVM=1, got %q", env["LLVM"])
	}
	if env["CROSS_COMPILE"] != "aarch64-linux-gnu-" {
		t.Errorf("expected CROSS_COMPILE=aarch64-linux-gnu-, got %q", env["CROSS_COMPILE"])
	}
	if env["CC"] != "clang" {
		t.Errorf("expected CC=clang, got %q", env["CC"])
	}
}

func TestResolveToolchain(t *testing.T) {
	tests := []struct {
		input    string
		expected db.ToolchainType
	}{
		{"", db.ToolchainGCC},
		{"gcc", db.ToolchainGCC},
		{"llvm", db.ToolchainLLVM},
		{"invalid", db.ToolchainGCC},
	}

	for _, tt := range tests {
		core := &db.CoreConfig{Toolchain: tt.input}
		result := db.ResolveToolchain(core)
		if result != tt.expected {
			t.Errorf("ResolveToolchain(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
