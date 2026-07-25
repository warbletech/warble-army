# Starling EX — Community Edition (warbleoss.org)

**Status:** Planned  
**Home:** [warbleoss.org](https://warbleoss.org)  
**Demo MCP (today):** [https://starling-ex.frakma.io/lite/mcp](https://starling-ex.frakma.io/lite/mcp)

## What CE is

**Starling EX Community Edition** is the self-hosted, Apache-2.0 (or equivalent) open flock for Kubernetes agent ops:

| Include in CE | Keep commercial (Warble Cloud) |
|---------------|--------------------------------|
| **Heron Ops** (cluster MCP) | Multi-tenant Starling Cloud control plane |
| **ShrikeOps** scanner CLI + server | Hosted score history, GitHub App SaaS |
| **Starling Lite** MCP facade + registry | Org SSO, billing, private Brain graph |
| Agent Gateway configs (agentgateway.dev) | Managed multi-cluster mesh |
| Reflexion actor/critic **scaffold** | Managed LLM routing / eval fleet |
| Helm / Kustomize install on any CNCF cluster | SLA, enterprise support |
| Docs + Office Hours via community | Private warble-core product source |

CE is **not** a thin open-core tease of every Starling Cloud API. It is a complete local/self-host path: scan → observe → propose → (optional) apply with guardrails.

## Proposed repo layout (warbleoss.org era)

```text
warble-tech/warble-army          # umbrella (submodules + MCP registry) — exists
warble-tech/shrikeops            # ShrikeOps — exists (public)
warble-tech/heron-ops            # rename of steadyhelm — planned
warble-tech/starling-ex-ce       # CE packaging: Helm chart, install docs, examples
warble-tech/starling             # product release channel (binaries) — optional CE binaries later
```

Website:

```text
warbleoss.org/
  /                          # flock home, birds, download
  /docs                      # install Heron Ops + ShrikeOps + Lite
  /starling-ex               # CE product page
  /community                 # Discord, Office Hours, RFCs
  /license                   # Apache-2.0 / dual-license notes
```

## Install story (target UX)

```bash
# One-liner (planned)
curl -fsSL https://warbleoss.org/install.sh | bash

# Or Helm
helm repo add warbleoss https://warbleoss.org/charts
helm install starling-ex-ce warbleoss/starling-ex-ce \
  --namespace warble-army --create-namespace
```

Exposes:

- MCP: `https://<your-host>/lite/mcp` (or ClusterIP)  
- Optional Agent Gateway admin UI  
- ShrikeOps scan API (optional)

## Phased roadmap

### Phase A — Brand & rename (now)

- [x] Flock brand map (`docs/BRAND.md`)
- [x] Heron Ops + ShrikeOps logo marks
- [x] Document SteadyHelm → Heron Ops transition
- [ ] Rename GitHub repo `steadyhelm` → `heron-ops` (or create `heron-ops` and archive steadyhelm)
- [ ] Binary alias: `heron-ops` primary, `steadyhelm` deprecated
- [ ] warbleoss.org landing (static or Cloudflare Pages)

### Phase B — CE package

- [ ] Helm chart `charts/starling-ex-ce` bundling starling-lite + heron-ops SA/RBAC + optional agentgateway
- [ ] `install.sh` / `make ce-up` against kind / k3s / GKE
- [ ] Public docs site content for warbleoss.org
- [ ] Good first issues + CONTRIBUTING under CE repos

### Phase C — Community loop

- [ ] Discord / Office Hours linked from warbleoss.org
- [ ] RFC process (reuse `warble-tech/community`)
- [ ] Showcase: “powered by Starling EX CE”
- [ ] Optional dual-track: CE tags vs Starling Cloud releases

## License stance

- Prefer **Apache-2.0** for CE components (aligned with ShrikeOps / SteadyHelm today).  
- CLA/DCO as used on warble-tech repos.  
- Commercial add-ons stay out of CE git history (no API keys, no proprietary Brain dumps).

## Boundary with frakma.io / warblecloud.com

| Surface | Audience |
|---------|----------|
| warbleoss.org | Open source users, contributors |
| starling-ex.frakma.io/lite | Live CE-compatible MCP demo |
| warblecloud.com | Paid Starling Cloud / Enterprise |
| frakma.io | FRAKMA platform, enterprise MLOps |

## Success metrics (CE)

1. `helm install` → working `tools/list` in &lt; 15 minutes on a clean cluster.  
2. External contributors land a PR in Heron Ops or ShrikeOps within first month of launch.  
3. Clear upgrade path documented: CE → Starling Cloud (optional, never required).
