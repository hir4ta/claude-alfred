---
name: buddy-checkpoint
description: >
  Use proactively every 15-20 tool calls or before committing changes to
  check session health, verify no anti-patterns are active, and get a quick
  status on progress. Especially important before git commits or when
  working on complex multi-file changes.
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
