# Current Priorities

**Active milestone**: M1 -- Stabilization & Polish
**Feature spec**: @.agent/roadmap/features/m1-stabilization.md

## Completed

- ~~**Fix Dockerfile** (M1.1)~~ -- Done on `feature/m1_1-1` (commit 0c144e8). Pending: container build validation.

## Next tasks (in order)

1. **WebUI i18n completion** (M1.2) -- Audit all views/components for hardcoded strings, complete locale JSON files in `src/webui/src/locales/{en,fr,de}/`.

2. **WebUI test setup** (M1.3) -- Set up Bun/Vitest test runner, add service layer unit tests, add `test:webui` to Taskfile.

3. **CI/CD hardening** (M1.4) -- Review `.github/workflows/build.yml`, ensure full pipeline: lint, fmt, test, build, artifact upload.

## Context for next session

- M1.1 (Dockerfile) done on branch `feature/m1_1-1`, not yet merged to main.
- M1.2 (i18n) is next. Requires auditing all WebUI views/components for hardcoded strings, then writing/completing locale JSON files for en, fr, de.
- The project has a working server (ldfd) and WebUI, but CLI is a stub and no build engine exists.
- All rules, feature specs, and project memory are in `.agent/` (not `.claude/`), per project convention.
