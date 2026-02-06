package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/spf13/viper"
)

// VersionDiscovery handles discovering available versions from upstream sources
type VersionDiscovery struct {
	httpClient    *http.Client
	versionRepo   *db.SourceVersionRepository
	componentRepo *db.ComponentRepository
	sourceRepo    *db.SourceRepository
}

// DiscoveredVersion represents a version found from an upstream source
type DiscoveredVersion struct {
	Version     string
	VersionType db.VersionType
	ReleaseDate *time.Time
	DownloadURL string
	IsStable    bool
}

// NewVersionDiscovery creates a new version discovery service
func NewVersionDiscovery(versionRepo *db.SourceVersionRepository, componentRepo *db.ComponentRepository, sourceRepo *db.SourceRepository) *VersionDiscovery {
	return &VersionDiscovery{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		versionRepo:   versionRepo,
		componentRepo: componentRepo,
		sourceRepo:    sourceRepo,
	}
}

// GetVersionRepo returns the version repository for external use
func (d *VersionDiscovery) GetVersionRepo() *db.SourceVersionRepository {
	return d.versionRepo
}

// DiscoveryMethod represents the method used to discover versions
type DiscoveryMethod string

const (
	DiscoveryMethodGitHub        DiscoveryMethod = "github"
	DiscoveryMethodKernelOrg     DiscoveryMethod = "kernel.org"
	DiscoveryMethodHTTPDirectory DiscoveryMethod = "http-directory"
)

// DetectDiscoveryMethod determines the discovery method from the URL
func (d *VersionDiscovery) DetectDiscoveryMethod(url string) DiscoveryMethod {
	lowerURL := strings.ToLower(url)

	if strings.Contains(lowerURL, "github.com") {
		return DiscoveryMethodGitHub
	}
	if strings.Contains(lowerURL, "kernel.org") {
		return DiscoveryMethodKernelOrg
	}

	return DiscoveryMethodHTTPDirectory
}

// DiscoverVersions discovers available versions from an upstream source
func (d *VersionDiscovery) DiscoverVersions(ctx context.Context, source *db.UpstreamSource) ([]DiscoveredVersion, error) {
	method := d.DetectDiscoveryMethod(source.URL)

	switch method {
	case DiscoveryMethodGitHub:
		return d.discoverGitHubVersions(ctx, source.URL)
	case DiscoveryMethodKernelOrg:
		return d.discoverKernelOrgVersions(ctx, source.URL)
	default:
		return d.discoverHTTPDirectoryVersions(ctx, source.URL)
	}
}

// discoverGitHubVersions fetches versions from GitHub Releases API
func (d *VersionDiscovery) discoverGitHubVersions(ctx context.Context, repoURL string) ([]DiscoveredVersion, error) {
	// Extract owner/repo from URL
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	// Fetch releases from GitHub API
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	versions, err := d.fetchGitHubReleases(ctx, apiURL)
	if err != nil {
		return nil, err
	}

	// Also fetch tags in case some versions aren't releases
	tagsURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", owner, repo)
	tagVersions, err := d.fetchGitHubTags(ctx, tagsURL)
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

// fetchGitHubReleases fetches releases from GitHub API
func (d *VersionDiscovery) fetchGitHubReleases(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
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

		resp, err := d.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch GitHub releases: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
				// Rate limited - log details and return error if no versions collected yet
				remaining := resp.Header.Get("X-RateLimit-Remaining")
				resetTime := resp.Header.Get("X-RateLimit-Reset")
				log.Warn("GitHub API rate limit hit",
					"status", resp.StatusCode,
					"remaining", remaining,
					"reset", resetTime,
					"url", url,
					"versions_collected", len(allVersions))
				if len(allVersions) == 0 {
					return nil, fmt.Errorf("GitHub API rate limit exceeded (status %d), no versions could be fetched", resp.StatusCode)
				}
				// Return what we have if we collected some versions
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
			versionType := determineGitHubVersionType(version, r.Prerelease)

			allVersions = append(allVersions, DiscoveredVersion{
				Version:     version,
				VersionType: versionType,
				ReleaseDate: &publishedAt,
				DownloadURL: r.HTMLURL,
				IsStable:    !r.Prerelease && !isPrerelease(version),
			})
		}

		page++
		if len(releases) < perPage {
			break
		}
	}

	return allVersions, nil
}

