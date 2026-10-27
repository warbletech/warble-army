#!/usr/bin/env bash
# Lint + render the project-sandbox chart and assert its hardening invariants.
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
C=charts/project-sandbox
die() { echo "test-project-sandbox: $*" >&2; exit 1; }
command -v helm >/dev/null || die "helm required"

helm lint "$C" -f "$C/ci-values.yaml" --strict >/dev/null
out=$(helm template t "$C" -f "$C/ci-values.yaml")
for want in \
  "pod-security.kubernetes.io/enforce: restricted" \
  "automountServiceAccountToken: false" \
  "name: default-deny-all" \
  "kind: ResourceQuota" \
  "kind: LimitRange" \
  "readOnlyRootFilesystem: true" \
  "allowPrivilegeEscalation: false" \
  "type: RuntimeDefault"; do
  grep -qF -- "$want" <<<"$out" || die "missing: $want"
done
[[ $(grep -c "automountServiceAccountToken: false" <<<"$out") -ge 2 ]] || die "SA and Job must both disable token automount"

# Negative cases must fail to render.
! helm template t "$C" >/dev/null 2>&1 || die "missing project should fail"
! helm template t "$C" --set project=Bad_Name >/dev/null 2>&1 || die "invalid project should fail"
! helm template t "$C" --set project=x --set 'network.egressCIDRs={0.0.0.0/0}' >/dev/null 2>&1 || die "open egress should fail"
! helm template t "$C" --set project=x --set task.enabled=true --set task.image=busybox:latest >/dev/null 2>&1 || die ":latest should fail"
echo "ok: project-sandbox chart"
