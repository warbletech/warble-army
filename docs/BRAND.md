# Warble Army — flock brand map

Open-source birds of Kubernetes ops. Commercial Starling Cloud remains under Warble; **Starling EX Community** ships under **[warbleoss.org](https://warbleoss.org)** and is **planned for CNCF donation**. Platform openness under the Linux Foundation lives in **Frakma-X** (LF wing)—see [GOVERNANCE.md](GOVERNANCE.md).

## The flock

| Product (public name) | Bird | Role | Repo / code today | Status |
|-----------------------|------|------|-------------------|--------|
| **ShrikeOps** | Shrike | Hunt misconfigs, CVEs, deprecated APIs; scored reports | `warbletech/shrikeops` | OSS (Apache-2.0) |
| **Heron Ops** | Heron | Live cluster MCP — observe pods, logs, events, metrics; gated writes | `warbletech/heron-ops` (was SteadyHelm) | **OSS renamed** |
| **Starling Lite** | Starling (CE) | MCP facade: scan + cluster + remediate registry | `warble-army/mcp/starling-lite` | OSS |
| **Agent Gateway** | — (infra) | Front door: MCP federation, CORS, metrics | `deploy/agentgateway` + agentgateway.dev | Deployed |
| **Starling EX** | Starling | Full agent-facing K8s ops product | private `warble-core` + public release channel | SaaS / Enterprise |
| **Starling EX Community** | Starling | Self-host CE flock (Lite + Heron + Shrike); **CNCF donation planned** | `charts/starling-ex-ce` + **warbleoss.org** | **Chart + site; CNCF path** |
| **Frakma-X** | — | **Linux Foundation wing** for open FRAKMA / platform work | *forming* | LF program (planned) |
| **Reflexion** | — | Actor/critic agent loop over MCP tools | `agents/reflexion` | Scaffold |
| **Army umbrella** | Flock | Submodules, MCP tool registry, CI, brand | `warbletech/warble-army` | Live |

### Bird metaphor (keep it consistent)

- **Shrike** — impales prey on thorns → pins findings to a scored report.  
- **Heron** — still water, sudden strike → calm observation of the cluster, precise tool use.  
- **Starling** — murmurations / flock coordination → multi-tool agents and product surface.  
- **Army / flock** — the collection of specialized birds working together.

## Logos

| Mark | Path |
|------|------|
| Heron Ops | [`brand/heron-ops/heron-ops-logo.jpg`](../brand/heron-ops/heron-ops-logo.jpg) |
| ShrikeOps | [`brand/shrike-ops/shrike-ops-logo.jpg`](../brand/shrike-ops/shrike-ops-logo.jpg) |

Use on white or light backgrounds. Wordmark optional: **Heron Ops** / **ShrikeOps** in Inter or system sans, weight 600.

## Naming rules

| Context | Prefer | Avoid |
|---------|--------|-------|
| User-facing docs, website, warbleoss.org | **Heron Ops**, **ShrikeOps**, **Starling EX CE** | SteadyHelm (legacy) |
| CLI binary (transition) | `heron-ops` (new), keep `steadyhelm` alias one major | Breaking rename without alias |
| Go module (transition) | `github.com/warbletech/heron-ops` after migrate | Silent dual modules forever |
| GitHub org for pure OSS | `warbletech` or future `warble-oss` | Mixing private SaaS secrets into CE repos |
| Public MCP URL (today) | `https://starling-ex.frakma.io/lite/mcp` | Advertising private endpoints as CE |

## Package identity (transition)

```text
SteadyHelm  ──rename──►  Heron Ops
  binary: steadyhelm  →  heron-ops  (+ steadyhelm symlink)
  module: github.com/warblecloud/steadyhelm
       →  github.com/warbletech/heron-ops
  pkg:    pkg/mcpserver (unchanged API surface)
```

ShrikeOps spelling stays **ShrikeOps** (product) / `shrikeops` (CLI & repo). Optional hyphenated **shrike-ops** only in prose for parallelism with heron-ops.

## Domains

| Domain | Purpose |
|--------|---------|
| [warbleoss.org](https://warbleoss.org) | **Starling EX Community** home (CNCF path candidate) |
| [warblecloud.com](https://warblecloud.com) | Commercial Starling / Warble Cloud product |
| [frakma.io](https://frakma.io) | FRAKMA platform / enterprise |
| starling-ex.frakma.io/lite | Public **Starling Lite** MCP (CE-friendly demo) |
| *Frakma-X (LF)* | Linux Foundation wing (when chartered) |

## Foundations (short)

| Wing | Foundation | Project surface |
|------|------------|-----------------|
| **Starling EX Community** | **CNCF** (planned donation) | Agent ↔ cluster ops (MCP, scan, remediate) |
| **Frakma-X** | **Linux Foundation** | Open platform / LLMOps stack collaboration |

Do not say “CNCF project” or “LF project” until the foundation accepts the project. Prefer: *“planning CNCF donation”* / *“Linux Foundation wing”*.

## See also

- [GOVERNANCE.md](GOVERNANCE.md) — commercial vs OSS vs CNCF vs LF  
- [STARLING_EX_CE.md](STARLING_EX_CE.md) — Community technical scope and roadmap  
- [architecture.md](architecture.md) — technical stack  
