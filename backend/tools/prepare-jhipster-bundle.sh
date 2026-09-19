#!/usr/bin/env bash
# GEN-02: prepare a downloadable JHipster bundle from a UML diagram JSON.
#
# Read-only safe: this script only writes the NEW bundle directory it creates.
# It never touches the Go runtime, its database, or any existing source.
#
# Usage:
#   prepare-jhipster-bundle.sh <diagram.json> <app-name> [out-dir]
#
# Example:
#   ./gobackend/tools/prepare-jhipster-bundle.sh /tmp/shop.json shop ./shop-jhipster
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <diagram.json> <app-name> [out-dir]" >&2
  exit 1
fi

DIAGRAM="$1"
APP="$2"
OUT="${3:-${APP}-jhipster}"

if [ ! -f "$DIAGRAM" ]; then
  echo "prepare-jhipster-bundle: diagram file not found: $DIAGRAM" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "prepare-jhipster-bundle: go toolchain not found in PATH" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../" && pwd)"
cd "$ROOT"

go run ./cmd/jdl-bundle -diagram "$DIAGRAM" -app "$APP" -out "$OUT"

cat <<EOF

Bundle prepared. JHipster is NOT installed here, so generation happens elsewhere:
  1. Copy $OUT to a machine with Node.js LTS, Java 17+, and network access.
  2. cd $OUT && ./generate.sh   (set PINNED_JHIPSTER to the exact generator version)
  3. Review report.json warnings BEFORE pointing the generated app at any database.

Rule: the generated project must NEVER point at the Go runtime database.
EOF
