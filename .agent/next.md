# Current Priorities

**Active milestone**: M1 -- Stabilization & Polish
**Feature spec**: @.agent/roadmap/features/m1-stabilization.md

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1` (commit 0c144e8). Pending: container build validation.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2` (commit 808086d). Replaced all hardcoded strings in Pagination, DownloadStatus, Modal, Badge, Register, ServerSettingsPanel with `t()` calls. Updated EN/FR/DE locale files (common.json, auth.json, settings.json).
- ~~**WebUI test setup** (M1.3)~~ -- Done on `feature/m1_1-3` (commit 91d6b5a). Vitest + jsdom, 117 unit tests across 7 test files covering services (downloads, forge, storage, settings, branding) and utilities (globFilter, sorting). `task test:webui` integrated. Component/integration tests remain open.

## Next tasks (in order)

1. **CI/CD hardening** (M1.4) -- Review `.github/workflows/build.yml`, ensure full pipeline: lint, fmt, test (server + webui), build, artifact upload. Verify documentation workflow.

## Context for next session

- M1.1, M1.2, and M1.3 are merged to main.
- M1.3 partial: component tests (Login, Distribution, Components) and integration tests (auth flow) are deferred. The test runner and service-layer tests are in place.
- M1.4 (CI/CD) is next. Requires reviewing GitHub Actions workflows, adding webui test step, ensuring build artifacts are uploaded.
- Bun is at `/home/flint/.bun/bin/bun` (not in default PATH). Taskfile uses bare `bun` which works when bun is in PATH.
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
