package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitLabProvider implements the Provider interface for GitLab repositories
type GitLabProvider struct {
	httpClient *http.Client
}

// NewGitLabProvider creates a new GitLab provider
func NewGitLabProvider(client *http.Client) *GitLabProvider {
	return &GitLabProvider{httpClient: client}
}

// Name returns the forge type
func (p *GitLabProvider) Name() ForgeType {
	return ForgeTypeGitLab
}

// Detect checks if the URL is a GitLab repository
func (p *GitLabProvider) Detect(urlStr string) bool {
	lower := strings.ToLower(urlStr)

	// Check for gitlab.com
	if strings.Contains(lower, "gitlab.com") {
		return true
	}

	// Check for common self-hosted patterns
	if strings.Contains(lower, "gitlab.") {
		return true
	}

	return false
}

// ParseRepoInfo extracts repository information from a GitLab URL
func (p *GitLabProvider) ParseRepoInfo(urlStr string) (*RepoInfo, error) {
	urlStr = strings.TrimSuffix(urlStr, ".git")
	urlStr = strings.TrimSuffix(urlStr, "/")

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Extract path parts (owner/repo or group/subgroup/repo)
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("unable to parse GitLab URL: %s", urlStr)
	}

	// For GitLab, the path could be nested (group/subgroup/repo)
	// We treat everything but the last part as "owner" and last as "repo"
	owner := strings.Join(pathParts[:len(pathParts)-1], "/")
	repo := pathParts[len(pathParts)-1]

	baseURL := fmt.Sprintf("%s://%s/%s/%s", parsed.Scheme, parsed.Host, owner, repo)
	apiBaseURL := fmt.Sprintf("%s://%s/api/v4", parsed.Scheme, parsed.Host)

	return &RepoInfo{
		Owner:      owner,
		Repo:       repo,
		BaseURL:    baseURL,
		APIBaseURL: apiBaseURL,
	}, nil
}

// DiscoverVersions fetches versions from GitLab Releases and Tags API
func (p *GitLabProvider) DiscoverVersions(ctx context.Context, urlStr string) ([]DiscoveredVersion, error) {
	repoInfo, err := p.ParseRepoInfo(urlStr)
	if err != nil {
		return nil, err
	}

	// GitLab uses URL-encoded project path
	projectPath := url.PathEscape(fmt.Sprintf("%s/%s", repoInfo.Owner, repoInfo.Repo))

	// Fetch releases
	releasesURL := fmt.Sprintf("%s/projects/%s/releases", repoInfo.APIBaseURL, projectPath)
	versions, err := p.fetchReleases(ctx, releasesURL)
	if err != nil {
		// Try tags if releases fail
		versions = nil
	}

	// Fetch tags
	tagsURL := fmt.Sprintf("%s/projects/%s/repository/tags", repoInfo.APIBaseURL, projectPath)
	tagVersions, err := p.fetchTags(ctx, tagsURL)
	if err != nil {
		tagVersions = nil
	}

	// Merge releases and tags
	versionMap := make(map[string]DiscoveredVersion)
	for _, v := range tagVersions {
		versionMap[v.Version] = v
	}
	for _, v := range versions {
		versionMap[v.Version] = v
	}

	result := make([]DiscoveredVersion, 0, len(versionMap))
	for _, v := range versionMap {
		result = append(result, v)
	}

	sortVersionsDescending(result)
	return result, nil
}

// fetchReleases fetches releases from GitLab API
func (p *GitLabProvider) fetchReleases(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion
	page := 1
	perPage := 100

	for {
		reqURL := fmt.Sprintf("%s?page=%d&per_page=%d", apiURL, page, perPage)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "ldfd/1.0")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				// No releases, not an error
				return nil, nil
			}
			return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var releases []struct {
			TagName     string    `json:"tag_name"`
			Name        string    `json:"name"`
			ReleasedAt  time.Time `json:"released_at"`
			Upcoming    bool      `json:"upcoming_release"` // GitLab's equivalent of prerelease
			Description string    `json:"description"`
		}

		if err := json.Unmarshal(body, &releases); err != nil {
			return nil, err
		}

		if len(releases) == 0 {
			break
		}

		for _, r := range releases {
			version := normalizeVersion(r.TagName)
			releasedAt := r.ReleasedAt
			isPre := r.Upcoming || isPrerelease(version)
			versionType := determineVersionType(version, isPre)

			allVersions = append(allVersions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				ReleaseDate:  &releasedAt,
				IsStable:     !isPre,
				IsPrerelease: isPre,
			})
		}

		page++
		if len(releases) < perPage {
			break
		}
	}

	return allVersions, nil
}

// fetchTags fetches tags from GitLab API
func (p *GitLabProvider) fetchTags(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion
	page := 1
	perPage := 100

	for {
		reqURL := fmt.Sprintf("%s?page=%d&per_page=%d", apiURL, page, perPage)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

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
			Name   string `json:"name"`
			Commit struct {
				CreatedAt time.Time `json:"created_at"`
			} `json:"commit"`
		}

		if err := json.Unmarshal(body, &tags); err != nil {
			return nil, err
		}

		if len(tags) == 0 {
			break
		}

		for _, t := range tags {
			version := normalizeVersion(t.Name)
			if !isVersionString(version) {
				continue
			}

			isPre := isPrerelease(version)
			versionType := determineVersionType(version, false)
			createdAt := t.Commit.CreatedAt

			allVersions = append(allVersions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				ReleaseDate:  &createdAt,
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

// DefaultURLTemplate returns the default URL template for GitLab
func (p *GitLabProvider) DefaultURLTemplate(repoInfo *RepoInfo) string {
	// GitLab's archive URL format
	return "{base_url}/-/archive/{tag}/{repo}-{tag}.tar.gz"
}

// DefaultVersionFilter calculates version filter from upstream releases
func (p *GitLabProvider) DefaultVersionFilter(ctx context.Context, repoInfo *RepoInfo) (string, error) {
	projectPath := url.PathEscape(fmt.Sprintf("%s/%s", repoInfo.Owner, repoInfo.Repo))
	releasesURL := fmt.Sprintf("%s/projects/%s/releases?per_page=100", repoInfo.APIBaseURL, projectPath)

	req, err := http.NewRequestWithContext(ctx, "GET", releasesURL, nil)
	if err != nil {
		return p.fallbackFilter(), nil
	}

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
		TagName  string `json:"tag_name"`
		Upcoming bool   `json:"upcoming_release"`
	}

	if err := json.Unmarshal(body, &releases); err != nil {
		return p.fallbackFilter(), nil
	}

	versions := make([]DiscoveredVersion, 0, len(releases))
	for _, r := range releases {
		versions = append(versions, DiscoveredVersion{
			Version:      normalizeVersion(r.TagName),
			IsPrerelease: r.Upcoming || isPrerelease(r.TagName),
		})
	}

	patterns := extractExcludePatterns(versions)
	if len(patterns) == 0 {
		return "", nil
	}

	return strings.Join(patterns, ","), nil
}

// fallbackFilter returns a safe default filter
func (p *GitLabProvider) fallbackFilter() string {
	return "!*-rc*,!*alpha*,!*beta*,!*-dev*,!*-pre*"
}

// GetDefaults returns both URL template and version filter for a URL
func (p *GitLabProvider) GetDefaults(ctx context.Context, urlStr string) (*ForgeDefaults, error) {
	repoInfo, err := p.ParseRepoInfo(urlStr)
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
