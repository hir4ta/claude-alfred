---
name: status
description: Show current qult session status including pending fixes, test/review gates, and changed files.
user_invocable: true
---

# /qult:status

Show the current qult quality gate status.

## Steps

1. Call `mcp__plugin_qult_qult__get_pending_fixes()` for current lint/typecheck errors
2. Call `mcp__plugin_qult_qult__get_session_status()` for session state
3. Call `mcp__plugin_qult_qult__get_gate_config()` for gate configuration

## Output format

Present a concise summary:

```
Pending fixes: 2 (lint: foo.ts, typecheck: bar.ts)
Tests: not passed
Review: not completed
Changed files: 5 (3 gated)
Gates: lint (biome), typecheck (tsc), test (vitest)
```

If no issues, report:
```
All clear. No pending fixes, tests passed, review completed.
```
