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

// GiteaProvider implements the Provider interface for Gitea repositories
// This is the base implementation used by Gitea, Codeberg, and Forgejo
type GiteaProvider struct {
	httpClient *http.Client
	forgeType  ForgeType
	detectFunc func(string) bool
}

// NewGiteaProvider creates a new Gitea provider
func NewGiteaProvider(client *http.Client) *GiteaProvider {
	return &GiteaProvider{
		httpClient: client,
		forgeType:  ForgeTypeGitea,
		detectFunc: func(urlStr string) bool {
			lower := strings.ToLower(urlStr)
			// Match gitea.* domains but not codeberg.org or forgejo.*
			if strings.Contains(lower, "codeberg.org") || strings.Contains(lower, "forgejo.") {
				return false
			}
			return strings.Contains(lower, "gitea.")
		},
	}
}

// NewCodebergProvider creates a provider for Codeberg.org
func NewCodebergProvider(client *http.Client) *GiteaProvider {
	return &GiteaProvider{
		httpClient: client,
		forgeType:  ForgeTypeCodeberg,
		detectFunc: func(urlStr string) bool {
			lower := strings.ToLower(urlStr)
			return strings.Contains(lower, "codeberg.org")
		},
	}
}

// NewForgejoProvider creates a provider for Forgejo instances
func NewForgejoProvider(client *http.Client) *GiteaProvider {
	return &GiteaProvider{
		httpClient: client,
		forgeType:  ForgeTypeForgejo,
		detectFunc: func(urlStr string) bool {
			lower := strings.ToLower(urlStr)
			return strings.Contains(lower, "forgejo.")
		},
	}
}

// Name returns the forge type
func (p *GiteaProvider) Name() ForgeType {
	return p.forgeType
}

// Detect checks if the URL matches this provider
func (p *GiteaProvider) Detect(urlStr string) bool {
	return p.detectFunc(urlStr)
}

// ParseRepoInfo extracts repository information from a Gitea/Codeberg/Forgejo URL
func (p *GiteaProvider) ParseRepoInfo(urlStr string) (*RepoInfo, error) {
	urlStr = strings.TrimSuffix(urlStr, ".git")
	urlStr = strings.TrimSuffix(urlStr, "/")

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("unable to parse URL: %s", urlStr)
	}

	owner := pathParts[0]
	repo := pathParts[1]

	baseURL := fmt.Sprintf("%s://%s/%s/%s", parsed.Scheme, parsed.Host, owner, repo)
	apiBaseURL := fmt.Sprintf("%s://%s/api/v1", parsed.Scheme, parsed.Host)

	return &RepoInfo{
		Owner:      owner,
		Repo:       repo,
		BaseURL:    baseURL,
		APIBaseURL: apiBaseURL,
	}, nil
}

// DiscoverVersions fetches versions from Gitea Releases and Tags API
func (p *GiteaProvider) DiscoverVersions(ctx context.Context, urlStr string) ([]DiscoveredVersion, error) {
	repoInfo, err := p.ParseRepoInfo(urlStr)
	if err != nil {
		return nil, err
	}

	// Fetch releases
	releasesURL := fmt.Sprintf("%s/repos/%s/%s/releases", repoInfo.APIBaseURL, repoInfo.Owner, repoInfo.Repo)
	versions, err := p.fetchReleases(ctx, releasesURL)
	if err != nil {
		versions = nil
	}

	// Fetch tags
	tagsURL := fmt.Sprintf("%s/repos/%s/%s/tags", repoInfo.APIBaseURL, repoInfo.Owner, repoInfo.Repo)
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

// fetchReleases fetches releases from Gitea API
func (p *GiteaProvider) fetchReleases(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion
	page := 1
	perPage := 50 // Gitea default limit

	for {
		reqURL := fmt.Sprintf("%s?page=%d&limit=%d", apiURL, page, perPage)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "ldfd/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				return nil, nil
			}
			return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
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
			return nil, err
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
			isPre := r.Prerelease || isPrerelease(version)
			versionType := determineVersionType(version, isPre)

			allVersions = append(allVersions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				ReleaseDate:  &publishedAt,
				DownloadURL:  r.HTMLURL,
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

// fetchTags fetches tags from Gitea API
func (p *GiteaProvider) fetchTags(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
	var allVersions []DiscoveredVersion
	page := 1
	perPage := 50

	for {
		reqURL := fmt.Sprintf("%s?page=%d&limit=%d", apiURL, page, perPage)
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "ldfd/1.0")
		req.Header.Set("Accept", "application/json")

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
				Created time.Time `json:"created"`
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
			created := t.Commit.Created

			allVersions = append(allVersions, DiscoveredVersion{
				Version:      version,
				VersionType:  versionType,
				ReleaseDate:  &created,
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

// DefaultURLTemplate returns the default URL template for Gitea-based forges
func (p *GiteaProvider) DefaultURLTemplate(repoInfo *RepoInfo) string {
	// Gitea/Codeberg/Forgejo archive URL format
	return "{base_url}/archive/{tag}.tar.gz"
}

// DefaultVersionFilter calculates version filter from upstream releases
func (p *GiteaProvider) DefaultVersionFilter(ctx context.Context, repoInfo *RepoInfo) (string, error) {
	releasesURL := fmt.Sprintf("%s/repos/%s/%s/releases?limit=100", repoInfo.APIBaseURL, repoInfo.Owner, repoInfo.Repo)

	req, err := http.NewRequestWithContext(ctx, "GET", releasesURL, nil)
	if err != nil {
		return p.fallbackFilter(), nil
	}

	req.Header.Set("User-Agent", "ldfd/1.0")
	req.Header.Set("Accept", "application/json")

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

	patterns := extractExcludePatterns(versions)
	if len(patterns) == 0 {
		return "", nil
	}

	return strings.Join(patterns, ","), nil
}

// fallbackFilter returns a safe default filter
func (p *GiteaProvider) fallbackFilter() string {
	return "!*-rc*,!*alpha*,!*beta*,!*-dev*,!*-pre*"
}

// GetDefaults returns both URL template and version filter for a URL
func (p *GiteaProvider) GetDefaults(ctx context.Context, urlStr string) (*ForgeDefaults, error) {
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
