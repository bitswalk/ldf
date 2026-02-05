# M1 -- Stabilization & Polish

**Priority**: Critical
**Status**: In progress
**Depends on**: Nothing

## Goal

Harden existing features, fix known issues, bring everything to production quality before adding new functionality.

## Tasks

### 1.1 Fix Dockerfile
- [x] Replace `make build` with `task build` in `tools/docker/Dockerfile`
- [x] Update base Go image from 1.21 to 1.24
- [x] Align exposed port to 8443 (or make configurable)
- [x] Fix binary paths (`build/bin/` output), config path (`docs/samples/ldfd.yml`), and CMD (`ldfd`)
- [x] Add go-task and bun installation in builder stage
- [x] Add WebUI dist copy to runtime stage
- [ ] Validate multi-stage build produces working container
- **Files**: `tools/docker/Dockerfile`

### 1.2 WebUI i18n Completion
- [x] Audit all components and views for hardcoded strings
- [x] Complete English locale files in `src/webui/src/locales/en/`
- [x] Complete French locale files in `src/webui/src/locales/fr/`
- [x] Complete German locale files in `src/webui/src/locales/de/`
- [x] Test locale switching end-to-end
- **Files**: `src/webui/src/locales/`, `src/webui/src/services/i18n.ts`, all views and components

### 1.3 WebUI Testing
- [x] Set up test runner (Bun test or Vitest)
- [x] Add unit tests for service layer (`src/webui/src/services/*.ts`)
- [ ] Add component tests for critical views (Login, Distribution, Components)
- [ ] Add integration tests for auth flow
- [x] Add `test:webui` task to `Taskfile.yml`
- **Files**: `src/webui/`, `Taskfile.yml`

### 1.4 CI/CD Hardening
- [x] Review and complete `.github/workflows/build.yml`
- [x] Pipeline must run: lint, fmt, test (server + webui), build all
- [x] Add artifact upload for built binaries
- [x] Verify `.github/workflows/documentation.yml` deploys mkdocs
- [ ] Verify workflows run green after push to origin
- **Files**: `.github/workflows/`

## Acceptance Criteria

- Docker build succeeds and produces a running container
- All WebUI text is translated (no raw i18n keys visible)
- WebUI test suite exists and passes
- CI pipeline runs green on push to main
