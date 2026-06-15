#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ICON_SVG="$ROOT/assets/brand/icon.svg"
OUT_DIR="$ROOT/build"

mkdir -p "$OUT_DIR"
qlmanage -t -s 1024 -o "$OUT_DIR" "$ICON_SVG" >/dev/null
mv -f "$OUT_DIR/icon.svg.png" "$OUT_DIR/appicon.png"
echo "Wrote $OUT_DIR/appicon.png"
