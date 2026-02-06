---
name: create-release
description: Manage the release lifecycle -- create a release branch with version bump, or finalize by merging to main and tagging. Use at the start or end of a release cycle.
argument-hint: "<version> <codename> [--finalize]"

---

# Create Release

Manage the full release lifecycle. Each release has a unique one-word codename.

Previous releases: Phoenix (1.0.0).

## Arguments

- `$ARGUMENTS[0]` -- Version number (e.g., `1.1.0`). Required.
- `$ARGUMENTS[1]` -- Release codename (e.g., `Gryphon`). Required when creating; ignored with `--finalize`.
- `--finalize` -- Finalize an existing release branch (merge to main + tag + push).

Without `--finalize`, creates a new release branch. With `--finalize`, ships the release.

---

## Phase 1: Create Release Branch (default)

Use at the **start** of a release cycle.

### 1. Validate preconditions

```bash
git status --porcelain
git branch --show-current
```

Must be on `main` with a clean working tree. If not, stop and ask the user.

### 2. Create the release branch

```bash
git checkout -b release/<version>
```

### 3. Update version references

Update `RELEASE_VERSION` and `RELEASE_NAME` in `Taskfile.yml`:

```yaml
RELEASE_NAME: <codename>
RELEASE_VERSION: <version>
```

Update `version` in `src/webui/package.json`:

```json
"version": "<version>"
```

### 4. Commit and push

```bash
git add -A
git commit -m "release: begin v<version> <codename>"
git push -u origin release/<version>
```

### 5. Report

Summarize:
- Release branch: `release/<version>`
- Codename: `<codename>`
- Version bumped to `<version>`
- Next steps: create feature/bugfix/fix branches from `release/<version>`, merge them back via PR

---

## Phase 2: Finalize Release (`--finalize`)

Use when all work on the release branch is **complete and tested**.

### 1. Validate preconditions

```bash
git status --porcelain
git branch --show-current
```

Must be on `release/<version>` with a clean working tree. If not, stop and ask the user.

### 2. Read release metadata

Extract `RELEASE_NAME` and `RELEASE_VERSION` from `Taskfile.yml` to use in tags and messages.

### 3. Run full test suite

```bash
task fmt
task lint
task test
```

All must pass. If anything fails, stop and fix before continuing.

### 4. Build and verify

```bash
task build
./build/bin/ldfctl version
```

Confirm the output shows the expected version and codename.

### 5. Merge release branch to main

```bash
git checkout main
git pull origin main
git merge --no-ff release/<version> -m "Merge release/<version> - v<version> <codename>"
```

### 6. Tag the release

```bash
git tag -a v<version> -m "v<version> - <codename>"
```

### 7. Push main and tag

```bash
git push origin main
git push origin v<version>
```

This triggers the automated release workflow (`.github/workflows/release.yml`), which builds production binaries, packages them with checksums, and creates the GitHub release.

### 8. Clean up

```bash
git branch -d release/<version>
git push origin --delete release/<version>
```

### 9. Report

Summarize:
- Release branch merged to `main`
- Tag pushed: `v<version>`
- Codename: `<codename>`
- Automated release workflow triggered
- Release branch deleted (local and remote)
- Check: `https://github.com/bitswalk/ldf/actions`
