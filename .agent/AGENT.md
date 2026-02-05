# Linux Distribution Factory (LDF)

A modern, modular alternative to YoctoProject and BuildRoot for creating custom Linux distributions.

## Project Structure

Monorepo with three main components:

- `src/ldfd/` -- Go REST API server (core), Gin framework, SQLite, JWT auth
- `src/webui/` -- SolidJS SPA, Bun, Vite, TailwindCSS 4.x
- `src/ldfctl/` -- Go CLI client (stub, not yet implemented)
- `src/common/` -- Shared Go libraries (errors, logs, config, paths, version)

## Tech Stack

| Component | Stack |
|-----------|-------|
| Server | Go 1.24, Gin, SQLite, JWT, Cobra, Viper, Charm Log |
| WebUI | SolidJS, TypeScript, Bun, Vite 7, TailwindCSS 4, Phosphor Icons, Departure Mono font |
| CLI | Go, Cobra, Viper (shared with server via `src/common/`) |
| Build | Taskfile (go-task), Docker, GitHub Actions |
| Docs | MkDocs Material |

## Environment Notes

- **Bun** is installed at `/home/flint/.bun/bin/bun` (not in default PATH). Use full path or `export PATH="$HOME/.bun/bin:$PATH"` before running bun commands.

## Common Commands

```bash
# Build everything (production)
task build

# Build dev (with debug symbols)
task build:dev

# Run server (dev)
task run:srv:dev

# Run tests
task test          # all
task test:srv      # server only

# WebUI dev server (port 3000)
cd src/webui && bun run dev

# Format & lint
task fmt
task lint

# Install dependencies
task deps           # Go
task deps:webui     # Bun
```

## Key Architecture Decisions

- API-first: ldfd is a pure REST API on port 8443, returns JSON only
- WebUI is a separate SPA that talks to ldfd, not embedded in the binary (yet)
- Storage is pluggable: local filesystem or S3-compatible (AWS, MinIO, GarageHQ)
- Forge detection supports GitHub, GitLab, Gitea, Codeberg, Forgejo, HTTP dirs
- Database migrations are sequential and immutable (`src/ldfd/db/migrations/`)
- Common code in `src/common/` is shared between ldfd and ldfctl -- check there before duplicating
- WebUI never uses `<div>`/`<span>` -- semantic HTML5 tags only

## Project State

- **Server (ldfd)**: Production-ready. 50+ endpoints, full CRUD, auth, RBAC, downloads, storage, forge detection. Well-tested (150+ test cases).
- **WebUI**: ~90% complete. 9 functional views, auth flow, i18n infrastructure (translation files incomplete).
- **CLI (ldfctl)**: Empty stub. Not implemented.
- **Build engine**: Does not exist. LDF currently manages distribution metadata and artifacts, not actual compilation/assembly.

## Roadmap & Planning

- Roadmap: @.agent/roadmap/roadmap.md
- Feature specs: @.agent/roadmap/features/
- Current priorities: @.agent/next.md
- Scoped rules: @.agent/rules/
