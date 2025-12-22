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

	if dist.ID == 0 {
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
	notFound, err := repo.GetByID(99999)
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
	authRepo := auth.NewRepository(database.DB())
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
	authRepo := auth.NewRepository(database.DB())
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
	err = repo.Delete(99999)
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
