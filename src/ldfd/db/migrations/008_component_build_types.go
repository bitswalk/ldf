package migrations

import (
	"database/sql"
)

func migration008ComponentBuildTypes() Migration {
	return Migration{
		Version:     8,
		Description: "Add is_kernel_module and is_userspace columns to components",
		Up: func(tx *sql.Tx) error {
			// Add is_kernel_module column - indicates if component requires kernel configuration
			// Default to false since most components are userspace-only
			_, err := tx.Exec(`ALTER TABLE components ADD COLUMN is_kernel_module BOOLEAN NOT NULL DEFAULT 0`)
			if err != nil {
				return err
			}

			// Add is_userspace column - indicates if component needs to be built as userspace binary
			// Default to true since most components are userspace tools
			_, err = tx.Exec(`ALTER TABLE components ADD COLUMN is_userspace BOOLEAN NOT NULL DEFAULT 1`)
			if err != nil {
				return err
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

			// Create indexes for efficient querying by build type
			_, err = tx.Exec(`CREATE INDEX idx_components_kernel_module ON components(is_kernel_module)`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`CREATE INDEX idx_components_userspace ON components(is_userspace)`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
