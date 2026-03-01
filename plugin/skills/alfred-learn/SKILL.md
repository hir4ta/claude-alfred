---
name: alfred-learn
description: >
  Tell alfred about your Claude Code preferences and working style.
  Records preferences that influence future suggestions and briefings.
user-invocable: true
allowed-tools: AskUserQuestion, mcp__claude-alfred__feedback, mcp__claude-alfred__state
---

Preference recording for personalized advice.

## Steps

1. Call state with detail="brief" to get current session stats and user profile
2. Ask the user about their preferences using AskUserQuestion:

   Question 1: "How do you prefer to work with Claude Code?"
   - "Plan first, then implement" (plan mode user)
   - "Jump straight into coding" (direct mode)
   - "Depends on the task" (adaptive)

   Question 2: "How do you feel about automated suggestions?"
   - "Show me everything — I'll filter" (aggressive)
   - "Only important things" (balanced)
   - "Minimal interruptions" (conservative)

   Question 3: "Which features do you use most?"
   - Options: hooks, skills, MCP tools, worktrees, agents, teams
   - multiSelect: true

3. Record each preference via feedback tool with appropriate pattern names:
   - feedback pattern="workflow_preference" rating="helpful" comment="[selected option]"
   - feedback pattern="suggestion_verbosity" rating="helpful" comment="[selected option]"

## Output

Confirm what was recorded:
- Workflow style: [selection]
- Suggestion level: [selection]
- Primary features: [selections]
- "These preferences will influence future alfred suggestions."

Keep it brief — max 5 lines.
