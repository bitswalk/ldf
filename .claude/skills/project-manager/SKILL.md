---
name: project-manager
description: Organizes project development by reading the GitHub Project roadmap, tracking milestone progress, and determining next tasks. Use when planning work, reviewing priorities, deciding what to work on next, or updating task status.
---

# Project Manager

Assists with project planning and coordination using GitHub Project, Issues, and Milestones.

## When to use this skill

- Planning what to work on next
- Reviewing milestone progress and priorities
- Breaking down a feature into actionable tasks
- Updating task status after completing work
- Deciding task order and dependencies

## How to use it

1. **Check GitHub state** -- query milestones, issues, and the project board:
   ```bash
   # Open milestones with issue counts
   gh api repos/bitswalk/ldf/milestones?state=open --jq '.[] | "\(.title): \(.open_issues) open, \(.closed_issues) closed"'
   
   # Open issues for next milestone
   gh issue list --milestone "M5 - Platform Maturity" --state open
   
   # Project board
   gh project item-list 9 --owner bitswalk
   ```

2. **Read auto memory** for context: check `MEMORY.md` for current state and notes

3. **Determine next task** based on:
   - Milestone priority (M5 > M6 > M7 for remaining work)
   - Task dependencies within a milestone
   - Issue labels and priority tags

4. **Update tracking** when work completes:
   - Close the GitHub issue: `gh issue close <number>`
   - Update `MEMORY.md` in auto memory when milestones complete or priorities shift
   - Use the TodoWrite tool to track in-progress work within a session

## Workflow

```
Query GH milestones/issues -> Read MEMORY.md -> Identify next open issue ->
  If release branch exists: branch from release/<version>
  If no release branch: branch from main
-> Work on it -> Close issue -> Update MEMORY.md
```

## Creating new work items

When new tasks arise that aren't tracked:

```bash
# Create issue with labels and milestone
gh issue create --title "Title" --body "Description" \
  --label "component: server" --label "type: feature" --label "priority: medium" \
  --milestone "M5 - Platform Maturity"

# Add to project board
gh project item-add 9 --owner bitswalk --url "https://github.com/bitswalk/ldf/issues/NUMBER"
```

## Release cycle workflow

### Minor releases (new features, milestones)

When planning a new feature release:

1. Identify the version number, codename, and all work items (milestone subtasks, bugs, fixes)
2. Create GitHub issues for all items, assign to the appropriate milestone
3. Use `/create-release <version> <codename>` to create the `release/<version>` branch
4. All feature/bugfix/fix branches for this release should be created from `release/<version>`
5. Merge completed work back into `release/<version>` (use `/milestone-complete`)
6. When all work is done and tested, use `/create-release <version> --finalize` to merge to main and tag
7. The tag push triggers automated release build and publish via GitHub Actions

Each minor release must have a unique one-word codename.

### Patch releases (bug fixes, polish)

When bugs or UI issues are discovered after a release:

1. **Create GitHub issues** for each fix with labels (`type: bug`, `priority:`, `component:`)
2. **Determine patch version**: increment the patch number of the current version (e.g., 1.2.0 -> 1.2.1)
3. **Create the release branch**: `/create-release <x.y.Z> <same-codename>` (reuse the minor release codename)
4. **Create bugfix branches** from the release branch: `bugfix/<issue-number>-<short-description>`
5. **Merge fixes** back into the release branch
6. **Finalize**: `/create-release <x.y.Z> --finalize`

**IMPORTANT**: Never commit fixes directly to `main`. Even a single bug fix should go through a patch release.

### Release history

| Version | Codename | Type |
|---------|----------|------|
| 1.0.0 | Phoenix | Major (initial) |
| 1.1.0 | Gryphon | Minor (M5.5) |
| 1.2.0 | Basilisk | Minor (M5.6) |

## Key references

- **GitHub Project**: https://github.com/orgs/bitswalk/projects/9 (project number 9, owner: bitswalk)
- **Milestones**: M1-M4 closed, M5 (Platform Maturity), M6 (TUI Client), M7 (Ecosystem)
- **Auto memory MEMORY.md**: project state, completed milestones, technical notes
- **Branch convention**: `feature/m<milestone>_<subtask>` (e.g., M5.1 -> `feature/m5_1`)
