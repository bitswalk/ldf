package migrations

import (
	"database/sql"
)

func migration014BoardProfiles() Migration {
	return Migration{
		Version:     14,
		Description: "Add board_profiles table with default profiles",
		Up: func(tx *sql.Tx) error {
			// Create board_profiles table
			_, err := tx.Exec(`
				CREATE TABLE board_profiles (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL UNIQUE,
					display_name TEXT NOT NULL,
					description TEXT DEFAULT '',
					arch TEXT NOT NULL,
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
			_, err = tx.Exec(`CREATE INDEX idx_board_profiles_name ON board_profiles(name)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_board_profiles_arch ON board_profiles(arch)`)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`CREATE INDEX idx_board_profiles_system ON board_profiles(is_system)`)
			if err != nil {
				return err
			}

			// Seed default profile: generic-x86_64
			_, err = tx.Exec(`
				INSERT INTO board_profiles (id, name, display_name, description, arch, config, is_system, owner_id)
				VALUES (
					'bp-generic-x86_64',
					'generic-x86_64',
					'Generic x86_64',
					'Generic profile for x86_64/amd64 systems with UEFI boot support',
					'x86_64',
					'{"kernel_overlay":{"CONFIG_EFI":"y","CONFIG_EFI_STUB":"y","CONFIG_FB_EFI":"y","CONFIG_FRAMEBUFFER_CONSOLE":"y","CONFIG_ACPI":"y","CONFIG_X86_ACPI_CPUFREQ":"y","CONFIG_CPU_FREQ_GOV_ONDEMAND":"y","CONFIG_PCIEPORTBUS":"y","CONFIG_HOTPLUG_PCI":"y","CONFIG_VIRTIO":"m","CONFIG_VIRTIO_PCI":"m","CONFIG_VIRTIO_BLK":"m","CONFIG_VIRTIO_NET":"m"},"kernel_cmdline":"console=tty0 console=ttyS0,115200"}',
					1,
					''
				)
			`)
			if err != nil {
				return err
			}

			// Seed default profile: rpi4
			_, err = tx.Exec(`
				INSERT INTO board_profiles (id, name, display_name, description, arch, config, is_system, owner_id)
				VALUES (
					'bp-rpi4',
					'rpi4',
					'Raspberry Pi 4 Model B',
					'Raspberry Pi 4 Model B (BCM2711, Cortex-A72, 1-8GB RAM)',
					'aarch64',
					'{"device_trees":[{"source":"arch/arm64/boot/dts/broadcom/bcm2711-rpi-4-b.dts","overlays":["arch/arm64/boot/dts/overlays/vc4-kms-v3d-pi4-overlay.dts"]}],"kernel_overlay":{"CONFIG_ARCH_BCM2835":"y","CONFIG_BCM2835_WDT":"y","CONFIG_DRM_VC4":"m","CONFIG_SND_BCM2835_SOC_I2S":"m","CONFIG_MMC_BCM2835":"y","CONFIG_SERIAL_8250_BCM2835AUX":"y","CONFIG_USB_DWC2":"m","CONFIG_USB_XHCI_PCI":"y","CONFIG_BRCMFMAC":"m","CONFIG_BT_HCIUART_BCM":"y","CONFIG_I2C_BCM2835":"y","CONFIG_SPI_BCM2835":"y","CONFIG_GPIO_BCM_VIRT":"y","CONFIG_THERMAL_BCM2835":"y"},"kernel_defconfig":"bcm2711_defconfig","boot_params":{"config_txt":"# Raspberry Pi 4 boot configuration\narm_64bit=1\ndtoverlay=vc4-kms-v3d-pi4\ndisable_overscan=1\ngpu_mem=256\nenable_uart=1\n"},"firmware":[{"name":"rpi-firmware","path":"/boot","description":"Raspberry Pi boot firmware (start4.elf, fixup4.dat, bootcode.bin)"}],"kernel_cmdline":"console=serial0,115200 console=tty1 root=/dev/mmcblk0p2 rootfstype=ext4 rootwait"}',
					1,
					''
				)
			`)
			return err
		},
	}
}
