package migrations

import (
	"database/sql"
)

func migration017ToolchainProfiles() Migration {
	return Migration{
		Version:     17,
		Description: "Add toolchain_profiles table with default profiles",
		Up: func(tx *sql.Tx) error {
			// Create toolchain_profiles table
			_, err := tx.Exec(`
				CREATE TABLE toolchain_profiles (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL UNIQUE,
					display_name TEXT NOT NULL,
					description TEXT DEFAULT '',
					type TEXT NOT NULL,
					config TEXT NOT NULL DEFAULT '{}',
					is_system INTEGER NOT NULL DEFAULT 0,
					owner_id TEXT DEFAULT '',
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				return err
			}

			// Create indexes
			_, err = tx.Exec(`CREATE INDEX idx_toolchain_profiles_name ON toolchain_profiles(name)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_toolchain_profiles_type ON toolchain_profiles(type)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_toolchain_profiles_system ON toolchain_profiles(is_system)`)
			if err != nil {
				return err
			}

			// Seed default profile: GCC (Native)
			_, err = tx.Exec(`
				INSERT INTO toolchain_profiles (id, name, display_name, description, type, config, is_system, owner_id)
				VALUES (
					'tp-gcc-native',
					'gcc-native',
					'GCC (Native)',
					'GNU Compiler Collection for native builds',
					'gcc',
					'{}',
					1,
					''
				)
			`)
			if err != nil {
				return err
			}

			// Seed default profile: LLVM/Clang (Native)
			_, err = tx.Exec(`
				INSERT INTO toolchain_profiles (id, name, display_name, description, type, config, is_system, owner_id)
				VALUES (
					'tp-llvm-native',
					'llvm-native',
					'LLVM/Clang (Native)',
					'LLVM compiler infrastructure with Clang frontend for native builds',
					'llvm',
					'{}',
					1,
					''
				)
			`)
			return err
		},
	}
}
