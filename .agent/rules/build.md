---
paths:
  - "Taskfile.yml"
  - "tools/**/*"
  - ".github/**/*"
  - "Dockerfile"
  - "go.mod"
  - "go.sum"
---

# Build & CI Rules

## Build System

- Taskfile (go-task) is the build orchestrator. See `Taskfile.yml` at project root.
- Release codename: Phoenix, version: 1.0.1
- Production builds strip debug symbols and use `-s -w` ldflags
- Dev builds include debug symbols and race detector

## Key Tasks

| Task | Purpose |
|------|---------|
| `task build` | Build all (server + CLI + webui) for production |
| `task build:dev` | Build all for development |
| `task build:srv` / `task build:srv:dev` | Server binary only |
| `task build:cli` / `task build:cli:dev` | CLI binary only |
| `task build:webui` | WebUI bundle via Bun + Vite |
| `task test` | Run all tests |
| `task test:srv` | Server tests only |
| `task clean` | Remove build artifacts |
| `task deps` / `task deps:webui` | Install Go / Bun dependencies |
| `task fmt` | Format Go code |
| `task lint` | Run golangci-lint |

## Output

- Binaries go to `build/bin/` (ldfd, ldfctl)
- WebUI bundle goes to `src/webui/dist/`

## Docker

- Dockerfile at `tools/docker/Dockerfile`
- Known issues: references `make build` instead of `task build`, Go version outdated (1.21 vs 1.24), port mismatch (8080 vs 8443)

## CI/CD

- GitHub Actions workflows in `.github/workflows/`
- `build.yml` -- build pipeline
- `documentation.yml` -- mkdocs deployment
