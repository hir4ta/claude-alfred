#!/bin/bash
set -euo pipefail

# Patch @opentui/core's dynamic platform import to a static one.
# Enables bun build --compile to resolve and embed the native .dylib.
#
# Usage: scripts/patch-opentui.sh <os> <arch>
#   e.g. scripts/patch-opentui.sh darwin arm64

if [ $# -ne 2 ]; then
  echo "Usage: $0 <os> <arch>"
  exit 1
fi

OS="$1"
ARCH="$2"
PLATFORM="${OS}-${ARCH}"
CORE_DIR="node_modules/@opentui/core"

TARGET_FILE=$(grep -l 'process.platform.*process.arch' "$CORE_DIR"/*.js 2>/dev/null | head -1 || true)

if [ -z "$TARGET_FILE" ]; then
  echo "Error: Could not find @opentui/core file with dynamic platform import"
  exit 1
fi

echo "Patching $TARGET_FILE for platform: $PLATFORM"

sed -i.bak "s|await import(\`@opentui/core-\${process.platform}-\${process.arch}/index.ts\`)|await import(\"@opentui/core-${PLATFORM}/index.ts\")|" "$TARGET_FILE"
rm -f "${TARGET_FILE}.bak"

MATCH_COUNT=$(grep -c "await import(\"@opentui/core-${PLATFORM}/index.ts\")" "$TARGET_FILE" || true)
if [ "$MATCH_COUNT" -ne 1 ]; then
  echo "Error: Expected 1 patched import, found $MATCH_COUNT"
  exit 1
fi

DYNAMIC_COUNT=$(grep -c '@opentui/core-\${process' "$TARGET_FILE" || true)
if [ "$DYNAMIC_COUNT" -ne 0 ]; then
  echo "Error: Dynamic platform import still present"
  exit 1
fi

echo "Patch applied: @opentui/core-${PLATFORM}"
