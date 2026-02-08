package migrations

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func migration019ToolchainComponentsCross() Migration {
	return Migration{
		Version:     19,
		Description: "Add cross-compilation toolchain components (GCC x86_64, LLVM cross)",
		Up:          migration019Up,
	}
}

func migration019Up(tx *sql.Tx) error {
	now := time.Now().UTC()

	type component struct {
		Name                     string
		Categories               []string
		DisplayName              string
		Description              string
		ArtifactPattern          string
		DefaultURLTemplate       string
		GithubNormalizedTemplate string
	}

	components := []component{
		{
			Name:                     "gcc-cross-x86_64",
			Categories:               []string{"toolchain"},
			DisplayName:              "GCC (Cross x86_64)",
			Description:              "GNU cross-compiler targeting x86_64-linux-gnu",
			ArtifactPattern:          "gcc-x86_64-linux-gnu-{version}.tar.xz",
			DefaultURLTemplate:       "{base_url}/gcc-{version}/gcc-{version}.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/releases/gcc-{version}.tar.gz",
		},
		{
			Name:                     "llvm-cross-aarch64",
			Categories:               []string{"toolchain"},
			DisplayName:              "LLVM/Clang (Cross aarch64)",
			Description:              "LLVM compiler infrastructure with Clang frontend for aarch64 cross-compilation",
			ArtifactPattern:          "llvm-project-{version}.src.tar.xz",
			DefaultURLTemplate:       "{base_url}/llvmorg-{version}/llvm-project-{version}.src.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/llvmorg-{version}.tar.gz",
		},
		{
			Name:                     "llvm-cross-x86_64",
			Categories:               []string{"toolchain"},
			DisplayName:              "LLVM/Clang (Cross x86_64)",
			Description:              "LLVM compiler infrastructure with Clang frontend for x86_64 cross-compilation",
			ArtifactPattern:          "llvm-project-{version}.src.tar.xz",
			DefaultURLTemplate:       "{base_url}/llvmorg-{version}/llvm-project-{version}.src.tar.xz",
			GithubNormalizedTemplate: "{base_url}/archive/refs/tags/llvmorg-{version}.tar.gz",
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
