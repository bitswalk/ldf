package forge

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/db"
)

// GenericProvider implements the Provider interface for generic HTTP sources
// This handles kernel.org-style directory listings and other HTTP-based sources
type GenericProvider struct {
	httpClient *http.Client
}

// NewGenericProvider creates a new generic provider
func NewGenericProvider(client *http.Client) *GenericProvider {
	return &GenericProvider{httpClient: client}
}

// Name returns the forge type
func (p *GenericProvider) Name() ForgeType {
	return ForgeTypeGeneric
}

// Detect always returns true as this is the fallback provider
func (p *GenericProvider) Detect(urlStr string) bool {
	// Generic provider is the fallback, always matches
	return true
}

// ParseRepoInfo extracts information from a generic URL
func (p *GenericProvider) ParseRepoInfo(urlStr string) (*RepoInfo, error) {
	urlStr = strings.TrimSuffix(urlStr, "/")

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// For generic sources, we don't have owner/repo concept
	// Use the host as "owner" and path as "repo" for template purposes
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	repo := "source"
	if len(pathParts) > 0 && pathParts[len(pathParts)-1] != "" {
		repo = pathParts[len(pathParts)-1]
	}

	return &RepoInfo{
		Owner:      parsed.Host,
		Repo:       repo,
		BaseURL:    urlStr,
		APIBaseURL: urlStr,
	}, nil
}

// DiscoverVersions fetches versions from HTTP directory listings
func (p *GenericProvider) DiscoverVersions(ctx context.Context, urlStr string) ([]DiscoveredVersion, error) {
	// Check if this looks like kernel.org
	if strings.Contains(strings.ToLower(urlStr), "kernel.org") {
		return p.discoverKernelOrgVersions(ctx, urlStr)
	}

	// Generic HTTP directory discovery
	return p.discoverHTTPDirectoryVersions(ctx, urlStr)
}

// discoverKernelOrgVersions discovers kernel versions from kernel.org
func (p *GenericProvider) discoverKernelOrgVersions(ctx context.Context, baseURL string) ([]DiscoveredVersion, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Fetch the base directory listing to find v{major}.x directories
	majorDirs, err := p.fetchKernelMajorDirectories(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch major directories: %w", err)
	}

	var allVersions []DiscoveredVersion

	// Fetch versions from each major directory
	for _, majorDir := range majorDirs {
		dirURL := fmt.Sprintf("%s/%s", baseURL, majorDir)
		versions, err := p.fetchKernelVersionsFromDirectory(ctx, dirURL)
		if err != nil {
			continue
		}
		allVersions = append(allVersions, versions...)
	}

	sortVersionsDescending(allVersions)
	return allVersions, nil
}

// fetchKernelMajorDirectories fetches the list of v{major}.x directories
func (p *GenericProvider) fetchKernelMajorDirectories(ctx context.Context, baseURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse directory listing for v{major}.x/ patterns
	dirPattern := regexp.MustCompile(`href="(v\d+\.x)/"`)
	matches := dirPattern.FindAllStringSubmatch(string(body), -1)

	var dirs []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			dirs = append(dirs, m[1])
			seen[m[1]] = true
		}
	}

	// Sort directories in descending order (v6.x before v5.x)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i] > dirs[j]
	})

	return dirs, nil
}

// fetchKernelVersionsFromDirectory fetches kernel versions from a v{major}.x directory
func (p *GenericProvider) fetchKernelVersionsFromDirectory(ctx context.Context, dirURL string) ([]DiscoveredVersion, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", dirURL+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse for linux-{version}.tar.xz patterns
	tarballPattern := regexp.MustCompile(`href="linux-(\d+\.\d+(?:\.\d+)?(?:-rc\d+)?)\.tar\.xz"`)
	matches := tarballPattern.FindAllStringSubmatch(string(body), -1)

	var versions []DiscoveredVersion
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			version := m[1]
			downloadURL := fmt.Sprintf("%s/linux-%s.tar.xz", dirURL, version)
			versionType := p.determineKernelVersionType(version)
			isPre := strings.Contains(version, "-rc")

			versions = append(versions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				DownloadURL:  downloadURL,
				IsStable:     !isPre,
				IsPrerelease: isPre,
			})
			seen[version] = true
		}
	}

	return versions, nil
}

