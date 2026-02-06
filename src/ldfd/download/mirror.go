package download

import (
	"net/http"
	"net/url"
	"os"
	urlpath "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// MirrorConfig holds global mirror/proxy settings
type MirrorConfig struct {
	ProxyURL  string // HTTP(S) proxy URL for all downloads
	LocalPath string // Local directory for offline mirror
}

// DefaultMirrorConfig returns default mirror configuration (everything disabled)
func DefaultMirrorConfig() MirrorConfig {
	return MirrorConfig{}
}

// MirrorResolver resolves download URLs through configured mirrors and proxies.
// Mirror resolution happens at the worker level so the original URL is preserved
// on the job for provenance, and workers can fall back to the original if a mirror fails.
type MirrorResolver struct {
	mirrors []db.MirrorConfigEntry
	config  MirrorConfig
}

// NewMirrorResolver creates a new mirror resolver with the given mirrors and config
func NewMirrorResolver(mirrors []db.MirrorConfigEntry, cfg MirrorConfig) *MirrorResolver {
	// Sort mirrors by priority (lower = tried first)
	sorted := make([]db.MirrorConfigEntry, len(mirrors))
	copy(sorted, mirrors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	return &MirrorResolver{
		mirrors: sorted,
		config:  cfg,
	}
}

// ResolveURL takes an original URL and returns the mirrored URL.
// Tries mirrors in priority order, returning the first prefix match.
// Returns the original URL if no mirror matches.
func (r *MirrorResolver) ResolveURL(originalURL string) string {
	for _, m := range r.mirrors {
		if !m.Enabled {
			continue
		}
		if strings.HasPrefix(originalURL, m.URLPrefix) {
			mirrored := m.MirrorURL + strings.TrimPrefix(originalURL, m.URLPrefix)
			log.Debug("Mirror URL resolved",
				"original", originalURL,
				"mirror", m.Name,
				"resolved", mirrored)
			return mirrored
		}
	}
	return originalURL
}

// ResolveLocalPath checks if the artifact exists in the local mirror directory.
// Returns the local file path if found, empty string otherwise.
// Checks both flat layout ({LocalPath}/{filename}) and structured layout
// ({LocalPath}/{sourceID}/{version}/{filename}).
func (r *MirrorResolver) ResolveLocalPath(originalURL, sourceID, version string) string {
	if r.config.LocalPath == "" {
		return ""
	}

	filename := urlpath.Base(originalURL)
	if filename == "" || filename == "." {
		return ""
	}

	// Try structured path first: {LocalPath}/{sourceID}/{version}/{filename}
	structuredPath := filepath.Join(r.config.LocalPath, sourceID, version, filename)
	if _, err := os.Stat(structuredPath); err == nil {
		log.Debug("Local mirror hit (structured)", "path", structuredPath)
		return structuredPath
	}

	// Try flat path: {LocalPath}/{filename}
	flatPath := filepath.Join(r.config.LocalPath, filename)
	if _, err := os.Stat(flatPath); err == nil {
		log.Debug("Local mirror hit (flat)", "path", flatPath)
		return flatPath
	}

	return ""
}

// GetHTTPTransport returns an http.Transport configured with proxy settings.
// Returns nil if no proxy is configured.
func (r *MirrorResolver) GetHTTPTransport() *http.Transport {
	if r.config.ProxyURL == "" {
		return nil
	}

	proxyURL, err := url.Parse(r.config.ProxyURL)
	if err != nil {
		log.Warn("Invalid proxy URL", "url", r.config.ProxyURL, "error", err)
		return nil
	}

	return &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
}

// HasMirrors returns true if any mirrors are configured
func (r *MirrorResolver) HasMirrors() bool {
	return len(r.mirrors) > 0
}

// HasLocalPath returns true if a local mirror path is configured
func (r *MirrorResolver) HasLocalPath() bool {
	return r.config.LocalPath != ""
}

// HasProxy returns true if a proxy URL is configured
func (r *MirrorResolver) HasProxy() bool {
	return r.config.ProxyURL != ""
}
