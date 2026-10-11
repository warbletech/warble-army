# warbleoss.org — landing content (source of truth for copy)

Implement in [`warbleoss.org/index.html`](../../warbleoss.org/index.html) (workspace) / `github.com/warbletech/warbleoss.org`.  
Portfolio order and open-source waves: monorepo [`OSS_PORTFOLIO_ALIGNMENT.md`](../../OSS_PORTFOLIO_ALIGNMENT.md).

---

## Banner

**Starling EX Community** — planning **CNCF** donation (Sandbox path) · **Frakma-X** — Linux Foundation wing for platform / LLMOps

---

## Hero

**Warble OSS**  
Open-source Kubernetes agent ops

Scan with **ShrikeOps**. Observe with **Heron Ops**. Coordinate with **Starling EX Community**.  
Birds of a feather — self-hosted, no lock-in. Adapt and extend the flock under Apache-2.0.

Pills:

- CNCF Sandbox — application draft  
- Frakma-X · LF wing (forming)  
- Self-host CE · no phone-home  

CTAs: [Try MCP live](https://starling-ex.frakma.io/lite/mcp) · [Get the army](https://github.com/warbletech/warble-army) · Easy first PR

---

## The flock

### ShrikeOps
Security scanner for manifests & Helm. Impales misconfigs before production.  
→ github.com/warbletech/shrikeops

### Heron Ops
Kubernetes MCP server. Live pods, logs, events, metrics for any agent IDE.  
→ (was SteadyHelm) github.com/warbletech/heron-ops

### Starling EX Community
Self-hosted CE: Starling Lite + Heron Ops + ShrikeOps, one chart.  
**Planned CNCF donation** (Warble stewards until then).  
→ warble-army charts/starling-ex-ce · GOVERNANCE.md

---

## Easy places to start (contributor funnel)

| Track | Repos | Why |
|-------|-------|-----|
| Good first | shrikeops-demo, shrikeops, heron-ops | Small surface, fixtures, flags, docs |
| Docs / site | warbleoss.org, community | No cluster required |
| Local stack | warble-bootstrap (public) | k3s / Rancher one-shot |
| LF wing | Frakma-X (planned); frakma-oss **private** | Not public OSS until chartered |

---

## Install (preview)

```bash
git clone --recurse-submodules https://github.com/warbletech/warble-army.git
cd warble-army && make starling-lite
./bin/starling-lite --mode=readonly --transport=http --addr=:8822
```

Live demo: `https://starling-ex.frakma.io/lite/mcp`

Helm:

```bash
helm upgrade --install starling-ex-ce ./charts/starling-ex-ce \
  --namespace warble-army --create-namespace
```

---

## Foundations

| | **Starling EX Community** | **Frakma-X** |
|--|---------------------------|--------------|
| Layer | Agent ↔ cluster ops (MCP) | Platform / LLMOps openness |
| Path | **CNCF donation (planned)** | **Linux Foundation wing** |
| Today | warbleoss.org + army repos | frakma-oss forming under LF |
| Funding later | LFX after Sandbox + community | LF program / separate fund |

## Community

- Discord Office Hours: Saturday 7–8 PM IST — https://discord.gg/WHsJkEFTt  
- RFCs / WGs / showcase — github.com/warbletech/community  

## Support / crowdfunding

- **Now:** use CE, PRs, adopters list, stars  
- **Next:** [LFX Crowdfunding](https://crowdfunding.lfx.linuxfoundation.org/) after 100% public OSS + multi-contributor traction  
- **Enterprise (optional):** warblecloud.com · frakma.io/enterprise  

## Commercial

Need multi-tenant SaaS, GitHub App, or managed FRAKMA?  
→ [warblecloud.com](https://warblecloud.com) · [frakma.io/enterprise](https://frakma.io/enterprise)

Community stays free and self-hosted. No license phone-home.
