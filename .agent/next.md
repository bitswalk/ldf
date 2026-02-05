# Current Priorities

**Active milestone**: M1 -- Stabilization & Polish (nearly complete)
**Feature spec**: @.agent/roadmap/features/m1-stabilization.md

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1` (commit 0c144e8). Pending: container build validation.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2` (commit 808086d). All hardcoded strings replaced with `t()` calls. EN/FR/DE locale files updated.
- ~~**WebUI test setup** (M1.3)~~ -- Done on `feature/m1_1-3` (commit 91d6b5a). Vitest + jsdom, 117 unit tests across 7 files. Component/integration tests deferred.
- ~~**CI/CD hardening** (M1.4)~~ -- Done on `feature/m1_1-4` (commit bd7b182). `build.yml` (lint, fmt, test-server, test-webui, build + artifacts) and `documentation.yml` (MkDocs gh-deploy). Pending: verify green after push.

## M1 Remaining

- M1.1: Validate Docker container build
- M1.3: Component tests and integration tests (deferred)
- M1.4: Verify CI runs green after push to origin

## Next milestone

**M2 -- API Documentation** -- Generate OpenAPI/Swagger docs from ldfd endpoints, serve via MkDocs or standalone.

## Context for next session

- All M1 tasks (M1.1-M1.4) are merged to main. Main is 11 commits ahead of origin.
- Push to origin will trigger the new CI workflows for the first time.
- Bun is at `/home/flint/.bun/bin/bun` (not in default PATH).
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