// fetchGitHubTags fetches tags from GitHub API
func (d *VersionDiscovery) fetchGitHubTags(ctx context.Context, apiURL string) ([]DiscoveredVersion, error) {
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

		resp, err := d.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
				// Rate limited - log and return what we have (tags are supplementary)
				remaining := resp.Header.Get("X-RateLimit-Remaining")
				resetTime := resp.Header.Get("X-RateLimit-Reset")
				log.Warn("GitHub API rate limit hit while fetching tags",
					"status", resp.StatusCode,
					"remaining", remaining,
					"reset", resetTime,
					"url", url,
					"tags_collected", len(allVersions))
			}
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

			versionType := determineGitHubVersionType(version, false)

			allVersions = append(allVersions, DiscoveredVersion{
				Version:     version,
				VersionType: versionType,
				IsStable:    !isPrerelease(version),
			})
		}

		page++
		if len(tags) < perPage {
			break
		}
	}

	return allVersions, nil
}

// discoverKernelOrgVersions discovers kernel versions from kernel.org
func (d *VersionDiscovery) discoverKernelOrgVersions(ctx context.Context, baseURL string) ([]DiscoveredVersion, error) {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Fetch the base directory listing to find v{major}.x directories
	majorDirs, err := d.fetchKernelMajorDirectories(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch major directories: %w", err)
	}

	var allVersions []DiscoveredVersion

	// Fetch versions from each major directory
	for _, majorDir := range majorDirs {
		dirURL := fmt.Sprintf("%s/%s", baseURL, majorDir)
		versions, err := d.fetchKernelVersionsFromDirectory(ctx, dirURL)
		if err != nil {
			// Log but continue with other directories
			continue
		}
		allVersions = append(allVersions, versions...)
	}

	sortVersionsDescending(allVersions)
	return allVersions, nil
}

// fetchKernelMajorDirectories fetches the list of v{major}.x directories from kernel.org
func (d *VersionDiscovery) fetchKernelMajorDirectories(ctx context.Context, baseURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := d.httpClient.Do(req)
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
	// Matches: v1.x/, v2.x/, v3.x/, v4.x/, v5.x/, v6.x/, etc.
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
func (d *VersionDiscovery) fetchKernelVersionsFromDirectory(ctx context.Context, dirURL string) ([]DiscoveredVersion, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", dirURL+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := d.httpClient.Do(req)
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
	// Matches: linux-6.12.4.tar.xz, linux-5.15.tar.xz, etc.
	tarballPattern := regexp.MustCompile(`href="linux-(\d+\.\d+(?:\.\d+)?(?:-rc\d+)?)\.tar\.xz"`)
	matches := tarballPattern.FindAllStringSubmatch(string(body), -1)

	var versions []DiscoveredVersion
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			version := m[1]
			downloadURL := fmt.Sprintf("%s/linux-%s.tar.xz", dirURL, version)
			versionType := determineKernelVersionType(version)

			versions = append(versions, DiscoveredVersion{
				Version:     version,
				VersionType: versionType,
				DownloadURL: downloadURL,
				IsStable:    !strings.Contains(version, "-rc"),
			})
			seen[version] = true
		}
	}

	return versions, nil
}

