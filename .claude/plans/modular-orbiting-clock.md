# M3.3 -- CLI Advanced Features (branch `feature/m3_3`)

## Summary

Add YAML output format, query parameter flags (limit/offset/status filtering), composite `release` commands, and improved shell completion. This rounds out the CLI UX before M3.4 testing.

## Scope

### 1. YAML output format

Add `--output yaml` support across all commands.

**Files:**
- `src/ldfctl/internal/output/output.go` -- add `PrintYAML(data interface{}) error` using `gopkg.in/yaml.v3`
- All command files -- update `getOutputFormat()` checks to handle `"yaml"` case
- `src/ldfctl/internal/cmd/root.go` -- update `--output` flag description to `"table, json, yaml"`

**Pattern:** Every command handler currently has:
```go
if getOutputFormat() == "json" {
    return output.PrintJSON(resp)
}
```
Update to:
```go
switch getOutputFormat() {
case "json":
    return output.PrintJSON(resp)
case "yaml":
    return output.PrintYAML(resp)
}
```

### 2. Query parameter flags (limit, offset, status)

The server already supports `?limit=N&offset=N` via `common.GetPaginationParams()` and `?status=` on distributions. The CLI client `Get()` takes a path string, so query params are appended directly.

**Client changes** -- update list methods to accept query params:

| Client file | Method | New params |
|-------------|--------|------------|
| `distributions.go` | `ListDistributions` | `limit, offset int, status string` |
| `components.go` | `ListComponents` | `limit, offset int` |
| `sources.go` | `ListSources` | `limit, offset int` |
| `downloads.go` | `ListDistributionDownloads` | `limit, offset int` |
| `components.go` | `GetComponentVersions` | `limit, offset int, versionType string` |
| `sources.go` | `ListSourceVersions` | `limit, offset int, versionType string` |

Use an options struct pattern to avoid breaking signatures:

```go
type ListOptions struct {
    Limit       int
    Offset      int
    Status      string
    VersionType string
}

func (o *ListOptions) QueryString() string {
    // Build ?limit=X&offset=Y&status=Z
}
```

Place in `src/ldfctl/internal/client/client.go`.

**Command changes** -- add flags to list commands:

| Command file | Command | New flags |
|-------------|---------|-----------|
| `distribution.go` | `list` | `--limit`, `--offset`, `--status` |
| `component.go` | `list` | `--limit`, `--offset` |
| `component.go` | `versions` | `--limit`, `--offset`, `--version-type` |
| `source.go` | `list` | `--limit`, `--offset` |
| `source.go` | `versions` | `--limit`, `--offset`, `--version-type` |
| `download.go` | `list` | `--limit`, `--offset` |

### 3. Composite `release` commands

A "release" is a CLI abstraction over the distribution API. The README shows:
```
ldfctl create release --distribution my-distro --version 0.0.1 --platform x86_64
ldfctl configure release --distribution my-distro --version 0.0.1 --channel alpha ...
```

The server has no separate "release" endpoint -- these are wrappers around distribution create/update with the `DistributionConfig` payload.

**New file:** `src/ldfctl/internal/cmd/release.go`

```
release
├── create   --name, --version, --platform, --visibility
├── configure --distribution <id>, --kernel, --init, --filesystem, --bootloader,
│             --partitioning-type, --partitioning-mode, --security,
│             --container, --virtualization, --target-type,
│             --desktop-env, --display-server, --package-manager
└── show     --distribution <id>  (show current config in readable format)
```

**`create`** = calls `CreateDistribution` with name + version + optional initial config.
**`configure`** = calls `GetDistribution` to fetch current config, merges flag values, calls `UpdateDistribution`.
**`show`** = calls `GetDistribution` and pretty-prints the config tree.

Register `releaseCmd` in root.go.

### 4. Shell completion enhancements

Cobra auto-generates the `completion` command (bash/zsh/fish/powershell). This already works. The enhancement is to add **custom completers** for positional args and flag values so users get context-aware suggestions.

**New file:** `src/ldfctl/internal/cmd/completion.go` -- helper functions for completions.

**Completions to add:**

| Command | Arg/Flag | Completer |
|---------|----------|-----------|
| `distribution get/update/delete/logs/stats/deletion-preview` | `<id>` arg | Fetch distribution list, return IDs+names |
| `component get/update/delete/versions/resolve-version` | `<id>` arg | Fetch component list, return IDs+names |
| `source get/update/delete/sync/versions/sync-status/clear-versions` | `<id>` arg | Fetch source list, return IDs+names |
| `role get/update/delete` | `<id>` arg | Fetch role list, return IDs |
| `--output` flag | value | Static: `table`, `json`, `yaml` |
| `--status` flag | value | Static: `pending`, `downloading`, `validating`, `ready`, `failed` |
| `--visibility` flag | value | Static: `public`, `private` |
| `component list --category` | value | Fetch categories from API |

