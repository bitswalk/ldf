# Current Priorities

**Active milestone**: M4 -- Build Engine Foundation -- in progress
**Feature spec**: @.agent/roadmap/features/m4-build-engine.md
**Plan**: @~/.claude/plans/modular-orbiting-clock.md

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1`.
- ~~**WebUI i18n completion** (M1.2)~~ -- Done on `feature/m1_1-2`.
- ~~**WebUI test setup** (M1.3)~~ -- Done on `feature/m1_1-3`. 117 unit tests. Component/integration tests deferred.
- ~~**CI/CD hardening** (M1.4)~~ -- Done on `feature/m1_1-4`. Pending: verify green after push.
- ~~**OpenAPI spec + Swagger UI** (M2.1 + M2.2)~~ -- Done on `feature/m2_1-1`. swaggo/swag annotations on all 74 operations (52 paths, 84 definitions). Swagger UI at `/swagger/index.html`. `docs:api` Taskfile task added.
- ~~**Project Documentation** (M2.3)~~ -- Done on `feature/m2_2-3`. 6 doc pages: index (rewrite), getting-started, architecture, configuration, deployment, sources. Deleted outdated structure.md. Added nav to mkdocs.yml.
- ~~**CLI Foundation** (M3.1)~~ -- Done on `feature/m3_1`. Cobra root command, HTTP API client, token storage, auth commands (login/logout/whoami), version command, resource commands (distribution, component, source, download, artifact, setting), table/JSON output formatting. 20 new files, 2553 lines. Separate CLI ldflags in Taskfile.yml. Fixed .gitignore `output/` rule. Updated AGENT.md branch naming convention.
- ~~**Core CLI Commands** (M3.2)~~ -- Done on `feature/m3_2`. Complete CLI coverage for all 71 ldfd API endpoints. Added 4 new resource groups: role (list/get/create/update/delete), forge (detect/preview-filter/types/filters), branding (get/info/upload/delete), langpack (list/get/upload/delete). Added health command. Extended existing resources: distribution (+logs/stats/deletion-preview), component (+categories/versions/resolve-version/--category), source (+versions/sync-status/clear-versions), download (+active), artifact (+url/storage-status/list-all), setting (+reset-db). 9 new files, 1537 lines added.
- ~~**CLI Advanced Features** (M3.3)~~ -- Done on `feature/m3_3`. YAML output support (goccy/go-yaml, all 61 handlers). Query parameter flags: --limit/--offset on list commands, --status on distribution list, --version-type on component/source versions. Composite release commands: create, configure (22 config flags for kernel/init/fs/security/runtime/target), show. Shell completion: ValidArgsFunction for distribution/component/source/role IDs, flag completions for --output/--status/--visibility/--category. Improved APIError with hints for 401/403/404/409. 3 new files, 1158 lines added.
- ~~**CLI Testing** (M3.4)~~ -- Done on `feature/m3_4`. 88 unit tests across 4 packages: client (21 tests: ListOptions, APIError, HTTP methods with httptest mock server), output (11 tests: PrintJSON/YAML/Table/Message/Error), config (5 tests: token save/load/clear, JSON serialization), cmd (51 tests: command registration, aliases, arg validation, flags, mock server execution, error handling, output formats). Coverage: output 89.7%, config 36.4%, cmd 22.8%, client 16.0%. Added test-cli job to CI pipeline (gated on by build job). Added test:cli to Taskfile.yml main test task. 4 new test files, 1490 lines added.
- ~~**Build Orchestrator** (M4.1)~~ -- Done on `feature/m4_1`. Stage-based build pipeline architecture mirroring download manager pattern. Build manager with configurable worker pool (default: 1), dispatcher polling every 10s. Build worker with sequential stage processing. Container executor wrapping Podman CLI. Database migration 012 (build_jobs, build_stages, build_logs tables). BuildJobRepository with full CRUD, status transitions, stage tracking, log management. 8 API endpoints: start build, list builds, get build with stages, query/stream logs (SSE), cancel, retry, list active (admin). CLI commands: build start/get/list/logs/cancel/retry/active with --arch/--format flags. Also fixed: nil interface panic in distribution delete (Go nil-interface pitfall), nil logger panics across 8 packages (initialized with logs.NewDefault()), .gitignore build/ pattern. 14 new files, 10 modified files, 2607 lines added.
- ~~**Kernel Compilation** (M4.2)~~ -- Done on `feature/m4_2`. First 4 build pipeline stages: Resolve (parse config, resolve component versions, map to download artifacts), Download check (verify downloads complete, check artifact existence), Prepare (create workspace, extract tar.gz/bz2/xz archives, generate kernel .config, build scripts), Compile (Podman container execution, x86_64/aarch64 cross-compilation, progress tracking). Kernel config generator with 3 modes: defconfig (arch default), options (defconfig + custom CONFIG_ options), custom (user-provided .config from storage). Recommended kernel options based on distribution config (filesystems, init system, security, virtualization, containers). Added KernelConfigMode type and extended KernelConfig struct. 5 new files, 4 modified, 1677 lines added.

## Next tasks

**M4.2 is complete.** Next subtask: **M4.3 -- Root Filesystem Assembly** (branch `feature/m4_3`).

M4.3 implements the assemble stage:
- Rootfs skeleton helpers (FHS directory structure, fstab, os-release)
- Component installers (init system: systemd/OpenRC, bootloader: GRUB2/systemd-boot, security: SELinux/AppArmor)
- Initramfs generator (kernel modules, /init script, cpio archive)
- Assemble stage orchestrating all rootfs assembly steps

Remaining after M4.3:
- M4.4 -- Image Generation (raw/qcow2/ISO image generators, package stage, end-to-end verification)
- M4.5 -- WebUI Build Integration (wire up build button, build dialog, builds list, build detail view, SSE log streaming)

## Context for next session

- M1, M2, M3 are fully complete and merged to main.
- M4 (Build Engine Foundation) is in progress: M4.1-M4.2 done, M4.3-M4.5 remaining.
- Main is ahead of origin (unpushed).
- Branch naming convention: `feature/m<milestone>_<subtask>` (e.g., M4.2 -> `feature/m4_2`).
- Build package at `src/ldfd/build/` -- note `.gitignore` was fixed to use `/build/` (top-level only).
- Build pipeline stages implemented: resolve, download, prepare, compile (4 of 6).
- Kernel config modes: defconfig, options (with ConfigOptions map[string]string), custom (CustomConfigPath to storage).
- Kernel config generator produces recommended options based on filesystem type, init system, security framework, virtualization, and container runtime.
- Stage interface: `Name() BuildStageName`, `Validate(StageContext) error`, `Execute(ctx, StageContext, ProgressFunc) error`.
- StageContext carries BuildID, Config, TargetArch, ImageFormat, workspace paths, LogWriter, resolved components.
- Build manager calls RegisterDefaultStages() on startup to register resolve/download/prepare/compile stages.
- Container executor at `build/container.go` wraps `podman run --rm` with mounts, env, stdout/stderr streams.
- Compile stage generates build script based on config mode, runs inside container, parses progress from output.
- Pre-existing test failures: ~10 source-related API tests return 404 (routing issue predating M4).
- CLI binary builds with `task build:cli` or `task build:cli:dev`.
- Token stored at `~/.ldfctl/token.json`, config at `~/.ldfctl/ldfctl.yaml`.
- Swagger UI at `/swagger/index.html`, MkDocs site at `ldf.bitswalk.com`.
- Bun is at `/home/flint/.bun/bin/bun` (not in default PATH).
- swag binary at `~/go/bin/swag`.
