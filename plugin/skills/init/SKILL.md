---
name: init
description: Set up qult quality gates for the current project. Creates .qult/ directory, detects gates, and places rules files.
user_invocable: true
---

# /qult:init

Set up qult for this project. Run this once per project.

## Steps

1. **Create `.qult/.state/` directory** if it doesn't exist
2. **Detect gates**: Run `/qult:detect-gates` to auto-detect lint, typecheck, and test tools
3. **Place rules files** in `Project/.claude/rules/`:
   - `qult.md` — MCP tool invocation rules (when DENIED, call `mcp__qult__get_pending_fixes`)
   - `qult-quality.md` — Test-driven development, scope management
   - `qult-plan.md` — Plan structure and task registration
4. **Add `.qult/` to `.gitignore`** if not already present
5. **Clean up legacy files**: Check for and remove old `~/.claude/skills/qult-*`, `~/.claude/agents/qult-*`, `~/.claude/rules/qult-*` files from pre-plugin installation

## Rules file content

### qult.md
```markdown
# qult Quality Gates

## MCP Tool Usage
- When a tool is DENIED by qult, call `mcp__qult__get_pending_fixes()` immediately to see errors
- Before committing, call `mcp__qult__get_session_status()` to check for blockers
- After a TaskCompleted hook runs, call `mcp__qult__get_session_status()` to see verification results
- If gates are not configured, run `/qult:detect-gates`

## Commit Gates
- All lint/typecheck errors must be fixed before committing
- Tests must pass before committing (when on_commit gates are configured)
- Independent review (`/qult:review`) is required for large changes or when a plan is active
```

### qult-quality.md
```markdown
# Quality Rules (qult)

## Test-Driven
- Write the test file FIRST, then implement
- At least 2 meaningful assertions per test case
- Do not mark implementation as complete until tests pass

## Task Scope
- Quick fix (no plan): keep changes focused, 1-2 files per logical change
- Planned work: follow the plan's task boundaries, scope is set by the plan
```

### qult-plan.md
```markdown
# Plan Rules (qult)

## Plan Structure
When writing a plan, use this structure:

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

Update task status to [done] as you complete each task.

## Task Registration
When transitioning from Plan mode to implementation:
1. Read the approved plan file
2. Create a task (TaskCreate) for each Task entry
3. Update each task status as you work
```

## Output
Confirm each step was completed successfully.
