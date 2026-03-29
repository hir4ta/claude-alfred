---
name: init
description: Set up qult quality gates for the current project. Creates .qult/ directory, detects gates, and places rules files.
user_invocable: true
---

# /qult:init

Set up qult for this project. Run this once per project.

## Steps

1. **Create `.qult/.state/` directory** if it doesn't exist
2. **Detect gates**: Run `/qult:detect-gates` to auto-detect lint, typecheck, and test tools
3. **Place rules files** in `Project/.claude/rules/` — use the exact content from `/qult:update` skill
4. **Add `.qult/` to `.gitignore`** if not already present
5. **Clean up legacy files**: Check for and remove old `~/.claude/skills/qult-*`, `~/.claude/agents/qult-*`, `~/.claude/rules/qult-*`, and `Project/.claude/rules/qult.md` (old name, now `qult-gates.md`)
6. **Copy hook binary**: Copy `${CLAUDE_PLUGIN_ROOT}/dist/hook.mjs` to `.qult/hook.mjs` via Bash: `cp "${CLAUDE_PLUGIN_ROOT}/dist/hook.mjs" .qult/hook.mjs`
7. **Register hooks in settings.local.json**: Read `.claude/settings.local.json` (create if missing). For each event in the hooks below, add or replace that event's entry in the existing `hooks` object. Preserve any other events or non-hooks keys already in the file. This ensures non-qult hooks are not destroyed.

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|Bash",
        "hooks": [
          {
            "type": "command",
            "command": "node \"$CLAUDE_PROJECT_DIR\"/.qult/hook.mjs post-tool",
            "timeout": 15
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Edit|Write|Bash|ExitPlanMode",
        "hooks": [
          {
            "type": "command",
            "command": "node \"$CLAUDE_PROJECT_DIR\"/.qult/hook.mjs pre-tool",
            "timeout": 5
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node \"$CLAUDE_PROJECT_DIR\"/.qult/hook.mjs stop",
            "timeout": 5
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node \"$CLAUDE_PROJECT_DIR\"/.qult/hook.mjs subagent-stop",
            "timeout": 5
          }
        ]
      }
    ],
    "TaskCompleted": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "node \"$CLAUDE_PROJECT_DIR\"/.qult/hook.mjs task-completed",
            "timeout": 15
          }
        ]
      }
    ]
  }
}
```

## Output

Confirm each step was completed successfully.
