#!/bin/bash
set -euo pipefail

# Build single-file executables for all supported platforms.
# Patches @opentui/core per platform, then compiles directly from source.
# Requires Bun 1.3+

OUT_DIR="dist/binaries"

mkdir -p "$OUT_DIR"

TARGETS=(
  "bun-darwin-arm64"
  "bun-darwin-x64"
  "bun-linux-arm64"
  "bun-linux-x64"
)

for target in "${TARGETS[@]}"; do
  platform="${target#bun-}"
  os="${platform%-*}"
  arch="${platform##*-}"
  outfile="${OUT_DIR}/alfred-${platform}"

  # Patch @opentui/core for this platform
  bash scripts/patch-opentui.sh "$os" "$arch"

  echo "Compiling for ${platform}..."
  bun build.ts --compile \
    --target="$target" \
    --outfile="$outfile"

  echo "  → ${outfile} ($(du -h "$outfile" | cut -f1))"

  # Restore original @opentui/core for next platform
  rm -rf node_modules/@opentui/core && bun install --frozen-lockfile 2>/dev/null
done

echo ""
echo "All binaries built in ${OUT_DIR}/"
ls -lh "$OUT_DIR"/
