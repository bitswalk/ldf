# Current Priorities

**Active milestone**: M2 complete. Next: M3 -- CLI Client (ldfctl)
**Feature spec**: @.agent/roadmap/roadmap.md (M3 section)

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1`.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2`.
- ~~**WebUI test setup** (M1.3)~~ -- Done on `feature/m1_1-3`. 117 unit tests. Component/integration tests deferred.
- ~~**CI/CD hardening** (M1.4)~~ -- Done on `feature/m1_1-4`. Pending: verify green after push.
- ~~**OpenAPI spec + Swagger UI** (M2.1 + M2.2)~~ -- Done on `feature/m2_1-1`. swaggo/swag annotations on all 74 operations (52 paths, 84 definitions). Swagger UI at `/swagger/index.html`. `docs:api` Taskfile task added.
- ~~**Project Documentation** (M2.3)~~ -- Done on `feature/m2_2-3`. 6 doc pages: index (rewrite), getting-started, architecture, configuration, deployment, sources. Deleted outdated structure.md. Added nav to mkdocs.yml.

## Next milestone

**M3 -- CLI Client (ldfctl)** -- Implement CLI commands for all API operations.

### M3 Sub-tasks (from roadmap)

- M3.1: CLI Foundation -- Cobra root command, Viper config, auth commands, output formatting
- M3.2: Core CLI Commands -- distribution, component, source, artifact, download, settings CRUD
- M3.3: CLI Advanced Features -- composite commands, shell completion, output formats
- M3.4: CLI Testing -- unit + integration tests

## Context for next session

- M1 and M2 are fully complete and merged to main.
- Main is ahead of origin (unpushed).
- Swagger UI at `/swagger/index.html`, MkDocs site at `ldf.bitswalk.com`.
- OpenAPI spec regeneration: `task docs:api`.
- Bun is at `/home/flint/.bun/bin/bun` (not in default PATH).
- swag binary at `~/go/bin/swag`.
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
