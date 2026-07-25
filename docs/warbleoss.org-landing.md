# warbleoss.org — landing content draft

Use this as copy for the static site (Cloudflare Pages / GitHub Pages).

---

## Hero

**Warble OSS**  
Open-source Kubernetes agent ops. Scan. Observe. Remediate.

Birds of a feather — **ShrikeOps**, **Heron Ops**, **Starling EX Community**  
*(planning donation to the CNCF)*. Platform openness under the Linux Foundation: **Frakma-X**.

[Get started](#install) · [GitHub](https://github.com/warble-tech/warble-army) · [Live MCP demo](https://starling-ex.frakma.io/lite/mcp)

---

## The flock

### ShrikeOps
Security scanner for manifests & Helm. Impales misconfigs before production.  
→ github.com/warble-tech/shrikeops

### Heron Ops
Kubernetes MCP server. Live pods, logs, events, metrics for any agent IDE.  
→ (was SteadyHelm) github.com/warble-tech/steadyhelm → heron-ops

### Starling EX Community
Self-hosted Community Edition: Starling Lite + Heron Ops + ShrikeOps, one chart.  
**Planned CNCF donation** (Warble stewards until then).  
→ github.com/warble-tech/warble-army/docs/STARLING_EX_CE.md · GOVERNANCE.md

---

## Install (preview)

```bash
git clone --recurse-submodules https://github.com/warble-tech/warble-army.git
cd warble-army && make starling-lite
./bin/starling-lite --mode=readonly --transport=http --addr=:8822
```

Live demo: `https://starling-ex.frakma.io/lite/mcp`

---

## Foundations

| | **Starling EX Community** | **Frakma-X** |
|--|---------------------------|--------------|
| Layer | Agent ↔ cluster ops (MCP) | Platform / LLMOps openness |
| Path | **CNCF donation (planned)** | **Linux Foundation wing** |
| Today | warbleoss.org + army repos | Forming under LF |

## Commercial

Need multi-tenant SaaS, GitHub App, or managed FRAKMA?  
→ [warblecloud.com](https://warblecloud.com) · [frakma.io/enterprise](https://frakma.io/enterprise)

Community stays free and self-hosted. No license phone-home.
