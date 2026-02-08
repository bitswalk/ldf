package migrations

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func migration018ToolchainComponents() Migration {
	return Migration{
		Version:     18,
		Description: "Seed toolchain components (GCC, LLVM, build essentials)",
		Up:          migration018Up,
	}
}

type toolchainComponent struct {
	Name                     string
	Categories               []string
	DisplayName              string
	Description              string
	ArtifactPattern          string
	DefaultURLTemplate       string
	GithubNormalizedTemplate string
}

func migration018Up(tx *sql.Tx) error {
	now := time.Now().UTC()

	components := []toolchainComponent{
		{
			Name:                     "gcc-native",
			Categories:               []string{"toolchain"},
			DisplayName:              "GCC (Native)",
			Description:              "GNU Compiler Collection for native builds",
			ArtifactPattern:          "gcc-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/gcc-{version}/gcc-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/releases/gcc-{version}.tar.gz",
		},
		{
			Name:                     "gcc-cross-aarch64",
			Categories:               []string{"toolchain"},
			DisplayName:              "GCC (Cross aarch64)",
			Description:              "GNU cross-compiler targeting aarch64-linux-gnu",
			ArtifactPattern:          "gcc-aarch64-linux-gnu-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/gcc-{version}/gcc-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/releases/gcc-{version}.tar.gz",
		},
		{
			Name:                     "llvm",
			Categories:               []string{"toolchain"},
			DisplayName:              "LLVM/Clang",
			Description:              "LLVM compiler infrastructure with Clang frontend",
			ArtifactPattern:          "llvm-project-{version}.src.tar.xz",
			DefaultURLTemplate:       "{base_url}/llvmorg-{version}/llvm-project-{version}.src.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/llvmorg-{version}.tar.gz",
		},
		{
			Name:                     "build-essentials",
			Categories:               []string{"toolchain"},
			DisplayName:              "Build Essentials",
			Description:              "Common build utilities (make, bc, flex, bison)",
			ArtifactPattern:          "build-essentials-{version}.tar.gz",
			DefaultURLTemplate:       "",
			GithubNormalizedTemplate: "",
		},
	}

	stmt, err := tx.Prepare(`
		INSERT INTO components (id, name, category, display_name, description, artifact_pattern,
			default_url_template, github_normalized_template, is_optional, is_system,
			is_kernel_module, is_userspace, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 1, 0, 1, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare toolchain component insert: %w", err)
	}
	defer stmt.Close()

	for _, c := range components {
		id := uuid.New().String()
		categoryStr := strings.Join(c.Categories, ",")
		if _, err := stmt.Exec(
			id,
			c.Name,
			categoryStr,
			c.DisplayName,
			c.Description,
			c.ArtifactPattern,
			c.DefaultURLTemplate,
			c.GithubNormalizedTemplate,
			now,
			now,
		); err != nil {
			return fmt.Errorf("failed to insert toolchain component %s: %w", c.Name, err)
		}
	}

	return nil
}
