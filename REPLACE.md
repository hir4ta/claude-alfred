# Go → TypeScript Rewrite Plan

## Background

alfred was started as a Go CLI tool (session monitor TUI) in early 2026. As it evolved into a Claude Code plugin with MCP server, hook handlers, web dashboard, and 15+ skills, the Go/TypeScript split became a maintenance burden rather than an advantage.

### Why Go was chosen originally
- Single binary distribution (no runtime dependency)
- Fast cold start for hook handlers
- ncruces/go-sqlite3 (pure Wasm, no CGO) for portable SQLite

### Why it no longer makes sense
- **All Claude Code users have Node.js** — npm distribution eliminates GoReleaser, Homebrew, binary caching, cross-compilation
- **Two languages for one product** — Go backend + TypeScript web dashboard means duplicated type definitions, separate build toolchains, different error handling patterns
- **MCP SDK** — TypeScript version is Anthropic's first-class implementation; Go SDK (mcp-go) is community-maintained
- **Plugin ecosystem** — Claude Code plugins are fundamentally Node.js; bin/run.sh bootstrap is a workaround for Go binaries
- **Contributor barrier** — Claude Code's target audience skews heavily TypeScript/JavaScript
- **Go's advantages don't apply here** — Hook timeouts are 5-10s (Node.js ~150ms startup is fine); memory usage irrelevant for dev tools; concurrency model unused (hooks are short-lived, MCP is single-request)

## Current Codebase (as of v0.76.0)

| Component | Language | Lines | Role |
|-----------|----------|-------|------|
| cmd/alfred/ | Go | ~4,000 | CLI dispatch, hook handlers, dashboard launcher |
| internal/store/ | Go | ~2,500 | SQLite persistence, knowledge files, FTS5, vectors |
| internal/mcpserver/ | Go | ~2,000 | MCP tool handlers (dossier, roster, ledger) |
| internal/spec/ | Go | ~2,500 | Spec lifecycle, validation, templates, steering |
| internal/epic/ | Go | ~500 | Epic YAML management |
| internal/embedder/ | Go | ~400 | Voyage AI client |
| internal/api/ | Go | ~800 | HTTP REST + SSE server |
| internal/dashboard/ | Go | ~600 | DataSource interface |
| internal/install/ | Go | ~500 | Plugin bundle generation |
| web/ | TypeScript | ~3,000 | React SPA (Vite + TanStack) |
| **Total** | | **~17,000** | |

## Target Architecture

```
src/
  cli/           — CLI entry point (commander.js or similar)
  hooks/         — Hook event handlers (SessionStart, PreCompact, etc.)
  mcp/           — MCP server (dossier, roster, ledger tools)
  store/         — SQLite persistence (sql.js or better-sqlite3)
  spec/          — Spec lifecycle, validation, templates
  epic/          — Epic YAML management
  embedder/      — Voyage AI client (fetch-based)
  api/           — HTTP REST + SSE server (Express or Hono)
  dashboard/     — DataSource interface
web/             — React SPA (unchanged, shares types directly)
```

### Key Technology Choices

| Concern | Go (current) | TypeScript (target) |
|---------|-------------|-------------------|
| SQLite | ncruces/go-sqlite3 (Wasm) | better-sqlite3 (native) or sql.js (Wasm) |
| MCP | mcp-go (community) | @anthropic-ai/sdk (official) |
| HTTP | chi/v5 | Hono or Express |
| Templates | text/template + go:embed | Template literals or Handlebars |
| Embedding | go:embed for SPA + templates | Bundler (esbuild/tsup) |
| Distribution | GoReleaser + Homebrew | npm publish |
| CLI | manual dispatch | commander.js or yargs |
| YAML | gopkg.in/yaml.v3 | js-yaml |

## Migration Strategy

### Phase 1: Foundation (store + types)
- Port SQLite schema, migrations, CRUD operations
- Port KnowledgeRow, SpecDir, EpicFile types
- Port FTS5, vector search, content_hash
- **Validation**: Run Go and TS implementations side-by-side against same DB

### Phase 2: MCP Server
- Port dossier, roster, ledger handlers
- Switch from mcp-go to @anthropic-ai/sdk
- Port spec validation (22 checks), confidence parsing, review system
- **Validation**: MCP tool responses match Go version for same inputs

### Phase 3: Hook Handlers
- Port SessionStart, PreCompact, UserPromptSubmit, PostToolUse
- Port transcript parsing, decision extraction, skill nudge
- Port autoappend, drift detection
- **Validation**: Hook stderr/stdout output matches Go version

### Phase 4: CLI + Dashboard
- Port CLI dispatch, dashboard launcher, steering-init
- Unify web/ types (direct imports instead of duplicated definitions)
- Port plugin-bundle generation
- **Validation**: `alfred dashboard`, `alfred version`, all subcommands work

### Phase 5: Distribution
- npm package with bin entry point
- Remove GoReleaser, Homebrew tap, binary bootstrap (bin/run.sh)
- Update Claude Code plugin to use npm directly
- **Validation**: `/plugin install alfred` works via npm

## Risks

| Risk | Mitigation |
|------|-----------|
| SQLite native addon portability | Use sql.js (Wasm) as fallback; better-sqlite3 prebuild covers most platforms |
| Performance regression in hooks | Benchmark critical paths; Node.js startup + sql.js should still be <500ms |
| Feature parity gaps during migration | Phase-by-phase with side-by-side validation; don't ship partial migration |
| Breaking change for existing users | Major version bump (v1.0.0); migration guide for .alfred/ data (schema is identical) |
| go:embed replacement | esbuild/tsup can inline templates; or read from node_modules at runtime |

## What NOT to change

- `.alfred/` directory structure (specs, knowledge, steering, templates, epics)
- DB schema V8 (knowledge_index, embeddings, FTS5, session_links)
- MCP tool interfaces (dossier, roster, ledger actions and parameters)
- Skill files (SKILL.md format is language-agnostic)
- Hook event protocol (JSON stdin, stdout output, stderr notifications)
- Markdown+frontmatter knowledge file format

## When to do this

Not now. V8 (knowledge-first architecture) just shipped. Let it stabilize for a few weeks/months. Ideal timing:
- After V8 has been used in production across multiple projects
- After any V8 bugs are shaken out
- When the next major feature push is planned (v1.0.0 candidate)
