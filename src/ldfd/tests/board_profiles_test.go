package tests

import (
	"net/http"
	"testing"

	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/bitswalk/ldf/src/ldfd/db"
)

// =============================================================================
// Board Profile Repository Tests
// =============================================================================

func setupBoardProfileTestDB(t *testing.T) (*db.Database, func()) {
	t.Helper()
	cfg := db.Config{
		PersistPath: "",
		LoadOnStart: false,
	}

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	return database, func() { _ = database.Shutdown() }
}

func TestBoardProfileRepository_SeededProfiles(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profiles, err := repo.List()
	if err != nil {
		t.Fatalf("failed to list profiles: %v", err)
	}

	if len(profiles) != 2 {
		t.Fatalf("expected 2 seeded profiles, got %d", len(profiles))
	}

	// Verify generic-x86_64
	x86, err := repo.GetByName("generic-x86_64")
	if err != nil {
		t.Fatalf("failed to get generic-x86_64: %v", err)
	}
	if x86 == nil {
		t.Fatal("expected generic-x86_64 profile to exist")
	}
	if x86.Arch != db.ArchX86_64 {
		t.Fatalf("expected arch x86_64, got %s", x86.Arch)
	}
	if !x86.IsSystem {
		t.Fatal("expected generic-x86_64 to be a system profile")
	}
	if x86.DisplayName == "" {
		t.Fatal("expected display name to be set")
	}

	// Verify rpi4
	rpi, err := repo.GetByName("rpi4")
	if err != nil {
		t.Fatalf("failed to get rpi4: %v", err)
	}
	if rpi == nil {
		t.Fatal("expected rpi4 profile to exist")
	}
	if rpi.Arch != db.ArchAARCH64 {
		t.Fatalf("expected arch aarch64, got %s", rpi.Arch)
	}
	if !rpi.IsSystem {
		t.Fatal("expected rpi4 to be a system profile")
	}
	if rpi.Config.KernelDefconfig != "bcm2711_defconfig" {
		t.Fatalf("expected bcm2711_defconfig, got %s", rpi.Config.KernelDefconfig)
	}
	if len(rpi.Config.DeviceTrees) == 0 {
		t.Fatal("expected rpi4 to have device trees")
	}
	if rpi.Config.BootParams.ConfigTxt == "" {
		t.Fatal("expected rpi4 to have config.txt boot params")
	}
}

func TestBoardProfileRepository_Create(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		Name:        "jetson-orin",
		DisplayName: "NVIDIA Jetson Orin",
		Description: "Jetson Orin developer kit",
		Arch:        db.ArchAARCH64,
		Config: db.BoardConfig{
			KernelDefconfig: "tegra_defconfig",
			KernelOverlay: map[string]string{
				"CONFIG_TEGRA_HOST1X": "y",
			},
			DeviceTrees: []db.DeviceTreeSpec{
				{Source: "arch/arm64/boot/dts/nvidia/tegra234-p3701-0000.dts"},
			},
		},
		IsSystem: false,
		OwnerID:  "user-123",
	}

	if err := repo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	if profile.ID == "" {
		t.Fatal("expected profile ID to be set")
	}
	if profile.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if profile.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestBoardProfileRepository_GetByID(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		Name:        "custom-board",
		DisplayName: "Custom Board",
		Arch:        db.ArchX86_64,
		Config: db.BoardConfig{
			KernelOverlay: map[string]string{
				"CONFIG_TEST": "y",
			},
		},
		OwnerID: "user-1",
	}
	if err := repo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	found, err := repo.GetByID(profile.ID)
	if err != nil {
		t.Fatalf("failed to get profile: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find profile")
	}
	if found.Name != "custom-board" {
		t.Fatalf("expected name 'custom-board', got '%s'", found.Name)
	}
	if found.Config.KernelOverlay["CONFIG_TEST"] != "y" {
		t.Fatal("expected kernel overlay to be preserved")
	}

	// Get non-existent
	notFound, err := repo.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent profile")
	}
}

