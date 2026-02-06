---
name: add-migration
description: Create a new database migration for ldfd with proper sequential numbering, function naming, and runner registration. Use when adding or modifying database tables.
argument-hint: "[description]"

---

# Add Migration

Create a new SQLite database migration following the project's established pattern.

## Arguments

- `$ARGUMENTS` -- Description of what the migration does (e.g., "add board_profiles table", "add status column to builds")

## Steps

### 1. Determine the next migration number

Read the existing migrations in `src/ldfd/db/migrations/` to find the highest number. The current latest is 013. The next migration should be the next sequential number (e.g., 014).

### 2. Create the migration file

Create `src/ldfd/db/migrations/<NNN>_<snake_case_description>.go`:

```go
package migrations

import (
	"database/sql"
)

func migration<NNN><CamelCaseDescription>() Migration {
	return Migration{
		Version:     <N>,  // integer without leading zeros
		Description: "<Human readable description>",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`<SQL statements>`)
			return err
		},
	}
}
```

For multiple SQL statements:

```go
Up: func(tx *sql.Tx) error {
    _, err := tx.Exec(`CREATE TABLE xxx (...)`)
    if err != nil {
        return err
    }

    _, err = tx.Exec(`CREATE INDEX idx_xxx_yyy ON xxx(yyy)`)
    if err != nil {
        return err
    }

    return nil
},
```

### 3. Register in runner.go

Edit `src/ldfd/db/migrations/runner.go` and add the new migration to the `registerAll()` method:

```go
func (r *Runner) registerAll() {
	r.migrations = []Migration{
		// ... existing migrations ...
		migration<NNN><CamelCaseDescription>(),  // <-- add this line
	}
	// ...
}
```

### 4. Verify

Run: `task build:srv` to confirm compilation.

## Naming conventions

- File: `<NNN>_<snake_case>.go` (e.g., `014_board_profiles.go`)
- Function: `migration<NNN><CamelCase>()` (e.g., `migration014BoardProfiles()`)
- Version field: plain integer, no leading zeros (e.g., `14`)
- Description: human-readable sentence (e.g., `"Create board_profiles table"`)

## Important rules

- Migrations are forward-only (no Down function)
- Migrations are immutable once committed -- never modify existing migrations
- Each migration runs inside a transaction (the `*sql.Tx` parameter)
- SQLite-specific: `ALTER TABLE` only supports `ADD COLUMN` and `RENAME`
- Use `INTEGER DEFAULT 0` for booleans in SQLite
- Use `TEXT` for strings, `INTEGER` for ints/booleans, `DATETIME` for timestamps
- Always include `DEFAULT` values for new columns added to existing tables
