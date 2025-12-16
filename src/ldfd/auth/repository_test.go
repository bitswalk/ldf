package auth

import (
	"database/sql"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use shared cache mode for in-memory database to allow concurrent access
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Set connection pool settings for concurrent access
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time

	// Create the roles and users tables
	// Role IDs must match constants in role.go
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

	CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role_id);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestCreateUser_RootUniqueness(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	// Create first root user - should succeed
	rootUser1 := NewUser("admin", "admin@example.com", "hashedpass", RoleIDRoot)
	err := repo.CreateUser(rootUser1)
	if err != nil {
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

	// Try to create second root user - should fail with ErrRootExists
	rootUser2 := NewUser("admin2", "admin2@example.com", "hashedpass", RoleIDRoot)
	err = repo.CreateUser(rootUser2)
	if err != ErrRootExists {
		t.Fatalf("expected ErrRootExists, got: %v", err)
	}
}

func TestCreateUser_DeveloperRole(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	// Create multiple developer users - should all succeed
	dev1 := NewUser("dev1", "dev1@example.com", "hashedpass", RoleIDDeveloper)
	if err := repo.CreateUser(dev1); err != nil {
		t.Fatalf("failed to create dev1: %v", err)
	}

	dev2 := NewUser("dev2", "dev2@example.com", "hashedpass", RoleIDDeveloper)
	if err := repo.CreateUser(dev2); err != nil {
		t.Fatalf("failed to create dev2: %v", err)
	}

	// Count users
	count, err := repo.CountUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 users, got %d", count)
	}
}

func TestCreateUser_EmailUniqueness(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	user1 := NewUser("user1", "same@example.com", "hashedpass", RoleIDDeveloper)
	if err := repo.CreateUser(user1); err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	// Try to create user with same email - should fail
	user2 := NewUser("user2", "same@example.com", "hashedpass", RoleIDDeveloper)
	err := repo.CreateUser(user2)
	if err != ErrEmailExists {
		t.Fatalf("expected ErrEmailExists, got: %v", err)
	}
}

func TestCreateUser_UsernameUniqueness(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	user1 := NewUser("samename", "user1@example.com", "hashedpass", RoleIDDeveloper)
	if err := repo.CreateUser(user1); err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	// Try to create user with same username - should fail
	user2 := NewUser("samename", "user2@example.com", "hashedpass", RoleIDDeveloper)
	err := repo.CreateUser(user2)
	if err != ErrUserExists {
		t.Fatalf("expected ErrUserExists, got: %v", err)
	}
}

func TestCreateUser_ConcurrentRootCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)

	// Try to create multiple root users concurrently
	const numGoroutines = 10
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user := NewUser(
				"admin"+string(rune('0'+idx)),
				"admin"+string(rune('0'+idx))+"@example.com",
				"hashedpass",
				RoleIDRoot,
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

	// Only one root user should have been created
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful root creation, got %d", successCount)
	}

	// Verify only one root user exists
	hasRoot, err := repo.HasRootUser()
	if err != nil {
		t.Fatalf("failed to check root user: %v", err)
	}
	if !hasRoot {
		t.Fatal("expected root user to exist")
	}

	count, err := repo.CountUsers()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user, got %d", count)
	}
}
