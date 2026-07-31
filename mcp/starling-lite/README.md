# Starling Lite

Thin **OSS MCP facade** that loads [`../tools/registry.json`](../tools/registry.json) and dispatches to:

| Backend | Tools |
|---------|--------|
| **SteadyHelm** (`pkg/mcpserver`) | `list_*`, `get_*`, `cluster_summary`, writes |
| **ShrikeOps** (`pkg/scan`) | `scan_manifest`, `scan_helm`, `scan_cluster_export` |
| **Local patcher** | `propose_remediation`, `apply_remediation` (dry-run first) |

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
