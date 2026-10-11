#!/usr/bin/env bash
# Starling EX Community Edition installer / local preview
# https://warbleoss.org · https://github.com/warbletech/warble-army
#
# Usage:
#   ./scripts/install.sh                 # same as local
#   ./scripts/install.sh local           # build + run MCP on :8822 (no cluster deploy)
#   ./scripts/install.sh docker          # docker image + run on :8822
#   ./scripts/install.sh helm            # helm upgrade --install chart (uses current kube context)
#   ./scripts/install.sh helm-template   # dry-run render only (safe preview)
#   ./scripts/install.sh smoke [URL]     # health + tools/list
#   ./scripts/install.sh down            # stop local process / docker container
#   ./scripts/install.sh status
#
# Env:
#   CE_PORT=8822  CE_NS=warble-army  CE_RELEASE=starling-ex-ce
#   CE_MODE=readonly  CE_IMAGE=starling-lite:ce
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CE_PORT="${CE_PORT:-8822}"
CE_NS="${CE_NS:-warble-army}"
CE_RELEASE="${CE_RELEASE:-starling-ex-ce}"
CE_MODE="${CE_MODE:-readonly}"
CE_IMAGE="${CE_IMAGE:-starling-lite:ce}"
CE_PID_FILE="${CE_PID_FILE:-/tmp/starling-ex-ce.pid}"
CE_CTR_NAME="${CE_CTR_NAME:-starling-ex-ce}"
CHART="$ROOT/charts/starling-ex-ce"
PUBLIC_DEMO="${PUBLIC_DEMO:-https://mcp.warbleoss.org}"

