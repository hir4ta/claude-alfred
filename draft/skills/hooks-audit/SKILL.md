---
name: hooks-audit
description: >
  Audit hooks.json configuration for correctness and security. Validates event
  names (22 valid events), handler types (command/http/prompt/agent), required
  fields per type, timeout ranges, matcher regex compilation, async restrictions,
  and security (no secrets in commands, allowedEnvVars for HTTP). Checks both
  project (.claude/hooks.json) and user (~/.claude/settings.json) hooks.
  Use when editing hooks, debugging hook failures, or auditing hook security.
  NOT for CLAUDE.md review (use /claude-md-audit). NOT for code review.
user-invocable: true
argument-hint: "[hooks.json path]"
allowed-tools: Read, Glob, Grep, Bash(jq *), WebFetch
context: fork
model: sonnet
---

# /hooks-audit — Hooks Configuration Auditor

## Phase 0: Refresh Knowledge

1. **WebFetch** `https://code.claude.com/docs/en/hooks` with prompt:
   "Extract all valid hook event names, handler types with required fields,
   timeout defaults, matcher syntax rules, async restrictions, exit code
   meanings, and security constraints."

## Phase 1: Discover Hooks

- Path given → read that file
- No arguments → scan:
  - `.claude/hooks.json`
  - `.claude/settings.json` (hooks section)
  - `.claude/settings.local.json` (hooks section)
  - `~/.claude/settings.json` (hooks section)
  - Plugin hooks (`.claude/plugins/*/hooks/hooks.json`)

## Phase 2: Evaluate

### Event Validity (CRITICAL)

| # | Check | Rule |
|---|---|---|
| EV1 | Event name is valid | Must be one of: SessionStart, InstructionsLoaded, UserPromptSubmit, PreToolUse, PermissionRequest, PostToolUse, PostToolUseFailure, Notification, SubagentStart, SubagentStop, Stop, TeammateIdle, TaskCompleted, ConfigChange, WorktreeCreate, WorktreeRemove, PreCompact, PostCompact, Elicitation, ElicitationResult, SessionEnd, Setup |
| EV2 | No typos in event names | Case-sensitive check |

### Handler Compliance (CRITICAL)

| # | Check | Rule |
|---|---|---|
| H1 | type field present | Required: "command", "http", "prompt", or "agent" |
| H2 | command type has command field | Non-empty string |
| H3 | http type has url field | Valid URL |
| H4 | prompt/agent type has prompt field | Non-empty string |
| H5 | async only on command type | async: true invalid on http/prompt/agent |
| H6 | once only in skill/agent hooks | Not valid in global hooks |

### Timeout Validation (HIGH)

| # | Check | Rule |
|---|---|---|
| T1 | Timeout in reasonable range | command: default 600s; prompt: 30s; agent: 60s |
| T2 | SessionEnd hooks fast | Global 1.5s limit for SessionEnd |
| T3 | No zero timeout | timeout: 0 means instant kill |

### Matcher Validation (HIGH)

| # | Check | Rule |
|---|---|---|
| M1 | Matcher regex compiles | Invalid regex = hook never fires |
| M2 | Matcher not overly broad | `.*` matches everything — warn |
| M3 | Tool matchers use correct format | MCP: `mcp__server__tool` |

### Security (CRITICAL)

| # | Check | Rule |
|---|---|---|
| SC1 | No secrets in command strings | API keys, tokens, passwords |
| SC2 | HTTP hooks use allowedEnvVars | No raw secrets in headers |
| SC3 | No eval/curl-pipe-sh patterns | Command injection risk |
| SC4 | Prompt hooks return valid JSON | Schema: `{"ok": bool, "reason": string}` |

### Best Practices (MEDIUM)

| # | Check | Rule |
|---|---|---|
| BP1 | statusMessage for slow hooks | User feedback during long operations |
| BP2 | No duplicate event handlers | Same event + matcher registered twice |
| BP3 | Blocking hooks are justified | Stop/PreToolUse hooks should be lightweight |

## Phase 3: Score & Report

```
## Hooks Audit: {path}

**Events: {N}** | **Handlers: {N}** | **Score: {X}/100** | **Status: {PASS|NEEDS_FIXES}**

| Event | Type | Timeout | Matcher | Issues |
|-------|------|---------|---------|--------|
| PreToolUse | command | 10s | Edit | OK |
| Stop | prompt | 30s | — | SC4: JSON response unreliable |
```

## Example

User: `/hooks-audit`

```
## Hooks Audit: .claude/hooks.json

**Events: 4** | **Handlers: 5** | **Score: 85/100** | **Status: PASS_WITH_WARNINGS**

| Event | Type | Timeout | Issues |
|-------|------|---------|--------|
| SessionStart | command | 5s | OK |
| PreToolUse | command | 600s | [HIGH] T1: 600s default — consider lower |
| Stop | prompt | 30s | [MEDIUM] BP1: No statusMessage |
| SessionEnd | command | 10s | [HIGH] T2: Exceeds 1.5s SessionEnd limit |
```

## Troubleshooting

- **hooks.json parse error**: Validate JSON syntax first with `jq . < hooks.json`.
- **Event name not found**: Check exact casing — events are PascalCase.
- **WebFetch fails**: Use embedded event list (EV1) as authoritative source.
