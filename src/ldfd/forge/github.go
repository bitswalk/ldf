package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// GitHubProvider implements the Provider interface for GitHub repositories
type GitHubProvider struct {
	httpClient *http.Client
}

// NewGitHubProvider creates a new GitHub provider
func NewGitHubProvider(client *http.Client) *GitHubProvider {
	return &GitHubProvider{httpClient: client}
}

// Name returns the forge type
func (p *GitHubProvider) Name() ForgeType {
	return ForgeTypeGitHub
}

// Detect checks if the URL is a GitHub repository
func (p *GitHubProvider) Detect(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "github.com")
}

// ParseRepoInfo extracts owner and repo from a GitHub URL
func (p *GitHubProvider) ParseRepoInfo(url string) (*RepoInfo, error) {
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	var owner, repo string

	// Handle git@ URLs
	if strings.HasPrefix(url, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(url, "git@github.com:"), "/")
		if len(parts) >= 2 {
			owner, repo = parts[0], parts[1]
		}
	} else if strings.Contains(url, "github.com") {
		// Handle https URLs
		parts := strings.Split(url, "github.com/")
		if len(parts) >= 2 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 2 {
				owner, repo = pathParts[0], pathParts[1]
			}
		}
	}

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("unable to parse GitHub URL: %s", url)
	}

	return &RepoInfo{
		Owner:      owner,
		Repo:       repo,
		BaseURL:    fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		APIBaseURL: "https://api.github.com",
	}, nil
}

// DiscoverVersions fetches versions from GitHub Releases and Tags API
func (p *GitHubProvider) DiscoverVersions(ctx context.Context, url string) ([]DiscoveredVersion, error) {
	repoInfo, err := p.ParseRepoInfo(url)
	if err != nil {
		return nil, err
	}

	// Fetch releases from GitHub API
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases", repoInfo.APIBaseURL, repoInfo.Owner, repoInfo.Repo)
	versions, err := p.fetchReleases(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	// Also fetch tags in case some versions aren't releases
	tagsURL := fmt.Sprintf("%s/repos/%s/%s/tags", repoInfo.APIBaseURL, repoInfo.Owner, repoInfo.Repo)
	tagVersions, err := p.fetchTags(ctx, tagsURL)
	if err != nil {
		// Tags are optional, don't fail if we can't get them
		tagVersions = nil
	}

	// Merge releases and tags (releases take priority for metadata)
	versionMap := make(map[string]DiscoveredVersion)
	for _, v := range tagVersions {
		versionMap[v.Version] = v
	}
	for _, v := range versions {
		versionMap[v.Version] = v // Overwrite tags with release info
	}

	// Convert map to slice and sort
	result := make([]DiscoveredVersion, 0, len(versionMap))
	for _, v := range versionMap {
		result = append(result, v)
	}

	sortVersionsDescending(result)
	return result, nil
}

// fetchReleases fetches releases from GitHub API
func (p *GitHubProvider) fetchReleases(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("%s?page=%d&per_page=%d", apiURL, page, perPage)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "ldfd/1.0")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GitHub releases: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusForbidden {
				// Rate limited, return what we have
				break
			}
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		var releases []struct {
			TagName     string    `json:"tag_name"`
			Name        string    `json:"name"`
			PublishedAt time.Time `json:"published_at"`
			Prerelease  bool      `json:"prerelease"`
			Draft       bool      `json:"draft"`
			HTMLURL     string    `json:"html_url"`
		}

		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, fmt.Errorf("failed to parse GitHub releases: %w", err)
		}

		if len(releases) == 0 {
			break
		}

		for _, r := range releases {
			if r.Draft {
				continue
			}

			version := normalizeVersion(r.TagName)
			publishedAt := r.PublishedAt
			versionType := determineVersionType(version, r.Prerelease)

			allVersions = append(allVersions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				ReleaseDate:  &publishedAt,
				DownloadURL:  r.HTMLURL,
				IsStable:     !r.Prerelease && !isPrerelease(version),
				IsPrerelease: r.Prerelease,
			})
		}

		page++
		if len(releases) < perPage {
			break
		}
	}

	return allVersions, nil
}