func TestBoardProfileRepository_GetByName(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		Name:        "unique-board",
		DisplayName: "Unique Board",
		Arch:        db.ArchAARCH64,
		OwnerID:     "user-1",
	}
	if err := repo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	found, err := repo.GetByName("unique-board")
	if err != nil {
		t.Fatalf("failed to get profile by name: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find profile")
	}
	if found.ID != profile.ID {
		t.Fatalf("expected ID '%s', got '%s'", profile.ID, found.ID)
	}

	// Get non-existent
	notFound, err := repo.GetByName("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notFound != nil {
		t.Fatal("expected nil for non-existent name")
	}
}

func TestBoardProfileRepository_ListByArch(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	// Create additional profiles
	aarch64Profile := &db.BoardProfile{
		Name:        "custom-aarch64",
		DisplayName: "Custom AArch64",
		Arch:        db.ArchAARCH64,
		OwnerID:     "user-1",
	}
	if err := repo.Create(aarch64Profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	// List aarch64 profiles (seeded rpi4 + our custom one)
	aarch64List, err := repo.ListByArch(db.ArchAARCH64)
	if err != nil {
		t.Fatalf("failed to list by arch: %v", err)
	}
	if len(aarch64List) != 2 {
		t.Fatalf("expected 2 aarch64 profiles, got %d", len(aarch64List))
	}

	// List x86_64 profiles (only seeded generic-x86_64)
	x86List, err := repo.ListByArch(db.ArchX86_64)
	if err != nil {
		t.Fatalf("failed to list by arch: %v", err)
	}
	if len(x86List) != 1 {
		t.Fatalf("expected 1 x86_64 profile, got %d", len(x86List))
	}
}

func TestBoardProfileRepository_ListSystem(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	// Create a non-system profile
	userProfile := &db.BoardProfile{
		Name:        "user-board",
		DisplayName: "User Board",
		Arch:        db.ArchX86_64,
		IsSystem:    false,
		OwnerID:     "user-1",
	}
	if err := repo.Create(userProfile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	systemList, err := repo.ListSystem()
	if err != nil {
		t.Fatalf("failed to list system profiles: %v", err)
	}

	// Should be exactly 2 (seeded profiles)
	if len(systemList) != 2 {
		t.Fatalf("expected 2 system profiles, got %d", len(systemList))
	}

	for _, p := range systemList {
		if !p.IsSystem {
			t.Fatalf("expected all listed profiles to be system, got non-system: %s", p.Name)
		}
	}
}

func TestBoardProfileRepository_ListByOwner(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	// Create profiles for different owners
	for i := 0; i < 3; i++ {
		p := &db.BoardProfile{
			Name:        "owner1-board-" + string(rune('a'+i)),
			DisplayName: "Owner1 Board",
			Arch:        db.ArchX86_64,
			OwnerID:     "owner-1",
		}
		if err := repo.Create(p); err != nil {
			t.Fatalf("failed to create profile: %v", err)
		}
	}

	otherProfile := &db.BoardProfile{
		Name:        "owner2-board",
		DisplayName: "Owner2 Board",
		Arch:        db.ArchAARCH64,
		OwnerID:     "owner-2",
	}
	if err := repo.Create(otherProfile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	owner1List, err := repo.ListByOwner("owner-1")
	if err != nil {
		t.Fatalf("failed to list by owner: %v", err)
	}
	if len(owner1List) != 3 {
		t.Fatalf("expected 3 profiles for owner-1, got %d", len(owner1List))
	}

	owner2List, err := repo.ListByOwner("owner-2")
	if err != nil {
		t.Fatalf("failed to list by owner: %v", err)
	}
	if len(owner2List) != 1 {
		t.Fatalf("expected 1 profile for owner-2, got %d", len(owner2List))
	}
}

func TestBoardProfileRepository_Update(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		Name:        "update-board",
		DisplayName: "Original Name",
		Description: "Original description",
		Arch:        db.ArchX86_64,
		Config: db.BoardConfig{
			KernelCmdline: "console=tty0",
		},
		OwnerID: "user-1",
	}
	if err := repo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	originalUpdatedAt := profile.UpdatedAt

	// Update fields
	profile.DisplayName = "Updated Name"
	profile.Description = "Updated description"
	profile.Config.KernelCmdline = "console=tty0 console=ttyS0,115200"

	if err := repo.Update(profile); err != nil {
		t.Fatalf("failed to update profile: %v", err)
	}

	found, err := repo.GetByID(profile.ID)
	if err != nil {
		t.Fatalf("failed to get updated profile: %v", err)
	}
	if found.DisplayName != "Updated Name" {
		t.Fatalf("expected 'Updated Name', got '%s'", found.DisplayName)
	}
	if found.Description != "Updated description" {
		t.Fatalf("expected 'Updated description', got '%s'", found.Description)
	}
	if found.Config.KernelCmdline != "console=tty0 console=ttyS0,115200" {
		t.Fatalf("expected updated cmdline, got '%s'", found.Config.KernelCmdline)
	}
	if !found.UpdatedAt.After(originalUpdatedAt) {
		t.Fatal("expected UpdatedAt to be updated")
	}
}

func TestBoardProfileRepository_Update_NotFound(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		ID:          "nonexistent-id",
		Name:        "nonexistent",
		DisplayName: "Nonexistent",
	}

	err := repo.Update(profile)
	if err == nil {
		t.Fatal("expected error when updating non-existent profile")
	}
}

func TestBoardProfileRepository_Delete(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		Name:        "delete-board",
		DisplayName: "Delete Board",
		Arch:        db.ArchX86_64,
		OwnerID:     "user-1",
	}
	if err := repo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	if err := repo.Delete(profile.ID); err != nil {
		t.Fatalf("failed to delete profile: %v", err)
	}

	found, err := repo.GetByID(profile.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected profile to be deleted")
	}
}

