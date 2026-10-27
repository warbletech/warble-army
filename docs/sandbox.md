# Project sandbox

Agents in the flock (Scout → Auditor → Healer → Critic → Apply, see [architecture](architecture.md)) sometimes need to **run** something for a target project, e.g. a ShrikeOps scan of a cloned repo, a remediation dry-run, or a candidate patch tried out before it is proposed. A project sandbox gives each project its own isolated, least-privilege namespace for that work.

## What a sandbox is

| Control | Implementation |
|---|---|
| Isolation boundary | Namespace `sbx-<project>` |
| Pod hardening | Pod Security Admission `restricted` (enforce + audit + warn) |
| Network | `default-deny-all` (ingress + egress) + DNS to kube-dns only; opt-in `egressCIDRs` (`0.0.0.0/0` rejected) |
| Identity | `sandbox-runner` ServiceAccount, `automountServiceAccountToken: false`, **no RBAC bindings** |
| Resources | `ResourceQuota` (cpu/memory/pods/secrets; 0 Services/LB/NodePort/PVC) + `LimitRange` defaults |
| Task Job | non-root 65532, read-only root FS, drop ALL caps, seccomp RuntimeDefault, no privilege escalation, `emptyDir` `/tmp` + `/work` with size limits, deadline + TTL, `backoffLimit: 0`, pinned image (no `:latest`) |

## Two ways to get one

**Helm** (`charts/project-sandbox`):

```bash
helm install sbx-payments charts/project-sandbox --set project=payments
# with a task:
helm install sbx-payments charts/project-sandbox -f charts/project-sandbox/ci-values.yaml --set project=payments
make sandbox-template PROJECT=payments   # render only
```

**MCP** (Starling Lite tool `sandbox_project`, readonly): returns the same manifests as dry-run YAML so an agent or a human can review them and then `kubectl apply --dry-run=server` them. Starling Lite never applies them itself, in line with [safety.md](safety.md).

```json
{"name":"sandbox_project","arguments":{"project":"payments","egress_cidrs":["10.20.0.0/16"],"image":"ghcr.io/warbletech/shrikeops:v0.1.0","command":["shrikeops","scan","/work"]}}
```

## Limits of this first pass
- Namespace + PSA is not a VM boundary. For untrusted code, add a sandboxed RuntimeClass (gVisor/Kata) as a follow-up.
- NetworkPolicy needs a CNI that enforces it (Calico, Cilium, GKE Dataplane V2).
- Getting source into `/work` (an init container with a read-only git clone, or an OCI artifact) is a follow-up.
- Wiring `apply_remediation` / eval-gated remediation (#18) to execute inside a sandbox is a follow-up.

Tests: `go test ./internal/sandbox/` (in `mcp/starling-lite`) and `make sandbox-test` (helm lint + render + invariants + negative cases; runs in CI).
