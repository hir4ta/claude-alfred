---
name: alfred-forecast
description: >
  Estimate task complexity from historical data and predict session trajectory.
  Shows expected tool count, success rate, workflow recommendation, health
  trend, and cascade risk.
user-invocable: true
allowed-tools: mcp__claude-alfred__alfred_plan, mcp__claude-alfred__alfred_state, mcp__claude-alfred__alfred_knowledge, mcp__claude-alfred__alfred_guidance
context: fork
agent: Explore
---

Task estimation and session prediction dashboard.

## Steps

1. Determine task type from the user's description (bugfix, feature, refactor, research, review)
2. Call alfred_plan with mode="estimate" and the task type for historical data
3. Call alfred_state for real-time session snapshot including predictions
4. Call alfred_state with detail="skill", skill_name="alfred-forecast" for health and phase data
5. If health < 0.7, call alfred_guidance with focus="alerts" for anti-pattern details

## Output

- Task type + expected tool count (median) + success rate
- Health: [score] [trend] | Phase: [current] → [next]
- Cascade risk: [low/medium/high]
- Recommended workflow steps
- One-sentence forecast

Keep it concise — max 8 lines.
