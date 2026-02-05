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

1. Read the roadmap for current state: `.agent/roadmap/roadmap.md`
2. Read the active priorities: `.agent/next.md`
3. Read the relevant feature spec: `.agent/roadmap/features/m{N}-*.md`
4. Determine the next actionable task based on:
   - Milestone priority (M1 > M2 > M3 > ... > M7)
   - Task dependencies within a milestone
   - Current completion status (checkboxes in feature files)
5. Update `.agent/next.md` when priorities shift
6. Mark tasks as done in feature files when completed

## File references

- **Roadmap overview**: [roadmap.md](../roadmap/roadmap.md)
- **Current priorities**: [next.md](../next.md)
- **Feature specs**: [features/](../roadmap/features/)
- **Project memory**: [AGENT.md](../AGENT.md)

## Workflow

```
Read next.md -> Read relevant feature file -> Identify next unchecked task -> Work on it -> Mark done -> Update next.md if needed
```
