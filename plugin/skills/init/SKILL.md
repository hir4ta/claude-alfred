---
name: init
description: >
  Initialize claude-buddy: download binary, sync sessions, and optionally
  configure semantic search. Run after /plugin install or /plugin marketplace update.
user-invocable: true
allowed-tools: Bash, AskUserQuestion
---

Initialize claude-buddy for this system.

## Steps

1. Find the plugin installation directory:
   ```bash
   find ~/.claude/plugins/cache -name "run.sh" -path "*/claude-buddy/*/bin/*" -type f 2>/dev/null | head -1
   ```

2. Ask the user if they want to enable semantic search (Voyage AI).
   If yes, ask for their API key and add it to their shell profile:
   ```bash
   echo 'export VOYAGE_API_KEY=<key>' >> ~/.$(basename "$SHELL")rc
   source ~/.$(basename "$SHELL")rc
   ```

3. Run the setup script (downloads binary + syncs sessions):
   ```bash
   sh <path-to-run.sh> setup
   ```

4. Verify the installation:
   ```bash
   sh <path-to-run.sh> version
   ```

## Output

- Installation status with version
- Semantic search status (enabled/disabled)
- Tell the user to restart Claude Code to activate hooks and MCP tools
