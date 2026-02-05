# Configuration

## Config File Locations

ldfd searches for a configuration file named `ldfd.yaml` (or `ldfd.yml`) in these paths, in order:

1. Path specified via `--config` flag
2. `/etc/ldfd/`
3. `/opt/ldfd/`
4. `~/.ldfd/`

```bash
# Use a specific config file
ldfd --config /path/to/ldfd.yml
```

## Configuration Precedence

Settings are resolved in this order (highest priority first):

1. **Environment variables** -- `LDFD_` prefix (e.g., `LDFD_SERVER_PORT=9000`)
2. **CLI flags** -- (e.g., `--port 9000`)
3. **Config file** -- YAML format
4. **Defaults** -- Hardcoded in the application

## Environment Variables

Environment variables use the `LDFD_` prefix with dots replaced by underscores:

| Config Key | Environment Variable |
|------------|---------------------|
| `server.port` | `LDFD_SERVER_PORT` |
| `log.level` | `LDFD_LOG_LEVEL` |
| `storage.s3.access_key` | `LDFD_STORAGE_S3_ACCESS_KEY` |

This is useful for passing secrets without writing them to config files:

```bash
export LDFD_STORAGE_S3_ACCESS_KEY="your-access-key"
export LDFD_STORAGE_S3_SECRET_KEY="your-secret-key"
ldfd
```

## CLI Flags

Common flags available on the `ldfd` command:

| Flag | Config Key | Default |
|------|-----------|---------|
| `--port`, `-p` | `server.port` | `8443` |
| `--bind`, `-b` | `server.bind` | `0.0.0.0` |
| `--config` | -- | (search paths) |
| `--log-output` | `log.output` | `auto` |
| `--log-level` | `log.level` | `info` |
| `--db-path` | `database.path` | `~/.ldfd/ldfd.db` |
| `--storage-type` | `storage.type` | `local` |
| `--storage-path` | `storage.local.path` | `~/.ldfd/artifacts` |
| `--s3-endpoint` | `storage.s3.endpoint` | (empty) |
| `--s3-region` | `storage.s3.region` | `us-east-1` |
| `--s3-bucket` | `storage.s3.bucket` | `ldf-distributions` |
| `--s3-access-key` | `storage.s3.access_key` | (empty) |
| `--s3-secret-key` | `storage.s3.secret_key` | (empty) |

## Settings Reference

All configurable settings with their types, defaults, and descriptions:

### Server

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `server.port` | int | `8443` | Yes | Port for the HTTP server |
| `server.bind` | string | `0.0.0.0` | Yes | Network address to bind to |

### Logging

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `log.output` | string | `auto` | No | Log destination: `stdout`, `journald`, or `auto` |
| `log.level` | string | `info` | No | Minimum log level: `debug`, `info`, `warn`, `error` |

Log level can be changed at runtime via the settings API without restarting the server.

### Database

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `database.path` | string | `~/.ldfd/ldfd.db` | Yes | Path to persist the in-memory SQLite database on shutdown |

### Storage

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `storage.type` | string | `local` | Yes | Storage backend: `local` or `s3` |
| `storage.local.path` | string | `~/.ldfd/artifacts` | Yes | Root directory for local artifact storage |

### S3 Storage

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `storage.s3.provider` | string | (empty) | Yes | S3 provider: `garage`, `minio`, `aws`, or `other` |
| `storage.s3.endpoint` | string | (empty) | Yes | S3 base domain (e.g., `s3.example.com`) |
| `storage.s3.region` | string | `us-east-1` | Yes | AWS/S3 region |
| `storage.s3.bucket` | string | `ldf-distributions` | Yes | S3 bucket name |
| `storage.s3.access_key` | string | (empty) | Yes | S3 access key ID (sensitive) |
| `storage.s3.secret_key` | string | (empty) | Yes | S3 secret access key (sensitive) |

Sensitive values are masked in API responses unless the `?reveal=true` query parameter is used.

### WebUI

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `webui.devmode` | bool | `false` | No | Enable developer mode (debug console and logs) |
| `webui.app_name` | string | (empty) | No | Custom application name in header and browser tab (max 32 characters) |

### Sync

| Key | Type | Default | Reboot | Description |
|-----|------|---------|--------|-------------|
| `sync.cache_duration` | int | `60` | No | Minimum minutes between automatic version syncs for a source (0 to disable caching) |

## Runtime Settings API

Settings can be read and modified at runtime via the REST API (requires root access):

```bash
# List all settings
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/v1/settings

# Get a single setting
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/v1/settings/log.level

# Update a setting
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  http://localhost:8443/v1/settings/log.level \
  -d '{"value": "debug"}'
```

Settings marked "Reboot: Yes" require a server restart to take effect. Settings marked "Reboot: No" are applied immediately.

## Complete Config File Example

```yaml
# ldfd.yml -- Full configuration reference
# See docs/samples/ldfd.yml for a copy of this file.

# Server settings
server:
  port: 8443
  bind: "0.0.0.0"

# Logging
log:
  output: "auto"    # auto, stdout, journald
  level: "info"     # debug, info, warn, error

# Database persistence
database:
  path: "~/.ldfd/ldfd.db"

# Artifact storage
storage:
  type: "local"     # local or s3

  local:
    path: "~/.ldfd/artifacts"

  # S3-compatible storage (uncomment to use)
  # s3:
  #   provider: "garage"    # garage, minio, aws, other
  #   endpoint: "s3.example.com"
  #   region: "us-east-1"
  #   bucket: "ldf-distributions"
  #   access_key: ""        # Use LDFD_STORAGE_S3_ACCESS_KEY env var instead
  #   secret_key: ""        # Use LDFD_STORAGE_S3_SECRET_KEY env var instead

# WebUI customization
webui:
  devmode: false
  app_name: ""

# Version sync behavior
sync:
  cache_duration: 60  # minutes (0 to disable)
```
