# M3 -- CLI Client (ldfctl)

**Priority**: High
**Status**: Not started (empty stub exists)
**Depends on**: M1

## Goal

Implement a fully functional CLI client that communicates with the ldfd API, enabling scriptable and automated workflows.

## Tasks

### 3.1 CLI Foundation
- [ ] Implement Cobra root command with global flags (--server, --token, --output, --config)
- [ ] Implement Viper config loading (reuse `src/common/cli/config.go`)
- [ ] Implement `ldfctl login` / `ldfctl logout` / `ldfctl whoami`
- [ ] Implement JSON/table output formatting
- [ ] Implement HTTP client for ldfd API communication
- **Files**: `src/ldfctl/`, `src/common/cli/`

### 3.2 Core Commands
- [ ] `ldfctl distribution list|create|get|update|delete`
- [ ] `ldfctl component list|create|get|update|delete`
- [ ] `ldfctl source list|create|get|update|delete|sync`
- [ ] `ldfctl artifact list|upload|download|delete`
- [ ] `ldfctl download list|create|cancel|retry`
- [ ] `ldfctl settings get|set`
- **Files**: `src/ldfctl/internal/cmd/`

### 3.3 Advanced Features
- [ ] `ldfctl create release` -- composite distribution release workflow
- [ ] `ldfctl configure release` -- set components for a distribution release
- [ ] `ldfctl build distribution` -- trigger build (when M4 exists)
- [ ] Shell completion (Bash, Zsh, Fish) via Cobra
- [ ] `--output json|table|yaml` for all commands
- **Files**: `src/ldfctl/internal/cmd/`

### 3.4 Testing
- [ ] Unit tests for command parsing and flag validation
- [ ] Integration tests against a test ldfd instance
- [ ] Wire into `task test:cli`
- **Files**: `src/ldfctl/tests/`, `Taskfile.yml`

## Acceptance Criteria

- All CRUD operations available via CLI
- Auth flow works (login, token storage, auto-refresh)
- Output formats: json, table, yaml
- Shell completion works for Bash/Zsh/Fish
- Tests pass in CI
