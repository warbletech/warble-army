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

## AI workloads (`ai-*` rules)

Starling Lite adds a deterministic, offline `ai-*` rule family to every `scan_manifest` / `scan_helm` /
`scan_cluster_export` result (`mcp/starling-lite/internal/scanx/ai.go`). Findings use the normal shape
(`category: ai_security`) and their ShrikeOps severity penalties are folded into the **security** score
and grade (incomplete scans stay `N/A`).

| Rule | Severity | Triggers on | Remediation (`propose_remediation`) |
|------|----------|-------------|-------------------------------------|
| `AI-MODEL-PICKLE` | high | container/initContainer command, args or env referencing `.bin/.pt/.pth/.pkl/.pickle/.ckpt` model files | hint: use `.safetensors`, run `scan_model`, `weights_only=True` |
| `AI-MODEL-UNPINNED` | medium | `HF_MODEL_ID`/`MODEL_ID`/`MODEL_URL`/… or `--model <repo>` / `huggingface-cli download` without a 40-char `HF_REVISION`/`MODEL_REVISION`/`--revision` SHA or `MODEL_SHA256`/`MODEL_DIGEST` | hint: pin revision SHA + verify digest |
| `AI-MODEL-TRUST-REMOTE-CODE` | high | `--trust-remote-code`, `trust_remote_code=True`, `TRUST_REMOTE_CODE=true` | hint |
| `AI-MCP-READWRITE-EXPOSED` | critical | MCP server (`*mcp*`, starling-lite, heron-ops) with `--mode=readwrite` / `MODE=readwrite`, reachable via Ingress → Service → pods with no auth annotation | hint: readonly or auth |
| `AI-MCP-INGRESS-NOAUTH` | medium | same, readonly MCP | hint: auth or keep internal |
| `AI-TOOL-INSTRUCTION-TEXT` | high | ConfigMap tool registry `description`/`instructions` containing instruction-like text (`ignore previous instructions`, `<IMPORTANT>`, `do not tell the user`, `~/.ssh`, exfiltration, …) | hint |
| `AI-AGENT-LETHAL-TRIFECTA` | critical | agent/tool ConfigMap listing write tools (`apply_*`, `delete_*`, `scale_*`, shell, …) **and** untrusted-content tools (`web_fetch`, `browse`, `read_url`, …) without `mode: readonly` | hint: split agents, readonly, human confirm |
| `AI-GPU-NO-LIMIT` | high | `nvidia.com/gpu` in requests only, or GPU node selector / toleration / `NVIDIA_VISIBLE_DEVICES` / `runtimeClassName: nvidia` without a limit | **patch**: `nvidia.com/gpu: 1` in requests+limits |
| `AI-HOSTPATH-MODEL-CACHE` | high | hostPath volume mounted at, or pointing to, a model/HF/torch cache path | hint: emptyDir / read-only PVC |
| `AI-PRIVILEGED-DEVICE-PLUGIN` | high | `privileged: true` on an NVIDIA/device-plugin/GPU-operator container | **patch**: existing privileged fix |

Auth annotations recognised on Ingress: nginx `auth-url`/`auth-type`/`auth-signin`/`auth-tls-secret`, Traefik
middlewares, ALB `auth-type`, Kong plugins, GKE `backend-config`. Fixtures: `internal/scanx/testdata/ai/{bad,good}`.

### `scan_model`

Static, picklescan-style check of model weights. It walks pickle opcodes (protocols 0–5, plus pickles inside
torch zip files) and lists every `GLOBAL`/`INST`/`STACK_GLOBAL` import; it **never unpickles**.

| Result | Meaning |
|--------|---------|
| `safe` | safetensors header |
| `pickle-allowlisted` | pickle with only known-safe globals (torch tensor rebuild, `collections.OrderedDict`, numpy) — still `AI-MODEL-PICKLE` |
| `suspicious` / `unparseable` | unknown globals or a stream that cannot be parsed → `AI-MODEL-PICKLE-UNKNOWN-GLOBAL` (medium) |
| `unsafe` | imports from `os`/`posix`/`subprocess`/`socket`/`runpy`/…, or `builtins.eval/exec/getattr/__import__/…` → `AI-MODEL-PICKLE-RCE` (critical) |

Input is a file inside the scan root or `bytes_base64`, capped by `STARLING_MODEL_MAX_BYTES` (default 256 MiB).
Fetching from Hugging Face is **not** implemented (follow-up: opt-in flag, size cap, timeout, no execution).

## Secrets

Hooks reject staged files that look like private keys or `.env` with live credentials. Do not put kubeconfig or cloud tokens in this umbrella repo.

## Project sandbox

Agent/remediation tasks that need to run something for a target project should run in a per-project sandbox namespace: PSA restricted, default-deny network, no SA token, quota, read-only root FS. See [sandbox.md](sandbox.md) (`charts/project-sandbox`, MCP tool `sandbox_project`).
