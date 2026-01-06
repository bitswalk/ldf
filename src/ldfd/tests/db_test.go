package tests

import (
	"testing"
	"time"

	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// =============================================================================
// Database Tests
// =============================================================================

func TestDatabase_New(t *testing.T) {
	cfg := db.Config{
		PersistPath: "", // No persistence for testing
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	if database.DB() == nil {
		t.Fatal("expected DB() to return non-nil connection")
	}
}

func TestDatabase_Settings(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	// Set a setting
	if err := database.SetSetting("test_key", "test_value"); err != nil {
		t.Fatalf("failed to set setting: %v", err)
	}

	// Get the setting
	value, err := database.GetSetting("test_key")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}
	if value != "test_value" {
		t.Fatalf("expected 'test_value', got '%s'", value)
	}

	// Update the setting
	if err := database.SetSetting("test_key", "updated_value"); err != nil {
		t.Fatalf("failed to update setting: %v", err)
	}

	value, err = database.GetSetting("test_key")
	if err != nil {
		t.Fatalf("failed to get updated setting: %v", err)
	}
	if value != "updated_value" {
		t.Fatalf("expected 'updated_value', got '%s'", value)
	}
}

func TestDatabase_GetAllSettings(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	// Set multiple settings
	database.SetSetting("key1", "value1")
	database.SetSetting("key2", "value2")
	database.SetSetting("key3", "value3")

	settings, err := database.GetAllSettings()
	if err != nil {
		t.Fatalf("failed to get all settings: %v", err)
	}

	if len(settings) != 3 {
		t.Fatalf("expected 3 settings, got %d", len(settings))
	}

	if settings["key1"] != "value1" {
		t.Fatalf("expected key1=value1, got %s", settings["key1"])
	}
}

func TestDatabase_GetSetting_NotFound(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	_, err = database.GetSetting("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent setting")
	}
}

// =============================================================================
// Distribution Repository Tests
// =============================================================================

func TestDistributionRepository_Create(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}

	if err := repo.Create(dist); err != nil {
		t.Fatalf("failed to create distribution: %v", err)
	}

	if dist.ID == "" {
		t.Fatal("expected distribution ID to be set after creation")
	}
}

func TestDistributionRepository_GetByID(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "test-distro",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPublic,
	}
	repo.Create(dist)

	found, err := repo.GetByID(dist.ID)
	if err != nil {
		t.Fatalf("failed to get distribution: %v", err)
	}
	if found.Name != "test-distro" {
		t.Fatalf("expected name 'test-distro', got '%s'", found.Name)
	}
	if found.Visibility != db.VisibilityPublic {
		t.Fatalf("expected visibility 'public', got '%s'", found.Visibility)
	}

	// Get non-existent
	notFound, err := repo.GetByID("nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent distribution")
	}
}

func TestDistributionRepository_GetByName(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "unique-name",
		Version:    "2.0.0",
		Status:     db.StatusReady,
		Visibility: db.VisibilityPrivate,
	}
	repo.Create(dist)

	found, err := repo.GetByName("unique-name")
	if err != nil {
		t.Fatalf("failed to get distribution: %v", err)
	}
	if found.Version != "2.0.0" {
		t.Fatalf("expected version '2.0.0', got '%s'", found.Version)
	}
}

func TestDistributionRepository_List(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	// Create multiple distributions
	for i := 0; i < 3; i++ {
		dist := &db.Distribution{
			Name:       "distro-" + string(rune('a'+i)),
			Version:    "1.0.0",
			Status:     db.StatusPending,
			Visibility: db.VisibilityPrivate,
		}
		repo.Create(dist)
	}

	// List all
	list, err := repo.List(nil)
	if err != nil {
		t.Fatalf("failed to list distributions: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 distributions, got %d", len(list))
	}

	// List with status filter
	status := db.StatusPending
	list, err = repo.List(&status)
	if err != nil {
		t.Fatalf("failed to list distributions with filter: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 pending distributions, got %d", len(list))
	}
}

func TestDistributionRepository_ListAccessible(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	// Create a real user to be the owner
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("testowner", "testowner@example.com", "hashedpassword", auth.RoleIDDeveloper)
	if err := authRepo.CreateUser(user); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	repo := db.NewDistributionRepository(database)

	// Create public distribution
	public := &db.Distribution{
		Name:       "public-distro",
		Version:    "1.0.0",
		Status:     db.StatusReady,
		Visibility: db.VisibilityPublic,
	}
	if err := repo.Create(public); err != nil {
		t.Fatalf("failed to create public distribution: %v", err)
	}

	// Create private distribution with owner
	private := &db.Distribution{
		Name:       "private-distro",
		Version:    "1.0.0",
		Status:     db.StatusReady,
		Visibility: db.VisibilityPrivate,
		OwnerID:    user.ID,
	}
	if err := repo.Create(private); err != nil {
		t.Fatalf("failed to create private distribution: %v", err)
	}

	// Verify owner_id was stored correctly
	found, err := repo.GetByID(private.ID)
	if err != nil {
		t.Fatalf("failed to get private distribution: %v", err)
	}
	if found == nil {
		t.Fatal("private distribution not found after creation")
	}
	if found.OwnerID != user.ID {
		t.Fatalf("owner_id not stored correctly: expected '%s', got '%s'", user.ID, found.OwnerID)
	}

	// Anonymous user should only see public
	list, err := repo.ListAccessible("", false, nil)
	if err != nil {
		t.Fatalf("failed to list accessible: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 public distribution for anonymous, got %d", len(list))
	}

	// Owner should see public + own private
	list, err = repo.ListAccessible(user.ID, false, nil)
	if err != nil {
		t.Fatalf("failed to list accessible: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 distributions for owner, got %d (public=%s, private owner=%s)", len(list), public.Visibility, found.OwnerID)
	}

	// Admin should see all
	list, err = repo.ListAccessible("other-user", true, nil)
	if err != nil {
		t.Fatalf("failed to list accessible: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 distributions for admin, got %d", len(list))
	}
}

func TestDistributionRepository_CanUserAccess(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	// Create a real user to be the owner
	authRepo := auth.NewUserManager(database.DB())
	owner := auth.NewUser("owner", "owner@example.com", "hashedpassword", auth.RoleIDDeveloper)
	if err := authRepo.CreateUser(owner); err != nil {
		t.Fatalf("failed to create owner user: %v", err)
	}

	repo := db.NewDistributionRepository(database)

	public := &db.Distribution{
		Name:       "public",
		Version:    "1.0.0",
		Status:     db.StatusReady,
		Visibility: db.VisibilityPublic,
	}
	if err := repo.Create(public); err != nil {
		t.Fatalf("failed to create public distribution: %v", err)
	}

	private := &db.Distribution{
		Name:       "private",
		Version:    "1.0.0",
		Status:     db.StatusReady,
		Visibility: db.VisibilityPrivate,
		OwnerID:    owner.ID,
	}
	if err := repo.Create(private); err != nil {
		t.Fatalf("failed to create private distribution: %v", err)
	}

	// Verify owner_id was stored
	found, err := repo.GetByID(private.ID)
	if err != nil {
		t.Fatalf("failed to get private distribution: %v", err)
	}
	if found == nil {
		t.Fatal("private distribution not found")
	}
	if found.OwnerID != owner.ID {
		t.Fatalf("owner_id not stored: expected '%s', got '%s'", owner.ID, found.OwnerID)
	}

	// Anyone can access public
	canAccess, err := repo.CanUserAccess(public.ID, "", false)
	if err != nil {
		t.Fatalf("error checking public access: %v", err)
	}
	if !canAccess {
		t.Fatal("anonymous should access public distribution")
	}

	// Anonymous cannot access private
	canAccess, err = repo.CanUserAccess(private.ID, "", false)
	if err != nil {
		t.Fatalf("error checking private access for anonymous: %v", err)
	}
	if canAccess {
		t.Fatal("anonymous should not access private distribution")
	}

	// Owner can access private
	canAccess, err = repo.CanUserAccess(private.ID, owner.ID, false)
	if err != nil {
		t.Fatalf("error checking access: %v", err)
	}
	if !canAccess {
		t.Fatalf("owner should access private distribution (ownerID=%s)", found.OwnerID)
	}

	// Admin can access anything
	canAccess, err = repo.CanUserAccess(private.ID, "other-user", true)
	if err != nil {
		t.Fatalf("error checking admin access: %v", err)
	}
	if !canAccess {
		t.Fatal("admin should access any distribution")
	}
}

