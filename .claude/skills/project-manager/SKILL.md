---
name: project-manager
description: Organizes project development by reading the roadmap, tracking milestone progress, and determining next tasks. Use when planning work, reviewing priorities, deciding what to work on next, or updating task status.
---

# Project Manager

Assists with project planning and coordination by maintaining a structured roadmap-to-feature pipeline.

## When to use this skill

- Planning what to work on next
- Reviewing milestone progress and priorities
- Breaking down a feature into actionable tasks
- Updating task status after completing work
- Deciding task order and dependencies

## How to use it

1. Read the auto memory for current state: check `MEMORY.md` and `roadmap.md` in the auto memory directory
2. Determine the next actionable task based on:
   - Milestone priority (M5 > M6 > M7 for remaining work)
   - Task dependencies within a milestone
   - Current completion status
3. Update `MEMORY.md` in auto memory when priorities shift or milestones complete
4. Use the TodoWrite tool to track in-progress work within a session

## Workflow

```
Read MEMORY.md -> Read roadmap.md -> Identify next unchecked task -> Work on it -> Mark done -> Update MEMORY.md
```

## Key references

- Auto memory MEMORY.md: project state, completed milestones, deferred items
- Auto memory roadmap.md: M5/M6/M7 task breakdown with acceptance criteria
- Branch convention: `feature/m<milestone>_<subtask>` (e.g., M5.1 -> `feature/m5_1`)