func TestBoardProfileRepository_Delete_NotFound(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	err := repo.Delete("nonexistent-id")
	if err == nil {
		t.Fatal("expected error when deleting non-existent profile")
	}
}

func TestBoardProfileRepository_ConfigJSONSerialization(t *testing.T) {
	database, cleanup := setupBoardProfileTestDB(t)
	defer cleanup()

	repo := db.NewBoardProfileRepository(database)

	profile := &db.BoardProfile{
		Name:        "config-test",
		DisplayName: "Config Test",
		Arch:        db.ArchAARCH64,
		Config: db.BoardConfig{
			DeviceTrees: []db.DeviceTreeSpec{
				{
					Source:   "arch/arm64/boot/dts/broadcom/bcm2711-rpi-4-b.dts",
					Overlays: []string{"arch/arm64/boot/dts/overlays/vc4-kms-v3d-overlay.dts"},
				},
			},
			KernelOverlay: map[string]string{
				"CONFIG_BCM2835_WDT": "y",
				"CONFIG_DRM_VC4":     "m",
			},
			KernelDefconfig: "bcm2711_defconfig",
			BootParams: db.BoardBootParams{
				BootloaderOverride: "uboot",
				UBootBoard:         "rpi_4",
				ExtraFiles: map[string]string{
					"/boot/cmdline.txt": "console=serial0,115200",
				},
				ConfigTxt: "arm_64bit=1\ndtoverlay=vc4-kms-v3d",
			},
			Firmware: []db.BoardFirmware{
				{
					Name:        "rpi-firmware",
					ComponentID: "fw-rpi",
					Path:        "/boot",
					Description: "Raspberry Pi firmware",
				},
			},
			KernelCmdline: "console=serial0,115200 console=tty1",
		},
		OwnerID: "user-1",
	}

	if err := repo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	found, err := repo.GetByID(profile.ID)
	if err != nil {
		t.Fatalf("failed to get profile: %v", err)
	}

	// Verify all config fields survived round-trip
	if len(found.Config.DeviceTrees) != 1 {
		t.Fatalf("expected 1 device tree, got %d", len(found.Config.DeviceTrees))
	}
	if found.Config.DeviceTrees[0].Source != "arch/arm64/boot/dts/broadcom/bcm2711-rpi-4-b.dts" {
		t.Fatalf("expected correct DT source, got '%s'", found.Config.DeviceTrees[0].Source)
	}
	if len(found.Config.DeviceTrees[0].Overlays) != 1 {
		t.Fatalf("expected 1 overlay, got %d", len(found.Config.DeviceTrees[0].Overlays))
	}
	if len(found.Config.KernelOverlay) != 2 {
		t.Fatalf("expected 2 kernel overlay entries, got %d", len(found.Config.KernelOverlay))
	}
	if found.Config.KernelOverlay["CONFIG_DRM_VC4"] != "m" {
		t.Fatalf("expected CONFIG_DRM_VC4=m, got '%s'", found.Config.KernelOverlay["CONFIG_DRM_VC4"])
	}
	if found.Config.KernelDefconfig != "bcm2711_defconfig" {
		t.Fatalf("expected bcm2711_defconfig, got '%s'", found.Config.KernelDefconfig)
	}
	if found.Config.BootParams.BootloaderOverride != "uboot" {
		t.Fatalf("expected uboot bootloader override, got '%s'", found.Config.BootParams.BootloaderOverride)
	}
	if found.Config.BootParams.UBootBoard != "rpi_4" {
		t.Fatalf("expected uboot board rpi_4, got '%s'", found.Config.BootParams.UBootBoard)
	}
	if found.Config.BootParams.ExtraFiles["/boot/cmdline.txt"] != "console=serial0,115200" {
		t.Fatal("expected extra files to be preserved")
	}
	if found.Config.BootParams.ConfigTxt != "arm_64bit=1\ndtoverlay=vc4-kms-v3d" {
		t.Fatalf("expected config.txt to be preserved, got '%s'", found.Config.BootParams.ConfigTxt)
	}
	if len(found.Config.Firmware) != 1 {
		t.Fatalf("expected 1 firmware entry, got %d", len(found.Config.Firmware))
	}
	if found.Config.Firmware[0].ComponentID != "fw-rpi" {
		t.Fatalf("expected firmware component ID 'fw-rpi', got '%s'", found.Config.Firmware[0].ComponentID)
	}
	if found.Config.KernelCmdline != "console=serial0,115200 console=tty1" {
		t.Fatalf("expected kernel cmdline to be preserved, got '%s'", found.Config.KernelCmdline)
	}
}

