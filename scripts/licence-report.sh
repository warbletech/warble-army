#!/usr/bin/env bash
# Regenerate the third-party dependency licence CSVs under docs/cncf/licenses/.
# Needs: Go, go-licenses v2 (go install github.com/google/go-licenses/v2@latest), python3,
# and initialised submodules (./scripts/submodules.sh init).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/docs/cncf/licenses"
mkdir -p "$OUT"

go_report() { # <module dir> <output name>
  (cd "$ROOT/$1" && go-licenses report ./... 2>/dev/null) \
    | grep -v 'github.com/warbletech/' \
    | sort -u > "$OUT/$2.csv"
  echo "wrote $OUT/$2.csv ($(wc -l < "$OUT/$2.csv") third-party packages)"
}

go_report mcp/starling-lite go-starling-lite
go_report heron-ops        go-heron-ops
go_report shrikeops/scanner go-shrikeops-scanner

# npm: package-lock.json v3 records each package's declared licence.
python3 - "$ROOT/shrikeops/frontend/package-lock.json" "$OUT/npm-shrikeops-frontend.csv" <<'PY'
import csv, json, sys
lock = json.load(open(sys.argv[1]))
rows = []
for path, meta in lock.get("packages", {}).items():
    if not path:
        continue
    lic = meta.get("license", "UNKNOWN")
    if isinstance(lic, dict):
        lic = lic.get("type", "UNKNOWN")
    scope = "dev" if meta.get("dev") else ("optional" if meta.get("optional") else "prod")
    rows.append((path.split("node_modules/")[-1], meta.get("version", ""), lic, scope))
with open(sys.argv[2], "w", newline="") as f:
    w = csv.writer(f)
    w.writerow(["package", "version", "license", "scope"])
    w.writerows(sorted(set(rows)))
print(f"wrote {sys.argv[2]} ({len(rows)} packages)")
PY
