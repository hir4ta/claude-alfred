---
name: buddy-recover
description: >
  Invoke on 2+ consecutive tool failures, Edit 'old_string not found', test
  FAIL after a fix attempt, or any compilation/import error. Do NOT retry —
  invoke this skill first for root cause analysis and past resolution diffs.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_recall, mcp__claude-buddy__buddy_alerts
---

Failure recovery advisor. Covers stuck loops, error recovery, and test failure debugging.

## Steps

1. Call buddy_skill_context with skill_name="buddy-recover" to get session health, recent failures, past solutions, and test correlations
2. If a past resolution diff exists, present the exact fix
3. If more detail needed, call buddy_patterns with the error message or failing test name
4. If the failure involves a specific file, call buddy_recall to find what worked before

## Output

- Root cause hypothesis (one sentence)
- ONE specific alternative approach or past fix to try
- If past resolution diff exists, show the exact change

Keep it under 5 lines. Be direct and actionable.
