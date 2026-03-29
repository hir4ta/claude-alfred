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
3. **Rules**: `Project/.claude/rules/qult.md` exists with MCP tool invocation rules
4. **MCP server**: Call `mcp__qult__get_gate_config()` to verify the MCP server is responding
5. **Pending fixes**: Call `mcp__qult__get_pending_fixes()` to check for stale errors
6. **Session state**: Call `mcp__qult__get_session_status()` to verify state tracking works

## Output format

Report each check as OK or FAIL with details:
```
[OK] .qult/.state/ directory exists
[OK] gates.json: 2 on_write, 1 on_commit gates
[FAIL] rules: .claude/rules/qult.md not found — run /qult:init
[OK] MCP server responding
```

## Fix suggestions

For each FAIL, suggest the fix:
- Missing `.qult/`: Run `/qult:init`
- Missing gates: Run `/qult:detect-gates`
- Missing rules: Run `/qult:init`
- MCP not responding: Check plugin is enabled via `/plugin`
