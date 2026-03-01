---
name: alfred-recover
description: >
  Invoke on 2+ consecutive tool failures, Edit 'old_string not found', test
  FAIL after a fix attempt, or any compilation/import error. Do NOT retry —
  invoke this skill first for root cause analysis and past resolution diffs.
user-invocable: false
allowed-tools: mcp__claude-alfred__alfred_state, mcp__claude-alfred__alfred_knowledge, mcp__claude-alfred__alfred_guidance, mcp__claude-alfred__alfred_diagnose
---

Failure recovery advisor. Covers stuck loops, error recovery, and test failure debugging.

## Steps

1. Call alfred_state with detail="skill", skill_name="alfred-recover" to get session health, recent failures, past solutions, and test correlations
2. If a past resolution diff exists, present the exact fix
3. If more detail needed, call alfred_knowledge with the error message or failing test name
4. If the failure involves a specific file, call alfred_knowledge with scope="recall" to find what worked before

## Output

- Root cause hypothesis (one sentence)
- ONE specific alternative approach or past fix to try
- If past resolution diff exists, show the exact change

Keep it under 5 lines. Be direct and actionable.
