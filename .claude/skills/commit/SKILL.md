---
name: commit
description: Format code, run linters and relevant tests, then create a conventional commit. Use after completing a piece of work to commit it cleanly.
argument-hint: "[optional commit message]"

---

# Commit

Run pre-commit checks and create a well-formatted conventional commit.

## Steps

### 1. Format and lint

```bash
task fmt
task lint
```

If lint fails, fix the issues before proceeding.

### 2. Run relevant tests

Determine which tests to run based on changed files:

- Changes in `src/ldfd/` -> `task test:srv`
- Changes in `src/ldfctl/` -> `task test:cli`
- Changes in `src/webui/` -> `cd src/webui && /home/flint/.bun/bin/bun test`
- Changes in `src/common/` -> `task test:srv && task test:cli` (affects both)

If tests fail, stop and fix before committing.

### 3. Stage changes

```bash
git add -A
```

Review what's staged with `git diff --cached --stat`.

### 4. Generate commit message

If `$ARGUMENTS` is provided, use it as the commit message.

Otherwise, generate a conventional commit message based on the staged changes:

**Format**: `<type>(<scope>): <description>`

**Types**:
- `feat` -- New feature
- `fix` -- Bug fix
- `refactor` -- Code refactoring
- `docs` -- Documentation only
- `test` -- Adding or updating tests
- `ci` -- CI/CD changes
- `build` -- Build system changes
- `chore` -- Other changes

**Scope** (inferred from changed files):
- `src/ldfd/api/` -> `api`
- `src/ldfd/db/` -> `db`
- `src/ldfd/build/` -> `build`
- `src/ldfd/` (other) -> `server`
- `src/ldfctl/` -> `cli`
- `src/webui/` -> `webui`
- `src/common/` -> `common`
- `docs/` -> `docs`
- `.github/` -> `ci`
- `Taskfile.yml` -> `build`
- Multiple scopes -> use the primary one, or omit scope

**Description**: Imperative mood, lowercase, no period. Max 72 chars.

For multi-line bodies, add a blank line after the subject, then describe what changed and why.

### 5. Commit

```bash
git commit -m "<message>"
```

### 6. Confirm

Show the commit with `git log --oneline -1` and `git diff --stat HEAD~1`.
