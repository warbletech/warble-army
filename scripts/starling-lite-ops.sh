#!/usr/bin/env bash
# Starling Lite ops helper (Linux/macOS)
# Build, smoke-test, Railway deploy, optional Cloudflare DNS for custom domain.
#
# Usage:
#   ./scripts/starling-lite-ops.sh help
#   ./scripts/starling-lite-ops.sh smoke
#   ./scripts/starling-lite-ops.sh smoke --url https://p4a4cpbh.up.railway.app
#   ./scripts/starling-lite-ops.sh railway-up
#   ./scripts/starling-lite-ops.sh railway-domain
#   ./scripts/starling-lite-ops.sh cloudflare-dns --host mcp --zone warbleoss.org
#   ./scripts/starling-lite-ops.sh all   # smoke public + print status
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RAILWAY_URL_DEFAULT="${RAILWAY_PUBLIC_URL:-https://starling-lite-production.up.railway.app}"
MCP_PATH="${MCP_PATH:-/mcp}"
HEALTH_PATH="${HEALTH_PATH:-/health}"

log()  { printf '==> %s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing dependency: $1"; }

cmd_help() {
  sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
}

cmd_build() {
  need go
  log "building starling-lite"
  mkdir -p "$ROOT/bin"
  (cd "$ROOT/mcp/starling-lite" && go build -o "$ROOT/bin/starling-lite" ./cmd/starling-lite)
  log "built $ROOT/bin/starling-lite"
}

cmd_run-local() {
  cmd_build
  export STARLING_MODE="${STARLING_MODE:-readonly}"
  export STARLING_TRANSPORT=http
  export PORT="${PORT:-8822}"
  log "listening on :$PORT (Ctrl-C to stop)"
  exec "$ROOT/bin/starling-lite" \
    --mode="$STARLING_MODE" \
    --transport=http \
    --registry="$ROOT/mcp/tools/registry.json"
}

cmd_smoke() {
  local url="${1:-$RAILWAY_URL_DEFAULT}"
  # allow --url flag style
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --url) url="$2"; shift 2 ;;
      *) url="$1"; shift ;;
    esac
  done
  need curl
  need python3
  url="${url%/}"
  log "health  $url$HEALTH_PATH"
  curl -fsS --max-time 30 "$url$HEALTH_PATH" | python3 -m json.tool
  log "tools/list  $url$MCP_PATH"
  local n
  n=$(curl -fsS --max-time 60 -X POST "$url$MCP_PATH" \
    -H 'Content-Type: application/json' \
    -H 'MCP-Protocol-Version: 2026-07-28' \
    -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
    | python3 -c 'import sys,json; r=json.load(sys.stdin); print(len(r.get("result",{}).get("tools",[])))')
  log "tools count: $n"
  [[ "$n" -gt 0 ]] || die "no tools returned"
  log "smoke OK"
}

cmd_railway-up() {
  need railway
  railway whoami >/dev/null || die "run: railway login"
  log "deploying from $ROOT (Dockerfile deploy/railway/Dockerfile)"
  if [[ ! -f "$ROOT/railway.toml" ]]; then
    cp "$ROOT/deploy/railway/railway.toml" "$ROOT/railway.toml" 2>/dev/null || true
  fi
  railway variable set \
    STARLING_MODE=readonly \
    STARLING_TRANSPORT=http \
    STARLING_REGISTRY=/etc/warble-army/registry.json \
    --skip-deploys 2>/dev/null || true
  railway up -y -d
  log "deploy queued — check: railway status"
  railway domain 2>/dev/null || true
  railway status || true
}

cmd_railway-domain() {
  need railway
  railway domain --json
  railway domain list --json
}

cmd_railway-status() {
  need railway
  railway status
  railway domain list 2>/dev/null || true
}

