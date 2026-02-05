# M4 -- Build Engine Foundation

**Priority**: High (core value proposition)
**Status**: Not started
**Depends on**: M1, M3 (partially)

## Goal

Implement the core Linux distribution build pipeline that transforms component definitions into bootable artifacts. This is the project's reason to exist.

## Tasks

### 4.1 Build Orchestrator
- [ ] Design pipeline stages: resolve -> download -> prepare -> compile -> assemble -> package
- [ ] Implement `src/ldfd/build/` package with stage-based orchestration
- [ ] Implement dependency resolution between components
- [ ] Add build job tracking to database (new migration)
- [ ] Expose API: `POST /v1/distributions/{id}/build`, `GET /v1/builds/{id}`, `GET /v1/builds/{id}/logs`
- [ ] Add build progress streaming (SSE or WebSocket)
- **Files**: `src/ldfd/build/`, `src/ldfd/db/migrations/`, `src/ldfd/api/builds/`

### 4.2 Kernel Compilation
- [ ] Kernel source download and extraction
- [ ] Kernel config generation (defconfig + custom options)
- [ ] Kernel compilation with cross-compilation support
- [ ] x86_64 and AARCH64 target support
- [ ] Kernel module handling
- **Files**: `src/ldfd/build/kernel/`

### 4.3 Root Filesystem Assembly
- [ ] Base rootfs creation (directory skeleton)
- [ ] Init system installation (systemd, OpenRC)
- [ ] Filesystem creation (ext4, XFS, Btrfs)
- [ ] Bootloader installation (GRUB2, systemd-boot, UKI)
- [ ] Security framework setup (SELinux, AppArmor)
- **Files**: `src/ldfd/build/rootfs/`, `src/ldfd/build/components/`

### 4.4 Image Generation
- [ ] Disk image creation (raw, qcow2)
- [ ] ISO generation
- [ ] Initramfs generation
- [ ] Image signing and checksum generation
- [ ] Store final artifacts in configured storage backend
- **Files**: `src/ldfd/build/image/`

### 4.5 WebUI Build Integration
- [ ] Wire up existing "Build" button on distribution detail page
- [ ] Add build start dialog (arch/format selection)
- [ ] Add builds list view (on distribution detail or separate page)
- [ ] Add build detail view with stage progress
- [ ] Implement SSE log streaming display
- [ ] Add build cancel/retry actions
- **Files**: `src/webui/src/api/`, `src/webui/src/views/DistributionDetail/`, `src/webui/src/components/`

## Acceptance Criteria

- Can build a minimal bootable Linux image from a distribution definition
- Build progress is trackable via API and WebUI
- At least x86_64 target works end-to-end
- Artifacts are stored and downloadable via existing storage/artifact system
- WebUI allows starting, monitoring, and managing builds
