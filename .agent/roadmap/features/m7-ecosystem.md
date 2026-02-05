# M7 -- Ecosystem & Distribution

**Priority**: Low (future)
**Status**: Not started
**Depends on**: M4, M5

## Goal

Package, distribute, and grow the project ecosystem.

## Tasks

### 7.1 Packaging & Distribution
- [ ] Release automation (GitHub Releases with goreleaser)
- [ ] Packages: .deb, .rpm, .apk
- [ ] Container images to GHCR / Docker Hub
- [ ] Homebrew formula
- [ ] One-liner install script
- **Files**: `.goreleaser.yml`, `tools/scripts/`, `.github/workflows/`

### 7.2 Plugin System
- [ ] Design plugin interface for custom components and build steps
- [ ] Plugin discovery and loading
- [ ] Plugin development guide
- **Files**: `src/ldfd/plugin/`, `docs/`

### 7.3 WebUI Embedding
- [ ] Embed compiled WebUI into ldfd binary (Go embed)
- [ ] Serve on configurable route from ldfd
- [ ] Alternative: `ldfctl --wui` starts local web server
- **Files**: `src/ldfd/core/server.go`, `src/webui/dist/`

### 7.4 Telemetry & Monitoring
- [ ] Prometheus metrics endpoint (`/metrics`)
- [ ] Health check enhancements
- [ ] Build duration and success rate tracking
- [ ] Optional opt-in usage telemetry
- **Files**: `src/ldfd/api/`, `src/ldfd/core/`

## Acceptance Criteria

- Binaries available as GitHub Releases with multi-platform packages
- Plugin interface documented and at least one example plugin exists
- WebUI servable directly from ldfd binary
