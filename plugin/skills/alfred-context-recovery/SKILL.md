---
name: alfred-context-recovery
description: >
  CRITICAL: Invoke immediately when you notice missing context, when you
  cannot recall recent decisions, or when conversation history seems
  truncated. Recovers the current task intent, working set files, recent
  decisions, and git branch state from session memory.
user-invocable: false
allowed-tools: mcp__claude-alfred__state, mcp__claude-alfred__knowledge
---

Automatic context recovery after compaction.

## Steps

1. Call state with detail="skill", skill_name="alfred-context-recovery" to get working set, decisions, and session state
2. If key details are missing, call knowledge with scope="recall" for:
   - Current task/goal
   - Files being actively edited
   - Recent decisions made
3. Call knowledge with type="decision" to restore architectural context if working on a complex task

## Output

Provide a compact recovery summary:
- Current goal: [one sentence]
- Active files: [list of files being edited]
- Recent decisions: [key decisions, max 3]
- Branch: [git branch if available]

Keep it under 8 lines. Focus on what's needed to continue work immediately.
