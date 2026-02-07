package build

import (
	"runtime"
	"testing"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

func TestDetectHostArch(t *testing.T) {
	host := DetectHostArch()

	switch runtime.GOARCH {
	case "amd64":
		if host != HostArchX86_64 {
			t.Errorf("expected HostArchX86_64 on amd64, got %s", host)
		}
	case "arm64":
		if host != HostArchAARCH64 {
			t.Errorf("expected HostArchAARCH64 on arm64, got %s", host)
		}
	default:
		// Fallback to x86_64
		if host != HostArchX86_64 {
			t.Errorf("expected HostArchX86_64 as fallback, got %s", host)
		}
	}
}

func TestIsNativeBuild(t *testing.T) {
	tests := []struct {
		name   string
		host   HostArch
		target db.TargetArch
		want   bool
	}{
		{"x86_64 native", HostArchX86_64, db.ArchX86_64, true},
		{"aarch64 native", HostArchAARCH64, db.ArchAARCH64, true},
		{"x86_64 to aarch64", HostArchX86_64, db.ArchAARCH64, false},
		{"aarch64 to x86_64", HostArchAARCH64, db.ArchX86_64, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNativeBuild(tt.host, tt.target)
			if got != tt.want {
				t.Errorf("IsNativeBuild(%s, %s) = %v, want %v", tt.host, tt.target, got, tt.want)
			}
		})
	}
}

func TestGetToolchain_Native(t *testing.T) {
	tc, err := GetToolchain(HostArchX86_64, db.ArchX86_64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.CrossCompilePrefix != "" {
		t.Errorf("expected empty CrossCompilePrefix for native build, got %q", tc.CrossCompilePrefix)
	}
	if tc.MakeArch != "x86" {
		t.Errorf("expected MakeArch=x86, got %q", tc.MakeArch)
	}

	tc, err = GetToolchain(HostArchAARCH64, db.ArchAARCH64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.CrossCompilePrefix != "" {
		t.Errorf("expected empty CrossCompilePrefix for native build, got %q", tc.CrossCompilePrefix)
	}
	if tc.MakeArch != "arm64" {
		t.Errorf("expected MakeArch=arm64, got %q", tc.MakeArch)
	}
}

func TestGetToolchain_CrossX86ToArm(t *testing.T) {
	tc, err := GetToolchain(HostArchX86_64, db.ArchAARCH64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.CrossCompilePrefix != "aarch64-linux-gnu-" {
		t.Errorf("expected CrossCompilePrefix=aarch64-linux-gnu-, got %q", tc.CrossCompilePrefix)
	}
	if tc.MakeArch != "arm64" {
		t.Errorf("expected MakeArch=arm64, got %q", tc.MakeArch)
	}
	if tc.ToolchainPkg != "gcc-aarch64-linux-gnu" {
		t.Errorf("expected ToolchainPkg=gcc-aarch64-linux-gnu, got %q", tc.ToolchainPkg)
	}
}

func TestGetToolchain_CrossArmToX86(t *testing.T) {
	tc, err := GetToolchain(HostArchAARCH64, db.ArchX86_64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.CrossCompilePrefix != "x86_64-linux-gnu-" {
		t.Errorf("expected CrossCompilePrefix=x86_64-linux-gnu-, got %q", tc.CrossCompilePrefix)
	}
	if tc.MakeArch != "x86" {
		t.Errorf("expected MakeArch=x86, got %q", tc.MakeArch)
	}
	if tc.ToolchainPkg != "gcc-x86-64-linux-gnu" {
		t.Errorf("expected ToolchainPkg=gcc-x86-64-linux-gnu, got %q", tc.ToolchainPkg)
	}
}

func TestGetToolchain_UnsupportedPair(t *testing.T) {
	_, err := GetToolchain("mips64", db.ArchX86_64)
	if err == nil {
		t.Error("expected error for unsupported host architecture")
	}
}

func TestValidateBuildEnvironment_Native(t *testing.T) {
	// Use the detected host arch to build a native test
	host := DetectHostArch()
	var target db.TargetArch
	if host == HostArchX86_64 {
		target = db.ArchX86_64
	} else {
		target = db.ArchAARCH64
	}

	env, err := ValidateBuildEnvironment(RuntimePodman, "ldf-builder:latest", target)
	if err != nil {
		t.Fatalf("unexpected error for native build: %v", err)
	}
	if !env.IsNative {
		t.Error("expected IsNative=true for native build")
	}
	if env.Toolchain.CrossCompilePrefix != "" {
		t.Errorf("expected empty CrossCompilePrefix for native build, got %q", env.Toolchain.CrossCompilePrefix)
	}
	if env.ContainerPlatformFlag != "" {
		t.Errorf("expected empty ContainerPlatformFlag for native build, got %q", env.ContainerPlatformFlag)
	}
	if env.UseQEMUEmulation {
		t.Error("expected UseQEMUEmulation=false for native build")
	}
}

func TestBuildEnvironmentPlatformFlag(t *testing.T) {
	// Cross-build: x86_64 -> aarch64 should get linux/arm64 platform flag
	// (only if QEMU is available, but toolchain should always resolve)
	host := HostArchX86_64
	target := db.ArchAARCH64

	tc, err := GetToolchain(host, target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.MakeArch != "arm64" {
		t.Errorf("expected MakeArch=arm64, got %q", tc.MakeArch)
	}

	// Check the container platform mapping
	platform, ok := containerPlatforms[db.ArchAARCH64]
	if !ok {
		t.Fatal("expected containerPlatforms entry for aarch64")
	}
	if platform != "linux/arm64" {
		t.Errorf("expected linux/arm64, got %q", platform)
	}

	platform, ok = containerPlatforms[db.ArchX86_64]
	if !ok {
		t.Fatal("expected containerPlatforms entry for x86_64")
	}
	if platform != "linux/amd64" {
		t.Errorf("expected linux/amd64, got %q", platform)
	}
}

func TestIsComponentCompatible(t *testing.T) {
	tests := []struct {
		name       string
		supported  []db.TargetArch
		targetArch db.TargetArch
		want       bool
	}{
		{
			name:       "empty list supports all",
			supported:  nil,
			targetArch: db.ArchX86_64,
			want:       true,
		},
		{
			name:       "matching architecture",
			supported:  []db.TargetArch{db.ArchX86_64, db.ArchAARCH64},
			targetArch: db.ArchAARCH64,
			want:       true,
		},
		{
			name:       "single matching architecture",
			supported:  []db.TargetArch{db.ArchX86_64},
			targetArch: db.ArchX86_64,
			want:       true,
		},
		{
			name:       "no match",
			supported:  []db.TargetArch{db.ArchAARCH64},
			targetArch: db.ArchX86_64,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &db.Component{SupportedArchitectures: tt.supported}
			got := isComponentCompatible(c, tt.targetArch)
			if got != tt.want {
				t.Errorf("isComponentCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}
