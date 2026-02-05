# M5 -- Platform Maturity

**Priority**: Medium
**Status**: Not started
**Depends on**: M4

## Goal

Bring the platform to enterprise-grade maturity with hardware profiles, cross-compilation, caching, and security hardening.

## Tasks

### 5.1 Board Profiles & Device Trees
- [ ] Design board profile schema (YAML-based)
- [ ] Implement board profile registry and API CRUD
- [ ] Add device tree compilation support
- [ ] Board-specific kernel config overlays
- [ ] Default profiles: generic-x86_64, Raspberry Pi 4
- **Files**: `src/ldfd/api/boards/`, `src/ldfd/build/board/`

### 5.2 Multi-Architecture
- [ ] Cross-compilation toolchain management
- [ ] x86_64 host -> AARCH64 target (and vice versa)
- [ ] QEMU-based testing for cross-built images
- [ ] Platform-specific package selection
- **Files**: `src/ldfd/build/platform/`

### 5.3 Advanced Download & Caching
- [ ] Artifact caching layer to avoid redundant downloads
- [ ] Mirror/proxy support for air-gapped environments
- [ ] Bandwidth throttling and scheduling
- [ ] Download deduplication across distributions
- **Files**: `src/ldfd/download/`, `src/ldfd/storage/`

### 5.4 Security Hardening
- [ ] API rate limiting
- [ ] Audit logging for all write operations
- [ ] HTTPS/TLS support for ldfd
- [ ] Token rotation policies
- [ ] Secrets management for S3 credentials and API tokens
- **Files**: `src/ldfd/api/middleware.go`, `src/ldfd/core/server.go`, `src/ldfd/auth/`

## Acceptance Criteria

- Can target at least 2 board profiles with device tree support
- Cross-compilation works for AARCH64 from x86_64 host
- Downloads are cached and deduplicated
- TLS works natively without a reverse proxy
