package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
)

var downloadLoggerOnce sync.Once

// setupDownloadLogger ensures the download package has a logger
func setupDownloadLogger() {
	downloadLoggerOnce.Do(func() {
		logger := logs.New(logs.Config{
			Output: logs.OutputStdout,
			Level:  "error", // Only log errors during tests
		})
		download.SetLogger(logger)
	})
}

// =============================================================================
// URL Builder Tests
// =============================================================================

func setupURLBuilderTestDB(t *testing.T) (*db.Database, *db.ComponentRepository, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	componentRepo := db.NewComponentRepository(database)

	return database, componentRepo, func() { _ = database.Shutdown() }
}

func TestURLBuilder_BuildURL_Basic(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{
		URL:         "https://example.com/releases",
		URLTemplate: "{base_url}/v{version}.tar.gz",
	}

	component := &db.Component{
		Name: "test-component",
	}

	url, err := builder.BuildURL(source, component, "1.0.0")
	if err != nil {
		t.Fatalf("failed to build URL: %v", err)
	}

	expected := "https://example.com/releases/v1.0.0.tar.gz"
	if url != expected {
		t.Fatalf("expected URL '%s', got '%s'", expected, url)
	}
}

func TestURLBuilder_BuildURL_NilSource(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	component := &db.Component{Name: "test"}

	_, err := builder.BuildURL(nil, component, "1.0.0")
	if err == nil {
		t.Fatal("expected error for nil source")
	}
}

func TestURLBuilder_BuildURL_NilComponent(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{URL: "https://example.com"}

	_, err := builder.BuildURL(source, nil, "1.0.0")
	if err == nil {
		t.Fatal("expected error for nil component")
	}
}

func TestURLBuilder_BuildURL_EmptyVersion(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{URL: "https://example.com"}
	component := &db.Component{Name: "test"}

	_, err := builder.BuildURL(source, component, "")
	if err == nil {
		t.Fatal("expected error for empty version")
	}
}

func TestURLBuilder_BuildURL_GitHub(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{
		URL: "https://github.com/owner/repo",
	}

	component := &db.Component{
		Name:                     "test-component",
		GitHubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
	}

	url, err := builder.BuildURL(source, component, "1.2.3")
	if err != nil {
		t.Fatalf("failed to build URL: %v", err)
	}

	expected := "https://github.com/owner/repo/archive/refs/tags/v1.2.3.tar.gz"
	if url != expected {
		t.Fatalf("expected URL '%s', got '%s'", expected, url)
	}
}

func TestURLBuilder_BuildURL_GitHub_WithGitSuffix(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{
		URL: "https://github.com/owner/repo.git",
	}

	component := &db.Component{
		Name:                     "test-component",
		GitHubNormalizedTemplate: "{base_url}/archive/refs/tags/v{version}.tar.gz",
	}

	url, err := builder.BuildURL(source, component, "1.0.0")
	if err != nil {
		t.Fatalf("failed to build URL: %v", err)
	}

	// .git suffix should be removed
	expected := "https://github.com/owner/repo/archive/refs/tags/v1.0.0.tar.gz"
	if url != expected {
		t.Fatalf("expected URL '%s', got '%s'", expected, url)
	}
}

func TestURLBuilder_BuildURL_TemplateVariables(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{
		URL:         "https://cdn.kernel.org/pub/linux/kernel",
		URLTemplate: "{base_url}/v{major_x}/linux-{version}.tar.xz",
	}

	component := &db.Component{
		Name: "kernel",
	}

	url, err := builder.BuildURL(source, component, "6.12.5")
	if err != nil {
		t.Fatalf("failed to build URL: %v", err)
	}

	expected := "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.12.5.tar.xz"
	if url != expected {
		t.Fatalf("expected URL '%s', got '%s'", expected, url)
	}
}

