# Warble Army

**Open flock of Kubernetes agents + MCP** — scan, observe, propose, remediate.

> Birds of a feather: **ShrikeOps** hunts bad config, **SteadyHelm** gives agents live cluster context via MCP, **Starling** is the product release channel for agent-facing ops, and demos keep the loop honest.

This repository is the **umbrella** that wires the flock together with submodules, a stable MCP tool layout, and maintenance hooks. Child product code lives in its own repos under [github.com/warble-tech](https://github.com/warble-tech).

```text
  IDE / Claude / Cursor / custom agent
              │  MCP (JSON-RPC 2.0)
              ▼
     mcp/starling-lite   ← tool registry (this repo)
      ├── scan_*         → ShrikeOps
      ├── cluster_*      → SteadyHelm
      └── remediate_*    → dry-run patches (guardrailed)
              │
     ┌────────┼────────┐
     ▼        ▼        ▼
 ShrikeOps  SteadyHelm  Starling (release / SaaS)
  scanner    live MCP    product binaries
```

---

## Submodules

| Path | Remote | Role |
|------|--------|------|
| [`shrikeops/`](shrikeops/) | [warble-tech/shrikeops](https://github.com/warble-tech/shrikeops) | Manifest / Helm security scanner |
| [`shrikeops-demo/`](shrikeops-demo/) | [warble-tech/shrikeops-demo](https://github.com/warble-tech/shrikeops-demo) | Golden/bad YAML + CI demo |
| [`steadyhelm/`](steadyhelm/) | [warble-tech/steadyhelm](https://github.com/warble-tech/steadyhelm) | **OSS Kubernetes MCP server** (stdio + HTTP) |
| [`starling/`](starling/) | [warble-tech/starling](https://github.com/warble-tech/starling) | Starling CLI **release channel** (binaries; product source private) |

Umbrella-owned (not submodules):

| Path | Purpose |
|------|---------|
| [`mcp/`](mcp/) | Stable MCP tool schemas + Starling Lite facade layout |
| [`docs/`](docs/) | Architecture, safety, submodule maintenance |
| [`scripts/`](scripts/) | Submodule sync, hook install, checks |
| [`.githooks/`](.githooks/) | Pre-commit / commit-msg / pre-push |

---

## Quick start

### 1. Clone with submodules

```bash
git clone --recurse-submodules https://github.com/warble-tech/warble-army.git
cd warble-army
```

If you already cloned without submodules:

```bash
./scripts/submodules.sh init
# or: git submodule update --init --recursive
```

### 2. Install git hooks (required for contributors)

```bash
./scripts/install-hooks.sh
```

Hooks live in `.githooks/` (`core.hooksPath`). They:

- block commits that leave submodules dirty or on detached unintended SHAs without intent
- enforce [Conventional Commits](https://www.conventionalcommits.org/)
- on push, verify submodule gitlinks still resolve

### 3. Run the OSS MCP (Starling Lite — recommended)

```bash
make starling-lite
./bin/starling-lite --mode=readonly --transport=stdio \
  --registry=mcp/tools/registry.json
```

Or SteadyHelm alone (cluster tools only):

```bash
make steadyhelm
./bin/steadyhelm --mode=readonly --transport=stdio
```

Cursor / Claude Desktop (full flock tools):

```json
{
  "mcpServers": {
    "warble-army": {
      "command": "/path/to/warble-army/bin/starling-lite",
      "args": ["--mode=readonly", "--transport=stdio", "--registry", "/path/to/warble-army/mcp/tools/registry.json"],
      "env": { "KUBECONFIG": "/home/you/.kube/config" }
    }
  }
}
```

See [`mcp/README.md`](mcp/README.md) and [`mcp/starling-lite/README.md`](mcp/starling-lite/README.md).

### 4. Reflexion agent demo

```bash
make agent-demo
# or manually:
./bin/starling-lite --transport=http --addr=:8822 &
python3 agents/reflexion/run.py --path shrikeops-demo/k8s/bad/root-container.yaml
```

### 5. Scan manifests (ShrikeOps)

```bash
cd shrikeops/scanner
go run ./cmd/cli/main.go scan ../../shrikeops-demo/k8s/bad/root-container.yaml
```

---

## Leverage roadmap (OSS)

| Phase | What | Status |
|-------|------|--------|
| **0** | Umbrella + submodules + hooks + master docs | **done** |
| **1** | SteadyHelm public embed API (`pkg/mcpserver`) | **done** |
| **2** | Starling Lite facade (`make starling-lite`) | **done** |
| **3** | Reflexion actor/critic scaffold (`agents/reflexion`) | **scaffold** |
| **4** | Plugin protocol + SARIF / GitHub Checks marketplace | planned |

**SaaS boundary:** Warble Cloud login, multi-tenant score history, and private `warble-core` stay commercial. Local kubeconfig + dry-run remediations stay OSS.

---

## Maintain submodules

```bash
./scripts/submodules.sh status     # dirty? ahead/behind?
./scripts/submodules.sh update     # fetch + checkout tracked branches
./scripts/submodules.sh bump       # advance gitlinks to latest remote tips (working tree)
./scripts/submodules.sh remote-add # re-add remotes if someone nuked origin
```

Details: [`docs/MAINTAINING.md`](docs/MAINTAINING.md).

After bumping pins, commit **only** the gitlink change on this umbrella:

```bash
git add shrikeops steadyhelm starling shrikeops-demo
git commit -m "chore(submodules): bump flock to latest"
```

---

## MCP tools (summary)

Authoritative registry: [`mcp/tools/registry.json`](mcp/tools/registry.json).

| Group | Examples | Backend |
|-------|----------|---------|
| **Cluster (read)** | `cluster_summary`, `list_pods`, `get_pod_logs` | SteadyHelm |
| **Cluster (write)** | `scale_deployment`, `delete_pod` | SteadyHelm (`--mode=readwrite`) |
| **Scan** | `scan_manifest`, `scan_helm` | ShrikeOps |
| **Remediate** | `propose_remediation`, `apply_remediation` | Starling Lite (dry-run default) |

Safety defaults: readonly mode, namespace allowlists, dry-run first. See [`docs/safety.md`](docs/safety.md).

---

## CI

GitHub Actions (`.github/workflows/check.yml`) on every push/PR:

1. Submodule status + MCP registry validation  
2. Build **SteadyHelm** + **Starling Lite**, `tools/list` + `scan_manifest` smoke  
3. **Reflexion agent** against HTTP MCP  

```bash
# local parity
make registry && make starling-lite && make test-mcp
```

## Docs

| Doc | Description |
|-----|-------------|
| [`docs/architecture.md`](docs/architecture.md) | Flock diagram, OSS vs SaaS |
| [`docs/MAINTAINING.md`](docs/MAINTAINING.md) | Submodules, remotes, hooks, release pins |
| [`docs/safety.md`](docs/safety.md) | RBAC, write gates, audit |
| [`docs/gke-pdb-unpermissive-2026-07.md`](docs/gke-pdb-unpermissive-2026-07.md) | GKE Active Assist PDB fix on `warble-cluster` |
| [`mcp/README.md`](mcp/README.md) | Tool layout + client setup |
| [`mcp/PROTOCOL.md`](mcp/PROTOCOL.md) | MCP JSON-RPC conventions for this flock |
| [`starling/INSTRUCTIONS.md`](starling/INSTRUCTIONS.md) | Product Starling setup (verify tools live) |
| [`steadyhelm/README.md`](steadyhelm/README.md) | SteadyHelm tools & deploy |
| [`shrikeops/README.md`](shrikeops/README.md) | Scanner install & API |

---

## Community

- Working groups: [agents](https://github.com/warble-tech/community/tree/main/working-groups/agents), [k8s](https://github.com/warble-tech/community/tree/main/working-groups/k8s)
- Discord · Office Hours — see [warble-tech/community](https://github.com/warble-tech/community)
- Product site: [warblecloud.com](https://warblecloud.com)

---

## License

- **Umbrella docs & schemas** in this repo: Apache-2.0 (unless noted).
- **Submodules:** each carries its own LICENSE (Apache-2.0 for ShrikeOps / SteadyHelm / Starling release).

© Warble Cloud / ChirpStack LLP.
