# Starling Lite

Thin **OSS MCP facade** that loads [`../tools/registry.json`](../tools/registry.json) and dispatches to:

| Backend | Tools |
|---------|--------|
| **SteadyHelm** (`pkg/mcpserver`) | `list_*`, `get_*`, `cluster_summary`, writes |
| **ShrikeOps** (`pkg/scan`) | `scan_manifest`, `scan_helm`, `scan_cluster_export` |
| **Local patcher** | `propose_remediation`, `propose_remediation_llm` (guarded LLM fallback), `apply_remediation` (dry-run first) |

### Scan results are only graded when every scanner ran

`scan_*` shells out to `pluto`, `polaris` and `kube-score` (via ShrikeOps). If any of them is missing on the
server, the result carries `"incomplete": true`, `score.grade: "N/A"`, `score.skipped_checks` and a
`warning` — treat it as *not scanned*, never as a pass. The deploy images under `deploy/` ship pinned versions
of all three.

### `propose_remediation` inputs

| Input | Effect |
|-------|--------|
| `kind`, `name`, `issue` (required) | Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet, ReplicationController, Job, CronJob |
| `container` | strategic-merge patch for that container |
| `finding` (ShrikeOps finding) | container taken from `finding.resource` (`…/container:<name>`) |
| `manifest` (resource YAML) | strategic-merge patch for every container of the matching resource |
| none of the above | RFC 6902 JSON patch on the **first** container (`patch_type: "json"`), with `missing_inputs` and `notes` |

Every proposal is `dry_run: true` and includes a `kubectl patch … --dry-run=server` hint.

### `propose_remediation_llm` (guarded LLM fallback)

Same inputs plus `manifest` (required), optional `finding`, `cluster_context` (Heron Ops `cluster_*` output)
and `server_dry_run`. Output is the same `Proposal`, plus `generator`, `provider` and `validation`.

1. **Rules first.** If `propose_remediation` has a built-in fix, it is returned with `generator: "rules"` and no LLM call.
2. **LLM fallback.** Otherwise the provider from `LLM_PROVIDER` drafts a patch. The manifest (comments stripped),
   finding and cluster context are sent as a JSON data block that the system prompt marks as untrusted.
3. **Guardrails (LLM output is untrusted):**
   - strict JSON `{patch, rationale}` only, 32 KiB completion / 16 KiB patch caps, rationale sanitised;
   - denylist: only the pod spec of existing containers may change; no `apiVersion`/`kind`/`metadata`
     changes, no `image`/`command`/`args`/`env`/volumes/`hostPath`/host ports/service accounts, no new
     containers, no `$patch` directives; `privileged`, `allowPrivilegeEscalation`, `hostNetwork|PID|IPC`
     must be `false`, `runAsUser` > 0, `capabilities` may only `drop`, no `Unconfined` seccomp;
   - must decode as a typed Kubernetes strategic-merge patch and apply to the original object;
   - ShrikeOps re-scan of original vs patched: rejected if the scan is incomplete, the score drops, a new
     finding appears, or nothing is resolved;
   - `kubectl apply --dry-run=server` only when a cluster backend is available (else `skipped`).
4. Accepted proposals carry `generator: "llm"`, a `[LLM-generated — review before applying]` rationale and
   `dry_run: true`. Rejections return an error with the reason and validation detail. Nothing is ever applied;
   `apply_remediation` gates are unchanged.

| Env | Meaning |
|-----|---------|
| `LLM_PROVIDER` | `vertex` \| `openai` \| `ollama` \| `avirka`; unset = LLM fallback disabled (fail closed) |
| `LLM_MODEL` | model id (defaults: openai `gpt-4o-mini`, vertex `gemini-2.5-flash`, ollama `llama3.1`; avirka: required) |
| `LLM_TIMEOUT_SECONDS` | per-call timeout, default 30, max 120 |
| `OPENAI_API_KEY`, `OPENAI_BASE_URL` | openai (key required) |
| `VERTEX_ACCESS_TOKEN`, `VERTEX_PROJECT`, `VERTEX_LOCATION` | vertex (token + project required; e.g. `gcloud auth print-access-token`) |
| `OLLAMA_HOST` | ollama, default `http://127.0.0.1:11434` (no key) |
| `AVIRKA_API_KEY`, `AVIRKA_BASE_URL` | avirka, OpenAI-compatible chat-completions endpoint (both required) |

Keys are read from the environment only and never logged. Tests use an in-process fake provider; no test
calls a real LLM. Golden tests over `shrikeops-demo/k8s/bad/*` live in `internal/llmfix/testdata/golden`
(regenerate with `go test ./internal/llmfix -run Golden -update`; they need the three scanners on `PATH`).

## Build & run

From the `warble-army` root:

```bash
make starling-lite
./bin/starling-lite --mode=readonly --transport=stdio \
  --registry=mcp/tools/registry.json
```

HTTP (agents / Reflexion):

```bash
./bin/starling-lite --mode=readonly --transport=http --addr=:8822
curl -s localhost:8822/health
```

### Always-on PaaS (Railway / Fly / Render)

Independent of GKE — scan tools work with **no** kubeconfig:

```bash
# Docker (same image Railway uses)
make railway-image
make railway-run   # http://127.0.0.1:8080/health  and  /mcp
```

See [`deploy/railway/README.md`](../../deploy/railway/README.md) for Railway deploy steps and env vars (`PORT`, `KUBECONFIG_B64`).

### MCP client config

```json
{
  "mcpServers": {
    "warble-army": {
      "command": "/path/to/warble-army/bin/starling-lite",
      "args": ["--mode=readonly", "--transport=stdio", "--registry", "/path/to/mcp/tools/registry.json"],
      "env": { "KUBECONFIG": "/home/you/.kube/config" }
    }
  }
}
```

## Layout

```text
starling-lite/
├── cmd/starling-lite/main.go
├── internal/
│   ├── config/
│   ├── protocol/      # JSON-RPC MCP types
│   ├── registry/      # load registry.json
│   ├── scanx/         # ShrikeOps + helm template
│   ├── remediate/     # propose / apply gates
│   └── server/        # stdio + HTTP + dispatch
└── go.mod             # replace → ../../steadyhelm, ../../shrikeops/scanner
```

## MCP protocol versions

| Version | Status in Starling Lite |
|---------|-------------------------|
| **2026-07-28** | Preferred — stateless core, `server/discover`, list cache hints, header routing |
| **2025-03-26** | Negotiated if client asks |
| **2024-11-05** | Still accepted (legacy initialize handshake) |

Community shorthand often calls **2026-07-28** “MCP 2.0”. Health JSON exposes `protocolVersions` / `protocolPreferred`.

## Safety

- Default `--mode=readonly` hides write tools.
- `apply_remediation` requires `confirm=true` when `dry_run=false`.
- Optional `--allow-namespaces=ns1,ns2` for writes.
- Real cluster apply is still operator/`kubectl` in Phase 2; Phase 3 Reflexion adds gated apply.

## Smoke

```bash
make test-mcp
make agent-demo
```
