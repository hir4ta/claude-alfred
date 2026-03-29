---
name: update
description: Update qult rules files in the current project after a plugin update. Overwrites existing rules with the latest version.
user_invocable: true
---

# /qult:update

Update qult rules files after a plugin update. Overwrites existing rules with the latest version.

Plugin update (`/plugin` > update) automatically updates hooks, skills, agents, and MCP server.
This skill updates the **project-level rules files** that were copied during `/qult:init`.

## Steps

1. **Overwrite rules files** in `Project/.claude/rules/`:

### qult-gates.md
```markdown
# qult Quality Gates

IMPORTANT: These rules are enforced by qult hooks. Follow them exactly.

## MCP Tool Usage

- When a tool is DENIED by qult, you MUST call `mcp__plugin_qult_qult__get_pending_fixes()` immediately
- Before committing, ALWAYS call `mcp__plugin_qult_qult__get_session_status()` to check for blockers
- After a TaskCompleted hook runs, call `mcp__plugin_qult_qult__get_session_status()` to see verification results
- If gates are not configured, run `/qult:detect-gates`

## Commit Gates

- NEVER commit with unresolved lint/typecheck errors
- Tests MUST pass before committing (when on_commit gates are configured)
- Independent review (`/qult:review`) is required for large changes or when a plan is active
```

### qult-quality.md
```markdown
# Quality Rules (qult)

## Test-Driven

- ALWAYS write the test file FIRST, then implement
- At least 2 meaningful assertions per test case
- NEVER mark implementation as complete until tests pass

## Task Scope

- Quick fix (no plan): keep changes focused, 1-2 files per logical change
- Planned work: follow the plan's task boundaries, scope is set by the plan
```

### qult-plan.md
```markdown
# Plan Rules (qult)

## Plan Structure

IMPORTANT: When writing a plan, you MUST use this structure:

\```
## Context
Why this change is needed.

## Tasks
### Task N: <name> [pending]
- **File**: <path>
- **Change**: <what to do>
- **Boundary**: <what NOT to change>
- **Verify**: <test file : test function>

## Success Criteria
- [ ] `<specific command>` -- expected outcome
\```

Update task status to [done] as you complete each task.

## Task Registration

When transitioning from Plan mode to implementation:
1. Read the approved plan file
2. Create a task (TaskCreate) for each Task entry
3. Update each task status as you work
```

2. **Remove legacy rule file** if it exists: `Project/.claude/rules/qult.md` (renamed to `qult-gates.md`)

3. **Re-detect gates** if `.qult/gates.json` is missing: run `/qult:detect-gates`

## Output

Confirm: `qult rules updated: qult-gates.md, qult-quality.md, qult-plan.md`
