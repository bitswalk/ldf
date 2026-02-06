---
name: check-ci
description: Check GitHub Actions CI pipeline status for the current branch or a recent push. Use after pushing code to validate that CI passes, or to diagnose failures.
argument-hint: "[branch or pr-number]"
---

# Check CI

Check the GitHub Actions CI pipeline status and diagnose any failures.

## Steps

### 1. Determine what to check

If `$ARGUMENTS` is provided, use it as the branch name or PR number.
Otherwise, use the current branch.

### 2. Get workflow run status

```bash
# Latest run for current branch
gh run list --branch <branch> --limit 3

# Or for a specific PR
gh pr checks <pr-number>
```

### 3. If any job failed, get details

```bash
# List jobs for a specific run
gh run view <run-id> --json jobs --jq '.jobs[] | {name, status, conclusion}'

# Get failed job logs
gh run view <run-id> --log-failed
```

### 4. Diagnose failures

The CI pipeline has 5 jobs:

| Job | What it checks | Common fixes |
|-----|---------------|--------------|
| **Lint** | `gofmt` + `golangci-lint` | Run `task fmt` then `task lint` locally |
| **Test Server** | `go test -v -race ./src/ldfd/...` | Run `task test:srv` locally |
| **Test CLI** | `go test -v -race ./src/ldfctl/...` | Run `task test:cli` locally |
| **Test WebUI** | `bun run test` in `src/webui/` | Run `cd src/webui && /home/flint/.bun/bin/bun test` locally |
| **Build** | `task build` (runs after all tests pass) | Run `task build` locally |

The Build job depends on all 4 test/lint jobs — it only runs if they all pass.

### 5. Fix and retry

If failures are found:
1. Reproduce locally with the matching command
2. Fix the issue
3. Commit and push
4. Re-check CI status

### 6. Report

Summarize:
- Overall status (pass/fail)
- Per-job status
- If failed: root cause and what was done to fix it
