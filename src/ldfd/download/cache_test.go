package download

import (
	"testing"
)

func TestBuildCachePath(t *testing.T) {
	c := &Cache{}

	tests := []struct {
		name     string
		sourceID string
		version  string
		artifact string
		want     string
	}{
		{
			name:     "standard path",
			sourceID: "src-123",
			version:  "1.0.0",
			artifact: "distribution/owner/dist/components/src-123/1.0.0/linux-1.0.0.tar.xz",
			want:     "cache/artifacts/src-123/1.0.0/linux-1.0.0.tar.xz",
		},
		{
			name:     "git source path",
			sourceID: "git-456",
			version:  "2.5.1",
			artifact: "distribution/owner/dist/sources/git-456/2.5.1/archive.tar.gz",
			want:     "cache/artifacts/git-456/2.5.1/archive.tar.gz",
		},
		{
			name:     "no extension artifact",
			sourceID: "src-789",
			version:  "3.0",
			artifact: "distribution/owner/dist/components/src-789/3.0/blob",
			want:     "cache/artifacts/src-789/3.0/blob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.buildCachePath(tt.sourceID, tt.version, tt.artifact)
			if got != tt.want {
				t.Errorf("buildCachePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultCacheConfig(t *testing.T) {
	cfg := DefaultCacheConfig()
	if !cfg.Enabled {
		t.Error("expected default cache config to be enabled")
	}
	if cfg.MaxSizeGB != 0 {
		t.Errorf("expected default MaxSizeGB to be 0 (unlimited), got %d", cfg.MaxSizeGB)
	}
}
