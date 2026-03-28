---
name: qult-plan-generator
description: "Generate a structured plan from a feature description. Spawns qult-plan-generator agent to analyze the codebase and produce a task-by-task plan. Use when starting new work or to expand a brief description into a detailed plan. NOT for modifying existing plans or reviewing plans (use /qult:plan-review for review)."
---

# /qult:plan-generator

Generate a structured implementation plan from a brief feature description.

## Stage 1: Plan generation (independent agent)

Spawn one `qult-plan-generator` agent with the user's feature description: `$ARGUMENTS`

The agent analyzes the codebase independently and outputs a complete plan in markdown format.

## Stage 2: Validation (conditional)

If the generated plan has 4 or more tasks:
1. Write the plan to `.claude/plans/plan-<timestamp>.md`
2. Run `/qult:plan-review` to validate plan quality
3. If findings exist, fix the plan and re-validate (max 1 fix cycle)

If the plan has fewer than 4 tasks, skip validation.

## Stage 3: Persist

Write the final plan to `.claude/plans/plan-<timestamp>.md` (if not already written in Stage 2).

Use format: `plan-YYYYMMDD-HHMMSS.md` for the filename.

## Output

Summary line: `Plan generated: .claude/plans/<filename> (N tasks)`

Then suggest: "Enter plan mode (Shift+Tab ×2) to review and approve the plan."
