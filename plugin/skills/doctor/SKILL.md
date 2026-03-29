---
name: doctor
description: Check qult health for the current project. Verifies state files, gates, rules, and MCP server connectivity.
user_invocable: true
---

# /qult:doctor

Diagnose qult setup issues in the current project.

## Checks

1. **`.qult/` directory**: Exists with `.state/` subdirectory
2. **Gates**: `.qult/gates.json` exists and has at least one gate in `on_write` or `on_commit`
3. **Rules**: `Project/.claude/rules/qult-gates.md` exists with MCP tool invocation rules
4. **Hook binary**: `.qult/hook.mjs` exists
5. **Settings hooks**: `.claude/settings.local.json` exists and contains a `hooks` key with all 5 events (PostToolUse, PreToolUse, Stop, SubagentStop, TaskCompleted)
6. **MCP server**: Call `mcp__plugin_qult_qult__get_gate_config()` to verify the MCP server is responding
7. **Pending fixes**: Call `mcp__plugin_qult_qult__get_pending_fixes()` to check for stale errors
8. **Session state**: Call `mcp__plugin_qult_qult__get_session_status()` to verify state tracking works

## Output format

Report each check as OK or FAIL with details:
```
[OK] .qult/.state/ directory exists
[OK] gates.json: 2 on_write, 1 on_commit gates
[OK] rules: .claude/rules/qult-gates.md exists
[OK] hook binary: .qult/hook.mjs exists
[FAIL] settings hooks: .claude/settings.local.json missing hooks — run /qult:init
[OK] MCP server responding
```

## Fix suggestions

For each FAIL, suggest the fix:
- Missing `.qult/`: Run `/qult:init`
- Missing gates: Run `/qult:detect-gates`
- Missing rules: Run `/qult:init`
- Missing hook binary: Run `/qult:init` or `/qult:update`
- Missing settings hooks: Run `/qult:init`
- MCP not responding: Check plugin is enabled via `/plugin`
