---
paths:
  - "src/common/**/*.go"
---

# Common Library Rules

## Purpose

`src/common/` provides shared Go packages used by both ldfd (server) and ldfctl (CLI). Changes here affect both projects.

## Packages

- `errors/` -- Structured error system: domain + code + HTTP status + wrapping. Used project-wide.
- `logs/` -- Charm Log wrapper. Three output modes: stdout, journald, auto. Supports levels: debug, info, warn, error.
- `cli/` -- Viper config initialization, env var binding, config file search paths.
- `paths/` -- Tilde (`~`) expansion and environment variable substitution.
- `version/` -- Build-time version injection via ldflags (release name, version, build date, git commit).

## Guidelines

- Keep packages minimal and well-scoped. Each package serves one purpose.
- Both ldfd and ldfctl use cobra/viper and charm log, but they don't share the same CLI options or behaviors. Common code is for shared mechanics (file lookup, parse logic), not application-specific behavior.
- Test changes in common/ against both ldfd and ldfctl builds.
