package tests

import (
	"database/sql"
	"testing"

	"github.com/bitswalk/ldf/src/ldfd/db/migrations"
	_ "github.com/mattn/go-sqlite3"
)

// =============================================================================
// Migration Runner Tests
// =============================================================================

func setupMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	return db
}

func TestMigrationRunner_Run(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)

	// Run all migrations
	if err := runner.Run(); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Verify schema_migrations table exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("schema_migrations table should exist: %v", err)
	}
	if count < 2 {
		t.Fatalf("expected at least 2 migrations applied, got %d", count)
	}
}

func TestMigrationRunner_Run_Idempotent(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)

	// Run migrations twice
	if err := runner.Run(); err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	if err := runner.Run(); err != nil {
		t.Fatalf("second run should succeed: %v", err)
	}

	// Should still have same number of migrations
	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if count < 2 {
		t.Fatalf("expected at least 2 migrations, got %d", count)
	}
}

func TestMigrationRunner_CurrentVersion(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)

	// Before running, version should be 0
	version, err := runner.CurrentVersion()
	if err != nil {
		t.Fatalf("failed to get current version: %v", err)
	}
	if version != 0 {
		t.Fatalf("expected version 0 before migrations, got %d", version)
	}

	// Run migrations
	runner.Run()

	// After running, version should be > 0
	version, err = runner.CurrentVersion()
	if err != nil {
		t.Fatalf("failed to get current version: %v", err)
	}
	if version < 2 {
		t.Fatalf("expected version >= 2 after migrations, got %d", version)
	}
}

func TestMigrationRunner_PendingCount(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)

	// Before running, should have pending migrations
	pending, err := runner.PendingCount()
	if err != nil {
		t.Fatalf("failed to get pending count: %v", err)
	}
	if pending < 2 {
		t.Fatalf("expected at least 2 pending migrations, got %d", pending)
	}

	// Run migrations
	runner.Run()

	// After running, should have 0 pending
	pending, err = runner.PendingCount()
	if err != nil {
		t.Fatalf("failed to get pending count: %v", err)
	}
	if pending != 0 {
		t.Fatalf("expected 0 pending migrations after run, got %d", pending)
	}
}

// =============================================================================
// Schema Tests
// =============================================================================

func TestMigration001_CreatesAllTables(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)
	runner.Run()

	tables := []string{
		"roles",
		"users",
		"distributions",
		"distribution_logs",
		"revoked_tokens",
		"settings",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("table %s should exist: %v", table, err)
		}
	}
}

func TestMigration001_CreatesIndexes(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)
	runner.Run()

	indexes := []string{
		"idx_distributions_status",
		"idx_distributions_name",
		"idx_distributions_owner",
		"idx_distributions_visibility",
		"idx_distribution_logs_dist_id",
		"idx_users_name",
		"idx_users_email",
		"idx_users_role",
		"idx_roles_name",
		"idx_revoked_tokens_user_id",
		"idx_revoked_tokens_expires_at",
	}

	for _, index := range indexes {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
			index,
		).Scan(&name)
		if err != nil {
			t.Fatalf("index %s should exist: %v", index, err)
		}
	}
}

func TestMigration002_SeedsDefaultRoles(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)
	runner.Run()

	// Verify all default roles exist
	expectedRoles := migrations.DefaultRoles()

	for _, expected := range expectedRoles {
		var id, name string
		var isSystem bool
		err := db.QueryRow(
			"SELECT id, name, is_system FROM roles WHERE id = ?",
			expected.ID,
		).Scan(&id, &name, &isSystem)

		if err != nil {
			t.Fatalf("role %s should exist: %v", expected.Name, err)
		}
		if name != expected.Name {
			t.Fatalf("expected role name %s, got %s", expected.Name, name)
		}
		if !isSystem {
			t.Fatalf("role %s should be a system role", expected.Name)
		}
	}
}

func TestMigration002_RolePermissions(t *testing.T) {
	db := setupMigrationTestDB(t)
	defer db.Close()

	runner := migrations.NewRunner(db)
	runner.Run()

	// Check root role has all permissions
	var canRead, canWrite, canDelete, canAdmin bool
	err := db.QueryRow(
		"SELECT can_read, can_write, can_delete, can_admin FROM roles WHERE id = ?",
		migrations.RoleIDRoot,
	).Scan(&canRead, &canWrite, &canDelete, &canAdmin)

	if err != nil {
		t.Fatalf("failed to get root role: %v", err)
	}
	if !canRead || !canWrite || !canDelete || !canAdmin {
		t.Fatal("root role should have all permissions")
	}

	// Check anonymous role has only read
	err = db.QueryRow(
		"SELECT can_read, can_write, can_delete, can_admin FROM roles WHERE id = ?",
		migrations.RoleIDAnonymous,
	).Scan(&canRead, &canWrite, &canDelete, &canAdmin)

	if err != nil {
		t.Fatalf("failed to get anonymous role: %v", err)
	}
	if !canRead {
		t.Fatal("anonymous role should have read permission")
	}
	if canWrite || canDelete || canAdmin {
		t.Fatal("anonymous role should only have read permission")
	}
}

// =============================================================================
// Migration Constants Tests
// =============================================================================

func TestMigrationRoleIDs_MatchAuthPackage(t *testing.T) {
	// These IDs must match the constants in auth/role.go
	expectedRootID := "908b291e-61fb-4d95-98db-0b76c0afd6b4"
	expectedDeveloperID := "91db9f27-b8a2-4452-9b80-5f6ab1096da8"
	expectedAnonymousID := "e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0"

	if migrations.RoleIDRoot != expectedRootID {
		t.Fatalf("RoleIDRoot mismatch: expected %s, got %s", expectedRootID, migrations.RoleIDRoot)
	}
	if migrations.RoleIDDeveloper != expectedDeveloperID {
		t.Fatalf("RoleIDDeveloper mismatch: expected %s, got %s", expectedDeveloperID, migrations.RoleIDDeveloper)
	}
	if migrations.RoleIDAnonymous != expectedAnonymousID {
		t.Fatalf("RoleIDAnonymous mismatch: expected %s, got %s", expectedAnonymousID, migrations.RoleIDAnonymous)
	}
}

func TestDefaultRoles_Count(t *testing.T) {
	roles := migrations.DefaultRoles()
	if len(roles) != 3 {
		t.Fatalf("expected 3 default roles, got %d", len(roles))
	}
}

func TestDefaultRoles_UniqueIDs(t *testing.T) {
	roles := migrations.DefaultRoles()
	ids := make(map[string]bool)

	for _, role := range roles {
		if ids[role.ID] {
			t.Fatalf("duplicate role ID: %s", role.ID)
		}
		ids[role.ID] = true
	}
}

func TestDefaultRoles_UniqueNames(t *testing.T) {
	roles := migrations.DefaultRoles()
	names := make(map[string]bool)

	for _, role := range roles {
		if names[role.Name] {
			t.Fatalf("duplicate role name: %s", role.Name)
		}
		names[role.Name] = true
	}
}
