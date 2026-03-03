---
name: harvest
description: >
  Manually refresh the alfred knowledge base. Normally auto-harvest keeps
  docs fresh automatically — use this for forced full crawl or targeted
  page updates.
user-invocable: true
allowed-tools: Bash, mcp__alfred__knowledge
context: current
---

The butler's procurement run — gathering the finest ingredients for the knowledge base.

## Steps

1. **[HOW]** Run the harvest CLI command:
   ```bash
   alfred harvest
   ```
   This crawls all documentation sources (official docs, changelog, engineering blog),
   upserts into the knowledge base, and generates embeddings.
   The command shows a TUI progress display.

2. **[WHAT]** Verify the result:
   - Call `knowledge` with query="Claude Code hooks" (limit=1) to confirm docs are fresh
   - Report the harvest result to the user

## Guardrails

- Do NOT use WebFetch or ingest MCP — the CLI handles everything natively
- Do NOT run harvest if VOYAGE_API_KEY is not set (the CLI will error)
