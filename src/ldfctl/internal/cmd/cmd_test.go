package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bitswalk/ldf/src/ldfctl/internal/client"
	"github.com/spf13/cobra"
)

// =============================================================================
// Test Helpers
// =============================================================================

// setupTestClient creates a mock HTTP server and injects a client pointing to it.
// Returns the server (for deferred Close) and the mux for registering handlers.
func setupTestClient(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(mux)
	apiClient = client.New(srv.URL)
	return srv
}

// resetGlobals resets global state between tests
func resetGlobals() {
	apiClient = nil
	outputFormat = "table"
}

// executeCommand runs a cobra command with the given args and returns stdout/stderr
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// =============================================================================
// Command Registration Tests
// =============================================================================

func TestRootCommand_HasSubcommands(t *testing.T) {
	expected := []string{
		"version", "health", "login", "logout", "whoami",
		"distribution", "component", "source", "download",
		"artifact", "setting", "role", "forge", "branding",
		"langpack", "release",
	}

	commands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		commands[cmd.Name()] = true
	}

	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected subcommand %q not found on root", name)
		}
	}
}

func TestDistributionCommand_HasSubcommands(t *testing.T) {
	expected := []string{
		"list", "get", "create", "update", "delete",
		"logs", "stats", "deletion-preview",
	}
	commands := make(map[string]bool)
	for _, cmd := range distributionCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected distribution subcommand %q not found", name)
		}
	}
}

func TestComponentCommand_HasSubcommands(t *testing.T) {
	expected := []string{
		"list", "get", "create", "update", "delete",
		"categories", "versions", "resolve-version",
	}
	commands := make(map[string]bool)
	for _, cmd := range componentCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected component subcommand %q not found", name)
		}
	}
}

func TestSourceCommand_HasSubcommands(t *testing.T) {
	expected := []string{
		"list", "get", "create", "update", "delete",
		"sync", "versions", "sync-status", "clear-versions",
	}
	commands := make(map[string]bool)
	for _, cmd := range sourceCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected source subcommand %q not found", name)
		}
	}
}

func TestDownloadCommand_HasSubcommands(t *testing.T) {
	expected := []string{"list", "get", "start", "cancel", "retry", "active"}
	commands := make(map[string]bool)
	for _, cmd := range downloadCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected download subcommand %q not found", name)
		}
	}
}

func TestArtifactCommand_HasSubcommands(t *testing.T) {
	expected := []string{"list", "upload", "download", "delete", "url", "storage-status", "list-all"}
	commands := make(map[string]bool)
	for _, cmd := range artifactCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected artifact subcommand %q not found", name)
		}
	}
}

func TestSettingCommand_HasSubcommands(t *testing.T) {
	expected := []string{"list", "get", "set", "reset-db"}
	commands := make(map[string]bool)
	for _, cmd := range settingCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected setting subcommand %q not found", name)
		}
	}
}

func TestRoleCommand_HasSubcommands(t *testing.T) {
	expected := []string{"list", "get", "create", "update", "delete"}
	commands := make(map[string]bool)
	for _, cmd := range roleCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected role subcommand %q not found", name)
		}
	}
}

func TestReleaseCommand_HasSubcommands(t *testing.T) {
	expected := []string{"create", "configure", "show"}
	commands := make(map[string]bool)
	for _, cmd := range releaseCmd.Commands() {
		commands[cmd.Name()] = true
	}
	for _, name := range expected {
		if !commands[name] {
			t.Errorf("expected release subcommand %q not found", name)
		}
	}
}

// =============================================================================
// Command Aliases Tests
// =============================================================================

func TestDistributionCommand_Alias(t *testing.T) {
	if len(distributionCmd.Aliases) == 0 || distributionCmd.Aliases[0] != "dist" {
		t.Error("expected distribution alias 'dist'")
	}
}

