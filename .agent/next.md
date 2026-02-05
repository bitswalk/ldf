# Current Priorities

**Active milestone**: M1 -- Stabilization & Polish
**Feature spec**: @.agent/roadmap/features/m1-stabilization.md

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1` (commit 0c144e8). Pending: container build validation.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2` (commit 808086d). Replaced all hardcoded strings in Pagination, DownloadStatus, Modal, Badge, Register, ServerSettingsPanel with `t()` calls. Updated EN/FR/DE locale files (common.json, auth.json, settings.json).

## Next tasks (in order)

1. **WebUI test setup** (M1.3) -- Set up Bun/Vitest test runner, add service layer unit tests, add `test:webui` to Taskfile.

2. **CI/CD hardening** (M1.4) -- Review `.github/workflows/build.yml`, ensure full pipeline: lint, fmt, test, build, artifact upload.

## Context for next session

- M1.1 and M1.2 are merged to main.
- M1.3 (testing) is next. Requires setting up a test runner (Vitest preferred with SolidJS), adding unit tests for services, component tests for critical views, and a `test:webui` Taskfile target.
- Bun is the package manager but may not be available in all environments -- ensure Taskfile handles this.
- The project has a working server (ldfd) and WebUI, but CLI is a stub and no build engine exists.
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
