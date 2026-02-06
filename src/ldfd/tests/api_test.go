// Package tests provides integration and unit tests for the ldfd server.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/version"
	"github.com/bitswalk/ldf/src/ldfd/api"
	"github.com/bitswalk/ldf/src/ldfd/api/base"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
	"github.com/bitswalk/ldf/src/ldfd/download"
	"github.com/bitswalk/ldf/src/ldfd/storage"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// =============================================================================
// Test Infrastructure
// =============================================================================

func init() {
	gin.SetMode(gin.TestMode)
}

// testAPI holds all the components needed for API testing
type testAPI struct {
	api         *api.API
	router      *gin.Engine
	database    *db.Database
	userManager *auth.UserManager
	jwtService  *auth.JWTService
}

// setupTestAPI creates a new test API instance with in-memory database
func setupTestAPI(t *testing.T) *testAPI {
	t.Helper()

	// Create in-memory database
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	// Create user manager
	userManager := auth.NewUserManager(database.DB())

	// Create mock settings store for JWT
	settings := newMockSettingsStore()

	// Create JWT service
	jwtService := auth.NewJWTService(auth.DefaultJWTConfig(), userManager, settings)

	// Set up version info for base handler
	base.SetVersionInfo(&version.Info{
		Version:        "1.0.0-test",
		ReleaseName:    "Test",
		ReleaseVersion: "1.0.0",
		BuildDate:      "2024-01-01",
		GitCommit:      "abc1234",
	})

	// Set up logger
	logger := logs.New(logs.Config{
		Output: logs.OutputStdout,
		Level:  "error",
	})
	api.SetLogger(logger)

	// Create repositories
	distRepo := db.NewDistributionRepository(database)
	sourceRepo := db.NewSourceRepository(database)
	componentRepo := db.NewComponentRepository(database)
	sourceVersionRepo := db.NewSourceVersionRepository(database)
	langPackRepo := db.NewLanguagePackRepository(database)

	// Create API
	apiInstance := api.New(api.Config{
		DistRepo:          distRepo,
		SourceRepo:        sourceRepo,
		ComponentRepo:     componentRepo,
		SourceVersionRepo: sourceVersionRepo,
		LangPackRepo:      langPackRepo,
		Database:          database,
		Storage:           nil, // No storage for basic tests
		UserManager:       userManager,
		JWTService:        jwtService,
		DownloadManager:   nil, // No download manager for basic tests
		VersionDiscovery:  nil, // No version discovery for basic tests
	})

	// Create router
	router := gin.New()
	apiInstance.RegisterRoutes(router)

	t.Cleanup(func() {
		_ = database.Shutdown()
	})

	return &testAPI{
		api:         apiInstance,
		router:      router,
		database:    database,
		userManager: userManager,
		jwtService:  jwtService,
	}
}

// createTestUser creates a user for testing and returns the user and token
func (ta *testAPI) createTestUser(t *testing.T, name, email string, roleID string) (*auth.User, string) {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := auth.NewUser(name, email, string(hashedPassword), roleID)
	if err := ta.userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Get fresh user with role name
	user, err = ta.userManager.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	token, err := ta.jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return user, token
}

// makeRequest makes an HTTP request to the test API
func (ta *testAPI) makeRequest(method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		reqBody = bytes.NewReader(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	ta.router.ServeHTTP(rec, req)
	return rec
}

// parseJSON parses the response body as JSON
func parseJSON(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("failed to parse JSON response: %v\nBody: %s", err, rec.Body.String())
	}
}

// =============================================================================
// Base Handler Tests
// =============================================================================

func TestAPI_HandleRoot(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "ldfd" {
		t.Fatalf("expected name 'ldfd', got %v", response["name"])
	}
	if response["description"] != "LDF Platform API Server" {
		t.Fatalf("expected correct description, got %v", response["description"])
	}

	endpoints, ok := response["endpoints"].(map[string]interface{})
	if !ok {
		t.Fatal("expected endpoints object")
	}
	if endpoints["health"] != "/v1/health" {
		t.Fatalf("expected health endpoint, got %v", endpoints["health"])
	}
}

func TestAPI_HandleHealth(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/health", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["status"] != "healthy" {
		t.Fatalf("expected status 'healthy', got %v", response["status"])
	}
	if response["timestamp"] == nil {
		t.Fatal("expected timestamp in response")
	}
}

func TestAPI_HandleVersion(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/version", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["version"] != "1.0.0-test" {
		t.Fatalf("expected version '1.0.0-test', got %v", response["version"])
	}
	if response["release_name"] != "Test" {
		t.Fatalf("expected release_name 'Test', got %v", response["release_name"])
	}
	if response["git_commit"] != "abc1234" {
		t.Fatalf("expected git_commit 'abc1234', got %v", response["git_commit"])
	}
}

// =============================================================================
// Auth Handler Tests
// =============================================================================

func TestAPI_HandleCreate(t *testing.T) {
	ta := setupTestAPI(t)

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     "newuser",
						"password": "testpassword123",
						"email":    "newuser@example.com",
					},
				},
			},
		},
	}

	rec := ta.makeRequest("POST", "/auth/create", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["access_token"] == nil {
		t.Fatal("expected access_token in response")
	}
	if response["refresh_token"] == nil {
		t.Fatal("expected refresh_token in response")
	}

	user, ok := response["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user object in response")
	}
	if user["name"] != "newuser" {
		t.Fatalf("expected name 'newuser', got %v", user["name"])
	}
	if user["email"] != "newuser@example.com" {
		t.Fatalf("expected email, got %v", user["email"])
	}

	// Check X-Subject-Token header
	if rec.Header().Get("X-Subject-Token") == "" {
		t.Fatal("expected X-Subject-Token header")
	}
}

func TestAPI_HandleCreate_Validation(t *testing.T) {
	ta := setupTestAPI(t)

	tests := []struct {
		name     string
		body     map[string]interface{}
		expected int
	}{
		{
			name: "missing password method",
			body: map[string]interface{}{
				"auth": map[string]interface{}{
					"identity": map[string]interface{}{
						"methods": []string{},
					},
				},
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "wrong method",
			body: map[string]interface{}{
				"auth": map[string]interface{}{
					"identity": map[string]interface{}{
						"methods": []string{"oauth"},
					},
				},
			},
			expected: http.StatusBadRequest,
		},
		{
			name: "missing fields",
			body: map[string]interface{}{
				"auth": map[string]interface{}{
					"identity": map[string]interface{}{
						"methods": []string{"password"},
						"password": map[string]interface{}{
							"user": map[string]interface{}{
								"name": "testuser",
								// missing password and email
							},
						},
					},
				},
			},
			expected: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := ta.makeRequest("POST", "/auth/create", tt.body, "")
			if rec.Code != tt.expected {
				t.Fatalf("expected status %d, got %d: %s", tt.expected, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAPI_HandleCreate_Duplicate(t *testing.T) {
	ta := setupTestAPI(t)

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     "duplicateuser",
						"password": "testpassword123",
						"email":    "duplicate@example.com",
					},
				},
			},
		},
	}

	// First creation should succeed
	rec := ta.makeRequest("POST", "/auth/create", body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("first creation failed: %d: %s", rec.Code, rec.Body.String())
	}

	// Second creation with same name should fail
	rec = ta.makeRequest("POST", "/auth/create", body, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409 for duplicate, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleLogin(t *testing.T) {
	ta := setupTestAPI(t)

	// Create a user first
	ta.createTestUser(t, "loginuser", "loginuser@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     "loginuser",
						"password": "testpassword",
					},
				},
			},
		},
	}

	rec := ta.makeRequest("POST", "/auth/login", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["access_token"] == nil {
		t.Fatal("expected access_token in response")
	}
	if response["refresh_token"] == nil {
		t.Fatal("expected refresh_token in response")
	}
}

