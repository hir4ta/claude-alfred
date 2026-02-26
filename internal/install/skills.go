package install

import (
	"os"
	"path/filepath"
)

type skillDef struct {
	Dir     string // directory name under ~/.claude/skills/
	Content string // SKILL.md content
}

var buddySkills = []skillDef{
	{
		Dir: "buddy-unstuck",
		Content: `---
name: buddy-unstuck
description: >
  Use proactively when experiencing repeated failures (3+ consecutive errors
  on the same file or tool), when stuck in a retry loop, or when the same
  approach keeps failing. Analyzes root cause and suggests alternative
  approaches based on past session knowledge.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_recall, mcp__claude-buddy__buddy_alerts
---

You are a debugging advisor. The user (Claude Code) is stuck in a failure loop.

## Steps

1. Call buddy_skill_context with skill_name="buddy-unstuck" to get session health, recent failures, and past solutions in one call
2. If more detail needed, call buddy_patterns with a query describing the current error
3. If the pattern involves a specific file, call buddy_recall to find what worked before

## Output

Provide exactly ONE alternative approach:
- What's likely causing the repeated failure (one sentence)
- A specific different approach to try (one sentence)
- If a past solution exists, reference it

Keep it under 5 lines. Be direct and actionable.
`,
	},
	{
		Dir: "buddy-checkpoint",
		Content: `---
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
`,
	},
	{
		Dir: "buddy-before-commit",
		Content: `---
name: buddy-before-commit
description: >
  Use automatically before any git commit to verify code quality and test
  status. Checks for active anti-patterns, unrun tests, and ensures no
  obvious issues will be committed.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_alerts, mcp__claude-buddy__buddy_current_state, Bash, Read
---

Pre-commit quality gate.

## Steps

1. Call buddy_skill_context with skill_name="buddy-before-commit" to get test/build status, unresolved failures, and quality summary in one call
2. If alerts are present, investigate and suggest fixes
3. If tests were not run and the project has tests, suggest running them

## Output

- If blocking issues found: list them (max 3) and suggest fixes
- If clean: respond "Pre-commit check passed" and proceed with the commit
- Never block the commit yourself — just advise
`,
	},
	{
		Dir: "buddy-impact",
		Content: `---
name: buddy-impact
description: >
  Analyze the blast radius of planned changes before editing. Shows importers,
  test coverage, and co-change history for a file.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_decisions, Read, Grep, Glob, Bash
---

Analyze the impact of changing a specific file or module.

## Steps

1. Identify the target file from the user's request
2. Call buddy_patterns to find related past changes and issues
3. Call buddy_decisions to check for architectural constraints
4. Use Grep to find importers/references to this file
5. Use Glob to find related test files

## Output

- Blast radius: number of files that reference this module
- Test coverage: which test files exist for this code
- Past issues: any known problems from pattern DB
- Recommendations: suggested approach for the change

Keep output under 10 lines. Be specific about file paths.
`,
	},
	{
		Dir: "buddy-review",
		Content: `---
name: buddy-review
description: >
  Review recent changes against pattern DB knowledge. Checks for known
  anti-patterns, past failures with similar code, and architectural decisions.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_alerts, mcp__claude-buddy__buddy_decisions, Bash, Read
---

Review recent code changes using pattern database knowledge.

## Steps

1. Call buddy_skill_context with skill_name="buddy-review" to get modified files, test status, and related patterns in one call
2. Run 'git diff --stat' to see detailed changes
3. Call buddy_patterns for specific files if more detail needed
4. Call buddy_decisions to check if changes align with past architectural choices

## Output

- List specific issues found (max 5)
- Reference past patterns or decisions that are relevant
- Suggest concrete fixes if issues found
- If clean, say "No issues found" and summarize what was reviewed
`,
	},
	{
		Dir: "buddy-estimate",
		Content: `---
name: buddy-estimate
description: >
  Estimate task complexity based on historical workflow data. Shows expected
  tool count, success rate, and recommended workflow.
user-invocable: true
allowed-tools: mcp__claude-buddy__buddy_estimate, mcp__claude-buddy__buddy_patterns
---

Estimate the complexity of a task using historical data.

## Steps

1. Determine the task type from the user's description (bugfix, feature, refactor, research, review)
2. Call buddy_estimate with the task type
3. Call buddy_patterns to find similar past tasks if available

## Output

- Task type classification
- Expected tool count (median from past sessions)
- Success rate for this type of task
- Recommended workflow steps
- Any relevant patterns from past sessions

Keep it concise — 5-8 lines max.
`,
	},
	{
		Dir: "buddy-error-recovery",
		Content: `---
name: buddy-error-recovery
description: >
  Use automatically after a tool failure to retrieve past resolution diffs
  and solution chains for the same error signature. Provides concrete fix
  suggestions based on cross-session failure→fix knowledge.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_recall
---

Automatic error recovery advisor. Triggered after tool failures.

## Steps

1. Call buddy_skill_context with skill_name="buddy-error-recovery" to get failure context, recent errors, and related solutions
2. If a past solution with resolution diff exists, present the exact fix
3. If no direct match, call buddy_patterns with the error message to find similar past solutions

## Output

- If an exact past fix exists: "Past fix found: change X to Y in file Z"
- If a similar pattern exists: "Similar error was resolved by: [approach]"
- If no past knowledge: "No past solutions found. Suggested approach: [one sentence]"

Keep it under 4 lines. Include file paths and concrete changes when available.
`,
	},
	{
		Dir: "buddy-context-recovery",
		Content: `---
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
`,
	},
	{
		Dir: "buddy-test-guidance",
		Content: `---
name: buddy-test-guidance
description: >
  Use automatically when tests have failed 2+ times consecutively.
  Analyzes failure patterns and suggests targeted debugging strategies
  based on test output correlation and past failure solutions.
user-invocable: false
allowed-tools: mcp__claude-buddy__buddy_skill_context, mcp__claude-buddy__buddy_patterns, mcp__claude-buddy__buddy_recall
---

Test failure debugging advisor.

## Steps

1. Call buddy_skill_context with skill_name="buddy-test-guidance" to get test status, recent failures, and correlated files
2. Call buddy_patterns with the failing test name or error message
3. If the failure involves a specific file, call buddy_recall to find past fixes for that file

## Output

Provide a targeted debugging strategy:
- Root cause hypothesis (one sentence based on error pattern)
- Specific debugging step to try next (one actionable instruction)
- If a past similar failure exists, reference the resolution

Keep it under 5 lines. Be specific about which file/function to investigate.
`,
	},
	{
		Dir: "buddy-predict",
		Content: `---
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
`,
	},
}

// removeSkills removes buddy skills from ~/.claude/skills/.
func removeSkills() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	for _, skill := range buddySkills {
		skillDir := filepath.Join(home, ".claude", "skills", skill.Dir)
		_ = os.RemoveAll(skillDir)
	}
}
