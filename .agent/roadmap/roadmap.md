# LDF Project Roadmap

**Project**: Linux Distribution Factory (LDF)
**Release Codename**: Phoenix
**Current Version**: 1.0.1
**Date**: 2026-02-05
**Status**: Active Development

---

## Executive Summary

LDF is a monorepo comprising three main components: a Go REST API server (`ldfd`), a SolidJS WebUI, and a Go CLI client (`ldfctl`). The API server and WebUI are functional and feature-rich. The CLI client is a stub. The actual Linux distribution build engine (kernel compilation, rootfs generation, image creation) does not exist yet -- the platform currently manages distribution metadata, component sources, artifact downloads, and storage.

This roadmap organizes remaining work into prioritized milestones, from stabilization of what exists today through to the long-term vision of a fully operational Linux distribution factory.

---

## Current State Assessment

### What Works

| Component | Status | Coverage |
|-----------|--------|----------|
| REST API (ldfd) | Production-ready | 50+ endpoints, full CRUD, auth, RBAC |
| Authentication | Complete | JWT, refresh tokens, role-based access |
| Download Manager | Complete | HTTP/Git downloads, retry, verification, worker pool |
| Forge Detection | Complete | GitHub, GitLab, Gitea, Codeberg, Forgejo, HTTP dirs |
| Storage Backends | Complete | Local filesystem + S3 (AWS, MinIO, GarageHQ) |
| Source Management | Complete | Unified system/user sources, version discovery, sync |
| WebUI | ~90% complete | 9 functional views, auth flow, settings, theming |
| Common Libraries | Complete | Errors, logging, config, paths, version |
| Build System | Working | Taskfile with dev/prod targets |
| Backend Tests | Good | 150+ test cases across API, auth, storage, downloads |

### What Is Missing or Incomplete

| Component | Status | Impact |
|-----------|--------|--------|
| CLI Client (ldfctl) | Empty stub | No CLI interface for operators |
| OpenAPI/Swagger | Not implemented | No auto-generated API docs |
| i18n Translation Files | Missing | WebUI shows raw keys instead of translated text |
| WebUI Tests | None | No frontend test coverage |
| Documentation | ~20% | Minimal content, missing guides and API reference |
| Dockerfile | Outdated | References `make build` instead of `task build` |
| Build Engine | Not started | No kernel compilation, rootfs, or image generation |
| TUI Client | Not started | Terminal UI mentioned in README but not begun |
| Board Profiles | Not started | Hardware-specific configurations not implemented |

---

## Milestone 1 -- Stabilization & Polish

**Goal**: Harden the existing platform, fix known issues, and bring all existing features to production quality.

**Priority**: Critical

### 1.1 Fix Dockerfile

- Update `Dockerfile` to use `task build` instead of `make build`
- Update base Go image from 1.21 to 1.24 to match `go.mod`
- Align exposed port (8080 in Dockerfile vs 8443 in ldfd default)
- Validate multi-stage build produces working container

### 1.2 WebUI i18n Completion

- Create/complete English locale files under `src/webui/src/locales/en/`
- Create/complete French locale files under `src/webui/src/locales/fr/`
- Create/complete German locale files under `src/webui/src/locales/de/`
- Audit all UI components for hardcoded strings and replace with i18n keys
- Test locale switching end-to-end

### 1.3 WebUI Testing

- Set up test runner (Bun test or Vitest via Vite)
- Add unit tests for service layer (`services/*.ts`)
- Add component tests for critical views (Login, Distribution, Components)
- Add integration tests for auth flow and API interactions
- Integrate frontend tests into `Taskfile.yml` (`test:webui` task)

### 1.4 CI/CD Hardening

- Review and complete `.github/workflows/build.yml`
- Ensure pipeline runs: lint, fmt, test (server + CLI + webui), build all
- Add artifact upload for built binaries
- Add workflow for documentation deployment

---

## Milestone 2 -- API Documentation & Developer Experience

**Goal**: Make the API discoverable, well-documented, and easy to integrate with.

**Priority**: High

### 2.1 OpenAPI 3.2 Specification

