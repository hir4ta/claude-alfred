---
name: buddy-predict
description: >
  Prediction dashboard showing next-tool predictions, cascade risk assessment,
  health trend trajectory, and workflow phase progress. Useful for understanding
  session dynamics and anticipating issues.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_current_state, mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_alerts
---

Session prediction and forecasting dashboard.

## Steps

1. Call buddy_current_state to get real-time session snapshot including predictions
2. Call buddy_skill_context with skill_name="buddy-checkpoint" for health and phase data
3. If health < 0.7, call buddy_alerts for detailed anti-pattern information

## Output

Present a prediction dashboard:
- Health: [score] [trend: improving/stable/declining]
- Next likely tools: [predicted tools with confidence]
- Cascade risk: [low/medium/high based on recent failure patterns]
- Phase: [current phase] → [expected next phase]
- Forecast: [one sentence about session trajectory]

Keep it under 8 lines. Use the data to provide actionable insight.
