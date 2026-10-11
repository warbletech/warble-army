# Starling Lite

Thin **OSS MCP facade** that loads [`../tools/registry.json`](../tools/registry.json) and dispatches to:

| Backend | Tools |
|---------|--------|
| **SteadyHelm** (`pkg/mcpserver`) | `list_*`, `get_*`, `cluster_summary`, writes |
| **ShrikeOps** (`pkg/scan`) | `scan_manifest`, `scan_helm`, `scan_cluster_export` |
| **Local patcher** | `propose_remediation`, `apply_remediation` (dry-run first) |

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
