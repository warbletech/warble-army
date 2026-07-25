# Warble Army

**Open flock of Kubernetes agents + MCP** — scan, observe, propose, remediate.

> Birds of a feather: **[ShrikeOps](brand/shrike-ops/)** hunts bad config, **[Heron Ops](brand/heron-ops/)** (formerly SteadyHelm) gives agents live cluster context via MCP, **Starling Lite / Starling EX CE** coordinates the flock. Community home: **[warbleoss.org](https://warbleoss.org)**.

This repository is the **umbrella** that wires the flock together with submodules, a stable MCP tool layout, and maintenance hooks. Child product code lives in its own repos under [github.com/warble-tech](https://github.com/warble-tech).

```text
  IDE / Claude / Cursor / custom agent
              │  MCP (JSON-RPC 2.0)
              ▼
     mcp/starling-lite   ← tool registry (this repo)
      ├── scan_*         → ShrikeOps
      ├── cluster_*      → Heron Ops (SteadyHelm → heron-ops)
      └── remediate_*    → dry-run patches (guardrailed)
              │
     ┌────────┼────────┐
     ▼        ▼        ▼
 ShrikeOps  Heron Ops   Starling EX CE (warbleoss.org)
  scanner   live MCP    self-host community edition
```

Public demo MCP: **[https://starling-ex.frakma.io/lite/mcp](https://starling-ex.frakma.io/lite/mcp)**  
Brand map: [`docs/BRAND.md`](docs/BRAND.md) · CE plan: [`docs/STARLING_EX_CE.md`](docs/STARLING_EX_CE.md)

---

## Submodules

| Path | Remote | Product | Role |
|------|--------|---------|------|
| [`shrikeops/`](shrikeops/) | [warble-tech/shrikeops](https://github.com/warble-tech/shrikeops) | **ShrikeOps** | Manifest / Helm security scanner |
| [`shrikeops-demo/`](shrikeops-demo/) | [warble-tech/shrikeops-demo](https://github.com/warble-tech/shrikeops-demo) | ShrikeOps | Golden/bad YAML + CI demo |
| [`heron-ops/`](heron-ops/) | [warble-tech/heron-ops](https://github.com/warble-tech/heron-ops) | **Heron Ops** | Live Kubernetes MCP (stdio + HTTP; was SteadyHelm) |
| [`starling/`](starling/) | [warble-tech/starling](https://github.com/warble-tech/starling) | Starling | CLI **release channel** (binaries; full product source private) |

Umbrella-owned (not submodules):

| Path | Purpose |
|------|---------|
| [`mcp/`](mcp/) | Stable MCP tool schemas + Starling Lite facade |
| [`deploy/agentgateway/`](deploy/agentgateway/) | Agent Gateway + public `/lite` Ingress |
| [`brand/`](brand/) | Heron Ops + ShrikeOps logos |
| [`docs/`](docs/) | Architecture, brand, CE plan, safety |
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
make heron-ops
./bin/heron-ops --mode=readonly --transport=stdio
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

### 5. Agent Gateway front door (recommended for multi-client)

[Agentgateway](https://agentgateway.dev) federates MCP, CORS, metrics, and optional LLM routing:

```bash
# Terminal A
make starling-lite
./bin/starling-lite --mode=readonly --transport=http --addr=:8822 \
  --registry=mcp/tools/registry.json

# Terminal B
make agentgateway-local
# MCP:   http://127.0.0.1:3000/mcp
# Admin: http://127.0.0.1:15000/ui/

python3 agents/reflexion/run.py --mcp http://127.0.0.1:3000/mcp \
  --path shrikeops-demo/k8s/bad/root-container.yaml
```

K8s (replaces the parked nginx stub in `warblecloud-ai`):

```bash
make agentgateway-image   # needs docker + AR push access
make agentgateway-apply
```

**Public (GKE):** [https://starling-ex.frakma.io/lite/mcp](https://starling-ex.frakma.io/lite/mcp)

Full docs: [`deploy/agentgateway/README.md`](deploy/agentgateway/README.md).

### 6. Scan manifests (ShrikeOps)

```bash
cd shrikeops/scanner
go run ./cmd/cli/main.go scan ../../shrikeops-demo/k8s/bad/root-container.yaml
```

---

## Leverage roadmap (OSS)

| Phase | What | Status |
|-------|------|--------|
| **0** | Umbrella + submodules + hooks + master docs | **done** |
| **1** | Heron Ops embed API (`pkg/mcpserver`, was SteadyHelm) | **done** |
| **2** | Starling Lite facade + public `/lite` MCP | **done** |
| **3** | Reflexion actor/critic scaffold | **scaffold** |
| **4** | Heron Ops rename + warbleoss.org CE packaging | **in progress** |
| **5** | Starling EX Community Edition on warbleoss.org | planned |
| **6** | Plugin protocol + SARIF / GitHub Checks | planned |

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
git add shrikeops heron-ops starling shrikeops-demo
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
| [`docs/BRAND.md`](docs/BRAND.md) | Flock brand map (Heron Ops, ShrikeOps, Starling) |
| [`docs/STARLING_EX_CE.md`](docs/STARLING_EX_CE.md) | Community Edition plan for warbleoss.org |
| [`docs/HERON_OPS_RENAME.md`](docs/HERON_OPS_RENAME.md) | SteadyHelm → Heron Ops checklist |
| [`docs/architecture.md`](docs/architecture.md) | Technical stack, OSS vs SaaS |
| [`docs/MAINTAINING.md`](docs/MAINTAINING.md) | Submodules, remotes, hooks, release pins |
| [`docs/safety.md`](docs/safety.md) | RBAC, write gates, audit |
| [`docs/gke-pdb-unpermissive-2026-07.md`](docs/gke-pdb-unpermissive-2026-07.md) | GKE Active Assist PDB fix on `warble-cluster` |
| [`mcp/README.md`](mcp/README.md) | Tool layout + client setup |
| [`mcp/PROTOCOL.md`](mcp/PROTOCOL.md) | MCP JSON-RPC conventions for this flock |
| [`starling/INSTRUCTIONS.md`](starling/INSTRUCTIONS.md) | Product Starling setup (verify tools live) |
| [`heron-ops/README.md`](heron-ops/README.md) | Heron Ops (cluster MCP) tools & deploy |
| [`charts/starling-ex-ce/`](charts/starling-ex-ce/) | Starling EX CE Helm chart |
| [warbleoss.org](https://warbleoss.org) | Community home + Try MCP |
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
