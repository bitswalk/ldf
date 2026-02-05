# Current Priorities

**Active milestone**: M2 -- API Documentation & Developer Experience
**Feature spec**: @.agent/roadmap/features/m2-api-docs.md

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1`.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2`.
- ~~**WebUI test setup** (M1.3)~~ -- Done on `feature/m1_1-3`. 117 unit tests. Component/integration tests deferred.
- ~~**CI/CD hardening** (M1.4)~~ -- Done on `feature/m1_1-4`. Pending: verify green after push.
- ~~**OpenAPI spec + Swagger UI** (M2.1 + M2.2)~~ -- Done on `feature/m2_1-1`. swaggo/swag annotations on all 74 operations (52 paths, 84 definitions). Swagger UI at `/swagger/index.html`. `docs:api` Taskfile task added.

## M2 Remaining

- **M2.3: Project Documentation** -- Expand `docs/` with getting-started, architecture, deployment, configuration, and sources guides. Update `mkdocs.yml` nav.

## Next milestone

**M3 -- CLI Client (ldfctl)** -- Implement CLI commands for all API operations.

## Context for next session

- M2.1 + M2.2 are merged to main. Main is ahead of origin (unpushed).
- Swagger UI is available at `/swagger/index.html` when ldfd runs.
- OpenAPI spec regeneration: `task docs:api` (requires `swag` in PATH or `~/go/bin/swag`).
- Generated spec files: `src/ldfd/docs/` (docs.go, swagger.json, swagger.yaml).
- Bun is at `/home/flint/.bun/bin/bun` (not in default PATH).
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
