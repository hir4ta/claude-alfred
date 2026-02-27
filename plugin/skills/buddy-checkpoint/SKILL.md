---
name: buddy-checkpoint
description: >
  Invoke after every 15 tool calls, before any git operation, and whenever
  switching between files or tasks. Quick health check that catches problems
  early. Especially important before git commits or when working on complex
  multi-file changes.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_current_state, mcp__claude-buddy__buddy_alerts
---

Quick session health check.

## Steps

1. Call buddy_skill_context with skill_name="buddy-checkpoint" to get session snapshot, health score, and alerts in one call
2. Only call buddy_alerts separately if health score < 0.7 and you need more detail

## Output

- If health >= 0.7 and no alerts: respond "Session healthy" and continue
- If health < 0.7: state the top issue in one sentence
- If active alerts: mention the most severe one with its suggestion
- Never output more than 3 lines
