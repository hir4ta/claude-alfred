---
name: buddy-before-commit
description: >
  Invoke before every git commit without exception. Checks for unrun tests,
  active anti-patterns, and code quality issues. A 5-second check that
  prevents bad commits. Do NOT skip this even for small changes.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_alerts, mcp__claude-buddy__buddy_current_state, Bash, Read
---

Pre-commit quality gate.

## Steps

1. Call buddy_skill_context with skill_name="buddy-before-commit" to get test/build status, unresolved failures, and quality summary in one call
2. If alerts are present, investigate and suggest fixes
3. If tests were not run and the project has tests, suggest running them

## Output

- If blocking issues found: list them (max 3) and suggest fixes
- If clean: respond "Pre-commit check passed" and proceed with the commit
- Never block the commit yourself — just advise