// =============================================================================
// Board Profile API Tests
// =============================================================================

func TestAPI_HandleBoardProfileList(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/board/profiles", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	count := int(response["count"].(float64))
	if count != 2 {
		t.Fatalf("expected 2 seeded profiles, got %d", count)
	}

	profiles, ok := response["profiles"].([]interface{})
	if !ok {
		t.Fatal("expected profiles array")
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
}

func TestAPI_HandleBoardProfileList_FilterByArch(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/board/profiles?arch=aarch64", nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	count := int(response["count"].(float64))
	if count != 1 {
		t.Fatalf("expected 1 aarch64 profile, got %d", count)
	}

	profiles := response["profiles"].([]interface{})
	profile := profiles[0].(map[string]interface{})
	if profile["arch"] != "aarch64" {
		t.Fatalf("expected arch aarch64, got %v", profile["arch"])
	}
}

func TestAPI_HandleBoardProfileGet(t *testing.T) {
	ta := setupTestAPI(t)

	// Get the seeded rpi4 profile
	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	rpi, err := boardProfileRepo.GetByName("rpi4")
	if err != nil {
		t.Fatalf("failed to get rpi4: %v", err)
	}

	rec := ta.makeRequest("GET", "/v1/board/profiles/"+rpi.ID, nil, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "rpi4" {
		t.Fatalf("expected name 'rpi4', got %v", response["name"])
	}
	if response["arch"] != "aarch64" {
		t.Fatalf("expected arch 'aarch64', got %v", response["arch"])
	}
}

func TestAPI_HandleBoardProfileGet_NotFound(t *testing.T) {
	ta := setupTestAPI(t)

	rec := ta.makeRequest("GET", "/v1/board/profiles/nonexistent-id", nil, "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileCreate(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "boardcreator", "boardcreator@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":         "jetson-nano",
		"display_name": "NVIDIA Jetson Nano",
		"description":  "Jetson Nano developer kit",
		"arch":         "aarch64",
		"config": map[string]interface{}{
			"kernel_defconfig": "tegra_defconfig",
			"kernel_overlay": map[string]string{
				"CONFIG_TEGRA_HOST1X": "y",
			},
		},
	}

	rec := ta.makeRequest("POST", "/v1/board/profiles", body, token)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["name"] != "jetson-nano" {
		t.Fatalf("expected name 'jetson-nano', got %v", response["name"])
	}
	if response["display_name"] != "NVIDIA Jetson Nano" {
		t.Fatalf("expected display_name 'NVIDIA Jetson Nano', got %v", response["display_name"])
	}
	if response["is_system"] != false {
		t.Fatal("expected is_system to be false for user-created profile")
	}
	if response["id"] == nil || response["id"] == "" {
		t.Fatal("expected id to be set")
	}
}

func TestAPI_HandleBoardProfileCreate_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	body := map[string]interface{}{
		"name":         "unauthorized-board",
		"display_name": "Unauthorized",
		"arch":         "x86_64",
	}

	rec := ta.makeRequest("POST", "/v1/board/profiles", body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileCreate_InvalidArch(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "boardbadarch", "boardbadarch@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":         "bad-arch-board",
		"display_name": "Bad Arch",
		"arch":         "mips",
	}

	rec := ta.makeRequest("POST", "/v1/board/profiles", body, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileCreate_DuplicateName(t *testing.T) {
	ta := setupTestAPI(t)

	_, token := ta.createTestUser(t, "boarddup", "boarddup@example.com", auth.RoleIDDeveloper)

	body := map[string]interface{}{
		"name":         "rpi4",
		"display_name": "Duplicate RPi4",
		"arch":         "aarch64",
	}

	rec := ta.makeRequest("POST", "/v1/board/profiles", body, token)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for duplicate name, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileUpdate(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "boardupdater", "boardupdater@example.com", auth.RoleIDDeveloper)

	// Create a user profile to update
	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	profile := &db.BoardProfile{
		Name:        "update-me",
		DisplayName: "Update Me",
		Arch:        db.ArchX86_64,
		OwnerID:     user.ID,
	}
	if err := boardProfileRepo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	body := map[string]interface{}{
		"display_name": "Updated Board",
		"description":  "Now with description",
	}

	rec := ta.makeRequest("PUT", "/v1/board/profiles/"+profile.ID, body, token)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]interface{}
	parseJSON(t, rec, &response)

	if response["display_name"] != "Updated Board" {
		t.Fatalf("expected display_name 'Updated Board', got %v", response["display_name"])
	}
	if response["description"] != "Now with description" {
		t.Fatalf("expected description 'Now with description', got %v", response["description"])
	}
}

func TestAPI_HandleBoardProfileUpdate_NotOwner(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "boardowner", "boardowner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "boardother", "boardother@example.com", auth.RoleIDDeveloper)

	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	profile := &db.BoardProfile{
		Name:        "owned-board",
		DisplayName: "Owned Board",
		Arch:        db.ArchX86_64,
		OwnerID:     owner.ID,
	}
	if err := boardProfileRepo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	body := map[string]interface{}{
		"display_name": "Hacked Name",
	}

	rec := ta.makeRequest("PUT", "/v1/board/profiles/"+profile.ID, body, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileUpdate_SystemProfileAsAdmin(t *testing.T) {
	ta := setupTestAPI(t)

	_, adminToken := ta.createTestUser(t, "boardadmin", "boardadmin@example.com", auth.RoleIDRoot)

	// Get seeded system profile
	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	rpi, err := boardProfileRepo.GetByName("rpi4")
	if err != nil {
		t.Fatalf("failed to get rpi4: %v", err)
	}

	body := map[string]interface{}{
		"description": "Updated by admin",
	}

	rec := ta.makeRequest("PUT", "/v1/board/profiles/"+rpi.ID, body, adminToken)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileUpdate_SystemProfileNonAdmin(t *testing.T) {
	ta := setupTestAPI(t)

	_, devToken := ta.createTestUser(t, "boarddev", "boarddev@example.com", auth.RoleIDDeveloper)

	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	rpi, err := boardProfileRepo.GetByName("rpi4")
	if err != nil {
		t.Fatalf("failed to get rpi4: %v", err)
	}

	body := map[string]interface{}{
		"description": "Should fail",
	}

	rec := ta.makeRequest("PUT", "/v1/board/profiles/"+rpi.ID, body, devToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileDelete(t *testing.T) {
	ta := setupTestAPI(t)

	user, token := ta.createTestUser(t, "boarddeleter", "boarddeleter@example.com", auth.RoleIDDeveloper)

	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	profile := &db.BoardProfile{
		Name:        "delete-me",
		DisplayName: "Delete Me",
		Arch:        db.ArchX86_64,
		OwnerID:     user.ID,
	}
	if err := boardProfileRepo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	rec := ta.makeRequest("DELETE", "/v1/board/profiles/"+profile.ID, nil, token)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify deletion
	found, err := boardProfileRepo.GetByID(profile.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Fatal("expected profile to be deleted")
	}
}

func TestAPI_HandleBoardProfileDelete_SystemProfile(t *testing.T) {
	ta := setupTestAPI(t)

	_, adminToken := ta.createTestUser(t, "boardadmindel", "boardadmindel@example.com", auth.RoleIDRoot)

	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	rpi, err := boardProfileRepo.GetByName("rpi4")
	if err != nil {
		t.Fatalf("failed to get rpi4: %v", err)
	}

	rec := ta.makeRequest("DELETE", "/v1/board/profiles/"+rpi.ID, nil, adminToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 for system profile deletion, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileDelete_NotOwner(t *testing.T) {
	ta := setupTestAPI(t)

	owner, _ := ta.createTestUser(t, "delowner", "delowner@example.com", auth.RoleIDDeveloper)
	_, otherToken := ta.createTestUser(t, "delother", "delother@example.com", auth.RoleIDDeveloper)

	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	profile := &db.BoardProfile{
		Name:        "not-yours",
		DisplayName: "Not Yours",
		Arch:        db.ArchX86_64,
		OwnerID:     owner.ID,
	}
	if err := boardProfileRepo.Create(profile); err != nil {
		t.Fatalf("failed to create profile: %v", err)
	}

	rec := ta.makeRequest("DELETE", "/v1/board/profiles/"+profile.ID, nil, otherToken)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPI_HandleBoardProfileDelete_Unauthorized(t *testing.T) {
	ta := setupTestAPI(t)

	boardProfileRepo := db.NewBoardProfileRepository(ta.database)
	rpi, err := boardProfileRepo.GetByName("rpi4")
	if err != nil {
		t.Fatalf("failed to get rpi4: %v", err)
	}

	rec := ta.makeRequest("DELETE", "/v1/board/profiles/"+rpi.ID, nil, "")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
