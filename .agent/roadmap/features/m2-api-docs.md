# M2 -- API Documentation & Developer Experience

**Priority**: High
**Status**: Not started
**Depends on**: M1 (partially, can overlap)

## Goal

Make the API discoverable, well-documented, and easy to integrate with through auto-generated OpenAPI specs and comprehensive project documentation.

## Tasks

### 2.1 OpenAPI 3.2 Specification
- [ ] Evaluate approach: code-gen (swaggo/swag) vs spec-first (manual YAML)
- [ ] Generate or write OpenAPI 3.2 spec for all v1 endpoints
- [ ] Include request/response schemas, auth requirements, error codes
- [ ] Version the spec under `src/api/openapi/` or equivalent
- **Files**: `src/ldfd/api/`, new spec files

### 2.2 Swagger UI Integration
- [ ] Add Swagger UI middleware to Gin
- [ ] Serve on `/swagger` route
- [ ] Wire OpenAPI spec to Swagger UI
- [ ] Ensure spec stays in sync with route changes (CI check or code-gen)
- **Files**: `src/ldfd/api/routes.go`, `src/ldfd/core/server.go`

### 2.3 Project Documentation
- [ ] Expand `docs/index.md` with getting-started guide
- [ ] Write architecture overview (`docs/architecture.md`)
- [ ] Write deployment guide (Docker, systemd, bare metal)
- [ ] Write configuration reference (all ldfd.yaml options)
- [ ] Document source management and forge integration
- [ ] Add developer contributing guide
- **Files**: `docs/`, `mkdocs.yml`

## Acceptance Criteria

- `/swagger` route serves interactive API documentation
- OpenAPI spec covers all v1 endpoints
- Documentation site has at least: getting started, architecture, deployment, config reference
