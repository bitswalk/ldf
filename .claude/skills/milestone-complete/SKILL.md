---
name: milestone-complete
description: Complete a milestone by running all tests, merging the feature branch to main, and updating project tracking. Use when finishing a milestone subtask.
argument-hint: "[milestone-id] [description]"

---

# Milestone Complete

Finalize a milestone subtask by running checks, merging, and updating tracking.

## Arguments

- `$ARGUMENTS[0]` -- Milestone ID (e.g., `M5.1`, `M5.2`)
- `$ARGUMENTS[1]` -- Short description (e.g., "Board profiles and device trees")

## Steps

### 1. Pre-flight checks

Run the full test suite and build:

```bash
task fmt
task lint
task test
task build
```

All must pass before proceeding. Fix any failures.

### 2. Commit any remaining changes

If there are uncommitted changes, commit them using conventional commit format.

### 3. Merge to main

```bash
git checkout main
git merge --no-ff <current-branch> -m "Merge branch '<current-branch>' - $1"
```

The branch name should follow the convention `feature/m<X>_<Y>` (e.g., `feature/m5_1` for M5.1).

### 4. Clean up feature branch

```bash
git branch -d <feature-branch>
```

### 5. Update auto memory

Update the MEMORY.md file in auto memory to reflect the completed milestone:
- Move the subtask to the "Completed" section with a summary of what was done
- Update the "Next" pointer to the next subtask
- Add any new deferred items or known issues discovered during the work

### 6. Report

Summarize:
- What was completed
- Files added/modified count
- Lines added
- Any deferred items or known issues
- What the next task should be

## Branch naming convention

- Milestone subtask: `feature/m<milestone>_<subtask>` (e.g., M5.1 -> `feature/m5_1`)
- When a branch combines multiple subtasks, use the first subtask number
- Create branches from `main`, merge back to `main` with `--no-ff`
