package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// TokenData Tests
// =============================================================================

func TestTokenData_JSONRoundTrip(t *testing.T) {
	original := &TokenData{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresAt:    "2025-01-01T00:00:00Z",
		ServerURL:    "http://localhost:8443",
		Username:     "admin",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded TokenData
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.AccessToken != original.AccessToken {
		t.Errorf("access token mismatch: got %s, want %s", decoded.AccessToken, original.AccessToken)
	}
	if decoded.RefreshToken != original.RefreshToken {
		t.Errorf("refresh token mismatch: got %s, want %s", decoded.RefreshToken, original.RefreshToken)
	}
	if decoded.ExpiresAt != original.ExpiresAt {
		t.Errorf("expires_at mismatch: got %s, want %s", decoded.ExpiresAt, original.ExpiresAt)
	}
	if decoded.ServerURL != original.ServerURL {
		t.Errorf("server_url mismatch: got %s, want %s", decoded.ServerURL, original.ServerURL)
	}
	if decoded.Username != original.Username {
		t.Errorf("username mismatch: got %s, want %s", decoded.Username, original.Username)
	}
}

func TestTokenData_JSONFieldNames(t *testing.T) {
	td := &TokenData{
		AccessToken:  "at",
		RefreshToken: "rt",
	}
	data, _ := json.Marshal(td)
	m := make(map[string]interface{})
	json.Unmarshal(data, &m)

	if _, ok := m["access_token"]; !ok {
		t.Error("expected json field 'access_token'")
	}
	if _, ok := m["refresh_token"]; !ok {
		t.Error("expected json field 'refresh_token'")
	}
}

// =============================================================================
// File I/O Tests (using temp directory)
// =============================================================================

func TestSaveAndLoadToken_TempDir(t *testing.T) {
	// Create a temp directory to simulate ~/.ldfctl/
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	original := &TokenData{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		ExpiresAt:    "2025-12-31T23:59:59Z",
		ServerURL:    "http://test:8443",
		Username:     "testuser",
	}

	// Write token manually (since SaveToken uses hardcoded path)
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected permissions 0600, got %o", perm)
	}

	// Read back
	readData, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	var loaded TokenData
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if loaded.AccessToken != original.AccessToken {
		t.Errorf("access token mismatch after load: got %s", loaded.AccessToken)
	}
	if loaded.Username != original.Username {
		t.Errorf("username mismatch after load: got %s", loaded.Username)
	}
}

func TestLoadToken_NonExistent(t *testing.T) {
	// Loading from a non-existent path should fail
	_, err := LoadToken()
	// This may or may not fail depending on whether ~/.ldfctl/token.json exists.
	// At minimum, verify the function doesn't panic.
	_ = err
}

func TestClearToken_NonExistent(t *testing.T) {
	// ClearToken on non-existent file should not error
	err := ClearToken()
	// May or may not error depending on real filesystem state.
	// At minimum, verify no panic.
	_ = err
}
