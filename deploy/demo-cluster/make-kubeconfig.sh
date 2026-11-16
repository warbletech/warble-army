#!/usr/bin/env bash
# Build the read-only kubeconfig for the public Starling Lite demo
# (warble-marketing#80) from the starling-lite-demo ServiceAccount in rbac.yaml.
#
#   deploy/demo-cluster/make-kubeconfig.sh --context <demo-cluster-context> [--out FILE]
#
# Reads the cluster (read-only kubectl calls), verifies the token's permissions
# with `kubectl auth can-i`, and writes a kubeconfig file (mode 0600). It does
# not apply anything and does not upload anything: setting KUBECONFIG_B64 on
# Railway is a separate, manual step (README.md, step 5).
set -euo pipefail

NS=starling-demo-reader
SA=starling-lite-demo
SECRET=starling-lite-demo-token
OUT=starling-lite-demo.kubeconfig
CTX=""

die() { echo "error: $*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --context) CTX="${2:-}"; shift 2 ;;
    --out) OUT="${2:-}"; shift 2 ;;
    -h|--help) sed -n '2,10p' "$0"; exit 0 ;;
    *) die "unknown argument: $1" ;;
  esac
done
[[ -n "$CTX" ]] || die "--context is required (the dedicated demo cluster, never a shared one)"
command -v kubectl >/dev/null || die "kubectl not found"

k() { kubectl --context "$CTX" "$@"; }

server="$(k config view --minify --raw -o jsonpath='{.clusters[0].cluster.server}')"
[[ -n "$server" ]] || die "no API server URL for context $CTX"
token="$(k -n "$NS" get secret "$SECRET" -o jsonpath='{.data.token}' | base64 -d)"
ca="$(k -n "$NS" get secret "$SECRET" -o jsonpath='{.data.ca\.crt}')"
[[ -n "$token" && -n "$ca" ]] || die "secret $NS/$SECRET has no token yet; apply rbac.yaml first"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
chmod 600 "$tmp"
cat >"$tmp" <<KC
apiVersion: v1
kind: Config
clusters:
- name: starling-demo
  cluster:
    server: ${server}
    certificate-authority-data: ${ca}
users:
- name: ${SA}
  user:
    token: ${token}
contexts:
- name: starling-demo
  context:
    cluster: starling-demo
    user: ${SA}
current-context: starling-demo
KC

# The token must be able to read what the demo shows and nothing more.
can() { kubectl --kubeconfig "$tmp" auth can-i "$@" 2>/dev/null || true; }
fail=0
expect() { # $1 yes|no, rest: can-i args
  local want="$1"; shift
  local got; got="$(can "$@")"
  if [[ "$got" != "$want" ]]; then echo "permission check failed: can-i $* = ${got:-error}, want $want" >&2; fail=1; fi
}
expect yes list pods --all-namespaces
expect yes list nodes
expect yes get configmaps -n demo
expect yes get pods --subresource=log -n demo
expect no  get secrets --all-namespaces
expect no  get configmaps -n kube-system
expect no  delete pods -n demo
expect no  patch deployments -n demo
expect no  create pods -n demo
[[ "$fail" == 0 ]] || die "token permissions do not match rbac.yaml; not writing $OUT"

install -m 600 "$tmp" "$OUT"
echo "wrote $OUT (read-only token for $NS/$SA on $server)"
echo "Next (manual, needs approval): set it as KUBECONFIG_B64 on the Railway service, see README.md step 5."
