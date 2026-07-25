# MCP protocol conventions (Warble Army)

Subset of [Model Context Protocol](https://spec.modelcontextprotocol.io) used by SteadyHelm and planned Starling Lite.

## Transport

| Transport | Use |
|-----------|-----|
| **stdio** | Claude Desktop, Cursor, local agents |
| **HTTP** | Sidecars, multi-agent mesh (`--addr=:8811`) |

## Lifecycle

1. Client → `initialize` (protocolVersion, clientInfo)
2. Server → capabilities + serverInfo
3. Client → `tools/list`
4. Client → `tools/call` with `{ "name", "arguments" }`
5. Server → text/structured content result or JSON-RPC error

## Tool naming

- `snake_case` only.
- Cluster reads: `list_*`, `get_*`, `cluster_summary`.
- Scans: `scan_*`.
- Mutations: verb_noun (`scale_deployment`, `delete_pod`, `apply_remediation`).
- Never overload one tool for scan + apply.

## Error codes (JSON-RPC)

| Code | Meaning |
|------|---------|
| `-32601` | Method not found |
| `-32602` | Invalid params (schema) |
| `-32001` | Backend unavailable (no kubeconfig, scanner down) |
| `-32003` | Write denied (readonly mode / allowlist) |
| `-32004` | Dry-run required / confirm missing |

## Versioning tools

Breaking param renames require a new tool name or major umbrella tag. Additive optional properties are fine without bump.