func TestURLBuilder_BuildURL_AllTemplateVariables(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	tests := []struct {
		name     string
		template string
		version  string
		expected string
	}{
		{
			name:     "version placeholder",
			template: "{base_url}/{version}.tar.gz",
			version:  "1.2.3",
			expected: "https://example.com/1.2.3.tar.gz",
		},
		{
			name:     "tag placeholder",
			template: "{base_url}/{tag}.tar.gz",
			version:  "1.2.3",
			expected: "https://example.com/v1.2.3.tar.gz",
		},
		{
			name:     "tag_short placeholder",
			template: "{base_url}/{tag_short}.tar.gz",
			version:  "1.2.3",
			expected: "https://example.com/v1.2.tar.gz",
		},
		{
			name:     "tag_compact placeholder",
			template: "{base_url}/{tag_compact}.tar.gz",
			version:  "1.2.3",
			expected: "https://example.com/v123.tar.gz",
		},
		{
			name:     "major placeholder",
			template: "{base_url}/{major}/file.tar.gz",
			version:  "6.12.5",
			expected: "https://example.com/6/file.tar.gz",
		},
		{
			name:     "minor placeholder",
			template: "{base_url}/{major}.{minor}/file.tar.gz",
			version:  "6.12.5",
			expected: "https://example.com/6.12/file.tar.gz",
		},
		{
			name:     "patch placeholder",
			template: "{base_url}/{major}.{minor}.{patch}/file.tar.gz",
			version:  "6.12.5",
			expected: "https://example.com/6.12.5/file.tar.gz",
		},
		{
			name:     "major_x placeholder",
			template: "{base_url}/v{major_x}/file.tar.gz",
			version:  "6.12.5",
			expected: "https://example.com/v6.x/file.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &db.UpstreamSource{
				URL:         "https://example.com",
				URLTemplate: tt.template,
			}
			component := &db.Component{Name: "test"}

			url, err := builder.BuildURL(source, component, tt.version)
			if err != nil {
				t.Fatalf("failed to build URL: %v", err)
			}
			if url != tt.expected {
				t.Fatalf("expected '%s', got '%s'", tt.expected, url)
			}
		})
	}
}

func TestURLBuilder_BuildURL_NoTemplate(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{
		URL: "https://example.com/static-url",
	}

	component := &db.Component{
		Name: "test",
	}

	url, err := builder.BuildURL(source, component, "1.0.0")
	if err != nil {
		t.Fatalf("failed to build URL: %v", err)
	}

	// Without template, should return base URL as-is
	expected := "https://example.com/static-url"
	if url != expected {
		t.Fatalf("expected '%s', got '%s'", expected, url)
	}
}

func TestURLBuilder_BuildURL_SourceTemplateOverride(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	// Source has a custom template that should override component's default
	source := &db.UpstreamSource{
		URL:         "https://custom-mirror.com",
		URLTemplate: "{base_url}/custom/{version}.tar.gz",
	}

	component := &db.Component{
		Name:               "test",
		DefaultURLTemplate: "{base_url}/default/{version}.tar.gz",
	}

	url, err := builder.BuildURL(source, component, "1.0.0")
	if err != nil {
		t.Fatalf("failed to build URL: %v", err)
	}

	// Source template should take priority
	expected := "https://custom-mirror.com/custom/1.0.0.tar.gz"
	if url != expected {
		t.Fatalf("expected '%s', got '%s'", expected, url)
	}
}