log()  { printf '==> %s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing: $1"; }

cmd_help() {
  sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'
}

cmd_local() {
  need go
  log "building starling-lite (local CE preview)"
  mkdir -p "$ROOT/bin"
  (cd "$ROOT/mcp/starling-lite" && go build -o "$ROOT/bin/starling-lite" ./cmd/starling-lite)

  if [[ -f "$CE_PID_FILE" ]] && kill -0 "$(cat "$CE_PID_FILE")" 2>/dev/null; then
    log "already running pid=$(cat "$CE_PID_FILE") — stop with: $0 down"
    return 0
  fi

  log "starting MCP on http://127.0.0.1:${CE_PORT}/mcp (mode=$CE_MODE)"
  nohup "$ROOT/bin/starling-lite" \
    --mode="$CE_MODE" \
    --transport=http \
    --addr=":${CE_PORT}" \
    --registry="$ROOT/mcp/tools/registry.json" \
    >"/tmp/starling-ex-ce.log" 2>&1 &
  echo $! >"$CE_PID_FILE"
  sleep 0.6
  cmd_smoke "http://127.0.0.1:${CE_PORT}"
  log "logs: tail -f /tmp/starling-ex-ce.log"
  log "public always-on demo (no install): $PUBLIC_DEMO/mcp"
}

cmd_docker() {
  need docker
  log "building CE image $CE_IMAGE"
  docker build -f "$ROOT/deploy/railway/Dockerfile" -t "$CE_IMAGE" "$ROOT"

  if docker ps -a --format '{{.Names}}' | grep -qx "$CE_CTR_NAME"; then
    log "removing existing container $CE_CTR_NAME"
    docker rm -f "$CE_CTR_NAME" >/dev/null
  fi

  log "running $CE_CTR_NAME on :${CE_PORT}"
  docker run -d --name "$CE_CTR_NAME" \
    -p "${CE_PORT}:8080" \
    -e PORT=8080 \
    -e STARLING_MODE="$CE_MODE" \
    -e STARLING_TRANSPORT=http \
    "$CE_IMAGE" >/dev/null

  sleep 1
  cmd_smoke "http://127.0.0.1:${CE_PORT}"
  log "docker logs -f $CE_CTR_NAME"
}

cmd_helm_template() {
  need helm
  [[ -d "$CHART" ]] || die "chart missing: $CHART"
  log "helm template dry-run (namespace=$CE_NS)"
  helm template "$CE_RELEASE" "$CHART" \
    --namespace "$CE_NS" \
    --set mode="$CE_MODE" \
    --set publicDemoUrl="$PUBLIC_DEMO/mcp" \
    --set docsUrl=https://warbleoss.org
  log "template OK — no cluster changes"
}

cmd_helm() {
  need helm
  need kubectl
  [[ -d "$CHART" ]] || die "chart missing: $CHART"

  local ctx
  ctx="$(kubectl config current-context 2>/dev/null || true)"
  log "kube context: ${ctx:-none}"
  if [[ -z "$ctx" ]]; then
    die "no kube context — use './scripts/install.sh local' or 'docker' for preview"
  fi

  # Guard: refuse accidental prod unless CE_FORCE=1
  if [[ "$ctx" == *gke* || "$ctx" == *prod* ]] && [[ "${CE_FORCE:-0}" != "1" ]]; then
    die "context looks shared/prod ($ctx). Set CE_FORCE=1 to install, or use local/docker preview"
  fi

  log "helm upgrade --install $CE_RELEASE (ns=$CE_NS)"
  helm upgrade --install "$CE_RELEASE" "$CHART" \
    --namespace "$CE_NS" \
    --create-namespace \
    --set mode="$CE_MODE" \
    --set publicDemoUrl="$PUBLIC_DEMO/mcp" \
    --set docsUrl=https://warbleoss.org \
    --wait --timeout 3m

  log "port-forward: kubectl -n $CE_NS port-forward svc/$CE_RELEASE 8822:8822"
  log "then: $0 smoke http://127.0.0.1:8822"
  kubectl -n "$CE_NS" get deploy,svc,pods -l "app.kubernetes.io/instance=$CE_RELEASE" 2>/dev/null \
    || kubectl -n "$CE_NS" get all 2>/dev/null || true
}

cmd_smoke() {
  local url="${1:-http://127.0.0.1:${CE_PORT}}"
  url="${url%/}"
  need curl
  log "health $url/health"
  curl -fsS --max-time 20 "$url/health" | (command -v python3 >/dev/null && python3 -m json.tool || cat)
  echo
  log "tools/list $url/mcp"
  local n
  n=$(curl -fsS --max-time 30 -X POST "$url/mcp" \
    -H 'Content-Type: application/json' \
    -H 'MCP-Protocol-Version: 2026-07-28' \
    -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
    | python3 -c 'import sys,json; r=json.load(sys.stdin); print(len(r.get("result",{}).get("tools",[])))')
  log "tools count: $n"
  [[ "$n" -gt 0 ]] || die "no tools returned"
  log "smoke OK"
}

docker_ok() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

cmd_down() {
  if [[ -f "$CE_PID_FILE" ]]; then
    local pid
    pid="$(cat "$CE_PID_FILE")"
    if kill -0 "$pid" 2>/dev/null; then
      log "stopping local pid $pid"
      kill "$pid" 2>/dev/null || true
    fi
    rm -f "$CE_PID_FILE"
  fi
  if docker_ok; then
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "$CE_CTR_NAME"; then
      log "removing docker container $CE_CTR_NAME"
      docker rm -f "$CE_CTR_NAME" >/dev/null || true
    fi
  fi
  log "local CE preview stopped"
}

cmd_status() {
  echo "ROOT=$ROOT"
  echo "CE_PORT=$CE_PORT CE_MODE=$CE_MODE"
  if [[ -f "$CE_PID_FILE" ]] && kill -0 "$(cat "$CE_PID_FILE")" 2>/dev/null; then
    echo "local: running pid=$(cat "$CE_PID_FILE")"
  else
    echo "local: not running"
  fi
  if docker_ok && docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$CE_CTR_NAME"; then
    echo "docker: running $CE_CTR_NAME"
  else
    echo "docker: not running (daemon unavailable or no container)"
  fi
  if command -v kubectl >/dev/null 2>&1; then
    echo "kube context: $(kubectl config current-context 2>/dev/null || echo none)"
  fi
  echo "public demo: $PUBLIC_DEMO"
}

main() {
  local cmd="${1:-local}"
  shift || true
  case "$cmd" in
    help|-h|--help) cmd_help ;;
    local|preview) cmd_local "$@" ;;
    docker) cmd_docker "$@" ;;
    helm) cmd_helm "$@" ;;
    helm-template|template|dry-run) cmd_helm_template "$@" ;;
    smoke|test) cmd_smoke "$@" ;;
    down|stop) cmd_down "$@" ;;
    status) cmd_status "$@" ;;
    *) die "unknown command: $cmd (try: help)" ;;
  esac
}

main "$@"
