// Package tests provides integration and unit tests for the ldfd server.
package tests

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	_ "github.com/mattn/go-sqlite3"
)

// setupAuthTestDB creates an in-memory database with auth schema for testing
func setupAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	db.SetMaxOpenConns(1)

	schema := `
	CREATE TABLE IF NOT EXISTS roles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		can_read BOOLEAN NOT NULL DEFAULT 1,
		can_write BOOLEAN NOT NULL DEFAULT 0,
		can_delete BOOLEAN NOT NULL DEFAULT 0,
		can_admin BOOLEAN NOT NULL DEFAULT 0,
		is_system BOOLEAN NOT NULL DEFAULT 0,
		parent_role_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	INSERT OR IGNORE INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system)
	VALUES ('908b291e-61fb-4d95-98db-0b76c0afd6b4', 'root', 'Admin role', 1, 1, 1, 1, 1);
	INSERT OR IGNORE INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system)
	VALUES ('91db9f27-b8a2-4452-9b80-5f6ab1096da8', 'developer', 'Developer role', 1, 1, 1, 0, 1);
	INSERT OR IGNORE INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system)
	VALUES ('e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0', 'anonymous', 'Anonymous role', 1, 0, 0, 0, 1);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (role_id) REFERENCES roles(id)
	);

	CREATE TABLE IF NOT EXISTS revoked_tokens (
		token_id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		revoked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role_id);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_used_at DATETIME,
		revoked BOOLEAN NOT NULL DEFAULT 0,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

// =============================================================================
// User Repository Tests
// =============================================================================

func TestUserRepository_CreateUser_RootUniqueness(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Create first root user - should succeed
	rootUser1 := auth.NewUser("admin", "admin@example.com", "hashedpass", auth.RoleIDRoot)
	if err := repo.CreateUser(rootUser1); err != nil {
		t.Fatalf("failed to create first root user: %v", err)
	}

	// Verify root user exists
	hasRoot, err := repo.HasRootUser()
	if err != nil {
		t.Fatalf("failed to check root user: %v", err)
	}
	if !hasRoot {
		t.Fatal("expected root user to exist")
	}

	// Try to create second root user - should fail
	rootUser2 := auth.NewUser("admin2", "admin2@example.com", "hashedpass", auth.RoleIDRoot)
	err = repo.CreateUser(rootUser2)
	if !errors.Is(err, errors.ErrRootUserExists) {
		t.Fatalf("expected ErrRootUserExists, got: %v", err)
	}
}

func TestUserRepository_CreateUser_DeveloperRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Create multiple developer users - should all succeed
	dev1 := auth.NewUser("dev1", "dev1@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(dev1); err != nil {
		t.Fatalf("failed to create dev1: %v", err)
	}

	dev2 := auth.NewUser("dev2", "dev2@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(dev2); err != nil {
		t.Fatalf("failed to create dev2: %v", err)
	}

	count, err := repo.CountUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 users, got %d", count)
	}
}

func TestUserRepository_CreateUser_EmailUniqueness(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	user1 := auth.NewUser("user1", "same@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(user1); err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2 := auth.NewUser("user2", "same@example.com", "hashedpass", auth.RoleIDDeveloper)
	err := repo.CreateUser(user2)
	if !errors.Is(err, errors.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got: %v", err)
	}
}

func TestUserRepository_CreateUser_UsernameUniqueness(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	user1 := auth.NewUser("samename", "user1@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(user1); err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2 := auth.NewUser("samename", "user2@example.com", "hashedpass", auth.RoleIDDeveloper)
	err := repo.CreateUser(user2)
	if !errors.Is(err, errors.ErrUserAlreadyExists) {
		t.Fatalf("expected ErrUserAlreadyExists, got: %v", err)
	}
}

func TestUserRepository_CreateUser_ConcurrentRootCreation(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	const numGoroutines = 10
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user := auth.NewUser(
				"admin"+string(rune('0'+idx)),
				"admin"+string(rune('0'+idx))+"@example.com",
				"hashedpass",
				auth.RoleIDRoot,
			)
			err := repo.CreateUser(user)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful root creation, got %d", successCount)
	}

	count, err := repo.CountUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user, got %d", count)
	}
}

func TestUserRepository_GetUserByName(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Get existing user
	found, err := repo.GetUserByName("testuser")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", found.Email)
	}

	// Get non-existing user
	_, err = repo.GetUserByName("nonexistent")
	if !errors.Is(err, errors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestUserRepository_GetUserByEmail(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	found, err := repo.GetUserByEmail("test@example.com")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if found.Name != "testuser" {
		t.Fatalf("expected name testuser, got %s", found.Name)
	}

	_, err = repo.GetUserByEmail("nonexistent@example.com")
	if !errors.Is(err, errors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}

// =============================================================================
// Role Repository Tests
// =============================================================================

func TestRoleRepository_GetRoleByID(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Get existing system role
	role, err := repo.GetRoleByID(auth.RoleIDRoot)
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role.Name != "root" {
		t.Fatalf("expected role name 'root', got %s", role.Name)
	}
	if !role.IsSystem {
		t.Fatal("expected root role to be a system role")
	}

	// Get non-existing role
	_, err = repo.GetRoleByID("nonexistent-id")
	if !errors.Is(err, errors.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got: %v", err)
	}
}

func TestRoleRepository_GetRoleByName(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	role, err := repo.GetRoleByName("developer")
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role.ID != auth.RoleIDDeveloper {
		t.Fatalf("expected role ID %s, got %s", auth.RoleIDDeveloper, role.ID)
	}

	_, err = repo.GetRoleByName("nonexistent")
	if !errors.Is(err, errors.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got: %v", err)
	}
}

func TestRoleRepository_ListRoles(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	roles, err := repo.ListRoles()
	if err != nil {
		t.Fatalf("failed to list roles: %v", err)
	}

	if len(roles) != 3 {
		t.Fatalf("expected 3 system roles, got %d", len(roles))
	}

	// Verify all system roles are present
	roleNames := make(map[string]bool)
	for _, r := range roles {
		roleNames[r.Name] = true
	}
	for _, name := range []string{"root", "developer", "anonymous"} {
		if !roleNames[name] {
			t.Fatalf("expected role %s to be present", name)
		}
	}
}

func TestRoleRepository_CreateRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Create custom role
	customRole := auth.NewRole("custom", "Custom role", auth.RolePermissions{
		CanRead:  true,
		CanWrite: true,
	}, "")

	if err := repo.CreateRole(customRole); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	// Verify role was created
	found, err := repo.GetRoleByName("custom")
	if err != nil {
		t.Fatalf("failed to get created role: %v", err)
	}
	if !found.Permissions.CanRead || !found.Permissions.CanWrite {
		t.Fatal("role permissions not set correctly")
	}

	// Try to create role with same name
	duplicate := auth.NewRole("custom", "Duplicate", auth.RolePermissions{}, "")
	err = repo.CreateRole(duplicate)
	if !errors.Is(err, errors.ErrRoleAlreadyExists) {
		t.Fatalf("expected ErrRoleAlreadyExists, got: %v", err)
	}
}

func TestRoleRepository_UpdateRole_SystemRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Try to update system role - should fail
	role, err := repo.GetRoleByID(auth.RoleIDRoot)
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}

	role.Description = "Modified description"
	err = repo.UpdateRole(role)
	if !errors.Is(err, errors.ErrSystemRoleModification) {
		t.Fatalf("expected ErrSystemRoleModification, got: %v", err)
	}
}

func TestRoleRepository_DeleteRole_SystemRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Try to delete system role - should fail
	err := repo.DeleteRole(auth.RoleIDRoot)
	if !errors.Is(err, errors.ErrSystemRoleDeletion) {
		t.Fatalf("expected ErrSystemRoleDeletion, got: %v", err)
	}
}

func TestRoleRepository_DeleteRole_CustomRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Create and then delete custom role
	customRole := auth.NewRole("todelete", "To be deleted", auth.RolePermissions{}, "")
	if err := repo.CreateRole(customRole); err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	if err := repo.DeleteRole(customRole.ID); err != nil {
		t.Fatalf("failed to delete role: %v", err)
	}

	// Verify role was deleted
	_, err := repo.GetRoleByID(customRole.ID)
	if !errors.Is(err, errors.ErrRoleNotFound) {
		t.Fatalf("expected ErrRoleNotFound, got: %v", err)
	}
}

// =============================================================================
// Token Repository Tests
// =============================================================================

func TestTokenRepository_RevokeAndCheckToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	repo := auth.NewUserManager(db)

	// Create a user first
	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Check token is not revoked initially
	revoked, err := repo.IsTokenRevoked("test-token-id")
	if err != nil {
		t.Fatalf("failed to check token: %v", err)
	}
	if revoked {
		t.Fatal("expected token to not be revoked initially")
	}

	// Revoke token
	if err := repo.RevokeToken("test-token-id", user.ID, user.CreatedAt.Add(24*60*60*1e9)); err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Check token is now revoked
	revoked, err = repo.IsTokenRevoked("test-token-id")
	if err != nil {
		t.Fatalf("failed to check token: %v", err)
	}
	if !revoked {
		t.Fatal("expected token to be revoked")
	}
}

// =============================================================================
// Role Model Tests
// =============================================================================

func TestRolePermissions(t *testing.T) {
	role := auth.NewRole("test", "Test role", auth.RolePermissions{
		CanRead:   true,
		CanWrite:  true,
		CanDelete: false,
		CanAdmin:  false,
	}, "")

	if !role.HasWriteAccess() {
		t.Fatal("expected HasWriteAccess to return true")
	}
	if role.HasDeleteAccess() {
		t.Fatal("expected HasDeleteAccess to return false")
	}
	if role.HasAdminAccess() {
		t.Fatal("expected HasAdminAccess to return false")
	}
}

func TestIsSystemRoleName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"root", true},
		{"developer", true},
		{"anonymous", true},
		{"custom", false},
		{"admin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if auth.IsSystemRoleName(tt.name) != tt.expected {
				t.Fatalf("IsSystemRoleName(%s) = %v, expected %v", tt.name, !tt.expected, tt.expected)
			}
		})
	}
}

func TestGetDefaultRoleID(t *testing.T) {
	if auth.GetDefaultRoleID() != auth.RoleIDDeveloper {
		t.Fatalf("expected default role to be developer, got %s", auth.GetDefaultRoleID())
	}
}

// =============================================================================
// JWT Service Tests
// =============================================================================

// mockSettingsStore implements auth.SettingsStore for testing
type mockSettingsStore struct {
	settings map[string]string
}

func newMockSettingsStore() *mockSettingsStore {
	return &mockSettingsStore{settings: make(map[string]string)}
}

func (m *mockSettingsStore) GetSetting(key string) (string, error) {
	if val, ok := m.settings[key]; ok {
		return val, nil
	}
	return "", nil // Return empty string for missing settings
}

func (m *mockSettingsStore) SetSetting(key, value string) error {
	m.settings[key] = value
	return nil
}

func setupJWTService(t *testing.T) (*auth.JWTService, *auth.UserManager, *auth.User) {
	t.Helper()
	db := setupAuthTestDB(t)
	t.Cleanup(func() { db.Close() })

	userManager := auth.NewUserManager(db)
	settings := newMockSettingsStore()
	jwtService := auth.NewJWTService(auth.DefaultJWTConfig(), userManager, settings)

	// Create a test user
	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return jwtService, userManager, user
}

func TestJWTService_GenerateToken(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Token should be a valid JWT (3 parts separated by dots)
	parts := 0
	for _, c := range token {
		if c == '.' {
			parts++
		}
	}
	if parts != 2 {
		t.Fatalf("expected JWT with 3 parts (2 dots), got %d dots", parts)
	}
}

func TestJWTService_GenerateToken_WithClaims(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate and check claims
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, claims.UserID)
	}
	if claims.UserName != user.Name {
		t.Fatalf("expected username %s, got %s", user.Name, claims.UserName)
	}
	if claims.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, claims.Email)
	}
	if claims.RoleID != user.RoleID {
		t.Fatalf("expected role ID %s, got %s", user.RoleID, claims.RoleID)
	}
	if claims.RoleName != "developer" {
		t.Fatalf("expected role name 'developer', got %s", claims.RoleName)
	}
}

func TestJWTService_GenerateTokenPair(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	pair, err := jwtService.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	if pair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if pair.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", pair.ExpiresIn)
	}
	if pair.ExpiresAt.IsZero() {
		t.Fatal("expected non-zero expires_at")
	}
}

func TestJWTService_ValidateToken(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, claims.UserID)
	}
	if !claims.Permissions.CanRead {
		t.Fatal("expected CanRead permission to be true")
	}
}

func TestJWTService_ValidateToken_InvalidToken(t *testing.T) {
	jwtService, _, _ := setupJWTService(t)

	_, err := jwtService.ValidateToken("invalid.token.here")
	if !errors.Is(err, errors.ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid, got: %v", err)
	}
}

func TestJWTService_ValidateToken_TamperedToken(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Tamper with the token by changing a character in the signature
	tamperedToken := token[:len(token)-5] + "XXXXX"

	_, err = jwtService.ValidateToken(tamperedToken)
	if !errors.Is(err, errors.ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid for tampered token, got: %v", err)
	}
}

func TestJWTService_ValidateToken_Revoked(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Revoke the token
	if err := jwtService.RevokeToken(token); err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Token should now be invalid
	_, err = jwtService.ValidateToken(token)
	if !errors.Is(err, errors.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked, got: %v", err)
	}
}

func TestJWTService_RevokeToken(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Token should be valid before revocation
	_, err = jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("token should be valid before revocation: %v", err)
	}

	// Revoke the token
	if err := jwtService.RevokeToken(token); err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// Token should be invalid after revocation
	_, err = jwtService.ValidateToken(token)
	if !errors.Is(err, errors.ErrTokenRevoked) {
		t.Fatalf("expected ErrTokenRevoked after revocation, got: %v", err)
	}
}

func TestJWTService_RevokeToken_InvalidToken(t *testing.T) {
	jwtService, _, _ := setupJWTService(t)

	err := jwtService.RevokeToken("invalid.token.here")
	if !errors.Is(err, errors.ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid, got: %v", err)
	}
}

func TestJWTService_GetTokenExpiry(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	token, err := jwtService.GenerateToken(user)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	expiry, err := jwtService.GetTokenExpiry(token)
	if err != nil {
		t.Fatalf("failed to get token expiry: %v", err)
	}

	if expiry.IsZero() {
		t.Fatal("expected non-zero expiry time")
	}

	// Expiry should be in the future
	if expiry.Before(time.Now()) {
		t.Fatal("expected expiry to be in the future")
	}

	// Expiry should be within the token duration (15 minutes default)
	maxExpiry := time.Now().Add(16 * time.Minute)
	if expiry.After(maxExpiry) {
		t.Fatalf("expiry too far in future: %v", expiry)
	}
}

func TestJWTService_GetTokenExpiry_InvalidToken(t *testing.T) {
	jwtService, _, _ := setupJWTService(t)

	_, err := jwtService.GetTokenExpiry("invalid.token.here")
	if !errors.Is(err, errors.ErrTokenInvalid) {
		t.Fatalf("expected ErrTokenInvalid, got: %v", err)
	}
}

func TestJWTService_RefreshAccessToken(t *testing.T) {
	jwtService, _, user := setupJWTService(t)

	// Generate initial token pair
	pair, err := jwtService.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("failed to generate token pair: %v", err)
	}

	// Use refresh token to get new access token
	newPair, returnedUser, err := jwtService.RefreshAccessToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("failed to refresh access token: %v", err)
	}

	if newPair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if returnedUser.ID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, returnedUser.ID)
	}

	// New access token should be valid
	claims, err := jwtService.ValidateToken(newPair.AccessToken)
	if err != nil {
		t.Fatalf("new access token should be valid: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected user ID %s in claims, got %s", user.ID, claims.UserID)
	}
}

func TestJWTService_RefreshAccessToken_InvalidToken(t *testing.T) {
	jwtService, _, _ := setupJWTService(t)

	_, _, err := jwtService.RefreshAccessToken("invalid-refresh-token")
	if !errors.Is(err, errors.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got: %v", err)
	}
}

func TestJWTService_SecretPersistence(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)
	settings := newMockSettingsStore()

	// Create first JWT service - should generate and store secret
	_ = auth.NewJWTService(auth.DefaultJWTConfig(), userManager, settings)

	// Verify secret was stored
	secret1, err := settings.GetSetting("jwt_secret")
	if err != nil {
		t.Fatalf("expected jwt_secret to be stored: %v", err)
	}
	if secret1 == "" {
		t.Fatal("expected non-empty jwt_secret")
	}

	// Create second JWT service - should reuse existing secret
	_ = auth.NewJWTService(auth.DefaultJWTConfig(), userManager, settings)

	secret2, _ := settings.GetSetting("jwt_secret")
	if secret1 != secret2 {
		t.Fatal("expected JWT services to use same secret key")
	}
}

func TestJWTService_DefaultConfig(t *testing.T) {
	cfg := auth.DefaultJWTConfig()

	if cfg.Issuer != "ldfd" {
		t.Fatalf("expected issuer 'ldfd', got %s", cfg.Issuer)
	}
	if cfg.TokenDuration != 15*time.Minute {
		t.Fatalf("expected token duration 15m, got %v", cfg.TokenDuration)
	}
	if cfg.RefreshTokenDuration != 7*24*time.Hour {
		t.Fatalf("expected refresh token duration 7d, got %v", cfg.RefreshTokenDuration)
	}
}

// =============================================================================
// Refresh Token Tests
// =============================================================================

func TestUserManager_CreateRefreshToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	// Create a user first
	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create refresh token
	plainToken, record, err := userManager.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	if plainToken == "" {
		t.Fatal("expected non-empty plain token")
	}
	if record == nil {
		t.Fatal("expected non-nil refresh token record")
	}
	if record.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, record.UserID)
	}
	if record.Revoked {
		t.Fatal("new token should not be revoked")
	}
	if record.ExpiresAt.Before(time.Now()) {
		t.Fatal("token should expire in the future")
	}
}

func TestUserManager_ValidateRefreshToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	plainToken, _, err := userManager.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// Validate the token
	record, err := userManager.ValidateRefreshToken(plainToken)
	if err != nil {
		t.Fatalf("failed to validate refresh token: %v", err)
	}

	if record.UserID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, record.UserID)
	}
}

func TestUserManager_ValidateRefreshToken_Invalid(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	_, err := userManager.ValidateRefreshToken("invalid-token")
	if !errors.Is(err, errors.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got: %v", err)
	}
}

func TestUserManager_ValidateRefreshToken_Revoked(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	plainToken, record, err := userManager.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// Revoke the token
	if err := userManager.RevokeRefreshToken(record.ID); err != nil {
		t.Fatalf("failed to revoke refresh token: %v", err)
	}

	// Validation should fail
	_, err = userManager.ValidateRefreshToken(plainToken)
	if !errors.Is(err, errors.ErrRefreshTokenRevoked) {
		t.Fatalf("expected ErrRefreshTokenRevoked, got: %v", err)
	}
}

func TestUserManager_RevokeRefreshToken(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_, record, err := userManager.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// Revoke by ID
	if err := userManager.RevokeRefreshToken(record.ID); err != nil {
		t.Fatalf("failed to revoke refresh token: %v", err)
	}
}

func TestUserManager_RevokeRefreshTokenByHash(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	plainToken, _, err := userManager.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// Revoke by plain token
	if err := userManager.RevokeRefreshTokenByHash(plainToken); err != nil {
		t.Fatalf("failed to revoke refresh token by hash: %v", err)
	}

	// Validation should fail
	_, err = userManager.ValidateRefreshToken(plainToken)
	if !errors.Is(err, errors.ErrRefreshTokenRevoked) {
		t.Fatalf("expected ErrRefreshTokenRevoked, got: %v", err)
	}
}

func TestUserManager_RevokeRefreshTokenByHash_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	err := userManager.RevokeRefreshTokenByHash("nonexistent-token")
	if !errors.Is(err, errors.ErrRefreshTokenInvalid) {
		t.Fatalf("expected ErrRefreshTokenInvalid, got: %v", err)
	}
}

func TestUserManager_RevokeAllUserRefreshTokens(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create multiple tokens
	token1, _, _ := userManager.CreateRefreshToken(user.ID)
	token2, _, _ := userManager.CreateRefreshToken(user.ID)
	token3, _, _ := userManager.CreateRefreshToken(user.ID)

	// Revoke all tokens for user
	if err := userManager.RevokeAllUserRefreshTokens(user.ID); err != nil {
		t.Fatalf("failed to revoke all user refresh tokens: %v", err)
	}

	// All tokens should be invalid
	for _, token := range []string{token1, token2, token3} {
		_, err := userManager.ValidateRefreshToken(token)
		if !errors.Is(err, errors.ErrRefreshTokenRevoked) {
			t.Fatalf("expected ErrRefreshTokenRevoked for token, got: %v", err)
		}
	}
}

func TestUserManager_GetUserRefreshTokenCount(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Initially should be 0
	count, err := userManager.GetUserRefreshTokenCount(user.ID)
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 tokens, got %d", count)
	}

	// Create 3 tokens
	userManager.CreateRefreshToken(user.ID)
	userManager.CreateRefreshToken(user.ID)
	_, record, _ := userManager.CreateRefreshToken(user.ID)

	count, err = userManager.GetUserRefreshTokenCount(user.ID)
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 tokens, got %d", count)
	}

	// Revoke one token
	userManager.RevokeRefreshToken(record.ID)

	count, err = userManager.GetUserRefreshTokenCount(user.ID)
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 tokens after revocation, got %d", count)
	}
}

func TestUserManager_CleanupExpiredTokens(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create and revoke a token (revoked tokens should be cleaned up)
	_, record, _ := userManager.CreateRefreshToken(user.ID)
	userManager.RevokeRefreshToken(record.ID)

	// Cleanup should not error
	if err := userManager.CleanupExpiredTokens(); err != nil {
		t.Fatalf("failed to cleanup: %v", err)
	}
}

func TestUserManager_UpdateRefreshTokenLastUsed(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	plainToken, record, err := userManager.CreateRefreshToken(user.ID)
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	// Update last used
	if err := userManager.UpdateRefreshTokenLastUsed(record.ID); err != nil {
		t.Fatalf("failed to update last used: %v", err)
	}

	// Validate and check last_used_at is set
	validated, err := userManager.ValidateRefreshToken(plainToken)
	if err != nil {
		t.Fatalf("failed to validate: %v", err)
	}
	if validated.LastUsedAt.IsZero() {
		t.Fatal("expected last_used_at to be set")
	}
}

// =============================================================================
// User Account Additional Tests
// =============================================================================

func TestUserManager_GetUserByID(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Get by ID
	found, err := userManager.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("failed to get user by ID: %v", err)
	}
	if found.Name != "testuser" {
		t.Fatalf("expected name 'testuser', got %s", found.Name)
	}
	if found.RoleName != "developer" {
		t.Fatalf("expected role name 'developer', got %s", found.RoleName)
	}
}

func TestUserManager_GetUserByID_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	_, err := userManager.GetUserByID("nonexistent-id")
	if !errors.Is(err, errors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestUserManager_GetUserWithRole(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	user := auth.NewUser("testuser", "test@example.com", "hashedpass", auth.RoleIDDeveloper)
	if err := userManager.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Get user with role
	foundUser, foundRole, err := userManager.GetUserWithRole(user.ID)
	if err != nil {
		t.Fatalf("failed to get user with role: %v", err)
	}

	if foundUser.ID != user.ID {
		t.Fatalf("expected user ID %s, got %s", user.ID, foundUser.ID)
	}
	if foundRole.ID != auth.RoleIDDeveloper {
		t.Fatalf("expected role ID %s, got %s", auth.RoleIDDeveloper, foundRole.ID)
	}
	if foundRole.Name != "developer" {
		t.Fatalf("expected role name 'developer', got %s", foundRole.Name)
	}
	if !foundRole.Permissions.CanRead {
		t.Fatal("expected CanRead permission")
	}
	if !foundRole.Permissions.CanWrite {
		t.Fatal("expected CanWrite permission")
	}
}

func TestUserManager_GetUserWithRole_NotFound(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	_, _, err := userManager.GetUserWithRole("nonexistent-id")
	if !errors.Is(err, errors.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got: %v", err)
	}
}

func TestUserManager_CountUsers(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	// Initially 0
	count, err := userManager.CountUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 users, got %d", count)
	}

	// Create users
	for i := 0; i < 5; i++ {
		user := auth.NewUser(
			"user"+string(rune('0'+i)),
			"user"+string(rune('0'+i))+"@example.com",
			"hashedpass",
			auth.RoleIDDeveloper,
		)
		userManager.CreateUser(user)
	}

	count, err = userManager.CountUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 users, got %d", count)
	}
}

func TestUserManager_HasRootUser(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()

	userManager := auth.NewUserManager(db)

	// Initially no root user
	hasRoot, err := userManager.HasRootUser()
	if err != nil {
		t.Fatalf("failed to check root user: %v", err)
	}
	if hasRoot {
		t.Fatal("expected no root user initially")
	}

	// Create root user
	root := auth.NewUser("admin", "admin@example.com", "hashedpass", auth.RoleIDRoot)
	if err := userManager.CreateUser(root); err != nil {
		t.Fatalf("failed to create root user: %v", err)
	}

	hasRoot, err = userManager.HasRootUser()
	if err != nil {
		t.Fatalf("failed to check root user: %v", err)
	}
	if !hasRoot {
		t.Fatal("expected root user to exist")
	}
}