// fetchTags fetches tags from GitHub API
func (p *GitHubProvider) fetchTags(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("%s?page=%d&per_page=%d", apiURL, page, perPage)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("User-Agent", "ldfd/1.0")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			break
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var tags []struct {
			Name string `json:"name"`
		}

		if err := json.Unmarshal(body, &tags); err != nil {
			return nil, err
		}

		if len(tags) == 0 {
			break
		}

		for _, t := range tags {
			version := normalizeVersion(t.Name)
			// Skip non-version tags
			if !isVersionString(version) {
				continue
			}

			isPre := isPrerelease(version)
			versionType := determineVersionType(version, false)

			allVersions = append(allVersions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				IsStable:     !isPre,
				IsPrerelease: isPre,
			})
		}

		page++
		if len(tags) < perPage {
			break
		}
	}

	return allVersions, nil
}

// DefaultURLTemplate returns the default URL template for GitHub
func (p *GitHubProvider) DefaultURLTemplate(repoInfo *RepoInfo) string {
	// GitHub's standard archive URL format
	return "{base_url}/archive/refs/tags/{tag}.tar.gz"
}

// DefaultVersionFilter calculates version filter from upstream releases
func (p *GitHubProvider) DefaultVersionFilter(ctx context.Context, repoInfo *RepoInfo) (string, error) {
	// Fetch releases to analyze prerelease patterns
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases", repoInfo.APIBaseURL, repoInfo.Owner, repoInfo.Repo)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?per_page=100", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return p.fallbackFilter(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.fallbackFilter(), nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return p.fallbackFilter(), nil
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	}

	if err := json.Unmarshal(body, &releases); err != nil {
		return p.fallbackFilter(), nil
	}

	// Build list of discovered versions with prerelease flag
	versions := make([]DiscoveredVersion, 0, len(releases))
	for _, r := range releases {
		if r.Draft {
			continue
		}
		versions = append(versions, DiscoveredVersion{
			Version:      normalizeVersion(r.TagName),
			IsPrerelease: r.Prerelease || isPrerelease(r.TagName),
		})
	}

	// Extract patterns from prereleases
	patterns := extractExcludePatterns(versions)

	if len(patterns) == 0 {
		// No prereleases found, no filtering needed
		return "", nil
	}

	return strings.Join(patterns, ","), nil
}

// fallbackFilter returns a safe default filter when API fails
func (p *GitHubProvider) fallbackFilter() string {
	return "!*-rc*,!*alpha*,!*beta*,!*-dev*,!*-pre*"
}

// GetDefaults returns both URL template and version filter for a URL
func (p *GitHubProvider) GetDefaults(ctx context.Context, url string) (*ForgeDefaults, error) {
	repoInfo, err := p.ParseRepoInfo(url)
	if err != nil {
		return nil, err
	}

	urlTemplate := p.DefaultURLTemplate(repoInfo)

	filter, err := p.DefaultVersionFilter(ctx, repoInfo)
	filterSource := "upstream"
	if err != nil || filter == p.fallbackFilter() {
		filterSource = "default"
	}

	return &ForgeDefaults{
		URLTemplate:   urlTemplate,
		VersionFilter: filter,
		FilterSource:  filterSource,
	}, nil
}

// sortVersionsDescending sorts versions in descending order
func sortVersionsDescending(versions []DiscoveredVersion) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})
}

// compareVersions compares two version strings
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 string
		if i < len(parts1) {
			p1 = parts1[i]
		}
		if i < len(parts2) {
			p2 = parts2[i]
		}

		n1, s1 := splitVersionPart(p1)
		n2, s2 := splitVersionPart(p2)

		if n1 != n2 {
			return n1 - n2
		}

		if s1 != s2 {
			if s1 == "" {
				return 1
			}
			if s2 == "" {
				return -1
			}
			return strings.Compare(s1, s2)
		}
	}

	return 0
}

// splitVersionPart splits a version part into numeric and suffix components
func splitVersionPart(part string) (num int, suffix string) {
	if part == "" {
		return 0, ""
	}

	i := 0
	for i < len(part) && part[i] >= '0' && part[i] <= '9' {
		num = num*10 + int(part[i]-'0')
		i++
	}

	if i < len(part) {
		suffix = part[i:]
	}

	return num, suffix
}