func TestDistributionRepository_UpdateStatus(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "status-test",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	repo.Create(dist)

	// Update to downloading
	if err := repo.UpdateStatus(dist.ID, db.StatusDownloading, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	found, _ := repo.GetByID(dist.ID)
	if found.Status != db.StatusDownloading {
		t.Fatalf("expected status downloading, got %s", found.Status)
	}
	if found.StartedAt == nil {
		t.Fatal("expected started_at to be set")
	}

	// Update to ready
	if err := repo.UpdateStatus(dist.ID, db.StatusReady, ""); err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	found, _ = repo.GetByID(dist.ID)
	if found.Status != db.StatusReady {
		t.Fatalf("expected status ready, got %s", found.Status)
	}
	if found.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestDistributionRepository_Update(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "update-test",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	repo.Create(dist)

	dist.Version = "2.0.0"
	dist.Visibility = db.VisibilityPublic

	if err := repo.Update(dist); err != nil {
		t.Fatalf("failed to update distribution: %v", err)
	}

	found, _ := repo.GetByID(dist.ID)
	if found.Version != "2.0.0" {
		t.Fatalf("expected version 2.0.0, got %s", found.Version)
	}
	if found.Visibility != db.VisibilityPublic {
		t.Fatalf("expected visibility public, got %s", found.Visibility)
	}
}

func TestDistributionRepository_Delete(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "delete-test",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	repo.Create(dist)

	if err := repo.Delete(dist.ID); err != nil {
		t.Fatalf("failed to delete distribution: %v", err)
	}

	found, _ := repo.GetByID(dist.ID)
	if found != nil {
		t.Fatal("expected distribution to be deleted")
	}

	// Delete non-existent
	err = repo.Delete("nonexistent-id")
	if err == nil {
		t.Fatal("expected error when deleting non-existent distribution")
	}
}

func TestDistributionRepository_Logs(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	dist := &db.Distribution{
		Name:       "log-test",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	repo.Create(dist)

	// Add logs
	repo.AddLog(dist.ID, "info", "Starting download")
	repo.AddLog(dist.ID, "info", "Download complete")
	repo.AddLog(dist.ID, "error", "Validation failed")

	logs, err := repo.GetLogs(dist.ID, 10)
	if err != nil {
		t.Fatalf("failed to get logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}

	// Verify all logs are present (order may vary due to same-millisecond timestamps)
	messages := make(map[string]bool)
	for _, log := range logs {
		messages[log.Message] = true
	}
	expectedMessages := []string{"Starting download", "Download complete", "Validation failed"}
	for _, msg := range expectedMessages {
		if !messages[msg] {
			t.Fatalf("expected log message '%s' to be present", msg)
		}
	}

	// Verify log levels
	levels := make(map[string]int)
	for _, log := range logs {
		levels[log.Level]++
	}
	if levels["info"] != 2 {
		t.Fatalf("expected 2 info logs, got %d", levels["info"])
	}
	if levels["error"] != 1 {
		t.Fatalf("expected 1 error log, got %d", levels["error"])
	}
}

func TestDistributionRepository_GetStats(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	// Create distributions with different statuses
	statuses := []db.DistributionStatus{
		db.StatusPending,
		db.StatusPending,
		db.StatusReady,
		db.StatusFailed,
	}

	for i, status := range statuses {
		dist := &db.Distribution{
			Name:       "stats-" + string(rune('a'+i)),
			Version:    "1.0.0",
			Status:     status,
			Visibility: db.VisibilityPrivate,
		}
		repo.Create(dist)
	}

	stats, err := repo.GetStats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats["pending"] != 2 {
		t.Fatalf("expected 2 pending, got %d", stats["pending"])
	}
	if stats["ready"] != 1 {
		t.Fatalf("expected 1 ready, got %d", stats["ready"])
	}
	if stats["failed"] != 1 {
		t.Fatalf("expected 1 failed, got %d", stats["failed"])
	}
}

func TestDistributionRepository_WithConfig(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	config := &db.DistributionConfig{
		Core: db.CoreConfig{
			Bootloader: "grub",
			Kernel: db.KernelConfig{
				Version: "6.1",
			},
		},
		System: db.SystemConfig{
			Init:           "systemd",
			PackageManager: "apt",
		},
	}

	dist := &db.Distribution{
		Name:       "config-test",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
		Config:     config,
	}
	repo.Create(dist)

	found, err := repo.GetByID(dist.ID)
	if err != nil {
		t.Fatalf("failed to get distribution: %v", err)
	}
	if found.Config == nil {
		t.Fatal("expected config to be present")
	}
	if found.Config.Core.Bootloader != "grub" {
		t.Fatalf("expected bootloader 'grub', got '%s'", found.Config.Core.Bootloader)
	}
	if found.Config.System.Init != "systemd" {
		t.Fatalf("expected init 'systemd', got '%s'", found.Config.System.Init)
	}
}

// =============================================================================
// Distribution Model Tests
// =============================================================================

func TestDistributionStatus_Constants(t *testing.T) {
	statuses := []db.DistributionStatus{
		db.StatusPending,
		db.StatusDownloading,
		db.StatusValidating,
		db.StatusReady,
		db.StatusFailed,
		db.StatusDeleted,
	}

	for _, s := range statuses {
		if s == "" {
			t.Fatal("status constant should not be empty")
		}
	}
}

func TestVisibility_Constants(t *testing.T) {
	if db.VisibilityPublic != "public" {
		t.Fatalf("expected 'public', got '%s'", db.VisibilityPublic)
	}
	if db.VisibilityPrivate != "private" {
		t.Fatalf("expected 'private', got '%s'", db.VisibilityPrivate)
	}
}

func TestDistribution_Timestamps(t *testing.T) {
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer database.Shutdown()

	repo := db.NewDistributionRepository(database)

	before := time.Now().Add(-time.Second)

	dist := &db.Distribution{
		Name:       "timestamp-test",
		Version:    "1.0.0",
		Status:     db.StatusPending,
		Visibility: db.VisibilityPrivate,
	}
	repo.Create(dist)

	after := time.Now().Add(time.Second)

	found, _ := repo.GetByID(dist.ID)
	if found.CreatedAt.Before(before) || found.CreatedAt.After(after) {
		t.Fatal("created_at should be set to current time")
	}
	if found.UpdatedAt.Before(before) || found.UpdatedAt.After(after) {
		t.Fatal("updated_at should be set to current time")
	}
}

// =============================================================================
// Component Repository Tests
// =============================================================================

func setupComponentTestDB(t *testing.T) (*db.Database, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	return database, func() { database.Shutdown() }
}

func TestComponentRepository_Create(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	component := &db.Component{
		Name:        "test-component",
		Category:    "core",
		DisplayName: "Test Component",
		Description: "A test component",
		IsOptional:  false,
		IsSystem:    true,
	}

	if err := repo.Create(component); err != nil {
		t.Fatalf("failed to create component: %v", err)
	}

	if component.ID == "" {
		t.Fatal("expected component ID to be set after creation")
	}

	// Verify timestamps
	if component.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if component.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestComponentRepository_GetByID(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	component := &db.Component{
		Name:        "get-by-id-component",
		Category:    "bootloader",
		DisplayName: "Get By ID Component",
		Description: "Testing GetByID",
		IsOptional:  true,
		IsSystem:    false,
	}
	repo.Create(component)

	// Get existing component
	found, err := repo.GetByID(component.ID)
	if err != nil {
		t.Fatalf("failed to get component: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find component")
	}
	if found.Name != "get-by-id-component" {
		t.Fatalf("expected name 'get-by-id-component', got '%s'", found.Name)
	}
	if found.Category != "bootloader" {
		t.Fatalf("expected category 'bootloader', got '%s'", found.Category)
	}
	if !found.IsOptional {
		t.Fatal("expected IsOptional to be true")
	}

	// Get non-existent component
	notFound, err := repo.GetByID("nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent component")
	}
}

func TestComponentRepository_GetByName(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	component := &db.Component{
		Name:        "unique-component-name",
		Category:    "init",
		DisplayName: "Unique Component",
		IsSystem:    true,
	}
	repo.Create(component)

	found, err := repo.GetByName("unique-component-name")
	if err != nil {
		t.Fatalf("failed to get component by name: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find component")
	}
	if found.ID != component.ID {
		t.Fatalf("expected ID '%s', got '%s'", component.ID, found.ID)
	}

	// Get non-existent
	notFound, err := repo.GetByName("nonexistent-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent component name")
	}
}

func TestComponentRepository_List(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	// Get initial count (system components from migrations)
	initialList, _ := repo.List()
	initialCount := len(initialList)

	// Create additional components
	components := []db.Component{
		{Name: "comp-a", Category: "core", DisplayName: "Comp A", IsSystem: true},
		{Name: "comp-b", Category: "bootloader", DisplayName: "Comp B", IsSystem: true},
		{Name: "comp-c", Category: "init", DisplayName: "Comp C", IsSystem: false},
	}

	for i := range components {
		if err := repo.Create(&components[i]); err != nil {
			t.Fatalf("failed to create component: %v", err)
		}
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("failed to list components: %v", err)
	}

	expectedCount := initialCount + 3
	if len(list) != expectedCount {
		t.Fatalf("expected %d components, got %d", expectedCount, len(list))
	}
}

func TestComponentRepository_ListByOwner(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	// Create a user to be the owner
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("compowner", "compowner@example.com", "hashedpassword", auth.RoleIDDeveloper)
	if err := authRepo.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	repo := db.NewComponentRepository(database)

	// Create components with and without owner
	ownedComp := &db.Component{
		Name:        "owned-component",
		Category:    "custom",
		DisplayName: "Owned Component",
		IsSystem:    false,
		OwnerID:     user.ID,
	}
	repo.Create(ownedComp)

	systemComp := &db.Component{
		Name:        "system-component",
		Category:    "core",
		DisplayName: "System Component",
		IsSystem:    true,
	}
	repo.Create(systemComp)

	// List by owner
	ownerList, err := repo.ListByOwner(user.ID)
	if err != nil {
		t.Fatalf("failed to list components by owner: %v", err)
	}

	if len(ownerList) != 1 {
		t.Fatalf("expected 1 owned component, got %d", len(ownerList))
	}
	if ownerList[0].Name != "owned-component" {
		t.Fatalf("expected 'owned-component', got '%s'", ownerList[0].Name)
	}
}

func TestComponentRepository_ListSystem(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	// Get initial system components count
	initialList, _ := repo.ListSystem()
	initialCount := len(initialList)

	// Create a mix of system and non-system components
	systemComp := &db.Component{
		Name:        "new-system-comp",
		Category:    "core",
		DisplayName: "New System Component",
		IsSystem:    true,
	}
	repo.Create(systemComp)

	userComp := &db.Component{
		Name:        "user-comp",
		Category:    "custom",
		DisplayName: "User Component",
		IsSystem:    false,
	}
	repo.Create(userComp)

	systemList, err := repo.ListSystem()
	if err != nil {
		t.Fatalf("failed to list system components: %v", err)
	}

	expectedCount := initialCount + 1
	if len(systemList) != expectedCount {
		t.Fatalf("expected %d system components, got %d", expectedCount, len(systemList))
	}

	// Verify all are system components
	for _, comp := range systemList {
		if !comp.IsSystem {
			t.Fatalf("expected all components to be system, got non-system: %s", comp.Name)
		}
	}
}

func TestComponentRepository_GetByCategory(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	// Create components with different categories
	comp1 := &db.Component{
		Name:        "filesystem-comp-1",
		Category:    "filesystem",
		DisplayName: "Filesystem 1",
		IsSystem:    true,
	}
	repo.Create(comp1)

	comp2 := &db.Component{
		Name:        "filesystem-comp-2",
		Category:    "filesystem",
		DisplayName: "Filesystem 2",
		IsSystem:    true,
	}
	repo.Create(comp2)

	comp3 := &db.Component{
		Name:        "network-comp",
		Category:    "network",
		DisplayName: "Network Component",
		IsSystem:    true,
	}
	repo.Create(comp3)

	// Get by category
	filesystemComps, err := repo.GetByCategory("filesystem")
	if err != nil {
		t.Fatalf("failed to get components by category: %v", err)
	}

	if len(filesystemComps) < 2 {
		t.Fatalf("expected at least 2 filesystem components, got %d", len(filesystemComps))
	}

	// Verify all returned are filesystem category
	for _, comp := range filesystemComps {
		if comp.Category != "filesystem" {
			// Check if filesystem is in categories list
			found := false
			for _, cat := range comp.Categories {
				if cat == "filesystem" {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected filesystem category, got '%s'", comp.Category)
			}
		}
	}
}

func TestComponentRepository_GetByCategoryAndNameContains(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	// Create components
	comp := &db.Component{
		Name:        "systemd-boot",
		Category:    "bootloader",
		DisplayName: "systemd-boot",
		IsSystem:    true,
	}
	repo.Create(comp)

	// Find by category and name contains
	found, err := repo.GetByCategoryAndNameContains("bootloader", "systemd")
	if err != nil {
		t.Fatalf("failed to get component: %v", err)
	}

	if found == nil {
		t.Fatal("expected to find component")
	}
	if found.Name != "systemd-boot" {
		t.Fatalf("expected 'systemd-boot', got '%s'", found.Name)
	}

	// Search that shouldn't match
	notFound, err := repo.GetByCategoryAndNameContains("bootloader", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-matching search")
	}
}

func TestComponentRepository_Update(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	component := &db.Component{
		Name:        "update-test-comp",
		Category:    "core",
		DisplayName: "Original Name",
		Description: "Original description",
		IsOptional:  false,
		IsSystem:    true,
	}
	repo.Create(component)

	originalUpdatedAt := component.UpdatedAt

	// Update the component
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	component.DisplayName = "Updated Name"
	component.Description = "Updated description"
	component.IsOptional = true

	if err := repo.Update(component); err != nil {
		t.Fatalf("failed to update component: %v", err)
	}

	// Verify update
	found, _ := repo.GetByID(component.ID)
	if found.DisplayName != "Updated Name" {
		t.Fatalf("expected 'Updated Name', got '%s'", found.DisplayName)
	}
	if found.Description != "Updated description" {
		t.Fatalf("expected 'Updated description', got '%s'", found.Description)
	}
	if !found.IsOptional {
		t.Fatal("expected IsOptional to be true")
	}
	if !found.UpdatedAt.After(originalUpdatedAt) {
		t.Fatal("expected UpdatedAt to be updated")
	}
}

func TestComponentRepository_Update_NotFound(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	component := &db.Component{
		ID:          "nonexistent-id",
		Name:        "test",
		Category:    "core",
		DisplayName: "Test",
	}

	err := repo.Update(component)
	if err == nil {
		t.Fatal("expected error when updating non-existent component")
	}
}

func TestComponentRepository_Delete(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	component := &db.Component{
		Name:        "delete-test-comp",
		Category:    "core",
		DisplayName: "Delete Test",
		IsSystem:    true,
	}
	repo.Create(component)

	// Delete
	if err := repo.Delete(component.ID); err != nil {
		t.Fatalf("failed to delete component: %v", err)
	}

	// Verify deleted
	found, _ := repo.GetByID(component.ID)
	if found != nil {
		t.Fatal("expected component to be deleted")
	}
}

func TestComponentRepository_Delete_NotFound(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	err := repo.Delete("nonexistent-id")
	if err == nil {
		t.Fatal("expected error when deleting non-existent component")
	}
}

func TestComponentRepository_GetCategories(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	// Create components with various categories
	categories := []string{"alpha", "beta", "gamma"}
	for i, cat := range categories {
		comp := &db.Component{
			Name:        "cat-comp-" + cat,
			Category:    cat,
			DisplayName: "Category Component " + cat,
			IsSystem:    true,
		}
		_ = i
		repo.Create(comp)
	}

	// Get all categories
	allCategories, err := repo.GetCategories()
	if err != nil {
		t.Fatalf("failed to get categories: %v", err)
	}

	// Verify our categories are present
	categorySet := make(map[string]bool)
	for _, c := range allCategories {
		categorySet[c] = true
	}

	for _, expected := range categories {
		if !categorySet[expected] {
			t.Fatalf("expected category '%s' to be present", expected)
		}
	}
}

func TestComponentRepository_DefaultVersionRule(t *testing.T) {
	database, cleanup := setupComponentTestDB(t)
	defer cleanup()

	repo := db.NewComponentRepository(database)

	// Create without specifying version rule
	comp := &db.Component{
		Name:        "version-rule-test",
		Category:    "core",
		DisplayName: "Version Rule Test",
		IsSystem:    true,
	}
	repo.Create(comp)

	found, _ := repo.GetByID(comp.ID)
	if found.DefaultVersionRule != db.VersionRuleLatestStable {
		t.Fatalf("expected default version rule 'latest-stable', got '%s'", found.DefaultVersionRule)
	}

	// Create with specific version rule
	comp2 := &db.Component{
		Name:               "version-rule-test-2",
		Category:           "core",
		DisplayName:        "Version Rule Test 2",
		IsSystem:           true,
		DefaultVersionRule: db.VersionRuleLatestLTS,
	}
	repo.Create(comp2)

	found2, _ := repo.GetByID(comp2.ID)
	if found2.DefaultVersionRule != db.VersionRuleLatestLTS {
		t.Fatalf("expected version rule 'latest-lts', got '%s'", found2.DefaultVersionRule)
	}
}

// =============================================================================
// Source Repository Tests
// =============================================================================

func setupSourceTestDB(t *testing.T) (*db.Database, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	return database, func() { database.Shutdown() }
}

func TestSourceRepository_Create(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "test-source",
		URL:             "https://example.com/source",
		ComponentIDs:    []string{"comp-1", "comp-2"},
		RetrievalMethod: "release",
		Priority:        10,
		Enabled:         true,
		IsSystem:        true,
	}

	if err := repo.Create(source); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	if source.ID == "" {
		t.Fatal("expected source ID to be set")
	}
	if source.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
}

func TestSourceRepository_GetByID(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	// Create a user for the owner
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("srcgetowner", "srcgetowner@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "get-by-id-source",
		URL:             "https://example.com/source",
		ComponentIDs:    []string{"comp-1"},
		RetrievalMethod: "git",
		URLTemplate:     "https://example.com/{version}",
		Priority:        5,
		Enabled:         true,
		IsSystem:        false,
		OwnerID:         user.ID,
	}
	repo.Create(source)

	found, err := repo.GetByID(source.ID)
	if err != nil {
		t.Fatalf("failed to get source: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find source")
	}
	if found.Name != "get-by-id-source" {
		t.Fatalf("expected name 'get-by-id-source', got '%s'", found.Name)
	}
	if found.RetrievalMethod != "git" {
		t.Fatalf("expected retrieval method 'git', got '%s'", found.RetrievalMethod)
	}
	if len(found.ComponentIDs) != 1 || found.ComponentIDs[0] != "comp-1" {
		t.Fatalf("expected component IDs ['comp-1'], got %v", found.ComponentIDs)
	}

	// Get non-existent
	notFound, err := repo.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent source")
	}
}

func TestSourceRepository_List(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	// Get initial count
	initialList, _ := repo.List()
	initialCount := len(initialList)

	// Create sources
	for i := 0; i < 3; i++ {
		source := &db.UpstreamSource{
			Name:            "list-source-" + string(rune('a'+i)),
			URL:             "https://example.com/source",
			RetrievalMethod: "release",
			Priority:        i,
			Enabled:         true,
			IsSystem:        true,
		}
		repo.Create(source)
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("failed to list sources: %v", err)
	}

	if len(list) != initialCount+3 {
		t.Fatalf("expected %d sources, got %d", initialCount+3, len(list))
	}
}

func TestSourceRepository_ListDefaults(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	// Create system source
	systemSource := &db.UpstreamSource{
		Name:            "system-source",
		URL:             "https://example.com/system",
		RetrievalMethod: "release",
		Priority:        1,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(systemSource)

	// Create user source
	userSource := &db.UpstreamSource{
		Name:            "user-source",
		URL:             "https://example.com/user",
		RetrievalMethod: "release",
		Priority:        2,
		Enabled:         true,
		IsSystem:        false,
		OwnerID:         "user-123",
	}
	repo.CreateUserSource(userSource)

	defaults, err := repo.ListDefaults()
	if err != nil {
		t.Fatalf("failed to list defaults: %v", err)
	}

	// Verify all are system sources
	for _, s := range defaults {
		if !s.IsSystem {
			t.Fatalf("expected all default sources to be system, got non-system: %s", s.Name)
		}
	}

	// Should include our system source
	found := false
	for _, s := range defaults {
		if s.Name == "system-source" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected to find 'system-source' in defaults")
	}
}

func TestSourceRepository_ListDefaultsByComponent(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)
	compRepo := db.NewComponentRepository(database)

	// Create a component
	comp := &db.Component{
		Name:        "test-comp-for-source",
		Category:    "core",
		DisplayName: "Test Component",
		IsSystem:    true,
	}
	compRepo.Create(comp)

	// Create sources for this component
	source1 := &db.UpstreamSource{
		Name:            "comp-source-1",
		URL:             "https://example.com/1",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Priority:        1,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(source1)

	// Create source for different component
	source2 := &db.UpstreamSource{
		Name:            "other-source",
		URL:             "https://example.com/other",
		ComponentIDs:    []string{"other-comp-id"},
		RetrievalMethod: "release",
		Priority:        2,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(source2)

	sources, err := repo.ListDefaultsByComponent(comp.ID)
	if err != nil {
		t.Fatalf("failed to list sources by component: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source for component, got %d", len(sources))
	}
	if sources[0].Name != "comp-source-1" {
		t.Fatalf("expected 'comp-source-1', got '%s'", sources[0].Name)
	}
}

func TestSourceRepository_GetDefaultByID(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	// Create system source
	source := &db.UpstreamSource{
		Name:            "default-source",
		URL:             "https://example.com",
		RetrievalMethod: "release",
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(source)

	// Create user source
	userSource := &db.UpstreamSource{
		Name:            "user-source",
		URL:             "https://example.com/user",
		RetrievalMethod: "release",
		Enabled:         true,
		IsSystem:        false,
		OwnerID:         "user-123",
	}
	repo.CreateUserSource(userSource)

	// GetDefaultByID should only return system sources
	found, err := repo.GetDefaultByID(source.ID)
	if err != nil {
		t.Fatalf("failed to get default source: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find default source")
	}

	// Should not find user source with GetDefaultByID
	notFound, err := repo.GetDefaultByID(userSource.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil when getting user source with GetDefaultByID")
	}
}

func TestSourceRepository_UpdateDefault(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "update-default-source",
		URL:             "https://example.com/original",
		RetrievalMethod: "release",
		Priority:        1,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(source)

	// Update
	source.URL = "https://example.com/updated"
	source.Priority = 5

	if err := repo.UpdateDefault(source); err != nil {
		t.Fatalf("failed to update default source: %v", err)
	}

	found, _ := repo.GetByID(source.ID)
	if found.URL != "https://example.com/updated" {
		t.Fatalf("expected updated URL, got '%s'", found.URL)
	}
	if found.Priority != 5 {
		t.Fatalf("expected priority 5, got %d", found.Priority)
	}
}

func TestSourceRepository_DeleteDefault(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "delete-default-source",
		URL:             "https://example.com",
		RetrievalMethod: "release",
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(source)

	if err := repo.DeleteDefault(source.ID); err != nil {
		t.Fatalf("failed to delete default source: %v", err)
	}

	found, _ := repo.GetByID(source.ID)
	if found != nil {
		t.Fatal("expected source to be deleted")
	}
}

func TestSourceRepository_DeleteDefault_NotFound(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	err := repo.DeleteDefault("nonexistent")
	if err == nil {
		t.Fatal("expected error when deleting non-existent default source")
	}
}

func TestSourceRepository_ListUserSources(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	// Create user
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("sourceowner", "sourceowner@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	repo := db.NewSourceRepository(database)

	// Create user sources
	for i := 0; i < 3; i++ {
		source := &db.UpstreamSource{
			Name:            "user-src-" + string(rune('a'+i)),
			URL:             "https://example.com/user",
			RetrievalMethod: "release",
			Priority:        i,
			Enabled:         true,
			OwnerID:         user.ID,
		}
		repo.CreateUserSource(source)
	}

	// Create source for different user
	otherSource := &db.UpstreamSource{
		Name:            "other-user-src",
		URL:             "https://example.com/other",
		RetrievalMethod: "release",
		Enabled:         true,
		OwnerID:         "other-user-id",
	}
	repo.CreateUserSource(otherSource)

	sources, err := repo.ListUserSources(user.ID)
	if err != nil {
		t.Fatalf("failed to list user sources: %v", err)
	}

	if len(sources) != 3 {
		t.Fatalf("expected 3 user sources, got %d", len(sources))
	}

	for _, s := range sources {
		if s.OwnerID != user.ID {
			t.Fatalf("expected owner ID '%s', got '%s'", user.ID, s.OwnerID)
		}
	}
}

func TestSourceRepository_ListUserSourcesByComponent(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("srccompowner", "srccompowner@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	compRepo := db.NewComponentRepository(database)
	comp := &db.Component{
		Name:        "user-src-comp",
		Category:    "core",
		DisplayName: "User Source Component",
		IsSystem:    true,
	}
	compRepo.Create(comp)

	repo := db.NewSourceRepository(database)

	// Create user source for component
	source := &db.UpstreamSource{
		Name:            "user-comp-source",
		URL:             "https://example.com",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Enabled:         true,
		OwnerID:         user.ID,
	}
	repo.CreateUserSource(source)

	sources, err := repo.ListUserSourcesByComponent(user.ID, comp.ID)
	if err != nil {
		t.Fatalf("failed to list user sources by component: %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
}

func TestSourceRepository_GetUserSourceByID(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	// Create a user for the owner
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("getusersrc", "getusersrc@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "get-user-source",
		URL:             "https://example.com",
		RetrievalMethod: "release",
		Enabled:         true,
		OwnerID:         user.ID,
	}
	repo.CreateUserSource(source)

	found, err := repo.GetUserSourceByID(source.ID)
	if err != nil {
		t.Fatalf("failed to get user source: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find user source")
	}
	if found.IsSystem {
		t.Fatal("expected IsSystem to be false")
	}
}

func TestSourceRepository_UpdateUserSource(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	// Create a user for the owner
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("updateusersrc", "updateusersrc@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "update-user-source",
		URL:             "https://example.com/original",
		RetrievalMethod: "release",
		Enabled:         true,
		OwnerID:         user.ID,
	}
	repo.CreateUserSource(source)

	source.URL = "https://example.com/updated"
	source.Enabled = false

	if err := repo.UpdateUserSource(source); err != nil {
		t.Fatalf("failed to update user source: %v", err)
	}

	found, _ := repo.GetByID(source.ID)
	if found.URL != "https://example.com/updated" {
		t.Fatalf("expected updated URL")
	}
	if found.Enabled {
		t.Fatal("expected Enabled to be false")
	}
}

func TestSourceRepository_DeleteUserSource(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	// Create a user for the owner
	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("deleteusersrc", "deleteusersrc@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "delete-user-source",
		URL:             "https://example.com",
		RetrievalMethod: "release",
		Enabled:         true,
		OwnerID:         user.ID,
	}
	repo.CreateUserSource(source)

	if err := repo.DeleteUserSource(source.ID); err != nil {
		t.Fatalf("failed to delete user source: %v", err)
	}

	found, _ := repo.GetByID(source.ID)
	if found != nil {
		t.Fatal("expected source to be deleted")
	}
}

func TestSourceRepository_DeleteUserSource_NotFound(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	err := repo.DeleteUserSource("nonexistent")
	if err == nil {
		t.Fatal("expected error when deleting non-existent user source")
	}
}

func TestSourceRepository_GetMergedSources(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("mergeuser", "mergeuser@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	repo := db.NewSourceRepository(database)

	// Create system source
	systemSource := &db.UpstreamSource{
		Name:            "merged-system-source",
		URL:             "https://example.com/system",
		RetrievalMethod: "release",
		Priority:        10,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(systemSource)

	// Create user source
	userSource := &db.UpstreamSource{
		Name:            "merged-user-source",
		URL:             "https://example.com/user",
		RetrievalMethod: "release",
		Priority:        5,
		Enabled:         true,
		OwnerID:         user.ID,
	}
	repo.CreateUserSource(userSource)

	// Get merged sources
	merged, err := repo.GetMergedSources(user.ID)
	if err != nil {
		t.Fatalf("failed to get merged sources: %v", err)
	}

	// Should have both sources
	foundSystem := false
	foundUser := false
	for _, s := range merged {
		if s.Name == "merged-system-source" {
			foundSystem = true
		}
		if s.Name == "merged-user-source" {
			foundUser = true
		}
	}

	if !foundSystem {
		t.Fatal("expected to find system source in merged list")
	}
	if !foundUser {
		t.Fatal("expected to find user source in merged list")
	}

	// Verify sorted by priority (ascending)
	for i := 1; i < len(merged); i++ {
		if merged[i].Priority < merged[i-1].Priority {
			t.Fatal("expected sources to be sorted by priority ascending")
		}
	}
}

func TestSourceRepository_GetMergedSourcesByComponent(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("mergecompuser", "mergecompuser@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	compRepo := db.NewComponentRepository(database)
	comp := &db.Component{
		Name:        "merge-comp",
		Category:    "core",
		DisplayName: "Merge Component",
		IsSystem:    true,
	}
	compRepo.Create(comp)

	repo := db.NewSourceRepository(database)

	// Create sources for component
	systemSource := &db.UpstreamSource{
		Name:            "merge-comp-system",
		URL:             "https://example.com/system",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Priority:        10,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(systemSource)

	userSource := &db.UpstreamSource{
		Name:            "merge-comp-user",
		URL:             "https://example.com/user",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Priority:        5,
		Enabled:         true,
		OwnerID:         user.ID,
	}
	repo.CreateUserSource(userSource)

	merged, err := repo.GetMergedSourcesByComponent(user.ID, comp.ID)
	if err != nil {
		t.Fatalf("failed to get merged sources by component: %v", err)
	}

	if len(merged) != 2 {
		t.Fatalf("expected 2 merged sources, got %d", len(merged))
	}
}

func TestSourceRepository_GetEffectiveSource(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	authRepo := auth.NewUserManager(database.DB())
	user := auth.NewUser("effectiveuser", "effectiveuser@example.com", "hash", auth.RoleIDDeveloper)
	authRepo.CreateUser(user)

	compRepo := db.NewComponentRepository(database)
	comp := &db.Component{
		Name:        "effective-comp",
		Category:    "core",
		DisplayName: "Effective Component",
		IsSystem:    true,
	}
	compRepo.Create(comp)

	repo := db.NewSourceRepository(database)

	// Create disabled low priority source
	disabledSource := &db.UpstreamSource{
		Name:            "disabled-source",
		URL:             "https://example.com/disabled",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Priority:        1,
		Enabled:         false, // Disabled
		IsSystem:        true,
	}
	repo.CreateDefault(disabledSource)

	// Create enabled higher priority source
	enabledSource := &db.UpstreamSource{
		Name:            "enabled-source",
		URL:             "https://example.com/enabled",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Priority:        10,
		Enabled:         true,
		IsSystem:        true,
	}
	repo.CreateDefault(enabledSource)

	effective, err := repo.GetEffectiveSource(comp.ID, user.ID)
	if err != nil {
		t.Fatalf("failed to get effective source: %v", err)
	}

	if effective == nil {
		t.Fatal("expected to find effective source")
	}

	// Should return enabled source even though disabled has lower priority
	if effective.Name != "enabled-source" {
		t.Fatalf("expected 'enabled-source', got '%s'", effective.Name)
	}
}

func TestSourceRepository_GetEffectiveSource_NoEnabled(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	compRepo := db.NewComponentRepository(database)
	comp := &db.Component{
		Name:        "no-enabled-comp",
		Category:    "core",
		DisplayName: "No Enabled Component",
		IsSystem:    true,
	}
	compRepo.Create(comp)

	repo := db.NewSourceRepository(database)

	// Create only disabled source
	source := &db.UpstreamSource{
		Name:            "only-disabled",
		URL:             "https://example.com",
		ComponentIDs:    []string{comp.ID},
		RetrievalMethod: "release",
		Priority:        1,
		Enabled:         false,
		IsSystem:        true,
	}
	repo.CreateDefault(source)

	effective, err := repo.GetEffectiveSource(comp.ID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if effective != nil {
		t.Fatal("expected nil when no enabled sources")
	}
}

func TestSourceRepository_Delete(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	source := &db.UpstreamSource{
		Name:            "delete-any-source",
		URL:             "https://example.com",
		RetrievalMethod: "release",
		Enabled:         true,
		IsSystem:        true,
	}
	repo.Create(source)

	if err := repo.Delete(source.ID); err != nil {
		t.Fatalf("failed to delete source: %v", err)
	}

	found, _ := repo.GetByID(source.ID)
	if found != nil {
		t.Fatal("expected source to be deleted")
	}
}

func TestSourceRepository_Delete_NotFound(t *testing.T) {
	database, cleanup := setupSourceTestDB(t)
	defer cleanup()

	repo := db.NewSourceRepository(database)

	err := repo.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error when deleting non-existent source")
	}
}

// =============================================================================
// Source Version Repository Tests
// =============================================================================

func setupSourceVersionTestDB(t *testing.T) (*db.Database, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	return database, func() { database.Shutdown() }
}

func TestSourceVersionRepository_Create(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	now := time.Now()
	version := &db.SourceVersion{
		SourceID:     "source-1",
		SourceType:   "default",
		Version:      "1.0.0",
		VersionType:  db.VersionTypeStable,
		ReleaseDate:  &now,
		DownloadURL:  "https://example.com/v1.0.0.tar.gz",
		Checksum:     "abc123",
		ChecksumType: "sha256",
		FileSize:     1024000,
		IsStable:     true,
	}

	if err := repo.Create(version); err != nil {
		t.Fatalf("failed to create version: %v", err)
	}

	if version.ID == "" {
		t.Fatal("expected version ID to be set")
	}
	if version.DiscoveredAt.IsZero() {
		t.Fatal("expected DiscoveredAt to be set")
	}
}

func TestSourceVersionRepository_GetByID(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	version := &db.SourceVersion{
		SourceID:    "source-1",
		SourceType:  "default",
		Version:     "2.0.0",
		VersionType: db.VersionTypeLongterm,
		DownloadURL: "https://example.com/v2.0.0.tar.gz",
		IsStable:    true,
	}
	repo.Create(version)

	found, err := repo.GetByID(version.ID)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find version")
	}
	if found.Version != "2.0.0" {
		t.Fatalf("expected version '2.0.0', got '%s'", found.Version)
	}
	if found.VersionType != db.VersionTypeLongterm {
		t.Fatalf("expected version type 'longterm', got '%s'", found.VersionType)
	}

	// Get non-existent
	notFound, err := repo.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent version")
	}
}

func TestSourceVersionRepository_GetByVersion(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	version := &db.SourceVersion{
		SourceID:   "source-1",
		SourceType: "default",
		Version:    "3.0.0",
		IsStable:   true,
	}
	repo.Create(version)

	found, err := repo.GetByVersion("source-1", "default", "3.0.0")
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find version")
	}
	if found.ID != version.ID {
		t.Fatalf("expected ID '%s', got '%s'", version.ID, found.ID)
	}

	// Get non-existent version
	notFound, err := repo.GetByVersion("source-1", "default", "99.99.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent version")
	}
}

func TestSourceVersionRepository_ListBySource(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create versions for source-1
	for _, v := range []string{"1.0.0", "1.1.0", "1.2.0"} {
		version := &db.SourceVersion{
			SourceID:   "source-1",
			SourceType: "default",
			Version:    v,
			IsStable:   true,
		}
		repo.Create(version)
	}

	// Create version for different source
	otherVersion := &db.SourceVersion{
		SourceID:   "source-2",
		SourceType: "default",
		Version:    "2.0.0",
		IsStable:   true,
	}
	repo.Create(otherVersion)

	versions, err := repo.ListBySource("source-1", "default")
	if err != nil {
		t.Fatalf("failed to list versions: %v", err)
	}

	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	for _, v := range versions {
		if v.SourceID != "source-1" {
			t.Fatalf("expected source ID 'source-1', got '%s'", v.SourceID)
		}
	}
}

func TestSourceVersionRepository_ListBySourceStable(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create stable versions
	for _, v := range []string{"1.0.0", "1.1.0"} {
		version := &db.SourceVersion{
			SourceID:   "source-stable",
			SourceType: "default",
			Version:    v,
			IsStable:   true,
		}
		repo.Create(version)
	}

	// Create unstable version
	unstable := &db.SourceVersion{
		SourceID:   "source-stable",
		SourceType: "default",
		Version:    "2.0.0-rc1",
		IsStable:   false,
	}
	repo.Create(unstable)

	stable, err := repo.ListBySourceStable("source-stable", "default")
	if err != nil {
		t.Fatalf("failed to list stable versions: %v", err)
	}

	if len(stable) != 2 {
		t.Fatalf("expected 2 stable versions, got %d", len(stable))
	}

	for _, v := range stable {
		if !v.IsStable {
			t.Fatalf("expected all versions to be stable, got unstable: %s", v.Version)
		}
	}
}

func TestSourceVersionRepository_ListBySourcePaginated(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create 10 versions
	for i := 0; i < 10; i++ {
		version := &db.SourceVersion{
			SourceID:   "source-paginated",
			SourceType: "default",
			Version:    "1." + string(rune('0'+i)) + ".0",
			IsStable:   true,
		}
		repo.Create(version)
	}

	// Get first page
	page1, total, err := repo.ListBySourcePaginated("source-paginated", "default", 5, 0, "")
	if err != nil {
		t.Fatalf("failed to list paginated: %v", err)
	}

	if total != 10 {
		t.Fatalf("expected total 10, got %d", total)
	}
	if len(page1) != 5 {
		t.Fatalf("expected 5 versions in page 1, got %d", len(page1))
	}

	// Get second page
	page2, _, err := repo.ListBySourcePaginated("source-paginated", "default", 5, 5, "")
	if err != nil {
		t.Fatalf("failed to list paginated: %v", err)
	}

	if len(page2) != 5 {
		t.Fatalf("expected 5 versions in page 2, got %d", len(page2))
	}
}

func TestSourceVersionRepository_ListBySourcePaginated_WithFilter(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create versions with different types
	stableV := &db.SourceVersion{
		SourceID:    "source-filter",
		SourceType:  "default",
		Version:     "1.0.0",
		VersionType: db.VersionTypeStable,
		IsStable:    true,
	}
	repo.Create(stableV)

	longtermV := &db.SourceVersion{
		SourceID:    "source-filter",
		SourceType:  "default",
		Version:     "2.0.0",
		VersionType: db.VersionTypeLongterm,
		IsStable:    true,
	}
	repo.Create(longtermV)

	mainlineV := &db.SourceVersion{
		SourceID:    "source-filter",
		SourceType:  "default",
		Version:     "3.0.0",
		VersionType: db.VersionTypeMainline,
		IsStable:    false,
	}
	repo.Create(mainlineV)

	// Filter by stable type
	stable, total, err := repo.ListBySourcePaginated("source-filter", "default", 10, 0, string(db.VersionTypeStable))
	if err != nil {
		t.Fatalf("failed to list with filter: %v", err)
	}

	if total != 1 {
		t.Fatalf("expected 1 stable version, got %d", total)
	}
	if len(stable) != 1 {
		t.Fatalf("expected 1 result, got %d", len(stable))
	}
	if stable[0].VersionType != db.VersionTypeStable {
		t.Fatalf("expected stable type, got '%s'", stable[0].VersionType)
	}
}

func TestSourceVersionRepository_GetLatestStable(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create versions (oldest first)
	v1 := &db.SourceVersion{
		SourceID:   "source-latest",
		SourceType: "default",
		Version:    "1.0.0",
		IsStable:   true,
	}
	repo.Create(v1)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	v2 := &db.SourceVersion{
		SourceID:   "source-latest",
		SourceType: "default",
		Version:    "2.0.0",
		IsStable:   true,
	}
	repo.Create(v2)

	time.Sleep(10 * time.Millisecond)

	// Unstable version (should not be returned)
	v3 := &db.SourceVersion{
		SourceID:   "source-latest",
		SourceType: "default",
		Version:    "3.0.0-rc1",
		IsStable:   false,
	}
	repo.Create(v3)

	latest, err := repo.GetLatestStable("source-latest", "default")
	if err != nil {
		t.Fatalf("failed to get latest stable: %v", err)
	}

	if latest == nil {
		t.Fatal("expected to find latest stable version")
	}
	if latest.Version != "2.0.0" {
		t.Fatalf("expected version '2.0.0', got '%s'", latest.Version)
	}
}

func TestSourceVersionRepository_BulkUpsert(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create initial version
	initial := &db.SourceVersion{
		SourceID:   "source-bulk",
		SourceType: "default",
		Version:    "1.0.0",
		IsStable:   true,
	}
	repo.Create(initial)

	// Bulk upsert with mix of new and existing
	versions := []db.SourceVersion{
		{SourceID: "source-bulk", SourceType: "default", Version: "1.0.0", IsStable: true, DownloadURL: "updated-url"}, // Existing
		{SourceID: "source-bulk", SourceType: "default", Version: "1.1.0", IsStable: true},                             // New
		{SourceID: "source-bulk", SourceType: "default", Version: "1.2.0", IsStable: true},                             // New
	}

	newCount, err := repo.BulkUpsert(versions)
	if err != nil {
		t.Fatalf("failed to bulk upsert: %v", err)
	}

	if newCount != 2 {
		t.Fatalf("expected 2 new versions, got %d", newCount)
	}

	// Verify total count
	all, _ := repo.ListBySource("source-bulk", "default")
	if len(all) != 3 {
		t.Fatalf("expected 3 total versions, got %d", len(all))
	}

	// Verify existing was updated
	existing, _ := repo.GetByVersion("source-bulk", "default", "1.0.0")
	if existing.DownloadURL != "updated-url" {
		t.Fatalf("expected updated URL, got '%s'", existing.DownloadURL)
	}
}

func TestSourceVersionRepository_BulkUpsert_Empty(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	newCount, err := repo.BulkUpsert([]db.SourceVersion{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCount != 0 {
		t.Fatalf("expected 0 new versions, got %d", newCount)
	}
}

func TestSourceVersionRepository_DeleteBySource(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create versions
	for i := 0; i < 3; i++ {
		v := &db.SourceVersion{
			SourceID:   "source-delete",
			SourceType: "default",
			Version:    "1." + string(rune('0'+i)) + ".0",
			IsStable:   true,
		}
		repo.Create(v)
	}

	if err := repo.DeleteBySource("source-delete", "default"); err != nil {
		t.Fatalf("failed to delete by source: %v", err)
	}

	remaining, _ := repo.ListBySource("source-delete", "default")
	if len(remaining) != 0 {
		t.Fatalf("expected 0 remaining versions, got %d", len(remaining))
	}
}

func TestSourceVersionRepository_CountBySource(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	for i := 0; i < 5; i++ {
		v := &db.SourceVersion{
			SourceID:   "source-count",
			SourceType: "default",
			Version:    "1." + string(rune('0'+i)) + ".0",
			IsStable:   true,
		}
		repo.Create(v)
	}

	count, err := repo.CountBySource("source-count", "default")
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}
}

// =============================================================================
// Version Sync Job Tests
// =============================================================================

func TestSourceVersionRepository_CreateSyncJob(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	job := &db.VersionSyncJob{
		SourceID:   "source-sync",
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}

	if err := repo.CreateSyncJob(job); err != nil {
		t.Fatalf("failed to create sync job: %v", err)
	}

	if job.ID == "" {
		t.Fatal("expected job ID to be set")
	}
	if job.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
}

func TestSourceVersionRepository_GetSyncJob(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	job := &db.VersionSyncJob{
		SourceID:   "source-get-job",
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}
	repo.CreateSyncJob(job)

	found, err := repo.GetSyncJob(job.ID)
	if err != nil {
		t.Fatalf("failed to get sync job: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find sync job")
	}
	if found.SourceID != "source-get-job" {
		t.Fatalf("expected source ID 'source-get-job', got '%s'", found.SourceID)
	}

	// Get non-existent
	notFound, err := repo.GetSyncJob("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent job")
	}
}

func TestSourceVersionRepository_GetLatestSyncJob(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create older job
	oldJob := &db.VersionSyncJob{
		SourceID:   "source-latest-job",
		SourceType: "default",
		Status:     db.SyncStatusCompleted,
	}
	repo.CreateSyncJob(oldJob)

	time.Sleep(10 * time.Millisecond)

	// Create newer job
	newJob := &db.VersionSyncJob{
		SourceID:   "source-latest-job",
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}
	repo.CreateSyncJob(newJob)

	latest, err := repo.GetLatestSyncJob("source-latest-job", "default")
	if err != nil {
		t.Fatalf("failed to get latest sync job: %v", err)
	}

	if latest == nil {
		t.Fatal("expected to find latest sync job")
	}
	if latest.ID != newJob.ID {
		t.Fatalf("expected latest job ID '%s', got '%s'", newJob.ID, latest.ID)
	}
}

func TestSourceVersionRepository_GetRunningSyncJob(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create completed job
	completedJob := &db.VersionSyncJob{
		SourceID:   "source-running",
		SourceType: "default",
		Status:     db.SyncStatusCompleted,
	}
	repo.CreateSyncJob(completedJob)

	// Create running job
	runningJob := &db.VersionSyncJob{
		SourceID:   "source-running",
		SourceType: "default",
		Status:     db.SyncStatusRunning,
	}
	repo.CreateSyncJob(runningJob)

	running, err := repo.GetRunningSyncJob("source-running", "default")
	if err != nil {
		t.Fatalf("failed to get running job: %v", err)
	}

	if running == nil {
		t.Fatal("expected to find running job")
	}
	if running.Status != db.SyncStatusRunning {
		t.Fatalf("expected running status, got '%s'", running.Status)
	}
}

func TestSourceVersionRepository_GetRunningSyncJob_None(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create only completed job
	completedJob := &db.VersionSyncJob{
		SourceID:   "source-no-running",
		SourceType: "default",
		Status:     db.SyncStatusCompleted,
	}
	repo.CreateSyncJob(completedJob)

	running, err := repo.GetRunningSyncJob("source-no-running", "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if running != nil {
		t.Fatal("expected nil when no running jobs")
	}
}

func TestSourceVersionRepository_UpdateSyncJob(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	job := &db.VersionSyncJob{
		SourceID:   "source-update-job",
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}
	repo.CreateSyncJob(job)

	// Update
	now := time.Now()
	job.Status = db.SyncStatusCompleted
	job.VersionsFound = 10
	job.VersionsNew = 5
	job.CompletedAt = &now

	if err := repo.UpdateSyncJob(job); err != nil {
		t.Fatalf("failed to update sync job: %v", err)
	}

	found, _ := repo.GetSyncJob(job.ID)
	if found.Status != db.SyncStatusCompleted {
		t.Fatalf("expected completed status, got '%s'", found.Status)
	}
	if found.VersionsFound != 10 {
		t.Fatalf("expected 10 versions found, got %d", found.VersionsFound)
	}
	if found.VersionsNew != 5 {
		t.Fatalf("expected 5 new versions, got %d", found.VersionsNew)
	}
}

func TestSourceVersionRepository_MarkSyncJobRunning(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	job := &db.VersionSyncJob{
		SourceID:   "source-mark-running",
		SourceType: "default",
		Status:     db.SyncStatusPending,
	}
	repo.CreateSyncJob(job)

	if err := repo.MarkSyncJobRunning(job.ID); err != nil {
		t.Fatalf("failed to mark running: %v", err)
	}

	found, _ := repo.GetSyncJob(job.ID)
	if found.Status != db.SyncStatusRunning {
		t.Fatalf("expected running status, got '%s'", found.Status)
	}
	if found.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
}

func TestSourceVersionRepository_MarkSyncJobCompleted(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	job := &db.VersionSyncJob{
		SourceID:   "source-mark-completed",
		SourceType: "default",
		Status:     db.SyncStatusRunning,
	}
	repo.CreateSyncJob(job)

	if err := repo.MarkSyncJobCompleted(job.ID, 15, 8); err != nil {
		t.Fatalf("failed to mark completed: %v", err)
	}

	found, _ := repo.GetSyncJob(job.ID)
	if found.Status != db.SyncStatusCompleted {
		t.Fatalf("expected completed status, got '%s'", found.Status)
	}
	if found.VersionsFound != 15 {
		t.Fatalf("expected 15 versions found, got %d", found.VersionsFound)
	}
	if found.VersionsNew != 8 {
		t.Fatalf("expected 8 new versions, got %d", found.VersionsNew)
	}
	if found.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestSourceVersionRepository_MarkSyncJobFailed(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	job := &db.VersionSyncJob{
		SourceID:   "source-mark-failed",
		SourceType: "default",
		Status:     db.SyncStatusRunning,
	}
	repo.CreateSyncJob(job)

	if err := repo.MarkSyncJobFailed(job.ID, "Connection timeout"); err != nil {
		t.Fatalf("failed to mark failed: %v", err)
	}

	found, _ := repo.GetSyncJob(job.ID)
	if found.Status != db.SyncStatusFailed {
		t.Fatalf("expected failed status, got '%s'", found.Status)
	}
	if found.ErrorMessage != "Connection timeout" {
		t.Fatalf("expected error message 'Connection timeout', got '%s'", found.ErrorMessage)
	}
	if found.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestSourceVersionRepository_ListSyncJobsBySource(t *testing.T) {
	database, cleanup := setupSourceVersionTestDB(t)
	defer cleanup()

	repo := db.NewSourceVersionRepository(database)

	// Create jobs
	for i := 0; i < 5; i++ {
		job := &db.VersionSyncJob{
			SourceID:   "source-list-jobs",
			SourceType: "default",
			Status:     db.SyncStatusCompleted,
		}
		repo.CreateSyncJob(job)
		time.Sleep(5 * time.Millisecond)
	}

	jobs, err := repo.ListSyncJobsBySource("source-list-jobs", "default", 3)
	if err != nil {
		t.Fatalf("failed to list sync jobs: %v", err)
	}

	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs (limit), got %d", len(jobs))
	}
}

// =============================================================================
// Language Pack Repository Tests
// =============================================================================

func setupLanguagePackTestDB(t *testing.T) (*db.Database, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	return database, func() { database.Shutdown() }
}

func TestLanguagePackRepository_Create(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "fr-FR",
		Name:       "French",
		Version:    "1.0.0",
		Author:     "Test Author",
		Dictionary: `{"hello": "bonjour", "goodbye": "au revoir"}`,
	}

	if err := repo.Create(pack); err != nil {
		t.Fatalf("failed to create language pack: %v", err)
	}

	if pack.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if pack.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestLanguagePackRepository_Create_Duplicate(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "de-DE",
		Name:       "German",
		Version:    "1.0.0",
		Dictionary: `{}`,
	}
	repo.Create(pack)

	// Try to create duplicate
	duplicate := &db.LanguagePack{
		Locale:     "de-DE",
		Name:       "German Duplicate",
		Version:    "2.0.0",
		Dictionary: `{}`,
	}
	err := repo.Create(duplicate)
	if err == nil {
		t.Fatal("expected error when creating duplicate locale")
	}
}

func TestLanguagePackRepository_Get(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "es-ES",
		Name:       "Spanish",
		Version:    "1.0.0",
		Author:     "Test Author",
		Dictionary: `{"hello": "hola"}`,
	}
	repo.Create(pack)

	found, err := repo.Get("es-ES")
	if err != nil {
		t.Fatalf("failed to get language pack: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find language pack")
	}
	if found.Name != "Spanish" {
		t.Fatalf("expected name 'Spanish', got '%s'", found.Name)
	}
	if found.Author != "Test Author" {
		t.Fatalf("expected author 'Test Author', got '%s'", found.Author)
	}
	if found.Dictionary != `{"hello": "hola"}` {
		t.Fatalf("unexpected dictionary content")
	}
}

func TestLanguagePackRepository_Get_NotFound(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	found, err := repo.Get("nonexistent-locale")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected nil for non-existent locale")
	}
}

func TestLanguagePackRepository_List(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	// Create multiple packs
	packs := []db.LanguagePack{
		{Locale: "en-US", Name: "English (US)", Version: "1.0.0", Dictionary: `{}`},
		{Locale: "en-GB", Name: "English (UK)", Version: "1.0.0", Dictionary: `{}`},
		{Locale: "ja-JP", Name: "Japanese", Version: "1.0.0", Dictionary: `{}`},
	}

	for i := range packs {
		repo.Create(&packs[i])
	}

	list, err := repo.List()
	if err != nil {
		t.Fatalf("failed to list language packs: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 language packs, got %d", len(list))
	}

	// Verify metadata only (no dictionary)
	for _, meta := range list {
		if meta.Locale == "" {
			t.Fatal("expected locale to be set")
		}
		if meta.Name == "" {
			t.Fatal("expected name to be set")
		}
	}
}

func TestLanguagePackRepository_Update(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "it-IT",
		Name:       "Italian",
		Version:    "1.0.0",
		Dictionary: `{"hello": "ciao"}`,
	}
	repo.Create(pack)

	originalUpdatedAt := pack.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	// Update
	pack.Version = "1.1.0"
	pack.Author = "New Author"
	pack.Dictionary = `{"hello": "ciao", "goodbye": "arrivederci"}`

	if err := repo.Update(pack); err != nil {
		t.Fatalf("failed to update language pack: %v", err)
	}

	found, _ := repo.Get("it-IT")
	if found.Version != "1.1.0" {
		t.Fatalf("expected version '1.1.0', got '%s'", found.Version)
	}
	if found.Author != "New Author" {
		t.Fatalf("expected author 'New Author', got '%s'", found.Author)
	}
	if !found.UpdatedAt.After(originalUpdatedAt) {
		t.Fatal("expected UpdatedAt to be updated")
	}
}

func TestLanguagePackRepository_Update_NotFound(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "nonexistent",
		Name:       "Test",
		Version:    "1.0.0",
		Dictionary: `{}`,
	}

	err := repo.Update(pack)
	if err == nil {
		t.Fatal("expected error when updating non-existent pack")
	}
}

func TestLanguagePackRepository_Delete(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "pt-BR",
		Name:       "Portuguese (Brazil)",
		Version:    "1.0.0",
		Dictionary: `{}`,
	}
	repo.Create(pack)

	if err := repo.Delete("pt-BR"); err != nil {
		t.Fatalf("failed to delete language pack: %v", err)
	}

	found, _ := repo.Get("pt-BR")
	if found != nil {
		t.Fatal("expected language pack to be deleted")
	}
}

func TestLanguagePackRepository_Delete_NotFound(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	err := repo.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error when deleting non-existent pack")
	}
}

func TestLanguagePackRepository_Exists(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	pack := &db.LanguagePack{
		Locale:     "ko-KR",
		Name:       "Korean",
		Version:    "1.0.0",
		Dictionary: `{}`,
	}
	repo.Create(pack)

	// Check existing
	exists, err := repo.Exists("ko-KR")
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if !exists {
		t.Fatal("expected pack to exist")
	}

	// Check non-existing
	notExists, err := repo.Exists("nonexistent")
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if notExists {
		t.Fatal("expected pack to not exist")
	}
}

func TestLanguagePackRepository_WithNullAuthor(t *testing.T) {
	database, cleanup := setupLanguagePackTestDB(t)
	defer cleanup()

	repo := db.NewLanguagePackRepository(database)

	// Create without author
	pack := &db.LanguagePack{
		Locale:     "zh-CN",
		Name:       "Chinese (Simplified)",
		Version:    "1.0.0",
		Author:     "", // Empty author
		Dictionary: `{}`,
	}
	repo.Create(pack)

	found, err := repo.Get("zh-CN")
	if err != nil {
		t.Fatalf("failed to get language pack: %v", err)
	}
	if found.Author != "" {
		t.Fatalf("expected empty author, got '%s'", found.Author)
	}
}
