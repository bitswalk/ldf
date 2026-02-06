---
name: create-release
description: Build release binaries, package them, and create a GitHub release with tag, release notes, and assets. Use when cutting a new version.
argument-hint: "<version> [release-name]"

---

# Create Release

Build, package, and publish a GitHub release.

## Arguments

- `$ARGUMENTS[0]` -- Version number (e.g., `1.0.0`, `1.1.0`). Required.
- `$ARGUMENTS[1]` -- Release codename (e.g., `Phoenix`). Optional -- defaults to current `RELEASE_NAME` in `Taskfile.yml`.

## Steps

### 1. Validate preconditions

Ensure we're on `main` with a clean working tree:

```bash
git status --porcelain
git branch --show-current
```

If there are uncommitted changes or we're not on `main`, stop and ask the user.

### 2. Update version references

Update `RELEASE_VERSION` in `Taskfile.yml`:

```yaml
RELEASE_VERSION: <version>
```

If a release name was provided, also update `RELEASE_NAME`.

Update `version` in `src/webui/package.json`:

```json
"version": "<version>"
```

### 3. Commit version bump

```bash
git add -A
git commit -m "release: v<version> <release-name>"
git push
```

### 4. Run tests

```bash
task fmt
task lint
go test ./...
```

All must pass. If anything fails, stop and fix before continuing.

### 5. Build release binaries

```bash
task build:srv
task build:cli
```

Verify version is stamped correctly:

```bash
./build/bin/ldfctl version
```

Confirm the output shows the expected version.

### 6. Package assets

Create release tarballs and checksums:

```bash
mkdir -p build/release
tar -czf build/release/ldfd-v<version>-linux-amd64.tar.gz -C build/bin ldfd
tar -czf build/release/ldfctl-v<version>-linux-amd64.tar.gz -C build/bin ldfctl
cd build/release && sha256sum *.tar.gz > checksums-v<version>-sha256.txt
```

### 7. Generate release notes

Build a GitHub-flavored markdown release body with:

- **Title**: `LDF v<version> -- <release-name>`
- **Highlights**: Summarize major changes since the last release. Use `git log --oneline <prev-tag>..HEAD` to review commits. Group by type (features, fixes, improvements).
- **Components table**: ldfd, ldfctl, WebUI
- **Checksums block**: Contents of the checksums file in a fenced code block
- **Installation snippet**: tar extract + verify commands

### 8. Create GitHub release

```bash
gh release create v<version> \
  --title "v<version> - <release-name>" \
  --notes "<release-notes>" \
  --target main \
  build/release/ldfd-v<version>-linux-amd64.tar.gz \
  build/release/ldfctl-v<version>-linux-amd64.tar.gz \
  build/release/checksums-v<version>-sha256.txt
```

### 9. Clean up

```bash
rm -rf build/release
```

### 10. Report

Summarize:
- Release URL (from `gh release create` output)
- Tag: `v<version>`
- Assets uploaded with sizes
- SHA-256 checksums