// discoverHTTPDirectoryVersions discovers versions from a generic HTTP directory listing
func (d *VersionDiscovery) discoverHTTPDirectoryVersions(ctx context.Context, baseURL string) ([]DiscoveredVersion, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := d.httpClient.Do(req)
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
	// Common patterns: name-version.tar.gz, name-version.tar.xz, v1.2.3/, etc.
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

					versionType := db.VersionTypeStable
					if isPrerelease(version) {
						versionType = db.VersionTypeMainline
					}

					versions = append(versions, DiscoveredVersion{
						Version:     version,
						VersionType: versionType,
						DownloadURL: downloadURL,
						IsStable:    !isPrerelease(version),
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

// GetSyncCacheDuration returns the configured sync cache duration from settings
func GetSyncCacheDuration() time.Duration {
	minutes := viper.GetInt("sync.cache_duration")
	if minutes < 0 {
		minutes = 0
	}
	return time.Duration(minutes) * time.Minute
}

// SyncAllSources performs a version sync for all enabled sources
// This is typically called at startup to refresh the version cache
func (d *VersionDiscovery) SyncAllSources(ctx context.Context, sourceRepo *db.SourceRepository) {
	d.SyncAllSourcesWithCacheDuration(ctx, sourceRepo, GetSyncCacheDuration())
}

// SyncAllSourcesWithCacheDuration performs a version sync for all enabled sources,
// skipping sources that were successfully synced within the cache duration
func (d *VersionDiscovery) SyncAllSourcesWithCacheDuration(ctx context.Context, sourceRepo *db.SourceRepository, cacheDuration time.Duration) {
	// Get all sources (both default and user)
	sources, err := sourceRepo.List()
	if err != nil {
		log.Error("Failed to list sources for startup sync", "error", err)
		return
	}

	if len(sources) == 0 {
		log.Debug("No sources configured, skipping startup version sync")
		return
	}

	log.Info("Starting version sync for all sources", "source_count", len(sources), "cache_duration", cacheDuration.String())

	for _, source := range sources {
		if !source.Enabled {
			log.Debug("Skipping disabled source", "source_id", source.ID, "source_name", source.Name)
			continue
		}

		// Check if a sync job is already running for this source
		sourceType := db.GetSourceType(&source)
		runningJob, err := d.versionRepo.GetRunningSyncJob(source.ID, sourceType)
		if err != nil {
			log.Error("Failed to check running sync job", "source_id", source.ID, "error", err)
			continue
		}
		if runningJob != nil {
			log.Debug("Sync already in progress for source", "source_id", source.ID, "source_name", source.Name)
			continue
		}

		// Check if source was recently synced successfully (cache check)
		if cacheDuration > 0 {
			latestJob, err := d.versionRepo.GetLatestSyncJob(source.ID, sourceType)
			if err != nil {
				log.Error("Failed to check latest sync job", "source_id", source.ID, "error", err)
				continue
			}
			if latestJob != nil && latestJob.Status == db.SyncStatusCompleted && latestJob.CompletedAt != nil {
				timeSinceSync := time.Since(*latestJob.CompletedAt)
				if timeSinceSync < cacheDuration {
					log.Debug("Skipping recently synced source",
						"source_id", source.ID,
						"source_name", source.Name,
						"last_sync", latestJob.CompletedAt.Format(time.RFC3339),
						"time_since_sync", timeSinceSync.Round(time.Second).String())
					continue
				}
			}
		}

		// Create a new sync job
		job := &db.VersionSyncJob{
			SourceID:   source.ID,
			SourceType: sourceType,
			Status:     db.SyncStatusPending,
		}

		if err := d.versionRepo.CreateSyncJob(job); err != nil {
			log.Error("Failed to create startup sync job", "source_id", source.ID, "error", err)
			continue
		}

		// Run sync in background (we copy the source to avoid closure issues)
		sourceCopy := source
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			log.Info("Starting startup version sync", "source_id", sourceCopy.ID, "source_name", sourceCopy.Name)
			if err := d.SyncVersions(syncCtx, &sourceCopy, sourceType, job); err != nil {
				log.Error("Startup version sync failed", "source_id", sourceCopy.ID, "error", err)
			} else {
				log.Info("Startup version sync completed", "source_id", sourceCopy.ID, "source_name", sourceCopy.Name)
			}
		}()
	}
}

// SyncVersions performs a full version sync for a source
func (d *VersionDiscovery) SyncVersions(ctx context.Context, source *db.UpstreamSource, sourceType string, job *db.VersionSyncJob) error {
	// Mark job as running
	if err := d.versionRepo.MarkSyncJobRunning(job.ID); err != nil {
		return fmt.Errorf("failed to mark job running: %w", err)
	}

	// Discover versions
	discovered, err := d.DiscoverVersions(ctx, source)
	if err != nil {
		errMsg := fmt.Sprintf("failed to discover versions: %v", err)
		_ = d.versionRepo.MarkSyncJobFailed(job.ID, errMsg)
		return err
	}

	// Convert discovered versions to SourceVersion records
	sourceVersions := make([]db.SourceVersion, len(discovered))
	for i, v := range discovered {
		sourceVersions[i] = db.SourceVersion{
			SourceID:    source.ID,
			SourceType:  sourceType,
			Version:     v.Version,
			VersionType: v.VersionType,
			ReleaseDate: v.ReleaseDate,
			DownloadURL: v.DownloadURL,
			IsStable:    v.IsStable,
		}
	}

	// Bulk upsert versions
	newCount, err := d.versionRepo.BulkUpsert(sourceVersions)
	if err != nil {
		errMsg := fmt.Sprintf("failed to save versions: %v", err)
		_ = d.versionRepo.MarkSyncJobFailed(job.ID, errMsg)
		return err
	}

	// Mark job as completed
	if err := d.versionRepo.MarkSyncJobCompleted(job.ID, len(discovered), newCount); err != nil {
		return fmt.Errorf("failed to mark job completed: %w", err)
	}

	// Auto-detect and update source default version if not already set
	if source.DefaultVersion == "" && len(discovered) > 0 {
		defaultVersion := d.detectDefaultVersion(ctx, source, discovered)
		if defaultVersion != "" {
			source.DefaultVersion = defaultVersion
			// Persist the default version to the database
			if d.sourceRepo != nil {
				if err := d.sourceRepo.Update(source); err != nil {
					log.Warn("Failed to persist source default version",
						"source_id", source.ID,
						"error", err)
				} else {
					log.Info("Auto-detected and saved default version for source",
						"source_id", source.ID,
						"source_name", source.Name,
						"default_version", defaultVersion)
				}
			} else {
				log.Info("Auto-detected default version for source (not persisted - no sourceRepo)",
					"source_id", source.ID,
					"source_name", source.Name,
					"default_version", defaultVersion)
			}
		}
	}

	// After successful sync, update default versions for all linked components
	if err := d.updateComponentDefaultVersions(source); err != nil {
		// Log warning but don't fail the sync for this
		log.Warn("Failed to update component default versions after sync", "source_id", source.ID, "error", err)
	}

	return nil
}

// detectDefaultVersion determines the default version for a source
// For GitHub sources, it fetches the latest release tag
// For other sources, it uses the first stable version from discovered versions
func (d *VersionDiscovery) detectDefaultVersion(ctx context.Context, source *db.UpstreamSource, discovered []DiscoveredVersion) string {
	method := d.DetectDiscoveryMethod(source.URL)

	switch method {
	case DiscoveryMethodGitHub:
		// Try to get the latest release from GitHub API
		latestVersion := d.fetchGitHubLatestRelease(ctx, source.URL)
		if latestVersion != "" {
			return latestVersion
		}
		// Fall back to first stable discovered version
		return d.findFirstStableVersion(discovered)

	case DiscoveryMethodKernelOrg:
		// For kernel.org, find the first stable (non-RC) version
		return d.findFirstStableVersion(discovered)

	default:
		// For other sources, use the first stable version
		return d.findFirstStableVersion(discovered)
	}
}

// fetchGitHubLatestRelease fetches the latest release tag from GitHub API
func (d *VersionDiscovery) fetchGitHubLatestRelease(ctx context.Context, repoURL string) string {
	owner, repo, err := parseGitHubURL(repoURL)
	if err != nil {
		return ""
	}

	// Use the /releases/latest endpoint which returns the most recent non-prerelease, non-draft release
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return ""
	}

	return normalizeVersion(release.TagName)
}

// findFirstStableVersion returns the first stable version from a list of discovered versions
// Assumes versions are already sorted in descending order
func (d *VersionDiscovery) findFirstStableVersion(discovered []DiscoveredVersion) string {
	for _, v := range discovered {
		if v.IsStable {
			return v.Version
		}
	}
	// If no stable version found, return the first version
	if len(discovered) > 0 {
		return discovered[0].Version
	}
	return ""
}

// updateComponentDefaultVersions updates the default version for all components linked to a source
// Only updates components that use auto-version rules (latest-stable or latest-lts), not pinned versions
func (d *VersionDiscovery) updateComponentDefaultVersions(source *db.UpstreamSource) error {
	if d.componentRepo == nil {
		return nil // Component repo not available, skip update
	}

	for _, componentID := range source.ComponentIDs {
		// Get component
		component, err := d.componentRepo.GetByID(componentID)
		if err != nil {
			log.Warn("Failed to get component for version update", "component_id", componentID, "error", err)
			continue
		}
		if component == nil {
			continue
		}

		// Only update if component uses auto-version rule (not pinned)
		if component.DefaultVersionRule == db.VersionRulePinned {
			continue
		}

		// Resolve the latest version based on the component's rule
		var latestVersion *db.SourceVersion
		switch component.DefaultVersionRule {
		case db.VersionRuleLatestStable:
			latestVersion, err = d.versionRepo.GetLatestStableByComponent(componentID)
		case db.VersionRuleLatestLTS:
			latestVersion, err = d.versionRepo.GetLatestLongtermByComponent(componentID)
		default:
			// Default to latest stable if rule is empty or unknown
			latestVersion, err = d.versionRepo.GetLatestStableByComponent(componentID)
		}

		if err != nil {
			log.Warn("Failed to resolve latest version for component",
				"component_id", componentID,
				"component_name", component.Name,
				"rule", component.DefaultVersionRule,
				"error", err)
			continue
		}

		if latestVersion == nil {
			continue
		}

		// Only update if the version has changed
		if latestVersion.Version != component.DefaultVersion {
			oldVersion := component.DefaultVersion
			component.DefaultVersion = latestVersion.Version
			if err := d.componentRepo.Update(component); err != nil {
				log.Warn("Failed to update component default version",
					"component_id", componentID,
					"component_name", component.Name,
					"error", err)
				continue
			}
			log.Info("Updated component default version",
				"component_name", component.Name,
				"rule", component.DefaultVersionRule,
				"old_version", oldVersion,
				"new_version", latestVersion.Version)
		}
	}

	return nil
}

// Helper functions

// parseGitHubURL extracts owner and repo from a GitHub URL
func parseGitHubURL(url string) (owner, repo string, err error) {
	// Handle various GitHub URL formats:
	// https://github.com/owner/repo
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git

	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	if strings.HasPrefix(url, "git@github.com:") {
		parts := strings.Split(strings.TrimPrefix(url, "git@github.com:"), "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	if strings.Contains(url, "github.com") {
		parts := strings.Split(url, "github.com/")
		if len(parts) >= 2 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 2 {
				return pathParts[0], pathParts[1], nil
			}
		}
	}

	return "", "", fmt.Errorf("unable to parse GitHub URL: %s", url)
}

// normalizeVersion removes common prefixes from version strings
func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

// isVersionString checks if a string looks like a version number
func isVersionString(s string) bool {
	// Simple check: starts with a digit
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

// isPrerelease checks if a version string indicates a prerelease
func isPrerelease(version string) bool {
	lower := strings.ToLower(version)
	return strings.Contains(lower, "-rc") ||
		strings.Contains(lower, "-alpha") ||
		strings.Contains(lower, "-beta") ||
		strings.Contains(lower, "-dev") ||
		strings.Contains(lower, "-pre") ||
		strings.Contains(lower, ".rc") ||
		strings.Contains(lower, "_rc") ||
		strings.Contains(lower, "alpha") ||
		strings.Contains(lower, "beta")
}

// determineGitHubVersionType determines the version type for a GitHub release
// Categories:
// - mainline: prereleases, rc, alpha, beta, dev versions
// - stable: regular releases (default)
func determineGitHubVersionType(version string, isGitHubPrerelease bool) db.VersionType {
	// If GitHub marks it as prerelease, or version string indicates prerelease
	if isGitHubPrerelease || isPrerelease(version) {
		return db.VersionTypeMainline
	}

	// Default to stable for regular releases
	return db.VersionTypeStable
}

// determineKernelVersionType determines the version type for a kernel version
// Based on kernel.org categorization: mainline, stable, longterm, linux-next
func determineKernelVersionType(version string) db.VersionType {
	lower := strings.ToLower(version)

	// linux-next versions
	if strings.HasPrefix(lower, "next-") {
		return db.VersionTypeLinuxNext
	}

	// RC versions are mainline
	if strings.Contains(lower, "-rc") {
		return db.VersionTypeMainline
	}

	// Known LTS (longterm) kernel series
	// Reference: https://kernel.org/category/releases.html
	ltsVersions := []string{
		"6.12", "6.6", "6.1", "5.15", "5.10", "5.4", "4.19", "4.14",
	}

	// Extract major.minor from version
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		majorMinor := parts[0] + "." + parts[1]
		for _, lts := range ltsVersions {
			if majorMinor == lts {
				return db.VersionTypeLongterm
			}
		}
	}

	// Default to stable for non-RC, non-LTS versions
	return db.VersionTypeStable
}

// sortVersionsDescending sorts versions in descending order using semver-like comparison
func sortVersionsDescending(versions []DiscoveredVersion) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})
}

// compareVersions compares two version strings
// Returns: >0 if v1 > v2, <0 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	// Split by dots and compare each part
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

		// Extract numeric prefix and any suffix
		n1, s1 := splitVersionPart(p1)
		n2, s2 := splitVersionPart(p2)

		if n1 != n2 {
			return n1 - n2
		}

		// Compare suffixes (empty suffix > non-empty suffix for prereleases)
		if s1 != s2 {
			if s1 == "" {
				return 1 // No suffix is greater (stable > prerelease)
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
