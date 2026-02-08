---
name: create-release
description: Manage the release lifecycle -- create a release branch with version bump, or finalize by merging to main and tagging. Use at the start or end of a release cycle.
argument-hint: "<version> <codename> [--finalize]"

---

# Create Release

Manage the full release lifecycle. Each release has a unique one-word codename.

Previous releases: Phoenix (1.0.0), Gryphon (1.1.0), Basilisk (1.2.0).

## Arguments

- `$ARGUMENTS[0]` -- Version number (e.g., `1.1.0` or `1.2.1`). Required.
- `$ARGUMENTS[1]` -- Release codename (e.g., `Gryphon`). Required when creating; ignored with `--finalize`.
- `--finalize` -- Finalize an existing release branch (merge to main + tag + push).

Without `--finalize`, creates a new release branch. With `--finalize`, ships the release.

**Patch releases** (e.g., `1.2.1`) reuse the codename of their minor release (e.g., Basilisk).
**Minor releases** (e.g., `1.3.0`) require a new unique codename.

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

### 7. Generate release notes (minor+ releases only)

For **minor releases** (x.Y.0) and **major releases** (X.0.0), generate curated release notes. Skip this step for patch releases (x.y.Z where Z > 0) -- they use auto-generated changelog.

Determine if this is a minor+ release by checking if the patch version is `0`.

Write a `RELEASE_NOTES.md` file in the repo root with the following structure. Do NOT include an H1 title -- the GitHub release title (`v<version>`) already serves as the heading.

```markdown
## Highlights

- Concise bullet points of the most important user-facing changes
- Group by theme (new features, improvements, fixes)
- Reference GitHub issues where relevant (#N)

## Components

| Component | Description |
|-----------|-------------|
| ldfd | LDF daemon / API server |
| ldfctl | Command-line client |
| WebUI | Web interface (served by ldfd) |

## Milestones Included

- **M<N>** -- <Title>: brief summary of what was delivered
- List all milestones completed since the previous minor/major release

## Installation

\```bash
# Download and extract
tar -xzf ldfd-v<version>-linux-amd64.tar.gz
tar -xzf ldfctl-v<version>-linux-amd64.tar.gz

# Verify
./ldfd version
./ldfctl version
\```
```

**Guidelines:**
- Keep it concise -- no redundant information (the H1 title is already the release tag)
- Highlights should focus on what's new since the last minor/major release, not repeat commit messages
- Use the milestone descriptions from MEMORY.md and GitHub issues for content
- The checksums section is not needed -- the workflow adds checksums as release assets

After writing, commit it:

```bash
git add RELEASE_NOTES.md
git commit --amend --no-edit
```

This amends the merge commit so the notes file is included in the tagged release.

### 8. Push main and tag

```bash
git push origin main
git push origin v<version>
```

This triggers the automated release workflow (`.github/workflows/release.yml`), which builds production binaries, packages them with checksums, and creates the GitHub release. For minor+ releases, the workflow uses `RELEASE_NOTES.md` as the release body.

### 9. Clean up

```bash
git branch -d release/<version>
git push origin --delete release/<version>
```

### 10. Report

Summarize:
- Release branch merged to `main`
- Tag pushed: `v<version>`
- Codename: `<codename>`
- Automated release workflow triggered
- Release notes: curated (minor+) or auto-generated (patch)
- Release branch deleted (local and remote)
- Check: `https://github.com/bitswalk/ldf/actions`