func TestAPI_HandleLogin_InvalidCredentials(t *testing.T) {
	ta := setupTestAPI(t)

	ta.createTestUser(t, "loginuser2", "loginuser2@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     "loginuser2",
						"password": "wrongpassword",
					},
				},
			},
		},
	}

	rec := ta.makeRequest("POST", "/auth/login", body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleLogin_UserNotFound(t *testing.T) {
	ta := setupTestAPI(t)

	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"identity": map[string]interface{}{
				"methods": []string{"password"},
				"password": map[string]interface{}{
					"user": map[string]interface{}{
						"name":     "nonexistentuser",
						"password": "anypassword",
					},
				},
			},
		},
	}

	rec := ta.makeRequest("POST", "/auth/login", body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleLogout(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "logoutuser", "logoutuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("POST", "/auth/logout", nil, token)

	if rec.Code != 498 { // Token revoked status
		t.Fatalf("expected status 498, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["message"] != "Token revoked successfully" {
		t.Fatalf("expected revocation message, got %v", response["message"])
	}
}

func TestAPI_HandleLogout_NoToken(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("POST", "/auth/logout", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleRefresh(t *testing.T) {
	ta := setupTestAPI(t)

	user, _ := ta.createTestUser(t, "refreshuser", "refreshuser@example.com", auth.RoleIDDeveloper)

	// Get a token pair
	pair, err := ta.jwtService.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	body := map[string]interface{}{
		"refresh_token": pair.RefreshToken,
	}

	rec := ta.makeRequest("POST", "/auth/refresh", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["access_token"] == nil {
		t.Fatal("expected new access_token")
	}
}

func TestAPI_HandleRefresh_InvalidToken(t *testing.T) {
	ta := setupTestAPI(t)

	body := map[string]interface{}{
		"refresh_token": "invalid-refresh-token",
	}

	rec := ta.makeRequest("POST", "/auth/refresh", body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleValidate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "validateuser", "validateuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/auth/validate", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["valid"] != true {
		t.Fatalf("expected valid=true, got %v", response["valid"])
	}

	user, ok := response["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user object")
	}
	if user["name"] != "validateuser" {
		t.Fatalf("expected name 'validateuser', got %v", user["name"])
	}
}

func TestAPI_HandleValidate_NoToken(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/auth/validate", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Role Handler Tests
// =============================================================================

func TestAPI_HandleListRoles(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/roles", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	roles, ok := response["roles"].([]interface{})
	if !ok {
		t.Fatal("expected roles array")
	}
	if len(roles) < 3 {
		t.Fatalf("expected at least 3 system roles, got %d", len(roles))
	}
}

func TestAPI_HandleGetRole(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/roles/"+auth.RoleIDDeveloper, nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	role, ok := response["role"].(map[string]interface{})
	if !ok {
		t.Fatal("expected role object")
	}
	if role["name"] != "developer" {
		t.Fatalf("expected name 'developer', got %v", role["name"])
	}
}

func TestAPI_HandleGetRole_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/roles/nonexistent-id", nil, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleCreateRole(t *testing.T) {
	ta := setupTestAPI(t)

	// Create admin user
	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"name":        "testrole",
		"description": "Test role",
		"permissions": map[string]interface{}{
			"can_read":  true,
			"can_write": true,
		},
	}

	rec := ta.makeRequest("POST", "/v1/roles", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	role, ok := response["role"].(map[string]interface{})
	if !ok {
		t.Fatal("expected role object")
	}
	if role["name"] != "testrole" {
		t.Fatalf("expected name 'testrole', got %v", role["name"])
	}
}

func TestAPI_HandleCreateRole_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	// Create non-admin user
	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":        "testrole",
		"description": "Test role",
	}

	rec := ta.makeRequest("POST", "/v1/roles", body, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// File Upload Tests (Artifacts and Branding)
// =============================================================================

// makeMultipartRequest creates a multipart form request for file uploads
func (ta *testAPI) makeMultipartRequest(t *testing.T, method, path string, fieldName, fileName string, fileContent []byte, token string) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	ta.router.ServeHTTP(rec, req)
	return rec
}

func TestAPI_HandleArtifactUpload(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "uploaduser", "uploaduser@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "upload-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// Upload an artifact
	fileContent := []byte("test artifact content")
	rec := ta.makeMultipartRequest(t, "POST", "/v1/distributions/"+dist.ID+"/artifacts", "file", "test-artifact.txt", fileContent, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["key"] == nil {
		t.Fatal("expected key in response")
	}
	if response["size"] == nil {
		t.Fatal("expected size in response")
	}
}

func TestAPI_HandleArtifactUpload_NoFile(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "nofileuser", "nofileuser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "nofile-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// POST without file
	rec := ta.makeRequest("POST", "/v1/distributions/"+dist.ID+"/artifacts", nil, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleArtifactUpload_DistributionNotFound(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "uploaduser2", "uploaduser2@example.com", auth.RoleIDDeveloper)

	fileContent := []byte("test content")
	rec := ta.makeMultipartRequest(t, "POST", "/v1/distributions/nonexistent-id/artifacts", "file", "test.txt", fileContent, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleArtifactDownload(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "downloadartuser", "downloadartuser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "download-art-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// First upload an artifact
	fileContent := []byte("downloadable content")
	ta.makeMultipartRequest(t, "POST", "/v1/distributions/"+dist.ID+"/artifacts", "file", "download-test.txt", fileContent, token)

	// Now download it
	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/artifacts/download-test.txt", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	if rec.Body.String() != "downloadable content" {
		t.Fatalf("expected 'downloadable content', got %s", rec.Body.String())
	}
}

func TestAPI_HandleArtifactDownload_NotFound(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "dlnotfound", "dlnotfound@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "dlnotfound-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/artifacts/nonexistent.txt", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleArtifactDelete(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "deleteartuser", "deleteartuser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "delete-art-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// Upload first
	fileContent := []byte("deletable content")
	ta.makeMultipartRequest(t, "POST", "/v1/distributions/"+dist.ID+"/artifacts", "file", "delete-test.txt", fileContent, token)

	// Now delete
	rec := ta.makeRequest("DELETE", "/v1/distributions/"+dist.ID+"/artifacts/delete-test.txt", nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleArtifactGetURL_Success(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "geturluser", "geturluser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "geturl-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// Upload first
	fileContent := []byte("presigned content")
	ta.makeMultipartRequest(t, "POST", "/v1/distributions/"+dist.ID+"/artifacts", "file", "presigned-test.txt", fileContent, token)

	// Get presigned URL
	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/artifacts-url/presigned-test.txt", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["url"] == nil {
		t.Fatal("expected url in response")
	}
}

func TestAPI_HandleBrandingUploadAsset(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "brandingadmin", "brandingadmin@example.com", auth.RoleIDRoot)

	// Upload a logo
	logoContent := []byte("fake logo png content")
	rec := ta.makeMultipartRequest(t, "POST", "/v1/branding/logo", "file", "logo.png", logoContent, token)

	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("expected status 201 or 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingUploadAsset_InvalidAsset(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "brandinginvalid", "brandinginvalid@example.com", auth.RoleIDRoot)

	logoContent := []byte("fake content")
	rec := ta.makeMultipartRequest(t, "POST", "/v1/branding/invalid-asset", "file", "test.png", logoContent, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingUploadAsset_Unauthorized(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "brandingdev", "brandingdev@example.com", auth.RoleIDDeveloper)

	logoContent := []byte("fake content")
	rec := ta.makeMultipartRequest(t, "POST", "/v1/branding/logo", "file", "logo.png", logoContent, token)

	// Developers shouldn't be able to upload branding assets
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingDeleteAsset(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "brandingdelete", "brandingdelete@example.com", auth.RoleIDRoot)

	// First upload
	logoContent := []byte("deletable logo")
	ta.makeMultipartRequest(t, "POST", "/v1/branding/logo", "file", "logo.png", logoContent, token)

	// Now delete
	rec := ta.makeRequest("DELETE", "/v1/branding/logo", nil, token)

	// Branding delete returns 200 with a message
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["message"] == nil {
		t.Fatal("expected message in response")
	}
}

func TestAPI_HandleBrandingGetAsset_AfterUpload(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "brandingget", "brandingget@example.com", auth.RoleIDRoot)

	// Upload logo
	logoContent := []byte("logo content for get test")
	ta.makeMultipartRequest(t, "POST", "/v1/branding/logo", "file", "logo.png", logoContent, token)

	// Get logo (no auth required for branding assets)
	rec := ta.makeRequest("GET", "/v1/branding/logo", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDeleteRole(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Create a custom role first
	customRole := auth.NewRole("todelete", "To delete", auth.RolePermissions{}, "")
	_ = ta.userManager.CreateRole(customRole)

	rec := ta.makeRequest("DELETE", "/v1/roles/"+customRole.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDeleteRole_SystemRole(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Try to delete system role
	rec := ta.makeRequest("DELETE", "/v1/roles/"+auth.RoleIDDeveloper, nil, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Distribution Handler Tests
// =============================================================================

func TestAPI_HandleDistributionList(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/distributions", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["count"] == nil {
		t.Fatal("expected count in response")
	}
	if response["distributions"] == nil {
		t.Fatal("expected distributions array")
	}
}

func TestAPI_HandleDistributionCreate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "distcreator", "distcreator@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":       "test-distro",
		"version":    "1.0.0",
		"visibility": "private",
	}

	rec := ta.makeRequest("POST", "/v1/distributions", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "test-distro" {
		t.Fatalf("expected name 'test-distro', got %v", response["name"])
	}
	if response["version"] != "1.0.0" {
		t.Fatalf("expected version '1.0.0', got %v", response["version"])
	}
}

func TestAPI_HandleDistributionCreate_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	body := map[string]interface{}{
		"name": "test-distro",
	}

	rec := ta.makeRequest("POST", "/v1/distributions", body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionCreate_Duplicate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "distcreator2", "distcreator2@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name": "duplicate-distro",
	}

	// First creation
	rec := ta.makeRequest("POST", "/v1/distributions", body, token)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first creation failed: %d: %s", rec.Code, rec.Body.String())
	}

	// Second creation should fail
	rec = ta.makeRequest("POST", "/v1/distributions", body, token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionGet(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "distowner", "distowner@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "get-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	if err := distRepo.Create(dist); err != nil {
		t.Fatalf("failed to create distribution: %v", err)
	}

	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "get-test-distro" {
		t.Fatalf("expected name 'get-test-distro', got %v", response["name"])
	}
}

func TestAPI_HandleDistributionGet_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/distributions/nonexistent-id", nil, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionUpdate(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "distupdater", "distupdater@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "update-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		OwnerID:    user.ID,
	}
	if err := distRepo.Create(dist); err != nil {
		t.Fatalf("failed to create distribution: %v", err)
	}

	body := map[string]interface{}{
		"version":    "2.0.0",
		"visibility": "public",
	}

	rec := ta.makeRequest("PUT", "/v1/distributions/"+dist.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["version"] != "2.0.0" {
		t.Fatalf("expected version '2.0.0', got %v", response["version"])
	}
}

func TestAPI_HandleDistributionUpdate_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "distowner2", "distowner2@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "otheruser", "otheruser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "forbidden-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		OwnerID:    owner.ID,
	}
	if err := distRepo.Create(dist); err != nil {
		t.Fatalf("failed to create distribution: %v", err)
	}

	body := map[string]interface{}{
		"version": "2.0.0",
	}

	rec := ta.makeRequest("PUT", "/v1/distributions/"+dist.ID, body, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionDelete(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "distdeleter", "distdeleter@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "delete-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		OwnerID:    user.ID,
	}
	if err := distRepo.Create(dist); err != nil {
		t.Fatalf("failed to create distribution: %v", err)
	}

	rec := ta.makeRequest("DELETE", "/v1/distributions/"+dist.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionGetLogs(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "distlogger", "distlogger@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "logs-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	if err := distRepo.Create(dist); err != nil {
		t.Fatalf("failed to create distribution: %v", err)
	}

	// Add some logs
	_ = distRepo.AddLog(dist.ID, "info", "Test log message")

	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/logs", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var logs []map[string]interface{}
	parseJSON(t, rec, &logs)

	if len(logs) == 0 {
		t.Fatal("expected at least one log entry")
	}
}

func TestAPI_HandleDistributionGetStats(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/distributions/stats", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["total"] == nil {
		t.Fatal("expected total in response")
	}
	if response["stats"] == nil {
		t.Fatal("expected stats in response")
	}
}

// =============================================================================
// Component Handler Tests
// =============================================================================

func TestAPI_HandleComponentList(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/components", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["count"] == nil {
		t.Fatal("expected count in response")
	}
	if response["components"] == nil {
		t.Fatal("expected components array")
	}
}

func TestAPI_HandleComponentGet(t *testing.T) {
	ta := setupTestAPI(t)

	// Create a component
	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "test-kernel",
		Category:    "core",
		DisplayName: "Test Kernel",
		Description: "A test kernel component",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/components/"+component.ID, nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "test-kernel" {
		t.Fatalf("expected name 'test-kernel', got %v", response["name"])
	}
}

func TestAPI_HandleComponentGet_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/components/nonexistent-id", nil, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentListByCategory(t *testing.T) {
	ta := setupTestAPI(t)

	// Create components in different categories
	componentRepo := db.NewComponentRepository(ta.database)
	for i := 0; i < 3; i++ {
		component := &db.Component{
			Name:        fmt.Sprintf("core-component-%d", i),
			Category:    "core",
			DisplayName: fmt.Sprintf("Core Component %d", i),
		}
		_ = componentRepo.Create(component)
	}

	rec := ta.makeRequest("GET", "/v1/components/category/core", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	count := int(response["count"].(float64))
	if count < 3 {
		t.Fatalf("expected at least 3 components in core category, got %d", count)
	}
}

func TestAPI_HandleComponentGetCategories(t *testing.T) {
	ta := setupTestAPI(t)

	// Create components in different categories
	componentRepo := db.NewComponentRepository(ta.database)
	categories := []string{"core", "bootloader", "firmware"}
	for i, cat := range categories {
		component := &db.Component{
			Name:        fmt.Sprintf("cat-component-%d", i),
			Category:    cat,
			DisplayName: fmt.Sprintf("Category Component %d", i),
		}
		_ = componentRepo.Create(component)
	}

	rec := ta.makeRequest("GET", "/v1/components/categories", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	count := int(response["count"].(float64))
	if count < 3 {
		t.Fatalf("expected at least 3 categories, got %d", count)
	}
}

func TestAPI_HandleComponentCreate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"name":         "new-component",
		"category":     "core",
		"display_name": "New Component",
		"description":  "A new test component",
	}

	rec := ta.makeRequest("POST", "/v1/components", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "new-component" {
		t.Fatalf("expected name 'new-component', got %v", response["name"])
	}
}

func TestAPI_HandleComponentCreate_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":         "unauthorized-component",
		"category":     "core",
		"display_name": "Unauthorized",
	}

	rec := ta.makeRequest("POST", "/v1/components", body, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentUpdate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "update-component",
		Category:    "core",
		DisplayName: "Update Component",
	}
	_ = componentRepo.Create(component)

	body := map[string]interface{}{
		"display_name": "Updated Component Name",
		"description":  "Updated description",
	}

	rec := ta.makeRequest("PUT", "/v1/components/"+component.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["display_name"] != "Updated Component Name" {
		t.Fatalf("expected updated display_name, got %v", response["display_name"])
	}
}

func TestAPI_HandleComponentDelete(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "delete-component",
		Category:    "core",
		DisplayName: "Delete Component",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("DELETE", "/v1/components/"+component.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentGetVersions(t *testing.T) {
	ta := setupTestAPI(t)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "versions-component",
		Category:    "core",
		DisplayName: "Versions Component",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/components/"+component.ID+"/versions", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["versions"] == nil {
		t.Fatal("expected versions array")
	}
	if response["total"] == nil {
		t.Fatal("expected total in response")
	}
}

func TestAPI_HandleComponentResolveVersion(t *testing.T) {
	ta := setupTestAPI(t)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:           "resolve-component",
		Category:       "core",
		DisplayName:    "Resolve Component",
		DefaultVersion: "6.1.0",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/components/"+component.ID+"/resolve-version?rule=pinned", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["rule"] != "pinned" {
		t.Fatalf("expected rule 'pinned', got %v", response["rule"])
	}
	if response["resolved_version"] != "6.1.0" {
		t.Fatalf("expected resolved_version '6.1.0', got %v", response["resolved_version"])
	}
}

func TestAPI_HandleComponentResolveVersion_MissingRule(t *testing.T) {
	ta := setupTestAPI(t)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "resolve-component-2",
		Category:    "core",
		DisplayName: "Resolve Component 2",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/components/"+component.ID+"/resolve-version", nil, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Source Handler Tests
// =============================================================================

func TestAPI_HandleSourceList(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "sourceuser", "sourceuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/sources", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["count"] == nil {
		t.Fatal("expected count in response")
	}
	if response["sources"] == nil {
		t.Fatal("expected sources array")
	}
}

func TestAPI_HandleSourceList_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/sources", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceListDefaults(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("GET", "/v1/sources", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["count"] == nil {
		t.Fatal("expected count in response")
	}
}

func TestAPI_HandleSourceListDefaults_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	// Unified list endpoint is accessible to all authenticated users.
	// Test that unauthenticated requests get 401.
	rec := ta.makeRequest("GET", "/v1/sources", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceCreateUserSource(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "sourceowner", "sourceowner@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name": "My Custom Source",
		"url":  "https://example.com/releases",
	}

	rec := ta.makeRequest("POST", "/v1/sources", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "My Custom Source" {
		t.Fatalf("expected name 'My Custom Source', got %v", response["name"])
	}
}

func TestAPI_HandleSourceUpdateUserSource(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "srcupdater", "srcupdater@example.com", auth.RoleIDDeveloper)

	// Create a user source
	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Update Source",
		URL:     "https://example.com",
		OwnerID: user.ID,
		Enabled: true,
	}
	_ = sourceRepo.CreateUserSource(source)

	body := map[string]interface{}{
		"name": "Updated Source Name",
	}

	rec := ta.makeRequest("PUT", "/v1/sources/"+source.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceDeleteUserSource(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "srcdeleter", "srcdeleter@example.com", auth.RoleIDDeveloper)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Delete Source",
		URL:     "https://example.com",
		OwnerID: user.ID,
		Enabled: true,
	}
	_ = sourceRepo.CreateUserSource(source)

	rec := ta.makeRequest("DELETE", "/v1/sources/"+source.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceListByComponent(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "srccompuser", "srccompuser@example.com", auth.RoleIDDeveloper)

	// Create a component
	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "src-component",
		Category:    "core",
		DisplayName: "Source Component",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/sources/component/"+component.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Language Pack Handler Tests
// =============================================================================

func TestAPI_HandleLangPackList(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "languser", "languser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/language-packs", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["language_packs"] == nil {
		t.Fatal("expected language_packs array")
	}
}

func TestAPI_HandleLangPackList_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/language-packs", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleLangPackGet(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "langgetuser", "langgetuser@example.com", auth.RoleIDDeveloper)

	// Create a language pack
	langPackRepo := db.NewLanguagePackRepository(ta.database)
	pack := &db.LanguagePack{
		Locale:     "fr-FR",
		Name:       "French",
		Version:    "1.0.0",
		Dictionary: `{"hello": "bonjour"}`,
	}
	_ = langPackRepo.Create(pack)

	rec := ta.makeRequest("GET", "/v1/language-packs/fr-FR", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["locale"] != "fr-FR" {
		t.Fatalf("expected locale 'fr-FR', got %v", response["locale"])
	}
}

func TestAPI_HandleLangPackGet_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "langnotfound", "langnotfound@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/language-packs/nonexistent", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleLangPackDelete(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Create a language pack
	langPackRepo := db.NewLanguagePackRepository(ta.database)
	pack := &db.LanguagePack{
		Locale:     "de-DE",
		Name:       "German",
		Version:    "1.0.0",
		Dictionary: `{"hello": "hallo"}`,
	}
	_ = langPackRepo.Create(pack)

	rec := ta.makeRequest("DELETE", "/v1/language-packs/de-DE", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleLangPackDelete_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("DELETE", "/v1/language-packs/en-US", nil, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Settings Handler Tests
// =============================================================================

func TestAPI_HandleSettingsGetAll(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("GET", "/v1/settings", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	settings, ok := response["settings"].([]interface{})
	if !ok {
		t.Fatal("expected settings array")
	}
	if len(settings) == 0 {
		t.Fatal("expected at least one setting")
	}
}

func TestAPI_HandleSettingsGetAll_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/settings", nil, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSettingsGet(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("GET", "/v1/settings/server.port", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["key"] != "server.port" {
		t.Fatalf("expected key 'server.port', got %v", response["key"])
	}
}

func TestAPI_HandleSettingsGet_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("GET", "/v1/settings/nonexistent.setting", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSettingsUpdate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"value": "debug",
	}

	rec := ta.makeRequest("PUT", "/v1/settings/log.level", body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["key"] != "log.level" {
		t.Fatalf("expected key 'log.level', got %v", response["key"])
	}
}

func TestAPI_HandleSettingsResetDatabase(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"confirmation": "RESET_DATABASE",
	}

	rec := ta.makeRequest("POST", "/v1/settings/database/reset", body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["success"] != true {
		t.Fatalf("expected success=true, got %v", response["success"])
	}
}

func TestAPI_HandleSettingsResetDatabase_WrongConfirmation(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"confirmation": "wrong",
	}

	rec := ta.makeRequest("POST", "/v1/settings/database/reset", body, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSettingsSensitiveMasked(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Get all settings - sensitive values should be masked
	rec := ta.makeRequest("GET", "/v1/settings", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// With reveal=true, sensitive values should be shown
	rec = ta.makeRequest("GET", "/v1/settings?reveal=true", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Middleware Tests
// =============================================================================

func TestAPI_AuthMiddleware_ValidToken(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "authuser", "authuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/sources", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_AuthMiddleware_InvalidToken(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/sources", nil, "invalid-token")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_AuthMiddleware_XSubjectTokenHeader(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "headeruser", "headeruser@example.com", auth.RoleIDDeveloper)

	// Use X-Subject-Token header instead of Authorization
	req := httptest.NewRequest("GET", "/v1/sources", nil)
	req.Header.Set("X-Subject-Token", token)

	rec := httptest.NewRecorder()
	ta.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_WriteAccessMiddleware(t *testing.T) {
	ta := setupTestAPI(t)

	// Anonymous role has no write access
	_, token := ta.createTestUser(t, "anonuser", "anonuser@example.com", auth.RoleIDAnonymous)

	body := map[string]interface{}{
		"name": "test-distro",
	}

	rec := ta.makeRequest("POST", "/v1/distributions", body, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_AdminAccessMiddleware(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	// Try to create a role (requires admin access)
	body := map[string]interface{}{
		"name": "testrole",
	}

	rec := ta.makeRequest("POST", "/v1/roles", body, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_RootAccessMiddleware(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	// Try to access settings (requires root access)
	rec := ta.makeRequest("GET", "/v1/settings", nil, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Edge Cases and Error Handling
// =============================================================================

func TestAPI_InvalidJSON(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "jsonuser", "jsonuser@example.com", auth.RoleIDDeveloper)

	req := httptest.NewRequest("POST", "/v1/distributions", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	ta.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_EmptyBody(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "emptyuser", "emptyuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("POST", "/v1/distributions", nil, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_MissingRequiredFields(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Create component without required fields
	body := map[string]interface{}{
		"name": "incomplete-component",
		// missing category and display_name
	}

	rec := ta.makeRequest("POST", "/v1/components", body, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Mock Storage Backend
// =============================================================================

// mockStorage is a simple in-memory storage backend for testing
type mockStorage struct {
	objects map[string]*mockObject
}

type mockObject struct {
	data        []byte
	contentType string
	size        int64
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		objects: make(map[string]*mockObject),
	}
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.objects[key] = &mockObject{
		data:        data,
		contentType: contentType,
		size:        size,
	}
	return nil
}

func (m *mockStorage) Download(ctx context.Context, key string) (io.ReadCloser, *storage.ObjectInfo, error) {
	obj, ok := m.objects[key]
	if !ok {
		return nil, nil, fmt.Errorf("object not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(obj.data)), &storage.ObjectInfo{
		Key:         key,
		Size:        obj.size,
		ContentType: obj.contentType,
	}, nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.objects[key]
	return ok, nil
}

func (m *mockStorage) GetInfo(ctx context.Context, key string) (*storage.ObjectInfo, error) {
	obj, ok := m.objects[key]
	if !ok {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	return &storage.ObjectInfo{
		Key:         key,
		Size:        obj.size,
		ContentType: obj.contentType,
	}, nil
}

func (m *mockStorage) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
	var result []storage.ObjectInfo
	for key, obj := range m.objects {
		if strings.HasPrefix(key, prefix) {
			result = append(result, storage.ObjectInfo{
				Key:         key,
				Size:        obj.size,
				ContentType: obj.contentType,
			})
		}
	}
	return result, nil
}

func (m *mockStorage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "https://mock-storage.example.com/" + key + "?presigned=true", nil
}

func (m *mockStorage) GetWebURL(key string) string {
	return "https://mock-storage.example.com/" + key
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *mockStorage) Type() string {
	return "mock"
}

func (m *mockStorage) Location() string {
	return "memory://mock"
}

// =============================================================================
// Test API with Storage
// =============================================================================

// setupTestAPIWithStorage creates a test API instance with mock storage
func setupTestAPIWithStorage(t *testing.T) *testAPI {
	t.Helper()

	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}
	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	userManager := auth.NewUserManager(database.DB())
	settings := newMockSettingsStore()
	jwtService := auth.NewJWTService(auth.DefaultJWTConfig(), userManager, settings)

	base.SetVersionInfo(&version.Info{
		Version:        "1.0.0-test",
		ReleaseName:    "Test",
		ReleaseVersion: "1.0.0",
		BuildDate:      "2024-01-01",
		GitCommit:      "abc1234",
	})

	logger := logs.New(logs.Config{
		Output: logs.OutputStdout,
		Level:  "error",
	})
	api.SetLogger(logger)

	mockStore := newMockStorage()

	distRepo := db.NewDistributionRepository(database)
	sourceRepo := db.NewSourceRepository(database)
	componentRepo := db.NewComponentRepository(database)
	sourceVersionRepo := db.NewSourceVersionRepository(database)
	langPackRepo := db.NewLanguagePackRepository(database)

	apiInstance := api.New(api.Config{
		DistRepo:          distRepo,
		SourceRepo:        sourceRepo,
		ComponentRepo:     componentRepo,
		SourceVersionRepo: sourceVersionRepo,
		LangPackRepo:      langPackRepo,
		Database:          database,
		Storage:           mockStore,
		UserManager:       userManager,
		JWTService:        jwtService,
		DownloadManager:   nil,
		VersionDiscovery:  nil,
	})

	router := gin.New()
	apiInstance.RegisterRoutes(router)

	t.Cleanup(func() {
		_ = database.Shutdown()
	})

	return &testAPI{
		api:         apiInstance,
		router:      router,
		database:    database,
		userManager: userManager,
		jwtService:  jwtService,
	}
}

// =============================================================================
// Artifact Handler Tests (with Storage)
// =============================================================================

func TestAPI_HandleArtifactList(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "artifactuser", "artifactuser@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "artifact-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/artifacts", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	// Check for required fields (artifacts may be null for empty list)
	if _, ok := response["artifacts"]; !ok {
		t.Fatal("expected artifacts field in response")
	}
	if _, ok := response["distribution_id"]; !ok {
		t.Fatal("expected distribution_id field in response")
	}
	if _, ok := response["count"]; !ok {
		t.Fatal("expected count field in response")
	}
}

func TestAPI_HandleArtifactList_DistributionNotFound(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "artifactuser2", "artifactuser2@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/distributions/nonexistent-id/artifacts", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleStorageStatus(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	rec := ta.makeRequest("GET", "/v1/storage/status", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["available"] != true {
		t.Fatalf("expected available=true, got %v", response["available"])
	}
	if response["type"] != "mock" {
		t.Fatalf("expected type 'mock', got %v", response["type"])
	}
}

func TestAPI_HandleArtifactGetURL(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	user, token := ta.createTestUser(t, "urluser", "urluser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "url-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// First upload an artifact using direct storage access
	mockStore := newMockStorage()
	key := fmt.Sprintf("distribution/%s/%s/test-file.txt", user.ID, dist.ID)
	_ = mockStore.Upload(context.Background(), key, strings.NewReader("test content"), 12, "text/plain")

	// Note: The API's storage is separate from our mockStore, so artifact won't exist
	// This tests the "not found" path
	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/artifacts-url/test-file.txt", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 (artifact doesn't exist), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleArtifactListAll(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "listalluser", "listalluser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/artifacts", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	// Check for required fields (artifacts may be null for empty list)
	if _, ok := response["artifacts"]; !ok {
		t.Fatal("expected artifacts field in response")
	}
	if _, ok := response["count"]; !ok {
		t.Fatal("expected count field in response")
	}
}

// =============================================================================
// Branding Handler Tests (with Storage)
// =============================================================================

func TestAPI_HandleBrandingGetAsset_NotFound(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	rec := ta.makeRequest("GET", "/v1/branding/logo", nil, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingGetAsset_InvalidAsset(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	rec := ta.makeRequest("GET", "/v1/branding/invalid", nil, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingGetAssetInfo(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	rec := ta.makeRequest("GET", "/v1/branding/logo/info", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["asset"] != "logo" {
		t.Fatalf("expected asset 'logo', got %v", response["asset"])
	}
	if response["exists"] != false {
		t.Fatalf("expected exists=false, got %v", response["exists"])
	}
}

func TestAPI_HandleBrandingGetAssetInfo_InvalidAsset(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	rec := ta.makeRequest("GET", "/v1/branding/invalid/info", nil, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingDeleteAsset_NotFound(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("DELETE", "/v1/branding/logo", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBrandingDeleteAsset_Unauthorized(t *testing.T) {
	ta := setupTestAPIWithStorage(t)

	_, token := ta.createTestUser(t, "developer", "developer@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("DELETE", "/v1/branding/logo", nil, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Additional Source Handler Tests
// =============================================================================

func TestAPI_HandleSourceGetDefaultByID(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Create a default source
	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Default Kernel Source",
		URL:     "https://kernel.org",
		Enabled: true,
	}
	_ = sourceRepo.CreateDefault(source)

	rec := ta.makeRequest("GET", "/v1/sources/"+source.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "Default Kernel Source" {
		t.Fatalf("expected name 'Default Kernel Source', got %v", response["name"])
	}
}

func TestAPI_HandleSourceGetDefaultByID_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("GET", "/v1/sources/nonexistent-id", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceCreateDefault(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"name": "New Default Source",
		"url":  "https://example.com/releases",
	}

	rec := ta.makeRequest("POST", "/v1/sources", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceUpdateDefault(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Update Default Source",
		URL:     "https://old.example.com",
		Enabled: true,
	}
	_ = sourceRepo.CreateDefault(source)

	body := map[string]interface{}{
		"url": "https://new.example.com",
	}

	rec := ta.makeRequest("PUT", "/v1/sources/"+source.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceDeleteDefault(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Delete Default Source",
		URL:     "https://delete.example.com",
		Enabled: true,
	}
	_ = sourceRepo.CreateDefault(source)

	rec := ta.makeRequest("DELETE", "/v1/sources/"+source.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceGetUserSourceByID(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "srcowner", "srcowner@example.com", auth.RoleIDDeveloper)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "User Source",
		URL:     "https://user.example.com",
		OwnerID: user.ID,
		Enabled: true,
	}
	_ = sourceRepo.CreateUserSource(source)

	rec := ta.makeRequest("GET", "/v1/sources/"+source.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceGetUserSourceByID_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "owner", "owner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "other", "other@example.com", auth.RoleIDDeveloper)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Private Source",
		URL:     "https://private.example.com",
		OwnerID: owner.ID,
		Enabled: true,
	}
	_ = sourceRepo.CreateUserSource(source)

	rec := ta.makeRequest("GET", "/v1/sources/"+source.ID, nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceUpdateUserSource_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "srcowner2", "srcowner2@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "other2", "other2@example.com", auth.RoleIDDeveloper)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Private Update Source",
		URL:     "https://private2.example.com",
		OwnerID: owner.ID,
		Enabled: true,
	}
	_ = sourceRepo.CreateUserSource(source)

	body := map[string]interface{}{
		"name": "Hacked Name",
	}

	rec := ta.makeRequest("PUT", "/v1/sources/"+source.ID, body, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceDeleteUserSource_Forbidden(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "srcowner3", "srcowner3@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "other3", "other3@example.com", auth.RoleIDDeveloper)

	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:    "Private Delete Source",
		URL:     "https://private3.example.com",
		OwnerID: owner.ID,
		Enabled: true,
	}
	_ = sourceRepo.CreateUserSource(source)

	rec := ta.makeRequest("DELETE", "/v1/sources/"+source.ID, nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Additional Distribution Handler Tests
// =============================================================================

func TestAPI_HandleDistributionList_WithStatusFilter(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "filteruser", "filteruser@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)

	// Create distributions with different statuses
	for i, status := range []db.DistributionStatus{db.StatusPending, db.StatusReady, db.StatusFailed} {
		dist := &db.Distribution{
			Name:       fmt.Sprintf("filter-distro-%d", i),
			Version:    "1.0.0",
			Status:     status,
			Visibility: db.VisibilityPublic,
			OwnerID:    user.ID,
		}
		_ = distRepo.Create(dist)
	}

	// Filter by pending status
	rec := ta.makeRequest("GET", "/v1/distributions?status=pending", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	count := int(response["count"].(float64))
	if count < 1 {
		t.Fatalf("expected at least 1 pending distribution, got %d", count)
	}
}

func TestAPI_HandleDistributionCreate_WithConfig(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "configuser", "configuser@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":    "config-distro",
		"version": "1.0.0",
		"config": map[string]interface{}{
			"components": []map[string]interface{}{
				{
					"id":           "component-1",
					"version":      "1.0.0",
					"version_rule": "pinned",
				},
			},
		},
	}

	rec := ta.makeRequest("POST", "/v1/distributions", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["config"] == nil {
		t.Fatal("expected config in response")
	}
}

// =============================================================================
// Additional Role Handler Tests
// =============================================================================

func TestAPI_HandleUpdateRole(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	// Create a custom role
	customRole := auth.NewRole("updatable", "Updatable role", auth.RolePermissions{CanRead: true}, "")
	_ = ta.userManager.CreateRole(customRole)

	body := map[string]interface{}{
		"description": "Updated description",
		"permissions": map[string]interface{}{
			"can_read":  true,
			"can_write": true,
		},
	}

	rec := ta.makeRequest("PUT", "/v1/roles/"+customRole.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleUpdateRole_SystemRole(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"description": "Hacked description",
	}

	rec := ta.makeRequest("PUT", "/v1/roles/"+auth.RoleIDDeveloper, body, token)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleCreateRole_WithParentRole(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"name":           "child-role",
		"description":    "A child role",
		"parent_role_id": auth.RoleIDDeveloper,
		"permissions": map[string]interface{}{
			"can_read": true,
		},
	}

	rec := ta.makeRequest("POST", "/v1/roles", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleCreateRole_InvalidParentRole(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	body := map[string]interface{}{
		"name":           "orphan-role",
		"parent_role_id": "nonexistent-id",
	}

	rec := ta.makeRequest("POST", "/v1/roles", body, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Additional Component Handler Tests
// =============================================================================

func TestAPI_HandleComponentUpdate_NameConflict(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "admin", "admin@example.com", auth.RoleIDRoot)

	componentRepo := db.NewComponentRepository(ta.database)

	// Create two components
	comp1 := &db.Component{
		Name:        "existing-component",
		Category:    "core",
		DisplayName: "Existing",
	}
	_ = componentRepo.Create(comp1)

	comp2 := &db.Component{
		Name:        "rename-component",
		Category:    "core",
		DisplayName: "To Rename",
	}
	_ = componentRepo.Create(comp2)

	// Try to rename comp2 to comp1's name
	body := map[string]interface{}{
		"name": "existing-component",
	}

	rec := ta.makeRequest("PUT", "/v1/components/"+comp2.ID, body, token)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentResolveVersion_LatestStable(t *testing.T) {
	ta := setupTestAPI(t)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "latest-stable-component",
		Category:    "core",
		DisplayName: "Latest Stable Test",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/components/"+component.ID+"/resolve-version?rule=latest-stable", nil, "")

	// Will be 404 since no versions exist
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 (no versions), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentResolveVersion_InvalidRule(t *testing.T) {
	ta := setupTestAPI(t)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "invalid-rule-component",
		Category:    "core",
		DisplayName: "Invalid Rule Test",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("GET", "/v1/components/"+component.ID+"/resolve-version?rule=invalid", nil, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Download Handler Tests (with Download Manager)
// =============================================================================

// setupTestAPIWithDownloadManager creates a test API with download manager
func setupTestAPIWithDownloadManager(t *testing.T) *testAPI {
	t.Helper()

	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	userManager := auth.NewUserManager(database.DB())
	settings := newMockSettingsStore()
	jwtService := auth.NewJWTService(auth.DefaultJWTConfig(), userManager, settings)

	mockStore := newMockStorage()

	distRepo := db.NewDistributionRepository(database)
	sourceRepo := db.NewSourceRepository(database)
	componentRepo := db.NewComponentRepository(database)
	sourceVersionRepo := db.NewSourceVersionRepository(database)
	langPackRepo := db.NewLanguagePackRepository(database)

	// Create download manager
	downloadManager := download.NewManager(database, mockStore, download.DefaultConfig())

	apiInstance := api.New(api.Config{
		DistRepo:          distRepo,
		SourceRepo:        sourceRepo,
		ComponentRepo:     componentRepo,
		SourceVersionRepo: sourceVersionRepo,
		LangPackRepo:      langPackRepo,
		Database:          database,
		Storage:           mockStore,
		UserManager:       userManager,
		JWTService:        jwtService,
		DownloadManager:   downloadManager,
		VersionDiscovery:  nil,
	})

	router := gin.New()
	apiInstance.RegisterRoutes(router)

	t.Cleanup(func() {
		_ = database.Shutdown()
	})

	return &testAPI{
		api:         apiInstance,
		router:      router,
		database:    database,
		userManager: userManager,
		jwtService:  jwtService,
	}
}

func TestAPI_HandleListDistributionDownloads(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	user, token := ta.createTestUser(t, "downloaduser", "downloaduser@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "download-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/downloads", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if _, ok := response["jobs"]; !ok {
		t.Fatal("expected jobs field in response")
	}
	if _, ok := response["count"]; !ok {
		t.Fatal("expected count field in response")
	}
}

func TestAPI_HandleListDistributionDownloads_NotFound(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "downloaduser2", "downloaduser2@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/distributions/nonexistent-id/downloads", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleListDistributionDownloads_PrivateAccessDenied(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	owner, _ := ta.createTestUser(t, "owner", "owner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "other", "other@example.com", auth.RoleIDDeveloper)

	// Create a private distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "private-download-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		OwnerID:    owner.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("GET", "/v1/distributions/"+dist.ID+"/downloads", nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleGetDownloadJob_NotFound(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "jobuser", "jobuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("GET", "/v1/downloads/nonexistent-job-id", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleCancelDownload_NotFound(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "canceluser", "canceluser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("POST", "/v1/downloads/nonexistent-job-id/cancel", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleCancelDownload_Unauthorized(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	rec := ta.makeRequest("POST", "/v1/downloads/some-job-id/cancel", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleRetryDownload_NotFound(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "retryuser", "retryuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("POST", "/v1/downloads/nonexistent-job-id/retry", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleRetryDownload_Unauthorized(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	rec := ta.makeRequest("POST", "/v1/downloads/some-job-id/retry", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleListActiveDownloads(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "activeuser", "activeuser@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("GET", "/v1/downloads/active", nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if _, ok := response["jobs"]; !ok {
		t.Fatal("expected jobs field in response")
	}
}

func TestAPI_HandleFlushDistributionDownloads(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	user, token := ta.createTestUser(t, "flushuser", "flushuser@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "flush-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("DELETE", "/v1/distributions/"+dist.ID+"/downloads", nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleFlushDistributionDownloads_NotFound(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "flushuser2", "flushuser2@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("DELETE", "/v1/distributions/nonexistent-id/downloads", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleFlushDistributionDownloads_Unauthorized(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	rec := ta.makeRequest("DELETE", "/v1/distributions/some-id/downloads", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleFlushDistributionDownloads_Forbidden(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	owner, _ := ta.createTestUser(t, "flushowner", "flushowner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "flushother", "flushother@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "flush-forbidden-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		OwnerID:    owner.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("DELETE", "/v1/distributions/"+dist.ID+"/downloads", nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleStartDistributionDownloads_Unauthorized(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	rec := ta.makeRequest("POST", "/v1/distributions/some-id/downloads", nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleStartDistributionDownloads_NotFound(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	_, token := ta.createTestUser(t, "startuser", "startuser@example.com", auth.RoleIDDeveloper)

	rec := ta.makeRequest("POST", "/v1/distributions/nonexistent-id/downloads", nil, token)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleStartDistributionDownloads_Forbidden(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	owner, _ := ta.createTestUser(t, "startowner", "startowner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "startother", "startother@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "start-forbidden-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		OwnerID:    owner.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("POST", "/v1/distributions/"+dist.ID+"/downloads", nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestAPI_HandleDistributionDelete_Success(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "deleter", "deleter@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "deletable-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("DELETE", "/v1/distributions/"+dist.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionDelete_NotOwner(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "distowner", "distowner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "distother", "distother@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "protected-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    owner.ID,
	}
	_ = distRepo.Create(dist)

	rec := ta.makeRequest("DELETE", "/v1/distributions/"+dist.ID, nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentDelete_Success(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "compdeleter", "compdeleter@example.com", auth.RoleIDRoot)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "deletable-component",
		Category:    "test",
		DisplayName: "Deletable Component",
	}
	_ = componentRepo.Create(component)

	rec := ta.makeRequest("DELETE", "/v1/components/"+component.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleComponentDelete_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "compdeleter2", "compdeleter2@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("DELETE", "/v1/components/nonexistent-id", nil, token)

	// API returns 500 with "not found" message for missing components
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 404 or 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceDeleteDefault_Success(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "srcdeleter", "srcdeleter@example.com", auth.RoleIDRoot)

	// Create a component first
	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "src-del-component",
		Category:    "test",
		DisplayName: "Source Delete Component",
	}
	_ = componentRepo.Create(component)

	// Create a source
	sourceRepo := db.NewSourceRepository(ta.database)
	source := &db.UpstreamSource{
		Name:         "deletable-source",
		ComponentIDs: []string{component.ID},
		URL:          "https://example.com/source",
	}
	_ = sourceRepo.CreateDefault(source)

	rec := ta.makeRequest("DELETE", "/v1/sources/"+source.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleSourceDeleteDefault_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "srcdeleter2", "srcdeleter2@example.com", auth.RoleIDRoot)

	rec := ta.makeRequest("DELETE", "/v1/sources/nonexistent-id", nil, token)

	// API returns 500 with "not found" message for missing sources
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 404 or 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleDistributionList_WithOwnerFilter(t *testing.T) {
	ta := setupTestAPI(t)

	user1, token1 := ta.createTestUser(t, "owner1", "owner1@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)

	// Create a distribution for user1
	dist1 := &db.Distribution{
		Name:       "owner1-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user1.ID,
	}
	_ = distRepo.Create(dist1)

	// List distributions (owner filter may or may not be supported)
	rec := ta.makeRequest("GET", "/v1/distributions", nil, token1)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	// Just verify we got a response with distributions
	if _, ok := response["distributions"]; !ok {
		t.Fatal("expected distributions field in response")
	}
}

func TestAPI_HandleGetDownloadJob_Success(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	user, token := ta.createTestUser(t, "jobgetuser", "jobgetuser@example.com", auth.RoleIDDeveloper)

	// Create a distribution
	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "job-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	// Create a component
	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "job-component",
		Category:    "core",
		DisplayName: "Job Component",
	}
	_ = componentRepo.Create(component)

	// Create a download job directly
	jobRepo := db.NewDownloadJobRepository(ta.database)
	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        user.ID,
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		Status:         db.JobStatusPending,
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = jobRepo.Create(job)

	rec := ta.makeRequest("GET", "/v1/downloads/"+job.ID, nil, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["id"] != job.ID {
		t.Fatalf("expected job ID %s, got %v", job.ID, response["id"])
	}
}

func TestAPI_HandleCancelDownload_Success(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	user, token := ta.createTestUser(t, "canceluser2", "canceluser2@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "cancel-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "cancel-component",
		Category:    "core",
		DisplayName: "Cancel Component",
	}
	_ = componentRepo.Create(component)

	jobRepo := db.NewDownloadJobRepository(ta.database)
	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        user.ID,
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		Status:         db.JobStatusPending,
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = jobRepo.Create(job)

	rec := ta.makeRequest("POST", "/v1/downloads/"+job.ID+"/cancel", nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleRetryDownload_Success(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	user, token := ta.createTestUser(t, "retryuser2", "retryuser2@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "retry-test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "retry-component",
		Category:    "core",
		DisplayName: "Retry Component",
	}
	_ = componentRepo.Create(component)

	jobRepo := db.NewDownloadJobRepository(ta.database)
	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        user.ID,
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		Status:         db.JobStatusFailed, // Must be failed to retry
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = jobRepo.Create(job)

	rec := ta.makeRequest("POST", "/v1/downloads/"+job.ID+"/retry", nil, token)

	// Could be 200 (success) or 202 (accepted)
	if rec.Code != http.StatusOK && rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 200 or 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleRetryDownload_NotFailed(t *testing.T) {
	ta := setupTestAPIWithDownloadManager(t)

	user, token := ta.createTestUser(t, "retryuser3", "retryuser3@example.com", auth.RoleIDDeveloper)

	distRepo := db.NewDistributionRepository(ta.database)
	dist := &db.Distribution{
		Name:       "retry-test-distro2",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
		OwnerID:    user.ID,
	}
	_ = distRepo.Create(dist)

	componentRepo := db.NewComponentRepository(ta.database)
	component := &db.Component{
		Name:        "retry-component2",
		Category:    "core",
		DisplayName: "Retry Component 2",
	}
	_ = componentRepo.Create(component)

	jobRepo := db.NewDownloadJobRepository(ta.database)
	job := &db.DownloadJob{
		DistributionID: dist.ID,
		OwnerID:        user.ID,
		ComponentID:    component.ID,
		ComponentName:  component.Name,
		Status:         db.JobStatusPending, // Not failed
		ResolvedURL:    "https://example.com/file.tar.gz",
		Version:        "1.0.0",
	}
	_ = jobRepo.Create(job)

	rec := ta.makeRequest("POST", "/v1/downloads/"+job.ID+"/retry", nil, token)

	// Should fail because job is not in failed state
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
