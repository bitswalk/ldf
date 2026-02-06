package download

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

func TestMirrorResolver_ResolveURL_NoMirrors(t *testing.T) {
	r := NewMirrorResolver(nil, DefaultMirrorConfig())
	url := "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.tar.xz"
	got := r.ResolveURL(url)
	if got != url {
		t.Errorf("expected original URL, got %q", got)
	}
}

func TestMirrorResolver_ResolveURL_PrefixMatch(t *testing.T) {
	mirrors := []db.MirrorConfigEntry{
		{
			ID:        "m1",
			Name:      "Local mirror",
			URLPrefix: "https://cdn.kernel.org/pub/",
			MirrorURL: "https://mirror.local/kernel/",
			Priority:  0,
			Enabled:   true,
		},
	}
	r := NewMirrorResolver(mirrors, DefaultMirrorConfig())

	got := r.ResolveURL("https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.tar.xz")
	want := "https://mirror.local/kernel/linux/kernel/v6.x/linux-6.12.tar.xz"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMirrorResolver_ResolveURL_NoMatch(t *testing.T) {
	mirrors := []db.MirrorConfigEntry{
		{
			ID:        "m1",
			Name:      "GitHub mirror",
			URLPrefix: "https://github.com/",
			MirrorURL: "https://gh-mirror.local/",
			Priority:  0,
			Enabled:   true,
		},
	}
	r := NewMirrorResolver(mirrors, DefaultMirrorConfig())

	url := "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.tar.xz"
	got := r.ResolveURL(url)
	if got != url {
		t.Errorf("expected original URL when no prefix matches, got %q", got)
	}
}

func TestMirrorResolver_ResolveURL_DisabledMirror(t *testing.T) {
	mirrors := []db.MirrorConfigEntry{
		{
			ID:        "m1",
			Name:      "Disabled",
			URLPrefix: "https://cdn.kernel.org/pub/",
			MirrorURL: "https://should-not-match.local/",
			Priority:  0,
			Enabled:   false,
		},
	}
	r := NewMirrorResolver(mirrors, DefaultMirrorConfig())

	url := "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.tar.xz"
	got := r.ResolveURL(url)
	if got != url {
		t.Errorf("disabled mirror should not match, got %q", got)
	}
}

func TestMirrorResolver_ResolveURL_PriorityOrder(t *testing.T) {
	mirrors := []db.MirrorConfigEntry{
		{
			ID:        "m2",
			Name:      "Low priority",
			URLPrefix: "https://cdn.kernel.org/",
			MirrorURL: "https://low.local/",
			Priority:  10,
			Enabled:   true,
		},
		{
			ID:        "m1",
			Name:      "High priority",
			URLPrefix: "https://cdn.kernel.org/",
			MirrorURL: "https://high.local/",
			Priority:  1,
			Enabled:   true,
		},
	}
	r := NewMirrorResolver(mirrors, DefaultMirrorConfig())

	got := r.ResolveURL("https://cdn.kernel.org/pub/test.tar.gz")
	want := "https://high.local/pub/test.tar.gz"
	if got != want {
		t.Errorf("expected higher priority mirror first, got %q, want %q", got, want)
	}
}

func TestMirrorResolver_ResolveLocalPath_Structured(t *testing.T) {
	tmpDir := t.TempDir()

	// Create structured path
	structDir := filepath.Join(tmpDir, "src-123", "1.0.0")
	if err := os.MkdirAll(structDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(structDir, "linux-1.0.0.tar.xz"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := MirrorConfig{LocalPath: tmpDir}
	r := NewMirrorResolver(nil, cfg)

	got := r.ResolveLocalPath("https://example.com/linux-1.0.0.tar.xz", "src-123", "1.0.0")
	if got == "" {
		t.Fatal("expected local path to be found")
	}
	if filepath.Base(got) != "linux-1.0.0.tar.xz" {
		t.Errorf("expected filename linux-1.0.0.tar.xz, got %q", filepath.Base(got))
	}
}

func TestMirrorResolver_ResolveLocalPath_Flat(t *testing.T) {
	tmpDir := t.TempDir()

	// Create flat path
	if err := os.WriteFile(filepath.Join(tmpDir, "linux-1.0.0.tar.xz"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := MirrorConfig{LocalPath: tmpDir}
	r := NewMirrorResolver(nil, cfg)

	got := r.ResolveLocalPath("https://example.com/linux-1.0.0.tar.xz", "src-123", "1.0.0")
	if got == "" {
		t.Fatal("expected local path to be found")
	}
}

func TestMirrorResolver_ResolveLocalPath_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := MirrorConfig{LocalPath: tmpDir}
	r := NewMirrorResolver(nil, cfg)

	got := r.ResolveLocalPath("https://example.com/nonexistent.tar.xz", "src-123", "1.0.0")
	if got != "" {
		t.Errorf("expected empty string for missing file, got %q", got)
	}
}

func TestMirrorResolver_ResolveLocalPath_NoLocalPath(t *testing.T) {
	r := NewMirrorResolver(nil, DefaultMirrorConfig())
	got := r.ResolveLocalPath("https://example.com/file.tar.xz", "src-123", "1.0.0")
	if got != "" {
		t.Errorf("expected empty string when no local path configured, got %q", got)
	}
}

func TestMirrorResolver_GetHTTPTransport_NoProxy(t *testing.T) {
	r := NewMirrorResolver(nil, DefaultMirrorConfig())
	tr := r.GetHTTPTransport()
	if tr != nil {
		t.Error("expected nil transport when no proxy configured")
	}
}

func TestMirrorResolver_GetHTTPTransport_WithProxy(t *testing.T) {
	cfg := MirrorConfig{ProxyURL: "http://proxy.local:8080"}
	r := NewMirrorResolver(nil, cfg)
	tr := r.GetHTTPTransport()
	if tr == nil {
		t.Fatal("expected non-nil transport when proxy configured")
	}
}

func TestMirrorResolver_HasMethods(t *testing.T) {
	// No config
	r := NewMirrorResolver(nil, DefaultMirrorConfig())
	if r.HasMirrors() {
		t.Error("expected HasMirrors=false")
	}
	if r.HasLocalPath() {
		t.Error("expected HasLocalPath=false")
	}
	if r.HasProxy() {
		t.Error("expected HasProxy=false")
	}

	// With config
	mirrors := []db.MirrorConfigEntry{{ID: "m1", Enabled: true}}
	cfg := MirrorConfig{ProxyURL: "http://proxy:8080", LocalPath: "/mirror"}
	r2 := NewMirrorResolver(mirrors, cfg)
	if !r2.HasMirrors() {
		t.Error("expected HasMirrors=true")
	}
	if !r2.HasLocalPath() {
		t.Error("expected HasLocalPath=true")
	}
	if !r2.HasProxy() {
		t.Error("expected HasProxy=true")
	}
}
