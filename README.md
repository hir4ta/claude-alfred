# claude-alfred

A proactive session companion for Claude Code — real-time anti-pattern detection, predictive health monitoring, causal failure diagnosis, automatic context recovery, AST-based code quality analysis with auto-fix, coverage-aware test correlation, adaptive personalization, and cross-project knowledge sharing. Works as a Claude Code plugin with hooks, MCP tools, skills, and agents.

## Install

**1. Add the plugin in Claude Code:**

```
/plugin marketplace add hir4ta/claude-alfred
/plugin install claude-alfred@claude-alfred
```

**2. Run initial setup in your terminal (one-time):**

```bash
curl -fsSL https://raw.githubusercontent.com/hir4ta/claude-alfred/main/setup.sh | sh
```

This downloads the binary and syncs all available sessions (JSONL parsing, pattern extraction, and embedding generation). Estimated time is shown before sync starts.

**3. Restart Claude Code** to activate hooks and MCP tools.

### Optional: Voyage AI for semantic search

Set `VOYAGE_API_KEY` before running setup to enable vector-based knowledge search across sessions. Without it, search falls back to FTS5 BM25 / LIKE.

```bash
export VOYAGE_API_KEY=your-api-key
curl -fsSL https://raw.githubusercontent.com/hir4ta/claude-alfred/main/setup.sh | sh
```

Already ran setup without the key? Just set the key and re-run — embeddings are generated incrementally.

Setting `VOYAGE_API_KEY` before initial setup allows you to pre-build vector embeddings from your session data, giving alfred semantic search capabilities from the first conversation.

Uses `voyage-4-large` (2048 dimensions) for maximum retrieval accuracy.

### Building from source

```bash
git clone https://github.com/hir4ta/claude-alfred
cd claude-alfred
go build -o claude-alfred .
```

## Upgrade

Update the plugin inside Claude Code:

```
/plugin marketplace update
```

That's it. The binary is automatically downloaded on the next Claude Code restart. Session sync and embedding generation happen in the background.

## Commands

### `claude-alfred` / `claude-alfred watch`

Monitor a Claude Code session in real-time. Run in a separate terminal or tmux pane.

