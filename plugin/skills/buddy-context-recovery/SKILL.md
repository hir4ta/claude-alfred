---
name: buddy-context-recovery
description: >
  Use automatically after context compaction to restore working context.
  Recovers the current task intent, working set files, recent decisions,
  and git branch state from session memory.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_recall, mcp__claude-buddy__buddy_decisions
---

Automatic context recovery after compaction.

## Steps

1. Call buddy_skill_context with skill_name="buddy-context-recovery" to get working set, decisions, and session state
2. If key details are missing, call buddy_recall with specific queries for:
   - Current task/goal
   - Files being actively edited
   - Recent decisions made
3. Call buddy_decisions to restore architectural context if working on a complex task

## Output

Provide a compact recovery summary:
- Current goal: [one sentence]
- Active files: [list of files being edited]
- Recent decisions: [key decisions, max 3]
- Branch: [git branch if available]

Keep it under 8 lines. Focus on what's needed to continue work immediately.