func TestComponentCommand_Alias(t *testing.T) {
	if len(componentCmd.Aliases) == 0 || componentCmd.Aliases[0] != "comp" {
		t.Error("expected component alias 'comp'")
	}
}

func TestSourceCommand_Alias(t *testing.T) {
	if len(sourceCmd.Aliases) == 0 || sourceCmd.Aliases[0] != "src" {
		t.Error("expected source alias 'src'")
	}
}

func TestDownloadCommand_Alias(t *testing.T) {
	if len(downloadCmd.Aliases) == 0 || downloadCmd.Aliases[0] != "dl" {
		t.Error("expected download alias 'dl'")
	}
}

// =============================================================================
// Arg Validation Tests
// =============================================================================

func TestDistributionGetCmd_RequiresArg(t *testing.T) {
	err := distributionGetCmd.Args(distributionGetCmd, []string{})
	if err == nil {
		t.Error("expected error for missing arg on distribution get")
	}
}

func TestDistributionGetCmd_AcceptsOneArg(t *testing.T) {
	err := distributionGetCmd.Args(distributionGetCmd, []string{"some-id"})
	if err != nil {
		t.Errorf("unexpected error for valid arg: %v", err)
	}
}

func TestDistributionGetCmd_RejectsTwoArgs(t *testing.T) {
	err := distributionGetCmd.Args(distributionGetCmd, []string{"a", "b"})
	if err == nil {
		t.Error("expected error for two args on distribution get")
	}
}

func TestDistributionDeleteCmd_RequiresArg(t *testing.T) {
	err := distributionDeleteCmd.Args(distributionDeleteCmd, []string{})
	if err == nil {
		t.Error("expected error for missing arg on distribution delete")
	}
}

func TestComponentGetCmd_RequiresArg(t *testing.T) {
	err := componentGetCmd.Args(componentGetCmd, []string{})
	if err == nil {
		t.Error("expected error for missing arg on component get")
	}
}

func TestSourceGetCmd_RequiresArg(t *testing.T) {
	err := sourceGetCmd.Args(sourceGetCmd, []string{})
	if err == nil {
		t.Error("expected error for missing arg on source get")
	}
}

func TestArtifactDeleteCmd_RequiresTwoArgs(t *testing.T) {
	err := artifactDeleteCmd.Args(artifactDeleteCmd, []string{"one"})
	if err == nil {
		t.Error("expected error for single arg on artifact delete (needs dist-id + path)")
	}
}

