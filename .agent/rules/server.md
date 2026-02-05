---
paths:
  - "src/ldfd/**/*.go"
  - "src/common/**/*.go"
---

# Server Rules (ldfd)

## Architecture

- Gin HTTP framework on port 8443 (default)
- SQLite database with sequential immutable migrations in `src/ldfd/db/migrations/`
- JWT authentication with refresh tokens and RBAC (roles: root, admin, standard)
- Handler pattern: routes in `api/routes.go`, handlers in `api/<domain>/`, types in `api/<domain>/types.go`
- Repository pattern for database access in `db/`
- Pluggable storage backends in `storage/` (local, S3)

## Conventions

- All API routes are versioned under `/v1/`
- Root `/` exposes discovery info (version, health, endpoints)
- Target OpenAPI 3.2 compatibility
- Swagger UI should be served on `/swagger` (not yet implemented)
- Use structured errors from `src/common/errors/` with domain + code
- Log via `src/common/logs/` -- supports stdout, journald, or auto-detect
- Config via Viper -- lookup order: `/etc/ldf/ldfd.yaml`, `/opt/ldf/ldfd.yaml`, `~/.config/ldf/ldfd.yaml`

## Shared Code

Before writing new utility code in `src/ldfd/`, check `src/common/` first. Shared packages:
- `common/errors` -- structured error types, HTTP mapping, error codes
- `common/logs` -- charm log wrapper with output mode selection
- `common/cli` -- Viper config init, env var binding, file search
- `common/paths` -- tilde expansion, env substitution
- `common/version` -- build-time version info injection

## Testing

- Tests live in `src/ldfd/tests/`
- Use Go standard `testing` package
- Run with `task test:srv`
