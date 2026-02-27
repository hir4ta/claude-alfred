---
name: buddy-forecast
description: >
  Estimate task complexity from historical data and predict session trajectory.
  Shows expected tool count, success rate, workflow recommendation, health
  trend, and cascade risk.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_estimate, mcp__claude-buddy__buddy_current_state, mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_alerts
context: fork
agent: Explore
---

Task estimation and session prediction dashboard.

## Steps

1. Determine task type from the user's description (bugfix, feature, refactor, research, review)
2. Call buddy_estimate with the task type for historical data
3. Call buddy_current_state for real-time session snapshot including predictions
4. Call buddy_skill_context with skill_name="buddy-forecast" for health and phase data
5. If health < 0.7, call buddy_alerts for anti-pattern details

## Output

- Task type + expected tool count (median) + success rate
- Health: [score] [trend] | Phase: [current] → [next]
- Cascade risk: [low/medium/high]
- Recommended workflow steps
- One-sentence forecast

Keep it concise — max 8 lines.
