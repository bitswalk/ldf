# Architecture

## System Overview

```
                          +------------------+
                          |     WebUI         |
                          | (SolidJS SPA)     |
                          +--------+---------+
                                   |
                                   | HTTP/JSON
                                   |
+------------------+      +--------+---------+      +------------------+
|    ldfctl        +----->+     ldfd          +----->+   Storage        |
|  (CLI client)    | HTTP |  (API server)     |      | Local / S3      |
+------------------+      +----+----+----+---+      +------------------+
                               |    |    |
                    +----------+    |    +----------+
                    |               |               |
             +------+---+   +------+---+   +-------+------+
             | SQLite   |   | Download |   | Forge        |
             | Database |   | Manager  |   | Registry     |
             +----------+   +----------+   +--------------+
```

LDF follows an API-first design. The server (`ldfd`) is a pure REST API that returns JSON. The WebUI and CLI client are separate consumers of this API.

## Repository Layout

```
ldf/
  src/
    ldfd/           # API server (Go, Gin)
      api/          # HTTP handlers (11 modules)
      core/         # Server bootstrap, config, middleware
      auth/         # JWT service, user management
      db/           # SQLite database, migrations, repositories
      download/     # Download manager, version discovery
      forge/        # Forge detection, version filtering
      storage/      # Storage backends (local, S3)
      docs/         # Generated OpenAPI spec
    webui/          # Web interface (SolidJS, Bun, Vite)
    ldfctl/         # CLI client (stub)
    common/         # Shared Go libraries
      cli/          # Cobra/Viper helpers
      logs/         # Logging abstraction
      paths/        # Path expansion utilities
      version/      # Version info struct
  tools/
    docker/         # Dockerfile
  docs/             # MkDocs documentation (this site)
```

## Server Internals

### Gin Router and Middleware

ldfd uses the [Gin](https://gin-gonic.com/) web framework. The middleware stack processes requests in this order:

1. **Recovery** -- Catches panics and returns 500
2. **CORS** -- Handles cross-origin requests for WebUI
3. **Logging** -- Logs request method, path, status, and latency

### API Modules

The API is organized into 11 handler modules under `src/ldfd/api/`:

| Module | Prefix | Endpoints | Description |
|--------|--------|-----------|-------------|
| base | `/`, `/v1/` | 3 | Root discovery, health, version |
| auth | `/auth/` | 5 | Login, logout, create, refresh, validate |
| roles | `/v1/roles` | 5 | Role CRUD for RBAC |
| distributions | `/v1/distributions` | 8 | Distribution lifecycle management |
| components | `/v1/components` | 12 | Component catalog and versions |
| sources | `/v1/sources` | 11 | Upstream source management |
| downloads | `/v1/distributions/{id}/downloads` | 7 | Download job management |
| artifacts | `/v1/distributions/{id}/artifacts` | 7 | Artifact upload, download, storage |
| settings | `/v1/settings` | 4 | Server configuration (root only) |
| forge | `/v1/forge` | 4 | Forge detection and filter preview |
| branding | `/v1/branding` | 4 | Logo and favicon management |
| langpacks | `/v1/language-packs` | 4 | Custom language pack management |

Total: **74 operations** across **52 URL paths**.

### Authentication

ldfd uses JWT (JSON Web Tokens) for authentication:

- **Login** returns an access token and a refresh token
- Access tokens are short-lived; refresh tokens allow obtaining new access tokens
- Tokens are passed via the `Authorization: Bearer <token>` header
- The `X-Subject-Token` header is used in login/refresh responses
- Role-based access control (RBAC): endpoints require either authenticated access or root-level access

### Download Manager

The download manager handles fetching component sources from upstream:

- Worker pool with configurable concurrency
- HTTP and Git download support
- Automatic retry on failure
- Checksum verification
- Progress tracking per job
- Jobs are persisted in the database

### Forge Registry

The forge registry detects repository hosting platforms and extracts version information:

- Supports GitHub, GitLab, Gitea, Codeberg, Forgejo, and generic HTTP directories
- Auto-detects forge type from repository URL
- Discovers available versions (tags/releases) from upstream
- Provides default URL templates and version filters per forge type

### Storage Backends

Artifact storage is pluggable:

- **Local** -- Files stored on the local filesystem (default: `~/.ldfd/artifacts`)
- **S3** -- S3-compatible object storage with support for AWS S3, MinIO, GarageHQ, and generic providers

See [Configuration](configuration.md) for storage setup details.

## Database

ldfd uses an **in-memory SQLite** database with optional persistence:

- On startup, the database is loaded from disk (if a persist file exists)
- On shutdown, the database is saved to disk
- Migrations run automatically on startup (`src/ldfd/db/migrations/`)
- The persist path is configurable via `database.path`

### Key Tables

| Table | Purpose |
|-------|---------|
| distributions | Distribution definitions and config |
| components | Component catalog entries |
| upstream_sources | Source definitions (system + user) |
| source_versions | Discovered upstream versions |
| download_jobs | Download job state and progress |
| users | User accounts |
| roles | RBAC role definitions |
| settings | Runtime configuration overrides |
| language_packs | Custom UI translations |

## WebUI

The WebUI is a single-page application built with:

- **SolidJS** -- Reactive UI framework
- **Bun** -- JavaScript runtime and package manager
- **Vite 7** -- Build tool and dev server
- **TailwindCSS 4** -- Utility-first CSS
- **Phosphor Icons** -- Icon set
- **Departure Mono** -- Monospace font

The WebUI communicates with ldfd over HTTP/JSON. It is not embedded in the server binary -- it runs as a separate static asset served by ldfd or a reverse proxy.

### Internationalization

The WebUI supports multiple languages via a translation system:

- Built-in locales: English, French, German
- Custom language packs can be uploaded via the `/v1/language-packs` API
- Language packs are `.tar.xz`, `.tar.gz`, or `.xz` archives containing JSON translation files
