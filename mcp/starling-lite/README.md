# Starling Lite (Phase 2 scaffold)

Thin MCP facade that exposes [`../tools/registry.json`](../tools/registry.json) and dispatches to:

| Backend | Tools |
|---------|--------|
| **SteadyHelm** | `list_*`, `get_*`, `cluster_summary`, `scale_deployment`, `delete_pod` |
| **ShrikeOps** | `scan_manifest`, `scan_helm`, `scan_cluster_export` |
| **Local patcher** | `propose_remediation`, `apply_remediation` |

## Status

- **Schemas:** ready under `mcp/tools/`
- **Binary:** not implemented yet — use SteadyHelm + ShrikeOps CLI until this lands

## Planned package layout

```text
starling-lite/
├── README.md          ← this file
├── cmd/starling-lite/
│   └── main.go        # flags: --mode, --transport, --registry, --addr
├── internal/
│   ├── dispatch/      # name → backend
│   ├── steadyhelm/    # embed or exec SteadyHelm handlers
│   ├── shrikeops/     # CLI or HTTP client
│   └── remediate/     # finding → patch
└── go.mod
```

## Acceptance criteria

1. `tools/list` matches registry names (minus write tools in readonly mode).
2. `scan_manifest` on `shrikeops-demo/k8s/bad/root-container.yaml` returns grade F-class findings.
3. `cluster_summary` works with valid kubeconfig.
4. `apply_remediation` with `dry_run=false` without `confirm=true` returns `-32004`.
