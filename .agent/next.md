# Current Priorities

**Active milestone**: M3 -- CLI Client (ldfctl) -- in progress
**Feature spec**: @.agent/roadmap/roadmap.md (M3 section)

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1`.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2`.
- ~~**WebUI test setup** (M1.3)~~ -- Done on `feature/m1_1-3`. 117 unit tests. Component/integration tests deferred.
- ~~**CI/CD hardening** (M1.4)~~ -- Done on `feature/m1_1-4`. Pending: verify green after push.
- ~~**OpenAPI spec + Swagger UI** (M2.1 + M2.2)~~ -- Done on `feature/m2_1-1`. swaggo/swag annotations on all 74 operations (52 paths, 84 definitions). Swagger UI at `/swagger/index.html`. `docs:api` Taskfile task added.
- ~~**Project Documentation** (M2.3)~~ -- Done on `feature/m2_2-3`. 6 doc pages: index (rewrite), getting-started, architecture, configuration, deployment, sources. Deleted outdated structure.md. Added nav to mkdocs.yml.
- ~~**CLI Foundation** (M3.1)~~ -- Done on `feature/m3_1`. Cobra root command, HTTP API client, token storage, auth commands (login/logout/whoami), version command, resource commands (distribution, component, source, download, artifact, setting), table/JSON output formatting. 20 new files, 2553 lines. Separate CLI ldflags in Taskfile.yml. Fixed .gitignore `output/` rule. Updated AGENT.md branch naming convention.
- ~~**Core CLI Commands** (M3.2)~~ -- Done on `feature/m3_2`. Complete CLI coverage for all 71 ldfd API endpoints. Added 4 new resource groups: role (list/get/create/update/delete), forge (detect/preview-filter/types/filters), branding (get/info/upload/delete), langpack (list/get/upload/delete). Added health command. Extended existing resources: distribution (+logs/stats/deletion-preview), component (+categories/versions/resolve-version/--category), source (+versions/sync-status/clear-versions), download (+active), artifact (+url/storage-status/list-all), setting (+reset-db). 9 new files, 1537 lines added.

## Next tasks

**M3.3: CLI Advanced Features**:
- Composite commands (create release, configure release)
- Shell completion (Bash, Zsh, Fish) via Cobra built-in
- `--output yaml` format support
- Add `--filter`, `--sort`, `--limit` flags where applicable
- Improve error messages and user feedback

**M3.4: CLI Testing**:
- Unit tests for command parsing
- Integration tests against test server
- Add to CI pipeline via `test:cli` Taskfile task

## Context for next session

- M1 and M2 are fully complete and merged to main.
- M3.1 (CLI Foundation) and M3.2 (Core CLI Commands) are complete and merged to main.
- Main is ahead of origin (unpushed).
- Branch naming convention updated: `feature/m<milestone>_<subtask>` (e.g., M3.2 -> `feature/m3_2`).
- CLI binary builds with `task build:cli` or `task build:cli:dev`.
- CLI ldflags target `src/ldfctl/internal/cmd` package variables.
- Token stored at `~/.ldfctl/token.json`, config at `~/.ldfctl/ldfctl.yaml`.
- Config env prefix: `LDFCTL_`, search paths: `/etc/ldfctl`, `~/.ldfctl`.
- `golang.org/x/term` added as dependency for password prompts.
- Swagger UI at `/swagger/index.html`, MkDocs site at `ldf.bitswalk.com`.
- Bun is at `/home/flint/.bun/bin/bun` (not in default PATH).
- swag binary at `~/go/bin/swag`.
