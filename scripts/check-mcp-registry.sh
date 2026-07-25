#!/usr/bin/env bash
# Validate mcp/tools JSON: parseable, unique names, registry covers groups.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TOOLS_DIR="mcp/tools"
REGISTRY="$TOOLS_DIR/registry.json"
GROUPS=(cluster scan remediate)

die() { echo "check-mcp-registry: $*" >&2; exit 1; }

command -v python3 >/dev/null || die "python3 required"

[[ -f "$REGISTRY" ]] || die "missing $REGISTRY"

python3 - <<'PY'
import json, sys
from pathlib import Path

root = Path("mcp/tools")
registry_path = root / "registry.json"
groups = ["cluster", "scan", "remediate"]

def load(p: Path):
    try:
        return json.loads(p.read_text())
    except Exception as e:
        print(f"invalid JSON {p}: {e}", file=sys.stderr)
        sys.exit(1)

reg = load(registry_path)
if "tools" not in reg or not isinstance(reg["tools"], list):
    print("registry.json must have tools: []", file=sys.stderr)
    sys.exit(1)

reg_names = []
for t in reg["tools"]:
    if "name" not in t:
        print("registry tool missing name", file=sys.stderr)
        sys.exit(1)
    reg_names.append(t["name"])
    if "inputSchema" not in t:
        print(f"{t['name']}: missing inputSchema", file=sys.stderr)
        sys.exit(1)

if len(reg_names) != len(set(reg_names)):
    from collections import Counter
    dups = [n for n, c in Counter(reg_names).items() if c > 1]
    print(f"duplicate names in registry: {dups}", file=sys.stderr)
    sys.exit(1)

group_names = set()
for g in groups:
    p = root / f"{g}.json"
    if not p.exists():
        print(f"missing {p}", file=sys.stderr)
        sys.exit(1)
    data = load(p)
    tools = data.get("tools") or []
    for t in tools:
        n = t.get("name")
        if not n:
            print(f"{p}: tool without name", file=sys.stderr)
            sys.exit(1)
        if n in group_names:
            print(f"duplicate across groups: {n}", file=sys.stderr)
            sys.exit(1)
        group_names.add(n)

reg_set = set(reg_names)
if reg_set != group_names:
    only_reg = sorted(reg_set - group_names)
    only_grp = sorted(group_names - reg_set)
    if only_reg:
        print(f"in registry but not in group files: {only_reg}", file=sys.stderr)
    if only_grp:
        print(f"in group files but not in registry: {only_grp}", file=sys.stderr)
    sys.exit(1)

print(f"ok: {len(reg_names)} tools across {len(groups)} groups")
PY
