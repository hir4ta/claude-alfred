---
description: Constraints and best practices when editing hooks.json
paths:
  - "**/hooks/hooks.json"
  - "**/.claude/hooks.json"
  - "**/hooks.json"
---

# Hooks Guidelines

## Timeout Constraints

Hook timeouts are enforced by Claude Code — exceeding them causes SIGTERM:

| Event | Max Timeout | Notes |
|---|---|---|
| PreToolUse | 2s | No I/O, no DB — pure string matching only |
| UserPromptSubmit | 3s | FTS-only search, no API calls |
| SessionStart | 5s | Local I/O + DB reads, background process spawn OK |
| SessionEnd | 3s | Final cleanup, local I/O + DB writes |
| PreCompact | 10s | Transcript parsing + file writes |

- Set internal timeouts slightly UNDER the external timeout (e.g., 4500ms for a 5s hook) to allow graceful cleanup
- Hook handlers MUST fail-open: never block Claude Code on errors
- Use `statusMessage` for every hook to give users feedback

## Matchers

- PreToolUse: scope with tool name matchers (e.g., `"Edit|Write|MultiEdit"`) to avoid unnecessary invocations
- SessionEnd: exclude `clear` reason unless persistence on clear is intentional
- Keep matcher patterns minimal — broad matchers waste resources

## General

- Hooks are short-lived processes — avoid Voyage API calls or network requests in the critical path
- Offload slow work to background processes (`cmd.Start()` + `cmd.Process.Release()`)
- Always include a top-level `description` field in hooks.json
