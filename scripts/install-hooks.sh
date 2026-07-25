#!/usr/bin/env bash
# Point this repo at .githooks/ (committed hooks).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: not a git repository: $ROOT" >&2
  exit 1
fi

HOOKS_DIR="$ROOT/.githooks"
if [[ ! -d "$HOOKS_DIR" ]]; then
  echo "error: missing $HOOKS_DIR" >&2
  exit 1
fi

chmod +x "$HOOKS_DIR"/* 2>/dev/null || true
chmod +x "$ROOT/scripts"/*.sh 2>/dev/null || true

git config core.hooksPath .githooks
echo "core.hooksPath = .githooks"
echo "hooks installed:"
ls -1 "$HOOKS_DIR"
echo
echo "Verify: git config --get core.hooksPath"
