# MCP — Warble Army tool layout

Stable **tool contracts** for the flock. Implementations may live in SteadyHelm (cluster), ShrikeOps (scan), or Starling Lite (facade). Agents and IDEs should bind to these names.

## Layout

```text
mcp/
├── README.md                 ← you are here
├── PROTOCOL.md               ← JSON-RPC / MCP conventions
├── config/
│   ├── steadyhelm.example.json
│   ├── starling-lite.example.json
│   └── cursor.mcp.json
└── tools/
    ├── registry.json         ← canonical tools/list payload
    ├── cluster.json          ← SteadyHelm-backed
    ├── scan.json             ← ShrikeOps-backed
    └── remediate.json        ← propose/apply (facade)
```

## Client quick config

### SteadyHelm only (available today)

```json
{
  "mcpServers": {
    "steadyhelm": {
      "command": "steadyhelm",
      "args": ["--mode=readonly", "--transport=stdio"]
    }
  }
}
```

Full examples: [`config/`](config/).

### Starling product CLI

```json
{
  "mcpServers": {
    "starling": {
      "command": "starling",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

Verify live tools:

```bash
# SteadyHelm
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | steadyhelm --mode=readonly --transport=stdio

# Product Starling (if installed)
starling mcp --stdio
```

### Starling Lite facade (Phase 2)

Umbrella process that loads `tools/registry.json` and dispatches:

| Prefix / name | Backend |
|---------------|---------|
| `list_*`, `get_*`, `cluster_summary`, write tools | SteadyHelm library / subprocess |
| `scan_manifest`, `scan_helm` | ShrikeOps scan API or CLI |
| `propose_remediation`, `apply_remediation` | Local patch generator |

Binary path (future): `mcp/starling-lite/` — schemas land first so agents can mock.

## Registry

[`tools/registry.json`](tools/registry.json) is the **source of truth** for names, descriptions, and JSON Schema inputs. Pre-commit validates it is valid JSON and that every tool appears in exactly one group file.

When SteadyHelm or ShrikeOps add tools:

1. Update the group JSON under `tools/`.
2. Rebuild `registry.json` (or edit in sync).
3. Bump submodule pin if implementation changed.
4. Commit umbrella: `docs(mcp): add tool <name>`.

## Groups

| File | Responsibility |
|------|----------------|
| [`tools/cluster.json`](tools/cluster.json) | Live cluster observe / limited mutate |
| [`tools/scan.json`](tools/scan.json) | Manifest & Helm scanning |
| [`tools/remediate.json`](tools/remediate.json) | Patch propose / gated apply |