// determineKernelVersionType determines the version type for kernel versions
func (p *GenericProvider) determineKernelVersionType(version string) db.VersionType {
	lower := strings.ToLower(version)

	// linux-next versions
	if strings.HasPrefix(lower, "next-") {
		return db.VersionTypeLinuxNext
	}

	// RC versions are mainline
	if strings.Contains(lower, "-rc") {
		return db.VersionTypeMainline
	}

	// Known LTS kernel series
	ltsVersions := []string{
		"6.12", "6.6", "6.1", "5.15", "5.10", "5.4", "4.19", "4.14",
	}

	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		majorMinor := parts[0] + "." + parts[1]
		for _, lts := range ltsVersions {
			if majorMinor == lts {
				return db.VersionTypeLongterm
			}
		}
	}

	return db.VersionTypeStable
}

// discoverHTTPDirectoryVersions discovers versions from a generic HTTP directory listing
func (p *GenericProvider) discoverHTTPDirectoryVersions(ctx context.Context, baseURL string) ([]DiscoveredVersion, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Try to extract version patterns from links
	linkPattern := regexp.MustCompile(`href="([^"]+)"`)
	matches := linkPattern.FindAllStringSubmatch(string(body), -1)

	var versions []DiscoveredVersion
	seen := make(map[string]bool)

	// Version extraction patterns
	versionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:^|[-_])v?(\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?)\.(tar\.(gz|xz|bz2)|zip)$`),
		regexp.MustCompile(`^v?(\d+\.\d+(?:\.\d+)?(?:-[a-zA-Z0-9]+)?)/?$`),
	}

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		link := m[1]

		for _, vp := range versionPatterns {
			if vm := vp.FindStringSubmatch(link); len(vm) > 1 {
				version := vm[1]
				if !seen[version] {
					downloadURL := baseURL
					if !strings.HasSuffix(downloadURL, "/") {
						downloadURL += "/"
					}
					downloadURL += link

					isPre := isPrerelease(version)
					versionType := db.VersionTypeStable
					if isPre {
						versionType = db.VersionTypeMainline
					}

					versions = append(versions, DiscoveredVersion{
						Version:      version,
						VersionType:  versionType,
						DownloadURL:  downloadURL,
						IsStable:     !isPre,
						IsPrerelease: isPre,
					})
					seen[version] = true
				}
				break
			}
		}
	}

	sortVersionsDescending(versions)
	return versions, nil
}

// DefaultURLTemplate returns the default URL template for generic sources
func (p *GenericProvider) DefaultURLTemplate(repoInfo *RepoInfo) string {
	// Check if this is kernel.org
	if strings.Contains(strings.ToLower(repoInfo.BaseURL), "kernel.org") {
		return "{base_url}/v{major_x}/linux-{version}.tar.xz"
	}

	// Generic template - user should customize
	return "{base_url}/{name}-{version}.tar.gz"
}

// DefaultVersionFilter returns the default version filter for generic sources
func (p *GenericProvider) DefaultVersionFilter(ctx context.Context, repoInfo *RepoInfo) (string, error) {
	// For kernel.org, use type-based filtering
	if strings.Contains(strings.ToLower(repoInfo.BaseURL), "kernel.org") {
		// Default to stable and longterm, exclude mainline (RC) and linux-next
		return "!*-rc*,!next-*", nil
	}

	// For generic sources, use common prerelease patterns
	return "!*-rc*,!*alpha*,!*beta*,!*-dev*,!*-pre*", nil
}

// GetDefaults returns both URL template and version filter for a URL
func (p *GenericProvider) GetDefaults(ctx context.Context, urlStr string) (*ForgeDefaults, error) {
	repoInfo, err := p.ParseRepoInfo(urlStr)
	if err != nil {
		return nil, err
	}

	urlTemplate := p.DefaultURLTemplate(repoInfo)
	filter, _ := p.DefaultVersionFilter(ctx, repoInfo)

	return &ForgeDefaults{
		URLTemplate:   urlTemplate,
		VersionFilter: filter,
		FilterSource:  "default", // Generic always uses defaults
	}, nil
}
