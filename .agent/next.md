# Current Priorities

**Active milestone**: M1 -- Stabilization & Polish
**Feature spec**: @.agent/roadmap/features/m1-stabilization.md

## Next tasks (in order)

1. **Fix Dockerfile** (M1.1) -- Update `tools/docker/Dockerfile`: replace `make build` with `task build`, bump Go 1.21 to 1.24, fix port 8080 to 8443.

2. **WebUI i18n completion** (M1.2) -- Audit all views/components for hardcoded strings, complete locale JSON files in `src/webui/src/locales/{en,fr,de}/`.

3. **WebUI test setup** (M1.3) -- Set up Bun/Vitest test runner, add service layer unit tests, add `test:webui` to Taskfile.

4. **CI/CD hardening** (M1.4) -- Review `.github/workflows/build.yml`, ensure full pipeline: lint, fmt, test, build, artifact upload.

## Context for next session

- Roadmap was written on 2026-02-05 based on commit b773279 (main branch, clean state).
- No code changes have been made yet -- only `.agent/` documentation files created.
- The project has a working server (ldfd) and WebUI, but CLI is a stub and no build engine exists.
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