# Cloudflare DNS: CNAME host → Railway service domain
# Requires: CLOUDFLARE_API_TOKEN and CLOUDFLARE_ZONE_ID
# Or: gcloud secrets cloudflare-api-token + cloudflare-zone-id
cmd_cloudflare-dns() {
  local host="mcp"
  local zone_name=""
  local target="${RAILWAY_CNAME_TARGET:-p4a4cpbh.up.railway.app}"
  local proxied="true"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --host) host="$2"; shift 2 ;;
      --zone) zone_name="$2"; shift 2 ;;
      --target) target="$2"; shift 2 ;;
      --dns-only) proxied="false"; shift ;;
      *) die "unknown arg: $1" ;;
    esac
  done

  need curl
  need python3

  local token="${CLOUDFLARE_API_TOKEN:-}"
  local zone_id="${CLOUDFLARE_ZONE_ID:-}"

  if [[ -z "$token" ]] && command -v gcloud >/dev/null 2>&1; then
    log "loading CLOUDFLARE_API_TOKEN from gcloud secret"
    token="$(gcloud secrets versions access latest --secret=cloudflare-api-token 2>/dev/null || true)"
  fi
  if [[ -z "$zone_id" ]] && command -v gcloud >/dev/null 2>&1; then
    if [[ "$zone_name" == *frakma* ]] || [[ "$zone_name" == "frakma.io" ]]; then
      zone_id="$(gcloud secrets versions access latest --secret=cloudflare-zone-id-frakma 2>/dev/null || true)"
    else
      zone_id="$(gcloud secrets versions access latest --secret=cloudflare-zone-id 2>/dev/null || true)"
    fi
  fi

  [[ -n "$token" ]] || die "set CLOUDFLARE_API_TOKEN (or gcloud secret cloudflare-api-token)"
  [[ -n "$zone_id" ]] || die "set CLOUDFLARE_ZONE_ID (or gcloud secret cloudflare-zone-id)"

  # Resolve zone by name if needed
  if [[ -n "$zone_name" && -z "$zone_id" ]]; then
    zone_id=$(curl -fsS "https://api.cloudflare.com/client/v4/zones?name=${zone_name}" \
      -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
      | python3 -c 'import sys,json; r=json.load(sys.stdin); print(r["result"][0]["id"] if r.get("result") else "")')
  fi

  local fqdn="$host"
  if [[ -n "$zone_name" && "$host" != *.* ]]; then
    fqdn="${host}.${zone_name}"
  fi

  log "upsert CNAME $fqdn → $target (proxied=$proxied)"

  # Find existing record
  local rec_id
  rec_id=$(curl -fsS "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records?type=CNAME&name=${fqdn}" \
    -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
    | python3 -c 'import sys,json; r=json.load(sys.stdin); print(r["result"][0]["id"] if r.get("result") else "")')

  local body
  body=$(python3 -c "import json; print(json.dumps({'type':'CNAME','name':'''$host''','content':'''$target''','ttl':1,'proxied': $proxied}))")

  if [[ -n "$rec_id" ]]; then
    curl -fsS -X PUT "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records/${rec_id}" \
      -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
      -d "$body" | python3 -m json.tool
  else
    curl -fsS -X POST "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records" \
      -H "Authorization: Bearer $token" -H "Content-Type: application/json" \
      -d "$body" | python3 -m json.tool
  fi

  # Also register custom domain on Railway if linked
  if command -v railway >/dev/null 2>&1 && railway status >/dev/null 2>&1; then
    log "adding Railway custom domain $fqdn (if not already)"
    railway domain "$fqdn" --json 2>/dev/null || railway domain "$fqdn" 2>/dev/null || true
    railway domain status "$fqdn" 2>/dev/null || true
  fi

  log "DNS set. Wait for propagation, then: $0 smoke --url https://$fqdn"
}

cmd_all() {
  cmd_railway-status 2>/dev/null || true
  cmd_smoke --url "$RAILWAY_URL_DEFAULT"
}

main() {
  local cmd="${1:-help}"
  shift || true
  case "$cmd" in
    help|-h|--help) cmd_help ;;
    build) cmd_build "$@" ;;
    run-local|local) cmd_run-local "$@" ;;
    smoke|test) cmd_smoke "$@" ;;
    railway-up|deploy) cmd_railway-up "$@" ;;
    railway-domain|domain) cmd_railway-domain "$@" ;;
    railway-status|status) cmd_railway-status "$@" ;;
    cloudflare-dns|cf-dns|dns) cmd_cloudflare-dns "$@" ;;
    all) cmd_all "$@" ;;
    *) die "unknown command: $cmd (try: help)" ;;
  esac
}

main "$@"
