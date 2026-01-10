package migrations

import (
	"database/sql"
	"strings"
)

func migration008ComponentBuildTypes() Migration {
	return Migration{
		Version:     8,
		Description: "Add is_kernel_module and is_userspace columns to components",
		Up: func(tx *sql.Tx) error {
			// Check if columns already exist (for fresh installs where migration 001 includes them)
			hasKernelModule, err := columnExists(tx, "components", "is_kernel_module")
			if err != nil {
				return err
			}
			hasUserspace, err := columnExists(tx, "components", "is_userspace")
			if err != nil {
				return err
			}

			// Add is_kernel_module column if it doesn't exist
			if !hasKernelModule {
				_, err = tx.Exec(`ALTER TABLE components ADD COLUMN is_kernel_module BOOLEAN NOT NULL DEFAULT 0`)
				if err != nil {
					return err
				}
			}

			// Add is_userspace column if it doesn't exist
			if !hasUserspace {
				_, err = tx.Exec(`ALTER TABLE components ADD COLUMN is_userspace BOOLEAN NOT NULL DEFAULT 1`)
				if err != nil {
					return err
				}
			}

			// Update kernel - it's kernel-only, not userspace
			_, err = tx.Exec(`UPDATE components SET is_kernel_module = 1, is_userspace = 0 WHERE name = 'kernel'`)
			if err != nil {
				return err
			}

			// Update filesystem components - they have both kernel drivers and userspace tools
			_, err = tx.Exec(`UPDATE components SET is_kernel_module = 1 WHERE name IN ('btrfs', 'xfs', 'ext4', 'f2fs', 'zfs')`)
			if err != nil {
				return err
			}

			// Update security components - they have kernel LSM modules and userspace tools
			_, err = tx.Exec(`UPDATE components SET is_kernel_module = 1 WHERE name IN ('selinux', 'apparmor')`)
			if err != nil {
				return err
			}

			// Update virtualization components that require KVM kernel module
			_, err = tx.Exec(`UPDATE components SET is_kernel_module = 1 WHERE name = 'qemu-kvm-libvirt'`)
			if err != nil {
				return err
			}

			// Create indexes if they don't exist (use IF NOT EXISTS for safety)
			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_components_kernel_module ON components(is_kernel_module)`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_components_userspace ON components(is_userspace)`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

// columnExists checks if a column exists in a table
func columnExists(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}