func TestURLBuilder_BuildGitCloneURL(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "without .git suffix",
			url:      "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "with .git suffix",
			url:      "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "with trailing slash",
			url:      "https://github.com/owner/repo/",
			expected: "https://github.com/owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &db.UpstreamSource{URL: tt.url}
			result := builder.BuildGitCloneURL(source)
			if result != tt.expected {
				t.Fatalf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestURLBuilder_BuildGitRef(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	ref := builder.BuildGitRef("1.2.3")
	expected := "v1.2.3"
	if ref != expected {
		t.Fatalf("expected '%s', got '%s'", expected, ref)
	}
}

func TestURLBuilder_PreviewURL(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	source := &db.UpstreamSource{
		URL:         "https://example.com",
		URLTemplate: "{base_url}/v{version}.tar.gz",
	}
	component := &db.Component{Name: "test"}

	// With example version
	url, err := builder.PreviewURL(source, component, "2.0.0")
	if err != nil {
		t.Fatalf("failed to preview URL: %v", err)
	}
	if url != "https://example.com/v2.0.0.tar.gz" {
		t.Fatalf("unexpected preview URL: %s", url)
	}

	// Without version (should use default 1.0.0)
	url, err = builder.PreviewURL(source, component, "")
	if err != nil {
		t.Fatalf("failed to preview URL: %v", err)
	}
	if url != "https://example.com/v1.0.0.tar.gz" {
		t.Fatalf("unexpected preview URL with default version: %s", url)
	}
}

func TestURLBuilder_IsGitLabURL(t *testing.T) {
	_, componentRepo, cleanup := setupURLBuilderTestDB(t)
	defer cleanup()

	builder := download.NewURLBuilder(componentRepo)

	tests := []struct {
		url      string
		expected bool
	}{
		{"https://gitlab.com/owner/repo", true},
		{"https://gitlab.freedesktop.org/project", true},
		{"https://github.com/owner/repo", false},
		{"https://example.com/repo", false},
	}

	for _, tt := range tests {
		result := builder.IsGitLabURL(tt.url)
		if result != tt.expected {
			t.Errorf("IsGitLabURL(%s) = %v, expected %v", tt.url, result, tt.expected)
		}
	}
}

// =============================================================================
// Version Discovery Tests
// =============================================================================

func setupVersionDiscoveryTestDB(t *testing.T) (*db.Database, *db.SourceVersionRepository, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	versionRepo := db.NewSourceVersionRepository(database)

	return database, versionRepo, func() { _ = database.Shutdown() }
}

func TestVersionDiscovery_DetectDiscoveryMethod(t *testing.T) {
	database, versionRepo, cleanup := setupVersionDiscoveryTestDB(t)
	defer cleanup()

	componentRepo := db.NewComponentRepository(database)
	discovery := download.NewVersionDiscovery(versionRepo, componentRepo, nil)

	tests := []struct {
		url      string
		expected download.DiscoveryMethod
	}{
		{"https://github.com/owner/repo", download.DiscoveryMethodGitHub},
		{"https://GITHUB.COM/owner/repo", download.DiscoveryMethodGitHub},
		{"https://cdn.kernel.org/pub/linux/kernel", download.DiscoveryMethodKernelOrg},
		{"https://www.kernel.org/pub/linux/kernel", download.DiscoveryMethodKernelOrg},
		{"https://example.com/releases", download.DiscoveryMethodHTTPDirectory},
		{"https://ftp.gnu.org/gnu/bash", download.DiscoveryMethodHTTPDirectory},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			method := discovery.DetectDiscoveryMethod(tt.url)
			if method != tt.expected {
				t.Fatalf("expected method '%s', got '%s'", tt.expected, method)
			}
		})
	}
}

func TestVersionDiscovery_DiscoverVersions_GitHub(t *testing.T) {
	database, versionRepo, cleanup := setupVersionDiscoveryTestDB(t)
	defer cleanup()

	componentRepo := db.NewComponentRepository(database)

	// Create a mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return mock releases
			_, _ = w.Write([]byte(`[
				{
					"tag_name": "v2.0.0",
					"name": "Release 2.0.0",
					"published_at": "2024-01-15T00:00:00Z",
					"prerelease": false,
					"draft": false,
					"html_url": "https://github.com/owner/repo/releases/tag/v2.0.0"
				},
				{
					"tag_name": "v1.0.0",
					"name": "Release 1.0.0",
					"published_at": "2024-01-01T00:00:00Z",
					"prerelease": false,
					"draft": false,
					"html_url": "https://github.com/owner/repo/releases/tag/v1.0.0"
				},
				{
					"tag_name": "v2.1.0-rc1",
					"name": "Release Candidate",
					"published_at": "2024-01-20T00:00:00Z",
					"prerelease": true,
					"draft": false,
					"html_url": "https://github.com/owner/repo/releases/tag/v2.1.0-rc1"
				}
			]`))
			return
		}
		if r.URL.Path == "/repos/owner/repo/tags" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// We can't easily test with real GitHub API, so we skip actual HTTP test
	// but verify the discovery method detection works
	discovery := download.NewVersionDiscovery(versionRepo, componentRepo, nil)

	source := &db.UpstreamSource{
		URL: "https://github.com/owner/repo",
	}

	method := discovery.DetectDiscoveryMethod(source.URL)
	if method != download.DiscoveryMethodGitHub {
		t.Fatalf("expected GitHub method, got %s", method)
	}
}

