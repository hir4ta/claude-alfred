---
name: alfred-gate
description: >
  Invoke every 15 tool calls, before git commits, and when switching files
  or tasks. Quick health + quality gate that catches problems early and
  prevents bad commits. Do NOT skip before git operations.
user-invocable: false
allowed-tools: mcp__claude-alfred__state, mcp__claude-alfred__guidance, Bash, Read
---

Session health check and pre-commit quality gate.

## Steps

1. Call state with detail="skill", skill_name="alfred-gate" to get health score, test/build status, unresolved failures, and alerts
2. If this is a pre-commit check, verify tests have been run and no active alerts exist
3. Only call guidance with focus="alerts" separately if health < 0.7 and you need more detail

## Output

- If health >= 0.7, no alerts, tests passing: "Gate passed" and continue
- If blocking issues: list them (max 3) with suggested fixes
- Never block operations yourself — advise only
- Max 3 lines
