# Sources

Sources represent upstream repositories from which LDF discovers and tracks component versions. Each source points to a URL (typically a Git forge) and periodically syncs available versions.

## Concepts

### What is a source?

A source is a link between a component in LDF and an upstream repository. For example, a "Linux kernel" component might have a source pointing to `https://github.com/torvalds/linux`. LDF queries the source to discover available versions (tags/releases).

### System sources vs user sources

- **System sources** -- Pre-configured sources that ship with LDF's default component catalog. These are managed by the platform and cannot be deleted by regular users.
- **User sources** -- Sources created by users to track custom or additional upstream repositories. Users can create, update, and delete their own sources.

Both types use the same API endpoints. Access control is handled based on ownership and the `is_system` flag.

## Forge Detection

When creating a source, LDF automatically detects the hosting platform (forge) from the URL:

| Forge | Example URL | Detection Pattern |
|-------|------------|-------------------|
| GitHub | `https://github.com/torvalds/linux` | `github.com` |
| GitLab | `https://gitlab.com/group/project` | `gitlab.com` or self-hosted |
| Gitea | `https://gitea.example.com/org/repo` | Gitea API detection |
| Codeberg | `https://codeberg.org/org/repo` | `codeberg.org` |
| Forgejo | `https://forgejo.example.com/org/repo` | Forgejo API detection |
| Generic | `https://example.com/releases/` | Fallback for HTTP directories |

You can also use the forge detection API to test URL detection:

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  http://localhost:8443/v1/forge/detect \
  -d '{"url": "https://github.com/torvalds/linux"}'
```

The response includes the detected forge type, repository info, and suggested defaults (URL template and version filter).

## Version Discovery

Once a source is created, LDF discovers available versions by querying the upstream forge:

- **GitHub/GitLab/Gitea/Codeberg/Forgejo** -- Fetches tags and releases via the platform API
- **Generic HTTP** -- Parses directory listings for version-like entries

Versions are cached in the database. The sync interval is controlled by the `sync.cache_duration` setting (default: 60 minutes). Set to `0` to disable caching and always fetch fresh data.

### Manual sync

Trigger a version sync for a specific source:

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/v1/sources/{id}/sync
```

Check sync status:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/v1/sources/{id}/sync/status
```

### Startup sync

When ldfd starts, it automatically triggers a background version sync for all sources. This ensures the version cache is fresh without waiting for the first scheduled sync.

## Version Filtering

Sources can have version filters that control which upstream versions are included. Filters use a comma-separated pattern syntax:

### Filter syntax

- `*` matches any sequence of characters
- `!pattern` excludes versions matching the pattern
- Multiple patterns are comma-separated
- Exclusions take precedence over inclusions

### Examples

| Filter | Effect |
|--------|--------|
| `!*-rc*` | Exclude release candidates |
| `!*-rc*,!*alpha*,!*beta*` | Exclude pre-release versions |
| `6.12.*,6.6.*,6.1.*` | Only include specific major.minor versions |
| `!*-rc*,!next-*` | Exclude RC and next-branch versions |

### Common filter presets

LDF provides built-in filter presets accessible via the API:

| Preset | Filter |
|--------|--------|
| `stable-only` | `!*-rc*,!*alpha*,!*beta*,!*-dev*,!*-pre*,!*-snapshot*,!*-nightly*` |
| `no-rc` | `!*-rc*` |
| `lts-only` | `6.12.*,6.6.*,6.1.*,5.15.*,5.10.*,5.4.*` |
| `latest-major` | `6.*` |
| `kernel-stable` | `!*-rc*,!next-*` |
| `kernel-lts` | `6.12.*,6.6.*,6.1.*,5.15.*,5.10.*,5.4.*,4.19.*,4.14.*,!*-rc*` |

List available presets:

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/v1/forge/common-filters
```

### Filter preview

Before applying a filter, you can preview its effect on actual upstream versions:

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  http://localhost:8443/v1/forge/preview-filter \
  -d '{
    "url": "https://github.com/torvalds/linux",
    "version_filter": "!*-rc*,!next-*"
  }'
```

The response shows each version with whether it would be included or excluded, and the reason.

## API Endpoints

All source endpoints require authentication.

### Source management

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/sources` | List all sources |
| `POST` | `/v1/sources` | Create a user source |
| `GET` | `/v1/sources/component/{componentId}` | List sources for a component |
| `GET` | `/v1/sources/{id}` | Get a source by ID |
| `PUT` | `/v1/sources/{id}` | Update a source |
| `DELETE` | `/v1/sources/{id}` | Delete a source |

### Version management

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/sources/{id}/versions` | List discovered versions |
| `GET` | `/v1/sources/{id}/versions/types` | Get version type breakdown |
| `DELETE` | `/v1/sources/{id}/versions` | Clear cached versions |

### Sync

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/sources/{id}/sync` | Trigger version sync |
| `GET` | `/v1/sources/{id}/sync/status` | Get sync job status |

### Forge utilities

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/forge/detect` | Detect forge type from URL |
| `POST` | `/v1/forge/preview-filter` | Preview filter effect on versions |
| `GET` | `/v1/forge/types` | List all forge types |
| `GET` | `/v1/forge/common-filters` | List common filter presets |
