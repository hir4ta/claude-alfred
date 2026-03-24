#!/bin/bash
set -euo pipefail

# alfred installer — downloads the latest binary and sets up the environment.
# Usage: curl -fsSL https://raw.githubusercontent.com/hir4ta/claude-alfred/main/install.sh | bash

REPO="hir4ta/claude-alfred"
INSTALL_DIR="${HOME}/.local/bin"
ALFRED_DIR="${HOME}/.claude-alfred"

# Detect OS and architecture
detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *)      echo "Unsupported OS: $os" >&2; exit 1 ;;
  esac

  case "$arch" in
    x86_64|amd64)  arch="x64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)             echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac

  echo "${os}-${arch}"
}

# Get latest release tag from GitHub
get_latest_version() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

main() {
  local platform version url

  platform="$(detect_platform)"
  echo "Detected platform: ${platform}"

  echo "Fetching latest version..."
  version="$(get_latest_version)"
  echo "Latest version: ${version}"

  url="https://github.com/${REPO}/releases/download/${version}/alfred-${platform}"

  # Download binary
  echo "Downloading alfred ${version} for ${platform}..."
  mkdir -p "${INSTALL_DIR}"
  curl -fsSL "${url}" -o "${INSTALL_DIR}/alfred"
  chmod +x "${INSTALL_DIR}/alfred"

  # Setup database
  echo "Setting up alfred..."
  mkdir -p "${HOME}/.claude-alfred"
  "${INSTALL_DIR}/alfred" doctor 2>/dev/null || true

  # Install user rules (ensures MCP tool usage + spec workflow instructions are always loaded)
  local rules_dir="${HOME}/.claude/rules"
  mkdir -p "${rules_dir}"

  cat > "${rules_dir}/alfred.md" << 'RULES'
# alfred MCP Tools

alfred's knowledge base contains curated Claude Code docs and best practices with vector search.

## knowledge — Search docs and best practices

**ALWAYS call knowledge BEFORE** answering questions about Claude Code. Do not guess or rely on training data.

Call when the user's question or task involves ANY of:
- Hooks, skills, rules, agents, plugins, MCP servers, CLAUDE.md, memory
- Permissions, settings, compaction, CLI features, IDE integrations
- Best practices for Claude Code configuration or workflow
- Evaluating whether code follows Claude Code conventions

Do NOT call for: general programming, project-specific code, non-Claude-Code topics.

## config-review — Audit .claude/ config against best practices

Call when:
- Reviewing or auditing `.claude/` configuration
- Evaluating CLAUDE.md quality or looking for improvements
- Checking overall Claude Code setup health
RULES

  cat > "${rules_dir}/alfred-protocol.md" << 'RULES'
# Alfred Protocol — Spec-Driven Development

When a `.alfred/specs/` directory exists in the project, follow this protocol strictly.

## Spec Creation (User-Initiated)

Spec creation is triggered by the user, not auto-proposed by Claude.
- User explicitly requests: "spec作って", "/alfred:brief", "specを作成して" etc.
- When requested: ask for size (S/M/L) → create via `dossier action=init`
- Implementation without a spec is normal and allowed

## Task Tracking

- After completing each task, explicitly call `dossier action=check task_id="T-X.Y"`
- Do NOT rely solely on auto-detection — always confirm task completion explicitly
- Record decisions via `ledger action=save sub_type=decision` as they happen

## Wave Completion — Mandatory Review Gate

When all tasks in a Wave are done:

1. **Commit** with Wave number in message
2. **Self-review** via `alfred:code-reviewer` agent or `/alfred:inspect`
3. **Fix** Critical/High findings before proceeding
4. **Gate clear** — `dossier action=gate sub_action=clear reason="<review summary>"` (30+ chars, include: review method, findings count, fix summary)
5. **Knowledge** — `ledger action=save` (pattern/decision/rule). If nothing to save, state why
6. **Next Wave** — Proceed immediately, do NOT stop and wait for user input

## Completing a Spec

- Call `dossier action=complete` to close the spec
- Prefer complete over delete — completed specs serve as searchable knowledge
RULES
  echo "Rules installed: ${rules_dir}/alfred.md, alfred-protocol.md"

  # Ensure ~/.local/bin is in PATH
  if ! echo "$PATH" | grep -q "${INSTALL_DIR}"; then
    local shell_rc=""
    case "$(basename "$SHELL")" in
      zsh)  shell_rc="${HOME}/.zshrc" ;;
      bash) shell_rc="${HOME}/.bashrc" ;;
      fish) shell_rc="${HOME}/.config/fish/config.fish" ;;
    esac
    if [ -n "$shell_rc" ]; then
      echo "" >> "$shell_rc"
      echo "# alfred" >> "$shell_rc"
      echo "export PATH=\"${INSTALL_DIR}:\$PATH\"" >> "$shell_rc"
      echo "Added ${INSTALL_DIR} to PATH in ${shell_rc}"
    fi
  fi

  echo ""
  echo "✓ alfred ${version} installed to ${INSTALL_DIR}/alfred"
  echo ""
  echo "Next steps:"
  echo "  1. Restart your shell or run: export PATH=\"${INSTALL_DIR}:\$PATH\""
  echo "  2. (Optional) Add to ~/.zshrc: export VOYAGE_API_KEY=your-key"
  echo "  3. In Claude Code:"
  echo "     /plugin marketplace add ${REPO}"
  echo "     /plugin install alfred"
  echo "  4. Restart Claude Code"
  echo ""
  echo "Commands:"
  echo "  alfred dashboard    # Web dashboard"
  echo "  alfred tui          # Terminal progress viewer"
  echo "  alfred doctor       # Check installation"
}

main
