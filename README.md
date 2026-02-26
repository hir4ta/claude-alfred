# claude-buddy

A proactive session companion for Claude Code — real-time anti-pattern detection, destructive command blocking, automatic context recovery, code quality feedback, git awareness, and AI-powered usage coaching. Works as both a standalone TUI and a Claude Code plugin with hooks.

## Install

```bash
brew install hir4ta/tap/claude-buddy
```

Or build from source:

```bash
git clone https://github.com/hir4ta/claude-buddy
cd claude-buddy
go build -o claude-buddy .
```

## Setup

[Ollama](https://ollama.com) is required for knowledge search features. Start it before running install:

```bash
# Install and start Ollama
brew install ollama
ollama serve &

# Pull embedding model (choose one)
ollama pull kun432/cl-nagoya-ruri-large    # Japanese
ollama pull nomic-embed-text               # English / other languages

# Register hooks, sync sessions, generate embeddings
claude-buddy install
```

This registers the MCP server, writes hooks to `~/.claude/settings.json`, syncs all existing sessions to the local SQLite database (`~/.claude-buddy/buddy.db`), and generates embeddings for knowledge search. Hooks are active the next time you start Claude Code.

## Upgrade

```bash
brew update && brew upgrade claude-buddy
```

After upgrading, re-run install to update hook paths:

```bash
claude-buddy install
```

## Language

claude-buddy detects your system locale (`LANG` / `LC_ALL` / `LC_MESSAGES`) and generates AI feedback in your language. UI labels remain in English.

To persist your language setting, add to your `~/.zshrc` (or `~/.bashrc`):

```bash
export LANG=ja_JP.UTF-8
```

Or set per-invocation:

```bash
LANG=ja_JP.UTF-8 claude-buddy
LANG=ko_KR.UTF-8 claude-buddy
```

> **Note**: On macOS, the terminal may default to `en_US.UTF-8` even if the system language is set to Japanese. Set `LANG` explicitly if feedback appears in the wrong language.

Supported languages: English, Japanese, Chinese, Korean, Spanish, French, German, Portuguese, Russian, Italian, Arabic, Hindi, Thai, Vietnamese, Turkish, Polish, Dutch, Swedish.

## Commands

### `claude-buddy` / `claude-buddy watch`

Monitor a Claude Code session in real-time. Run in a separate terminal or tmux pane.

```bash
# Terminal 1
claude-buddy

# Terminal 2
claude
```

**Features:**

- **Header**: Session ID, turn count, tool usage, elapsed time, pulsing activity indicator
- **Anti-pattern detection**: Real-time alerts for retry loops, context thrashing, excessive tools, destructive commands, and more
  - Warning (yellow) and Action (red) level alert bars
- **Task progress**: TaskCreate/TaskUpdate tracking with shimmer animation
  - `○` pending / `▶` in_progress (animated) / `✔` completed
- **Message stream**: Live display of user input, assistant responses, tool summaries
  - `[user]` / `[answer]` / `[assistant]` / `[task+]` / `[agent]` / `[plan]` / `[msg]`
  - Expand any message with Enter to view full content
- **AI Feedback**: Every turn, LLM evaluates your session against official best practices
  - Situation / Observation / Suggestion with severity levels (info, insight, warning, action)

**Key bindings:**

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `↑` / `k` | Scroll up |
| `↓` / `j` | Scroll down |
| `Enter` | Expand/collapse message |
| `g` / `G` | Jump to top/bottom |
| `?` | Help overlay |

---

### `claude-buddy browse`

Browse past session history with the same expand/collapse interface.

```bash
claude-buddy browse
```

---

### `claude-buddy install`

One-time setup: registers the MCP server, writes hooks to `~/.claude/settings.json`, syncs sessions, and generates embeddings (if Ollama available).

```bash
claude-buddy install
```

---

### `claude-buddy serve`

Run as an MCP server (stdio) for Claude Code integration.

```bash
claude-buddy serve
```

**MCP Tools:**

| Tool | Description |
|------|-------------|
| `buddy_stats` | Session statistics (turns, tool frequency, duration) |
| `buddy_tips` | AI-powered feedback and improvement suggestions |
| `buddy_sessions` | List recent sessions with metadata |
| `buddy_resume` | Restore previous session context (goal, intent, compaction history, files changed/referenced, decisions) |
| `buddy_recall` | Search across past session history |
| `buddy_decisions` | Extract design decisions from past sessions |
| `buddy_alerts` | Real-time anti-pattern detection (retry loops, context thrashing, etc.) |
| `buddy_patterns` | Cross-project knowledge search with vector semantic search (Ollama) |
| `buddy_suggest` | Structured recommendations with session health, alerts, and feature utilization |
| `buddy_current_state` | Real-time session snapshot (stats, burst state, health score, predictions) |
| `buddy_estimate` | Task complexity estimation based on historical workflow data |
| `buddy_next_step` | Recommended next actions based on session context and recent tool history |
| `buddy_skill_context` | Aggregated session context tailored for a specific skill |

---

### `claude-buddy analyze [session_id]`

AI-powered session analysis via `claude -p`.

```bash
claude-buddy analyze          # Latest session
claude-buddy analyze de999fa4 # Specific session by ID prefix
```

Requires `claude` CLI in PATH.

### `claude-buddy uninstall`

Remove hooks and MCP server registration:

```bash
claude-buddy uninstall
```

## Hooks

`claude-buddy install` writes hooks directly to `~/.claude/settings.json`. These hooks actively monitor your session through Claude Code's lifecycle events:

| Hook Event | Behavior |
|---|---|
| **SessionStart** | Auto-restores session context (working set, decisions, git branch), captures git state |
| **PreToolUse** | Blocks destructive commands; warns on stale reads, git-dirty files, past failures; surfaces related decisions |
| **PostToolUse** | Tracks tool/file patterns, code quality heuristics, test failure correlation, suggestion effectiveness |
| **UserPromptSubmit** | Classifies intent/task type, injects relevant past knowledge, delivers queued nudges |
| **PreCompact** | Serializes working set (files, intent, decisions, git branch) for automatic restoration |
| **Stop** | Detects incomplete work (TODO/FIXME, unresolved failures), warns about uncommitted git changes |
| **PostToolUseFailure** | Tracks failure cascades, searches past solutions, starts resolution chains |
| **SubagentStart** | Injects session context into subagent launches |
| **SubagentStop** | Records subagent outcomes and delivery context |
| **SessionEnd** | Persists user profile, co-change data, workflow sequences; cleans up session state |

**Anti-pattern detectors** (hook-based, real-time):

- **RetryLoop**: 3+ consecutive identical tool calls
- **ExcessiveTools**: 25+ tool calls without user input
- **FileReadLoop**: Same file read 5+ times with no edits
- **ExploreLoop**: 10+ tools, no writes, 5+ minutes elapsed
- **Destructive commands**: `rm -rf`, `git push --force`, `git reset --hard`, `git checkout -- .`, `chmod 777`, etc.

**Proactive advisor signals** (context-injected via `additionalContext`):

- **Stale read warning**: File not re-read before editing, or last Read was 8+ tool calls ago
- **Past failure warning**: Similar Bash command failed earlier in the session
- **Git dirty file warning**: Editing a file with pre-existing uncommitted changes
- **Code quality heuristics**: Detects unchecked Go errors, debug prints, bare Python excepts, hardcoded secrets, TODO without ticket numbers, console.log in production JS/TS
- **Test failure correlation**: Connects test failures to recently edited files
- **Workflow guidance**: Suggests test-first approach for bugfix/refactor tasks
- **Past knowledge surfacing**: Surfaces related decisions and error solutions from previous sessions

**Automatic context recovery** (survives compaction):

Working set (currently edited files, intent, task type, key decisions, git branch) is automatically serialized before compaction and restored afterward. No manual intervention required.

**Suggestion effectiveness tracking**:

Nudge delivery and resolution are tracked across sessions. Patterns delivered 20+ times with <10% resolution rate are automatically suppressed to reduce noise.

**Skills** (invocable via slash commands):

| Skill | Description |
|---|---|
| `/buddy-unstuck` | Escape retry loops and suggest alternative approaches |
| `/buddy-checkpoint` | Session health check with active anti-pattern summary |
| `/buddy-before-commit` | Pre-commit quality verification |
| `/buddy-impact` | Blast radius analysis for planned file changes |
| `/buddy-review` | Review recent changes against pattern DB knowledge |
| `/buddy-estimate` | Task complexity estimation from historical data |
| `/buddy-predict` | Prediction dashboard (next tool, cascade risk, health trend) |

## Architecture

```
claude-buddy/
├── main.go                    # Entry point + subcommand routing
├── internal/
│   ├── parser/                # JSONL parser (type definitions + parsing)
│   ├── watcher/               # File watching (fsnotify + tail)
│   ├── analyzer/              # Live stats + Feedback type + anti-pattern detector
│   ├── coach/                 # AI feedback generation via claude -p
│   ├── hookhandler/           # Hook handlers (advisor signals, code heuristics, test correlation)
│   ├── sessiondb/             # Ephemeral per-session SQLite (working set, burst state, nudges)
│   ├── embedder/              # Ollama integration for semantic search
│   ├── locale/                # System locale detection (18 languages)
│   ├── tui/                   # Bubble Tea TUI (watch / browse / select)
│   ├── mcpserver/             # MCP server (stdio, 13 tools)
│   ├── store/                 # SQLite persistence (vector search + LIKE search + incremental sync)
│   └── install/               # Hook registration + MCP registration + initial sync
├── go.mod
└── go.sum
```

## Dependencies

| Library | Purpose |
|---------|---------|
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | TUI styling |
| [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) | File change watching |
| [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) | MCP server SDK |
| [ncruces/go-sqlite3](https://github.com/ncruces/go-sqlite3) | SQLite driver (pure Go, WASM-based) |

## Ollama

Ollama powers `buddy_patterns` and hook-based knowledge injection via vector semantic search. The embedding model is auto-selected based on your system locale: `kun432/cl-nagoya-ruri-large` (1024d) for Japanese, `nomic-embed-text` (768d) for other languages.

Ollama availability is checked once at session start and cached — subsequent hook calls use a single HTTP round-trip for embedding.
