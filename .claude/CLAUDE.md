# Linux Distribution Factory (LDF)

A modern, modular alternative to YoctoProject and BuildRoot for creating custom Linux distributions.

## Project Structure

Monorepo with four main components:

- `src/ldfd/` -- Go REST API server (core), Gin framework, SQLite, JWT auth
- `src/webui/` -- SolidJS SPA, Bun, Vite, TailwindCSS 4.x
- `src/ldfctl/` -- Go CLI client, Cobra, Viper
- `src/common/` -- Shared Go libraries (errors, logs, config, paths, version)

## Tech Stack

| Component | Stack |
|-----------|-------|
| Server | Go 1.24, Gin, SQLite, JWT, Cobra, Viper, Charm Log |
| WebUI | SolidJS, TypeScript, Bun, Vite 7, TailwindCSS 4, Phosphor Icons, Departure Mono font |
| CLI | Go 1.24, Cobra, Viper (shared with server via `src/common/`) |
| Build | Taskfile (go-task), Docker, GitHub Actions |
| Docs | MkDocs Material |

## Branch Workflow

### Branch Types

- **Release branches**: `release/<version>` (e.g., `release/1.1.0`) -- integration branch for a release cycle
- **Feature branches**: `feature/m<milestone>_<subtask>` (e.g., M5.1 -> `feature/m5_1`) -- for milestone work
- **Bugfix branches**: `bugfix/<issue-number>-<short-description>` (e.g., `bugfix/46-setting-list-decode`) -- for bug fixes
- **Fix branches**: `fix/<short-description>` -- for non-issue-tracked fixes (CI, lint, etc.)

### Release Cycle

Each release has a unique one-word codename (e.g., v1.0.0 = Phoenix).

1. **Plan**: Create the milestone on the roadmap; identify and create issues for all work items.
2. **Branch**: Use `/create-release <version> <codename>` to create `release/<version>` from `main` with version bump.
3. **Develop**: Create feature/bugfix/fix branches **from the release branch** (not from `main`).
4. **Integrate**: Merge completed branches back **into the release branch** via PR or merge.
5. **Stabilize**: Test the release branch as a whole; fix any integration issues.
6. **Ship**: Use `/create-release <version> --finalize` to merge to `main`, tag `v<version>`, and push.
7. **Automate**: The `v*` tag push triggers `.github/workflows/release.yml` which builds, packages, and publishes the GitHub release.

### General Rules

- When a release branch exists for your milestone, branch from it and merge back to it -- **not** `main`.
- When no release branch exists (hotfixes, standalone work), branch from and merge to `main` as before.
- **Never** switch to a new branch without merging or stashing the current one first.

## Environment Notes

- **Bun** is at `/home/flint/.bun/bin/bun` (not in default PATH)
- **swag** is at `~/go/bin/swag`
- Token stored at `~/.ldfctl/token.json`

## Common Commands

```bash
task build              # Build all (production)
task build:dev          # Build all (dev, debug symbols + race detector)
task build:srv          # Server binary only
task build:cli          # CLI binary only
task build:webui        # WebUI bundle via Bun + Vite
task test               # All tests
task test:srv           # Server tests only
task test:cli           # CLI tests only
task fmt                # Format Go code
task lint               # Run golangci-lint
task deps               # Install Go deps
task deps:webui         # Install Bun deps
cd src/webui && /home/flint/.bun/bin/bun run dev  # WebUI dev server (port 3000)
```

## Key Architecture Decisions

- API-first: ldfd is a pure REST API on port 8443, returns JSON only
- WebUI is a separate SPA that talks to ldfd, not embedded in the binary
- Storage is pluggable: local filesystem or S3-compatible (AWS, MinIO, GarageHQ)
- Forge detection supports GitHub, GitLab, Gitea, Codeberg, Forgejo, HTTP dirs
- Database migrations are sequential and immutable (`src/ldfd/db/migrations/`)
- Common code in `src/common/` is shared between ldfd and ldfctl -- check there before duplicating
- WebUI never uses `<div>`/`<span>` -- semantic HTML5 tags only
- Swagger UI served at `/swagger/index.html` (swaggo/swag annotations on all 74 operations)

## Project Management

- **GitHub Project**: [LDF Roadmap](https://github.com/orgs/bitswalk/projects/9) -- linked to `bitswalk/ldf`
- **GitHub Milestones**: M1-M4 (closed), M5-M7 (open) -- each with subtask issues
- **GitHub Issues**: All work items tracked as issues with labels and milestones
- **GitHub Wiki**: Available for long-form documentation and guides
- **Labels**: `priority:` (critical/high/medium/low), `component:` (server/cli/webui/build/common), `type:` (feature/infrastructure/security)

### Workflow

- When starting a new task, find or create the corresponding GitHub issue
- Reference issues in commit messages (e.g., `fixes #35`)
- Use `gh issue list --milestone "M5 - Platform Maturity"` to see current work
- Use `gh project item-list 9 --owner bitswalk` to see the roadmap board
- Close issues when work is merged to main
- Use the `/project-manager` skill when planning work or deciding what to work on next