func TestArtifactDeleteCmd_AcceptsTwoArgs(t *testing.T) {
	err := artifactDeleteCmd.Args(artifactDeleteCmd, []string{"dist-id", "path/file"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReleaseConfigureCmd_RequiresArg(t *testing.T) {
	err := releaseConfigureCmd.Args(releaseConfigureCmd, []string{})
	if err == nil {
		t.Error("expected error for missing arg on release configure")
	}
}

// =============================================================================
// Flag Tests
// =============================================================================

func TestDistributionListCmd_HasListFlags(t *testing.T) {
	flags := distributionListCmd.Flags()
	for _, name := range []string{"limit", "offset", "status"} {
		if flags.Lookup(name) == nil {
			t.Errorf("expected flag --%s on distribution list", name)
		}
	}
}

func TestComponentListCmd_HasListFlags(t *testing.T) {
	flags := componentListCmd.Flags()
	for _, name := range []string{"limit", "offset", "category"} {
		if flags.Lookup(name) == nil {
			t.Errorf("expected flag --%s on component list", name)
		}
	}
}

func TestSourceListCmd_HasListFlags(t *testing.T) {
	flags := sourceListCmd.Flags()
	for _, name := range []string{"limit", "offset"} {
		if flags.Lookup(name) == nil {
			t.Errorf("expected flag --%s on source list", name)
		}
	}
}

func TestSourceVersionsCmd_HasVersionFlags(t *testing.T) {
	flags := sourceVersionsCmd.Flags()
	for _, name := range []string{"limit", "offset", "version-type"} {
		if flags.Lookup(name) == nil {
			t.Errorf("expected flag --%s on source versions", name)
		}
	}
}

func TestReleaseConfigureCmd_HasConfigFlags(t *testing.T) {
	flags := releaseConfigureCmd.Flags()
	expected := []string{
		"kernel", "bootloader", "init", "filesystem",
		"security", "container", "virtualization",
		"target-type", "package-manager",
	}
	for _, name := range expected {
		if flags.Lookup(name) == nil {
			t.Errorf("expected flag --%s on release configure", name)
		}
	}
}

func TestReleaseCreateCmd_HasRequiredFlags(t *testing.T) {
	flags := releaseCreateCmd.Flags()
	nameFlag := flags.Lookup("name")
	if nameFlag == nil {
		t.Fatal("expected --name flag on release create")
	}
	versionFlag := flags.Lookup("version")
	if versionFlag == nil {
		t.Fatal("expected --version flag on release create")
	}
}

func TestSettingResetDBCmd_HasYesFlag(t *testing.T) {
	flag := settingResetDBCmd.Flags().Lookup("yes")
	if flag == nil {
		t.Error("expected --yes flag on setting reset-db")
	}
}

func TestRootCmd_OutputFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("output")
	if flag == nil {
		t.Fatal("expected --output persistent flag on root")
	}
	if flag.DefValue != "table" {
		t.Errorf("expected default output format 'table', got %q", flag.DefValue)
	}
}

func TestRootCmd_ServerFlag(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("server")
	if flag == nil {
		t.Fatal("expected --server persistent flag on root")
	}
	if flag.Shorthand != "s" {
		t.Errorf("expected shorthand 's' for --server, got %q", flag.Shorthand)
	}
}

// =============================================================================
// Command Execution Tests (with mock server)
// =============================================================================

func TestDistributionList_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 2,
			"distributions": []map[string]string{
				{"id": "d1", "name": "alpine", "version": "3.19", "status": "ready", "visibility": "public"},
				{"id": "d2", "name": "debian", "version": "12", "status": "pending", "visibility": "private"},
			},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runDistributionList(distributionListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDistributionGet_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions/d1", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"id": "d1", "name": "alpine", "version": "3.19", "status": "ready",
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runDistributionGet(distributionGetCmd, []string{"d1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDistributionDelete_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions/d1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runDistributionDelete(distributionDeleteCmd, []string{"d1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDistributionList_WithQueryParams(t *testing.T) {
	defer resetGlobals()

	var capturedQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions", func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":         0,
			"distributions": []interface{}{},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "table"
	distributionListCmd.Flags().Set("limit", "10")
	distributionListCmd.Flags().Set("offset", "5")
	distributionListCmd.Flags().Set("status", "ready")
	defer func() {
		distributionListCmd.Flags().Set("limit", "0")
		distributionListCmd.Flags().Set("offset", "0")
		distributionListCmd.Flags().Set("status", "")
	}()

	err := runDistributionList(distributionListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedQuery == "" {
		t.Error("expected query parameters to be sent")
	}
	for _, param := range []string{"limit=10", "offset=5", "status=ready"} {
		if !containsParam(capturedQuery, param) {
			t.Errorf("expected %q in query %q", param, capturedQuery)
		}
	}
}

func containsParam(query, param string) bool {
	for _, p := range splitQuery(query) {
		if p == param {
			return true
		}
	}
	return false
}

func splitQuery(query string) []string {
	result := []string{}
	for _, part := range bytes.Split([]byte(query), []byte("&")) {
		result = append(result, string(part))
	}
	return result
}

func TestComponentList_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/components", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":      1,
			"components": []map[string]string{{"id": "c1", "name": "linux", "category": "kernel"}},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runComponentList(componentListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSourceList_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sources", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":   1,
			"sources": []map[string]string{{"id": "s1", "name": "kernel.org", "url": "https://kernel.org"}},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runSourceList(sourceListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealthCommand_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runHealth(healthCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDistributionStats_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions/stats", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total": 5,
			"stats": map[string]int{"ready": 3, "pending": 2},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runDistributionStats(distributionStatsCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadActive_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/downloads/active", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 0,
			"jobs":  []interface{}{},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "table"
	err := runDownloadActive(downloadActiveCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStorageStatus_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/storage/status", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "local", "available": true, "path": "/data",
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runArtifactStorageStatus(artifactStorageStatusCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoleList_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/roles", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"roles": []map[string]interface{}{
				{"id": "r1", "name": "admin", "description": "Admin role",
					"permissions": map[string]bool{"can_read": true, "can_write": true, "can_delete": true, "can_admin": true}},
			},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runRoleList(roleListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestDistributionGet_ServerError(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not_found", "message": "distribution not found"})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	err := runDistributionGet(distributionGetCmd, []string{"bad-id"})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestComponentGet_ServerError(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/components/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal", "message": "database error"})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	err := runComponentGet(componentGetCmd, []string{"bad-id"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// =============================================================================
// Output Format Tests
// =============================================================================

func TestGetOutputFormat(t *testing.T) {
	defer resetGlobals()

	outputFormat = "json"
	if f := getOutputFormat(); f != "json" {
		t.Errorf("expected json, got %s", f)
	}

	outputFormat = "yaml"
	if f := getOutputFormat(); f != "yaml" {
		t.Errorf("expected yaml, got %s", f)
	}

	outputFormat = "table"
	if f := getOutputFormat(); f != "table" {
		t.Errorf("expected table, got %s", f)
	}
}

// =============================================================================
// ensureMap helper test
// =============================================================================

func TestEnsureMap(t *testing.T) {
	m := make(map[string]interface{})

	ensureMap(m, "core")
	if _, ok := m["core"].(map[string]interface{}); !ok {
		t.Error("expected map[string]interface{} at key 'core'")
	}

	// Should not overwrite existing map
	m["core"].(map[string]interface{})["test"] = "value"
	ensureMap(m, "core")
	coreMap := m["core"].(map[string]interface{})
	if coreMap["test"] != "value" {
		t.Error("ensureMap should not overwrite existing map contents")
	}

	// Should replace non-map value
	m["bad"] = "string-value"
	ensureMap(m, "bad")
	if _, ok := m["bad"].(map[string]interface{}); !ok {
		t.Error("expected ensureMap to replace non-map value with map")
	}
}

// =============================================================================
// Version Info Tests
// =============================================================================

func TestVersionInfo_Defaults(t *testing.T) {
	if Version != "dev" {
		t.Errorf("expected default Version 'dev', got %q", Version)
	}
	if BuildDate != "unknown" {
		t.Errorf("expected default BuildDate 'unknown', got %q", BuildDate)
	}
}

// =============================================================================
// Forge Commands Tests
// =============================================================================

func TestForgeTypes_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/forge/types", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"types": []string{"github", "gitlab", "tarball"},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runForgeTypes(forgeTypesCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestForgeFilters_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/forge/common-filters", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"filters": map[string]string{"stable": "^[0-9]+\\.[0-9]+$", "lts": "^[0-9]+\\..*-lts$"},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runForgeFilters(forgeFiltersCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Setting Commands Tests
// =============================================================================

func TestSettingList_MockServer(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/settings", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"settings": map[string]interface{}{
				"max_downloads": map[string]interface{}{"key": "max_downloads", "value": 5, "type": "int"},
			},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "json"
	err := runSettingList(settingListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Empty List Tests (table output with zero results)
// =============================================================================

func TestDistributionList_EmptyResult(t *testing.T) {
	defer resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/distributions", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":         0,
			"distributions": []interface{}{},
		})
	})
	srv := setupTestClient(t, mux)
	defer srv.Close()

	outputFormat = "table"
	// Should print "No distributions found." without error
	err := runDistributionList(distributionListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Suppress unused import warning
var _ = fmt.Sprintf