- Generate or write OpenAPI 3.2 spec for all v1 endpoints
- Evaluate approach: code-generation (swaggo/swag) vs spec-first (manual YAML)
- Include request/response schemas, auth requirements, error codes
- Version the spec file under `src/api/openapi/`

### 2.2 Swagger UI Integration

- Serve Swagger UI on `/swagger` route as stated in project rules
- Wire OpenAPI spec to Swagger UI middleware
- Ensure spec stays in sync with route changes

### 2.3 Project Documentation

- Expand `docs/index.md` with getting-started guide
- Write architecture overview (`docs/architecture.md`)
- Write API reference (auto-generated from OpenAPI spec)
- Write deployment guide (Docker, systemd, bare metal)
- Write configuration reference (all ldfd.yaml and ldfctl.yaml options)
- Document source management and forge integration
- Add developer contributing guide (`CONTRIBUTING.md`)

---

## Milestone 3 -- CLI Client (ldfctl)

**Goal**: Implement a fully functional CLI client that communicates with the ldfd API.

**Priority**: High

### 3.1 CLI Foundation

- Implement Cobra root command with shared flags (server URL, auth token, output format)
- Implement Viper config loading (reuse `src/common/cli/config.go`)
- Implement auth commands: `ldfctl login`, `ldfctl logout`, `ldfctl whoami`
- Implement JSON/table output formatting

### 3.2 Core CLI Commands

- `ldfctl distribution list|create|get|update|delete`
- `ldfctl component list|create|get|update|delete`
- `ldfctl source list|create|get|update|delete|sync`
- `ldfctl artifact list|upload|download|delete`
- `ldfctl download list|create|cancel|retry`
- `ldfctl settings get|set`

### 3.3 CLI Advanced Features

- `ldfctl create release` -- composite command for distribution release workflow
- `ldfctl configure release` -- set components for a distribution release
- `ldfctl build distribution` -- trigger build pipeline (when build engine exists)
- Shell completion (Bash, Zsh, Fish) via Cobra built-in
- `--output json|table|yaml` formatting for all commands

### 3.4 CLI Testing

- Unit tests for command parsing and flag validation
- Integration tests against a test ldfd server
- Add to CI pipeline via `test:cli` Taskfile task

---

## Milestone 4 -- Build Engine Foundation

**Goal**: Implement the core Linux distribution build pipeline that transforms component definitions into bootable artifacts.

