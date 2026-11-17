# Read-only demo cluster for mcp.warbleoss.org

Tracking: [warble-marketing#80](https://github.com/warbletech/warble-marketing/issues/80).
Done when `cluster_summary` answers on the public demo.

Today the Railway service behind `https://mcp.warbleoss.org` has no kubeconfig, so it
runs **scan-only**: `/health` reports `"cluster": false`, `tools/list` offers only the
scan and dry-run remediation tools, and a cluster tool call explains how to run locally.
This directory has what it takes to attach a cluster. **Nothing here is applied by CI**
(CI only exercises it against a throwaway kind cluster).

| File | What |
|------|------|
| `rbac.yaml` | `starling-lite-demo` ServiceAccount + token Secret in `starling-demo-reader`; cluster-wide `get/list/watch` on namespaces, nodes, pods, events, services, workloads, HPAs and metrics; ConfigMaps and pod logs only in the demo namespaces. No secrets, no write verbs, no wildcards. |
| `make-kubeconfig.sh` | Builds a kubeconfig from that token, checks it with `kubectl auth can-i` (can list pods, cannot read secrets or kube-system ConfigMaps, cannot delete or patch), writes it with mode 0600. Applies and uploads nothing. |

## Steps (each one needs an owner's OK: they create cloud resources or change production)

1. **Dedicated cluster.** A small GKE cluster used only for this demo (never a shared
   or production cluster: the token can list every pod and event in it). Its control
   plane must be reachable from Railway: Railway has no fixed egress IPs on every plan,
   so either a public endpoint with no authorized-networks restriction, or Railway static
   IPs added to authorized networks.
2. **Samples.** From `shrikeops-demo` (needs shrikeops-demo#6 and #7):
   `ALLOW_ANY_CONTEXT=1 scripts/broken-workloads.sh apply all` against that cluster.
   They stay broken on purpose (CrashLoopBackOff, OOMKilled, ImagePullBackOff, Pending, ...).
   Optional: metrics-server for `list_pod_metrics` / `list_node_metrics` (GKE ships it).
3. **RBAC.** `kubectl --context <demo> apply -f deploy/demo-cluster/rbac.yaml`
   (after the samples, so the demo namespaces exist).
4. **Kubeconfig.** `deploy/demo-cluster/make-kubeconfig.sh --context <demo> --out demo.kubeconfig`.
   Treat the file as a secret; delete it after step 5.
5. **Attach.** Set it on the Railway service, which redeploys it:
   `base64 -w0 demo.kubeconfig | railway variables set KUBECONFIG_B64=@/dev/stdin`
6. **Verify.**
   `python3 scripts/ci/starling-lite-smoke.py --url https://mcp.warbleoss.org --require-cluster`
   checks `/health` reports `"cluster": true` and `cluster_summary` answers.

Revoke: delete the `starling-lite-demo-token` Secret (or unset `KUBECONFIG_B64` on Railway
to go back to scan-only).

## Before going live

- Ingress auth / rate limits for the public endpoint are still open
  ([warble-marketing#81](https://github.com/warbletech/warble-marketing/issues/81),
  warble-army#36). With a cluster attached, anyone can read pod specs, events and the demo
  namespaces' logs and ConfigMaps, so keep secrets out of the demo cluster entirely.
- The Railway image pins `STARLING_SCAN_ROOT` to an empty directory, so `scan_manifest
  path=...` cannot read the materialized kubeconfig under `/tmp`.