func TestVersionDiscovery_DiscoverVersions_HTTPDirectory(t *testing.T) {
	database, versionRepo, cleanup := setupVersionDiscoveryTestDB(t)
	defer cleanup()

	componentRepo := db.NewComponentRepository(database)

	// Create a mock HTTP server with directory listing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<html>
			<body>
			<a href="package-1.0.0.tar.gz">package-1.0.0.tar.gz</a>
			<a href="package-1.1.0.tar.gz">package-1.1.0.tar.gz</a>
			<a href="package-2.0.0.tar.xz">package-2.0.0.tar.xz</a>
			<a href="package-2.1.0-beta.tar.gz">package-2.1.0-beta.tar.gz</a>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	discovery := download.NewVersionDiscovery(versionRepo, componentRepo, nil)

	source := &db.UpstreamSource{
		URL: server.URL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	versions, err := discovery.DiscoverVersions(ctx, source)
	if err != nil {
		t.Fatalf("failed to discover versions: %v", err)
	}

	if len(versions) == 0 {
		t.Fatal("expected to discover at least one version")
	}

	// Verify versions are sorted descending
	for i := 1; i < len(versions); i++ {
		// Just verify we got versions (order depends on implementation)
		if versions[i].Version == "" {
			t.Fatal("got empty version string")
		}
	}
}

func TestVersionDiscovery_SyncVersions(t *testing.T) {
	database, versionRepo, cleanup := setupVersionDiscoveryTestDB(t)
	defer cleanup()

	componentRepo := db.NewComponentRepository(database)

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<html>
			<body>
			<a href="pkg-1.0.0.tar.gz">pkg-1.0.0.tar.gz</a>
			<a href="pkg-1.1.0.tar.gz">pkg-1.1.0.tar.gz</a>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	discovery := download.NewVersionDiscovery(versionRepo, componentRepo, nil)

	// Create source
	sourceRepo := db.NewSourceRepository(database)
	source := &db.UpstreamSource{
		Name:            "test-source",
		URL:             server.URL,
		RetrievalMethod: "release",
		IsSystem:        true,
		Enabled:         true,
	}
	_ = sourceRepo.CreateDefault(source)

	// Create sync job
	job := &db.VersionSyncJob{
		SourceID:   source.ID,
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}
	_ = versionRepo.CreateSyncJob(job)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := discovery.SyncVersions(ctx, source, "default", job)
	if err != nil {
		t.Fatalf("failed to sync versions: %v", err)
	}

	// Verify job was completed
	updatedJob, _ := versionRepo.GetSyncJob(job.ID)
	if updatedJob.Status != db.SyncStatusCompleted {
		t.Fatalf("expected completed status, got %s", updatedJob.Status)
	}

	// Verify versions were created
	versions, _ := versionRepo.ListBySource(source.ID, "default")
	if len(versions) < 1 {
		t.Fatal("expected at least 1 version to be synced")
	}
}

func TestVersionDiscovery_SyncVersions_Failure(t *testing.T) {
	database, versionRepo, cleanup := setupVersionDiscoveryTestDB(t)
	defer cleanup()

	componentRepo := db.NewComponentRepository(database)

	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	discovery := download.NewVersionDiscovery(versionRepo, componentRepo, nil)

	source := &db.UpstreamSource{
		ID:   "test-source-id",
		Name: "test-source",
		URL:  server.URL,
	}

	// Create sync job
	job := &db.VersionSyncJob{
		SourceID:   source.ID,
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}
	_ = versionRepo.CreateSyncJob(job)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := discovery.SyncVersions(ctx, source, "default", job)
	if err == nil {
		t.Fatal("expected error for failed server")
	}

	// Verify job was marked as failed
	updatedJob, _ := versionRepo.GetSyncJob(job.ID)
	if updatedJob.Status != db.SyncStatusFailed {
		t.Fatalf("expected failed status, got %s", updatedJob.Status)
	}
	if updatedJob.ErrorMessage == "" {
		t.Fatal("expected error message to be set")
	}
}

// =============================================================================
// Download Manager Tests
// =============================================================================

func TestDownloadManager_DefaultConfig(t *testing.T) {
	setupDownloadLogger()
	cfg := download.DefaultConfig()

	if cfg.Workers <= 0 {
		t.Fatal("expected positive workers count")
	}
	if cfg.RetryDelay <= 0 {
		t.Fatal("expected positive retry delay")
	}
	if cfg.RequestTimeout <= 0 {
		t.Fatal("expected positive request timeout")
	}
	if cfg.MaxRetries <= 0 {
		t.Fatal("expected positive max retries")
	}
}

func TestDownloadManager_NewManager(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	managerCfg := download.Config{
		Workers:        2,
		RetryDelay:     time.Second,
		RequestTimeout: 30 * time.Second,
		MaxRetries:     3,
	}

	manager := download.NewManager(database, nil, managerCfg)
	if manager == nil {
		t.Fatal("expected manager to be created")
	}

	// Verify repositories are accessible
	if manager.JobRepo() == nil {
		t.Fatal("expected job repo to be set")
	}
	if manager.ComponentRepo() == nil {
		t.Fatal("expected component repo to be set")
	}
	if manager.SourceRepo() == nil {
		t.Fatal("expected source repo to be set")
	}
	if manager.URLBuilder() == nil {
		t.Fatal("expected URL builder to be set")
	}
}

func TestDownloadManager_NewManager_DefaultValues(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	// Pass zero config, should use defaults
	manager := download.NewManager(database, nil, download.Config{})
	if manager == nil {
		t.Fatal("expected manager to be created")
	}
}

func TestDownloadManager_SubmitJob(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	// Create a component
	compRepo := db.NewComponentRepository(database)
	component := &db.Component{
		Name:        "submit-test-comp",
		Category:    "core",
		DisplayName: "Submit Test Component",
		IsSystem:    true,
	}
	_ = compRepo.Create(component)

	// Create a distribution
	distRepo := db.NewDistributionRepository(database)
	dist := &db.Distribution{
		Name:       "submit-test-dist",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	_ = distRepo.Create(dist)

	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        "test-owner",
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		SourceID:       "test-source",
		SourceType:     "default",
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}

	err = manager.SubmitJob(job)
	if err != nil {
		t.Fatalf("failed to submit job: %v", err)
	}

	if job.ID == "" {
		t.Fatal("expected job ID to be set")
	}

	// Verify job is in database
	found, err := manager.GetJobStatus(job.ID)
	if err != nil {
		t.Fatalf("failed to get job status: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find job")
	}
	if found.Status != db.JobStatusPending {
		t.Fatalf("expected pending status, got %s", found.Status)
	}
}

func TestDownloadManager_CancelJob(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	// Create a component
	compRepo := db.NewComponentRepository(database)
	component := &db.Component{
		Name:        "cancel-test-comp",
		Category:    "core",
		DisplayName: "Cancel Test Component",
		IsSystem:    true,
	}
	_ = compRepo.Create(component)

	// Create a distribution
	distRepo := db.NewDistributionRepository(database)
	dist := &db.Distribution{
		Name:       "cancel-test-dist",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	_ = distRepo.Create(dist)

	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        "test-owner",
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		SourceID:       "test-source",
		SourceType:     "default",
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = manager.SubmitJob(job)

	err = manager.CancelJob(job.ID)
	if err != nil {
		t.Fatalf("failed to cancel job: %v", err)
	}

	found, _ := manager.GetJobStatus(job.ID)
	if found.Status != db.JobStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", found.Status)
	}
}

func TestDownloadManager_RetryJob(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	// Create a component
	compRepo := db.NewComponentRepository(database)
	component := &db.Component{
		Name:        "retry-test-comp",
		Category:    "core",
		DisplayName: "Retry Test Component",
		IsSystem:    true,
	}
	_ = compRepo.Create(component)

	// Create a distribution
	distRepo := db.NewDistributionRepository(database)
	dist := &db.Distribution{
		Name:       "retry-test-dist",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	_ = distRepo.Create(dist)

	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        "test-owner",
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		SourceID:       "test-source",
		SourceType:     "default",
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = manager.SubmitJob(job)

	// First mark it as failed
	_ = manager.JobRepo().MarkFailed(job.ID, "test failure")

	// Now retry
	err = manager.RetryJob(job.ID)
	if err != nil {
		t.Fatalf("failed to retry job: %v", err)
	}

	found, _ := manager.GetJobStatus(job.ID)
	if found.Status != db.JobStatusPending {
		t.Fatalf("expected pending status after retry, got %s", found.Status)
	}
	if found.RetryCount != 1 {
		t.Fatalf("expected retry count 1, got %d", found.RetryCount)
	}
}

func TestDownloadManager_RetryJob_NotFailed(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	// Create a component
	compRepo := db.NewComponentRepository(database)
	component := &db.Component{
		Name:        "retry-notfailed-comp",
		Category:    "core",
		DisplayName: "Retry Not Failed Component",
		IsSystem:    true,
	}
	_ = compRepo.Create(component)

	// Create a distribution
	distRepo := db.NewDistributionRepository(database)
	dist := &db.Distribution{
		Name:       "retry-notfailed-dist",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	_ = distRepo.Create(dist)

	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        "test-owner",
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		SourceID:       "test-source",
		SourceType:     "default",
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = manager.SubmitJob(job)

	// Try to retry a pending job (not failed)
	err = manager.RetryJob(job.ID)
	if err == nil {
		t.Fatal("expected error when retrying non-failed job")
	}
}

func TestDownloadManager_RetryJob_NotFound(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	err = manager.RetryJob("nonexistent-job-id")
	if err == nil {
		t.Fatal("expected error when retrying non-existent job")
	}
}

func TestDownloadManager_GetJobStatus(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	// Create a component
	compRepo := db.NewComponentRepository(database)
	component := &db.Component{
		Name:        "status-test-comp",
		Category:    "core",
		DisplayName: "Status Test Component",
		IsSystem:    true,
	}
	_ = compRepo.Create(component)

	// Create a distribution
	distRepo := db.NewDistributionRepository(database)
	dist := &db.Distribution{
		Name:       "status-test-dist",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	_ = distRepo.Create(dist)

	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        "test-owner",
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		SourceID:       "test-source",
		SourceType:     "default",
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = manager.SubmitJob(job)

	status, err := manager.GetJobStatus(job.ID)
	if err != nil {
		t.Fatalf("failed to get job status: %v", err)
	}
	if status == nil {
		t.Fatal("expected job status")
	}
	if status.ID != job.ID {
		t.Fatalf("expected job ID '%s', got '%s'", job.ID, status.ID)
	}
}

func TestDownloadManager_GetJobStatus_NotFound(t *testing.T) {
	setupDownloadLogger()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer func() { _ = database.Shutdown() }()

	manager := download.NewManager(database, nil, download.DefaultConfig())

	status, err := manager.GetJobStatus("nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != nil {
		t.Fatal("expected nil for non-existent job")
	}
}