**Priority**: High (this is the project's reason to exist)

### 4.1 Build Orchestrator

- Design build pipeline stages: resolve -> download -> prepare -> compile -> assemble -> package
- Implement `src/ldfd/build/` package with stage-based orchestration
- Implement dependency resolution between components
- Add build job tracking to database (new migration)
- Expose build API endpoints: `POST /v1/distributions/{id}/build`, `GET /v1/builds/{id}`

### 4.2 Kernel Compilation

- Implement kernel source download and extraction
- Implement kernel config generation (defconfig + custom options)
- Implement kernel compilation with cross-compilation support
- Support x86_64 and AARCH64 targets
- Implement kernel module handling

### 4.3 Root Filesystem Assembly

- Implement base rootfs creation (directory skeleton)
- Implement init system installation (systemd, OpenRC)
- Implement filesystem creation (ext4, XFS, Btrfs)
- Implement bootloader installation (GRUB2, systemd-boot, UKI)
- Implement security framework setup (SELinux, AppArmor)

### 4.4 Image Generation

- Implement disk image creation (raw, qcow2)
- Implement ISO generation
- Implement initramfs generation
- Implement image signing and checksum generation
- Store final artifacts in configured storage backend

---

## Milestone 5 -- Platform Maturity

**Goal**: Bring the platform to a mature, production-grade state.

**Priority**: Medium

### 5.1 Board Profiles & Device Trees

- Design board profile schema (YAML-based)
- Implement board profile registry and management
- Add device tree compilation support
- Add board-specific kernel config overlays
- Provide default profiles: generic-x86_64, Raspberry Pi 4, etc.
- Expose API endpoints for board profile CRUD

### 5.2 Multi-Architecture Support

- Implement cross-compilation toolchain management
- Support x86_64 host building AARCH64 targets (and vice versa)
- Add QEMU-based testing for cross-built images
- Platform-specific package selection and configuration

### 5.3 Advanced Download & Caching

- Implement artifact caching layer to avoid redundant downloads
- Implement mirror/proxy support for air-gapped environments
- Add bandwidth throttling and scheduling
- Implement download deduplication across distributions

### 5.4 Security Hardening

- Add API rate limiting
- Implement audit logging for all write operations
- Add HTTPS/TLS support for ldfd (currently HTTP only)
- Implement token rotation policies
- Add secrets management for S3 credentials and API tokens

---

## Milestone 6 -- TUI Client

**Goal**: Implement the Terminal User Interface using Bubble Tea as referenced in the README.

**Priority**: Medium

### 6.1 TUI Foundation

- Set up Bubble Tea framework in `src/ldfctl/internal/tui/`
- Implement navigation model (main menu, views, modals)
- Implement API client integration (reuse ldfctl HTTP client)
- Implement auth flow in TUI

### 6.2 TUI Views

- Dashboard: overview of distributions, recent builds, system status
- Distribution list/detail/create/edit views
- Component browser with version selection
- Source management views
- Build progress with real-time log streaming
- Settings panel

### 6.3 TUI Polish

- Keyboard shortcuts and help overlay
- Theme support (match WebUI theming)
- Responsive layout for different terminal sizes
- Integrate as `ldfctl --tui` entry point

---

## Milestone 7 -- Ecosystem & Distribution

**Goal**: Package, distribute, and grow the project ecosystem.

**Priority**: Low (future)

### 7.1 Packaging & Distribution

- Create release automation (GitHub Releases with goreleaser)
- Build packages for major distributions: .deb, .rpm, .apk
- Publish container images to GitHub Container Registry / Docker Hub
- Homebrew formula for macOS users
- Provide `install.sh` one-liner script

### 7.2 Plugin System

- Design plugin interface for custom components and build steps
- Implement plugin discovery and loading
- Document plugin development guide

### 7.3 WebUI as Embedded Asset

- Embed compiled WebUI into ldfd binary (Go embed)
- Serve WebUI directly from ldfd on a configurable route
- Alternatively, support `ldfctl --wui` starting a local web server

### 7.4 Telemetry & Monitoring

- Add Prometheus metrics endpoint (`/metrics`)
- Implement health check endpoint enhancements
- Add build duration and success rate tracking
- Optional usage telemetry (opt-in)

---

## Priority Matrix

| Priority | Milestone | Effort | Business Value |
|----------|-----------|--------|----------------|
| Critical | M1: Stabilization & Polish | Low | Fixes blockers, production readiness |
| High | M2: API Documentation | Medium | Developer onboarding, API adoption |
| High | M3: CLI Client | Medium | Operator tooling, automation |
| High | M4: Build Engine | Very High | Core value proposition |
| Medium | M5: Platform Maturity | High | Enterprise features |
| Medium | M6: TUI Client | Medium | Developer experience |
| Low | M7: Ecosystem | Medium | Community growth |

---

## Recommended Execution Order

```
M1 (Stabilization) ──> M2 (API Docs) ──> M3 (CLI) ──> M4 (Build Engine)
                                                              |
                                                              v
                                          M5 (Platform Maturity) ──> M7 (Ecosystem)
                                                              |
                                                              v
                                                      M6 (TUI Client)
```

M1 and M2 can partially overlap. M3 can start while M2 is finishing. M4 is the largest body of work and is the critical path. M5, M6, and M7 can be parallelized once M4 foundations are in place.

---

## Notes

- This roadmap reflects the state of the codebase as of 2026-02-05 (commit b773279 on main).
- The build engine (M4) is the single most important milestone -- it transforms LDF from a distribution metadata manager into an actual distribution factory.
- The CLI (M3) is a prerequisite for scriptable/automated workflows and should be delivered before or alongside the build engine.
- Each milestone should be tracked as a GitHub milestone with individual issues for each task.
