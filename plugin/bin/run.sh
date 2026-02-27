#!/bin/sh
BUDDY_VERSION="0.13.4"
BIN_DIR="$(cd "$(dirname "$0")" && pwd)"
BUDDY_BIN="${BIN_DIR}/claude-buddy"

case "$1" in
  setup)
    set -e
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case "$ARCH" in
      x86_64)  ARCH="amd64" ;;
      aarch64) ARCH="arm64" ;;
    esac
    URL="https://github.com/hir4ta/claude-buddy/releases/download/v${BUDDY_VERSION}/claude-buddy_${OS}_${ARCH}.tar.gz"
    curl -fsSL "$URL" | tar -xz -C "${BIN_DIR}" claude-buddy
    chmod +x "$BUDDY_BIN"
    echo "claude-buddy ${BUDDY_VERSION} installed"
    exec "$BUDDY_BIN" install
    ;;
  *)
    if [ ! -f "$BUDDY_BIN" ]; then
      case "$1" in
        hook-handler)
          echo '{"additionalContext":"[claude-buddy] Not initialized. Run /claude-buddy:init to set up."}'
          exit 0
          ;;
        *)
          echo "claude-buddy not initialized. Run /claude-buddy:init to set up." >&2
          exit 1
          ;;
      esac
    fi
    exec "$BUDDY_BIN" "$@"
    ;;
esac
