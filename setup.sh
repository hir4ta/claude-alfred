#!/bin/sh
# claude-alfred initial setup
# Usage: curl -fsSL https://raw.githubusercontent.com/hir4ta/claude-alfred/main/setup.sh | sh
#
# What this does:
#   1. Finds the plugin in ~/.claude/plugins/cache/
#   2. Downloads the claude-alfred binary
#   3. Syncs past sessions and generates embeddings
#
# Requires VOYAGE_API_KEY for vector search:
#   export VOYAGE_API_KEY=pa-xxx
#   curl -fsSL https://raw.githubusercontent.com/hir4ta/claude-alfred/main/setup.sh | sh

set -e

echo "claude-alfred setup"
echo "==================="
echo ""

# Find the latest plugin installation.
RUN_SH=$(find ~/.claude/plugins/cache -name "run.sh" -path "*/claude-alfred/*/bin/*" -type f 2>/dev/null | sort -V | tail -1)

if [ -z "$RUN_SH" ]; then
  echo "Error: claude-alfred plugin not found." >&2
  echo "" >&2
  echo "Install it first in Claude Code:" >&2
  echo "  /plugin marketplace add hir4ta/claude-alfred" >&2
  echo "  /plugin install claude-alfred@claude-alfred" >&2
  exit 1
fi

PLUGIN_DIR=$(dirname "$(dirname "$RUN_SH")")
echo "Plugin found: $PLUGIN_DIR"
echo ""

# Voyage AI check (required).
if [ -z "$VOYAGE_API_KEY" ]; then
  echo "Error: VOYAGE_API_KEY is required but not set." >&2
  echo "  export VOYAGE_API_KEY=pa-xxx" >&2
  echo "  Then re-run this setup script." >&2
  exit 1
fi
echo "VOYAGE_API_KEY: set"
echo ""

# Show estimated time.
INFO=$(sh "$RUN_SH" count-sessions 2>/dev/null || echo '{}')
SESSIONS=$(echo "$INFO" | grep -o '"sessions":[0-9]*' | grep -o '[0-9]*')
EST_MIN=$(echo "$INFO" | grep -o '"est_minutes":[0-9]*' | grep -o '[0-9]*')
if [ -n "$SESSIONS" ] && [ "$SESSIONS" -gt 0 ]; then
  echo "Found $SESSIONS sessions to sync (~${EST_MIN} min)"
else
  echo "No existing sessions found (fresh install)"
fi
echo ""

# Run setup (downloads binary + syncs sessions + generates embeddings).
sh "$RUN_SH" setup

echo ""
echo "Done! Restart Claude Code to activate hooks and MCP tools."
