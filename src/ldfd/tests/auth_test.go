// Package tests provides integration and unit tests for the ldfd server.
package tests

import (
	"database/sql"
	"sync"
	"testing"

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
