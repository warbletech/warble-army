# Safety — MCP and remediations

## Defaults

1. **Readonly** MCP mode unless explicitly configured otherwise.
2. **Dry-run** for every remediation proposal before apply.
3. **Namespace allowlist** when write tools are enabled.
4. **Audit** tool calls (JSONL) when running agent loops.

## SteadyHelm

| Mode | Tools |
|------|--------|
| `readonly` (default) | list/get/logs/metrics/summary only |
| `readwrite` | + `scale_deployment`, `delete_pod` |

Run with least privilege: scoped ServiceAccount + ClusterRole limited to needed verbs. See `steadyhelm/README.md` sample RBAC.

## Starling Lite (planned apply path)

| Tool | Gate |
|------|------|
| `propose_remediation` | Always allowed; returns patch, never mutates |
| `apply_remediation` | Requires `mode=readwrite`, allowlist, and `confirm=true` |

## LLM-assisted remediation (`propose_remediation_llm`)

| Gate | Rule |
|------|------|
| Order | Deterministic rules first; LLM only as fallback, output labelled `generator: llm` |
| Input | Manifest/finding/cluster context sent as untrusted JSON data (prompt-injection framing) |
| Output | Strict JSON, size caps, field denylist, typed strategic-merge patch |
| Verify | ShrikeOps re-scan: no score regression, no new findings; `kubectl apply --dry-run=server` when a cluster exists |
| Effect | Dry-run proposal only; `apply_remediation` gates unchanged |
| Config | `LLM_PROVIDER` + keys from env only; missing key = fail closed |

## Secrets

Hooks reject staged files that look like private keys or `.env` with live credentials. Do not put kubeconfig or cloud tokens in this umbrella repo.

## Project sandbox

Agent/remediation tasks that need to run something for a target project should run in a per-project sandbox namespace: PSA restricted, default-deny network, no SA token, quota, read-only root FS. See [sandbox.md](sandbox.md) (`charts/project-sandbox`, MCP tool `sandbox_project`).