Pattern using Cobra's `ValidArgsFunction`:
```go
cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    // Call API to get list, return matching items
}
```

And `RegisterFlagCompletionFunc` for flag values.

### 5. Improved error messages

Enhance `APIError.Error()` in `client.go` to provide more user-friendly messages for common status codes:
- 401: "Authentication required. Run 'ldfctl login' first."
- 403: "Permission denied. You don't have access to this resource."
- 404: "Resource not found."
- 409: "Resource already exists."
- 500: "Server error. Check server logs for details."

## Implementation plan

### Step 1: Add YAML output support

1. `go get gopkg.in/yaml.v3`
2. Edit `src/ldfctl/internal/output/output.go` -- add `PrintYAML`
3. Edit `src/ldfctl/internal/cmd/root.go` -- update `--output` flag description
4. Edit all 13 command files -- replace `if json` checks with `switch` including yaml case

### Step 2: Add ListOptions and query parameter support

1. Edit `src/ldfctl/internal/client/client.go` -- add `ListOptions` struct + `QueryString()` method
2. Edit client files (distributions, components, sources, downloads) -- update list methods to accept `*ListOptions`
3. Edit command files -- add `--limit`, `--offset`, `--status`, `--version-type` flags; pass to client

### Step 3: Add release composite commands

1. Create `src/ldfctl/internal/cmd/release.go` -- `create`, `configure`, `show` subcommands
2. Edit `src/ldfctl/internal/cmd/root.go` -- register `releaseCmd`

### Step 4: Add shell completion helpers

1. Create `src/ldfctl/internal/cmd/completion.go` -- reusable completer functions
2. Edit command files that take resource IDs -- add `ValidArgsFunction`
3. Register flag completions for `--output`, `--status`, `--visibility`, `--category`

### Step 5: Improve error messages

1. Edit `src/ldfctl/internal/client/client.go` -- enhance `APIError.Error()` with human-friendly hints

### Step 6: Register, build, verify

1. Register `releaseCmd` in root.go
2. `go build ./src/ldfctl/...` + `go vet ./src/ldfctl/...`
3. Verify `--help` output for all new/changed commands
4. Verify `ldfctl completion bash` works

## Files created/modified

| File | Action |
|------|--------|
| `src/ldfctl/internal/output/output.go` | Edit (add PrintYAML) |
| `src/ldfctl/internal/client/client.go` | Edit (add ListOptions, improve APIError) |
| `src/ldfctl/internal/client/distributions.go` | Edit (ListDistributions accepts ListOptions) |
| `src/ldfctl/internal/client/components.go` | Edit (ListComponents, GetComponentVersions accept ListOptions) |
| `src/ldfctl/internal/client/sources.go` | Edit (ListSources, ListSourceVersions accept ListOptions) |
| `src/ldfctl/internal/client/downloads.go` | Edit (ListDistributionDownloads accepts ListOptions) |
| `src/ldfctl/internal/cmd/root.go` | Edit (register releaseCmd, update --output desc) |
| `src/ldfctl/internal/cmd/release.go` | New (composite release commands) |
| `src/ldfctl/internal/cmd/completion.go` | New (completion helper functions) |
| `src/ldfctl/internal/cmd/distribution.go` | Edit (yaml output, --limit/--offset/--status flags, completions) |
| `src/ldfctl/internal/cmd/component.go` | Edit (yaml output, --limit/--offset flags, completions) |
| `src/ldfctl/internal/cmd/source.go` | Edit (yaml output, --limit/--offset flags, completions) |
| `src/ldfctl/internal/cmd/download.go` | Edit (yaml output, --limit/--offset flags) |
| `src/ldfctl/internal/cmd/artifact.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/setting.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/role.go` | Edit (yaml output, completions) |
| `src/ldfctl/internal/cmd/forge.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/branding.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/langpack.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/health.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/auth.go` | Edit (yaml output) |
| `src/ldfctl/internal/cmd/version.go` | Edit (yaml output) |

## Verification

- `go build ./src/ldfctl/...` compiles
- `go vet ./src/ldfctl/...` passes
- `ldfctl --help` shows release command
- `ldfctl distribution list --limit 5 --offset 0 --status ready --output yaml` works
- `ldfctl component list --limit 10 --output yaml` works
- `ldfctl release create --help` shows all config flags
- `ldfctl release configure --help` shows all config flags
- `ldfctl completion bash` outputs valid completion script
- Tab completion provides distribution/component/source IDs (when server running)
- Error messages show friendly hints (test with expired token → "Run 'ldfctl login' first")