```bash
# Terminal 1
claude-alfred

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

### `claude-alfred browse`

Browse past session history with the same expand/collapse interface.

```bash
claude-alfred browse
```

---

### `claude-alfred serve`

Run as an MCP server (stdio) for Claude Code integration.

```bash
claude-alfred serve
```

**MCP Tools (6 consolidated):**

| Tool | Description |
|------|-------------|
| `alfred_state` | Session health, statistics, predictions, session list, context recovery, skill context, accuracy metrics (`detail`: brief/standard/outlook/sessions/resume/skill/accuracy) |
| `alfred_knowledge` | Search past patterns, decisions, cross-project insights, and pre-compact history (`scope`: project/global/recall) |
| `alfred_guidance` | Workflow recommendations, alerts, next steps, pending nudges (`focus`: all/alerts/recommendations/next_steps/pending) |
| `alfred_plan` | Task estimation, progress tracking, strategic workflow planning (`mode`: estimate/progress/strategy) |
| `alfred_diagnose` | Error diagnosis + concrete fix patches with before/after code and verification commands |
| `alfred_feedback` | Rate suggestion quality (helpful/partially_helpful/not_helpful/misleading) to improve future relevance |

All tools support `format=concise` for reduced token consumption (summary + key data only).

---

### `claude-alfred analyze [session_id]`

Session analysis report.

```bash
claude-alfred analyze          # Latest session
claude-alfred analyze de999fa4 # Specific session by ID prefix
```

### `claude-alfred uninstall`

Remove hooks and MCP server registration:

```bash
claude-alfred uninstall
```

### `claude-alfred plugin-bundle [output_dir]`

Generate the plugin directory from Go source definitions. Used for development and CI verification.

```bash
claude-alfred plugin-bundle ./plugin
```

## Plugin

claude-alfred is distributed as a Claude Code plugin. The plugin provides:

- **13 hooks**: SessionStart, PreToolUse, PostToolUse, PostToolUseFailure, UserPromptSubmit, PreCompact, SessionEnd, SubagentStart, SubagentStop, Notification, TeammateIdle, TaskCompleted, PermissionRequest
- **5 skills**: alfred-recover, alfred-gate, alfred-analyze, alfred-forecast, alfred-context-recovery
- **1 agent**: alfred (persistent memory, session advisor)
- **MCP server**: 6 consolidated tools for session analysis, feedback, and cross-project knowledge search

### Skills

| Skill | Invocation | Description |
|---|---|---|
| alfred-recover | auto | Failure recovery: stuck loops, error resolution diffs, test debugging |
| alfred-gate | auto | Session health check + pre-commit quality gate |
| alfred-analyze | `/claude-alfred:alfred-analyze` | Blast radius analysis and change review (runs in forked context) |
| alfred-forecast | `/claude-alfred:alfred-forecast` | Task estimation and session prediction dashboard (runs in forked context) |
| alfred-context-recovery | auto | Restore working context after compaction |

## Hooks

Hooks actively monitor your session through Claude Code's lifecycle events:

| Hook Event | Behavior |
|---|---|
| **SessionStart** | Auto-restores session context (working set, decisions, git branch), captures git state |
| **PreToolUse** | Blocks destructive commands; episode early-warning (retry cascade, explore stuck, etc.); velocity wall look-ahead; auto-applies high-confidence code fixes; warns on stale reads, git-dirty files, past failures; surfaces related decisions with resolution diffs |
| **PostToolUse** | Tracks tool/file patterns, code quality heuristics, coverage-aware test failure correlation, suggestion effectiveness with pending verification |
| **UserPromptSubmit** | Classifies intent/task type, injects relevant past knowledge, delivers queued nudges |
| **PreCompact** | Serializes working set (files, intent, decisions, git branch) for automatic restoration |
| **PostToolUseFailure** | Causal WHY explanations for failures, deterministic Go compile error patterns, tracks failure cascades, searches past solutions, starts resolution chains, false-positive detection for nudge resolution |
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

- **Episode early-warning**: Detects emerging anti-patterns (retry cascade, explore stuck, edit-fail spiral, test-fixup fail, context overload) *before* tool execution, not after
- **Velocity wall look-ahead**: Predicts health decline using EWMV variance gating + OLS trend regression, warns ~30 tool calls before threshold breach
- **Auto-apply code fixes**: High-confidence (>=0.9) AST-based patches auto-applied on Edit for Go files (nil-error-wrap, defer-in-loop). Revert tracking for dynamic confidence adjustment
- **Causal WHY explanations**: Every failure diagnostic includes a WHY line explaining the root cause with causal links to recently edited files
- **Deterministic compile error patterns**: 9 Go-specific regex patterns (undefined, type mismatch, unused import, missing return, etc.) matched in <1ms
- **Stale read warning**: File not re-read before editing, or last Read was 8+ tool calls ago
- **Past failure warning**: Similar Bash command failed earlier in the session, with resolution diff display showing previous `old→new` fixes
- **Git dirty file warning**: Editing a file with pre-existing uncommitted changes
- **Code quality analysis**: Go via `go/ast`, Python/JS/TS/Rust via tree-sitter AST — detects unchecked errors, debug prints, bare excepts, mutable defaults, loose equality, hardcoded secrets, TODO without ticket numbers, and more. Includes concrete fix patch generation via CodeFixer
- **Regression probability**: Heuristic combining dependency depth, co-change frequency, exported symbol breadth, and test coverage to estimate change risk. Integrated into blast score for high-impact file warnings
- **Test coverage mapping**: AST-based function→test mapping generates specific `go test -run TestName ./pkg/` suggestions instead of generic "run tests". Coverage map used for causal test failure correlation
- **Test failure correlation**: Connects test failures to recently edited files via coverage map (function-level precision) with fallback to file-list heuristics
- **Workflow guidance**: Learned playbooks from past sessions with concrete file names and test commands; suggests test-first approach for bugfix/refactor tasks
- **Past knowledge surfacing**: Surfaces related decisions, error solutions, and resolution chains (tool sequences) from previous sessions
- **Cross-project learning**: Patterns and decisions are synced to a global DB (`~/.claude-alfred/global.db`) for reuse across projects

**Automatic context recovery** (survives compaction):

Working set (currently edited files, intent, task type, key decisions, git branch) is automatically serialized before compaction and restored afterward. No manual intervention required.

**Suggestion effectiveness tracking**:

Nudge delivery and resolution are tracked across sessions with two-step pending verification (mark pending on resolution action → confirm on next tool success/failure) to reduce false positives. Implicit negative signals are recorded when 4+ tools elapse without resolution. Graduated demotion replaces binary suppression: Stage 1 (15+ deliveries, <15% rate → 50% frequency reduction), Stage 2 (20+, <10% → 80% reduction), Stage 3 (30+, <5% → full suppression). Auto-feedback is skipped when it contradicts explicit user feedback. Explicit feedback via `alfred_feedback` MCP tool is integrated into Thompson Sampling with KL regularization and 3-tier fallback (per-user → contextual → base pattern) for priority adjustment. Token cost estimation tracks relative cost per suggestion delivery. Per-user pattern effectiveness is tracked per project for personalized prioritization.

**Deep intent model**:

4-layer understanding of the user's task: TaskType (bugfix/feature/refactor/test/explore/debug/review/docs), Domain (auth/database/ui/api/config/infra), WorkflowPhase (explore/design/implement/test/integrate), and RiskProfile (conservative/balanced/aggressive). Used for phase-aware suggestion gating and personalized advice.

**User profiling and personalization**:

Behavioral clustering (conservative/balanced/aggressive) based on read-write ratio, test frequency, and session velocity. Profile influences anti-pattern detection thresholds (conservative: 0.7x for earlier warnings, aggressive: 1.5x for higher tolerance), suggestion priority, and delivery timing via phase-aware gating.

## Architecture

```
claude-alfred/
├── main.go                    # Entry point + subcommand routing
├── plugin/                    # Claude Code plugin (generated by plugin-bundle)
│   ├── .claude-plugin/        # Plugin manifest
│   ├── hooks/                 # Hook definitions (14 events)
│   ├── bin/                   # Guard + setup wrapper script
│   ├── skills/                # 5 skills
│   ├── agents/                # Alfred agent
│   └── .mcp.json              # MCP server config
├── .claude-plugin/            # Marketplace manifest
├── internal/
│   ├── parser/                # JSONL parser (type definitions + parsing)
│   ├── watcher/               # File watching (fsnotify + tail)
│   ├── analyzer/              # Live stats + Feedback type + anti-pattern detector
│   ├── coach/                 # AI feedback generation via claude -p
│   ├── hookhandler/           # Hook handlers (advisor signals, code heuristics, test correlation)
│   ├── sessiondb/             # Ephemeral per-session SQLite (working set, burst state, nudges)
│   ├── embedder/              # Voyage AI integration for semantic search
│   ├── tui/                   # Bubble Tea TUI (watch / browse / select)
│   ├── mcpserver/             # MCP server (stdio, 6 consolidated tools)
│   ├── store/                 # SQLite persistence (vector search + LIKE search + incremental sync + global DB + accuracy metrics)
│   └── install/               # Plugin bundle + guard/setup wrapper + initial sync
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
| [odvcencio/gotreesitter](https://github.com/odvcencio/gotreesitter) | Pure Go tree-sitter runtime for multi-language AST analysis |

## Semantic Search

Voyage AI (`voyage-4-large`, 2048 dimensions) powers `alfred_knowledge` and hook-based knowledge injection via vector semantic search. Set `VOYAGE_API_KEY` to enable.

Without `VOYAGE_API_KEY`, knowledge search falls back to FTS5 BM25 / LIKE — all features work, just without semantic matching. FTS5 uses phrase-first search for multi-word queries (higher precision), falling back to OR-based search, with title-match reordering for relevance.
