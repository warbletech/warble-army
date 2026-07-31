# Governance & foundation map

How Warble’s commercial products, open-source flock, and foundation wings relate.  
**Status notes are intentional:** CNCF donation and LF programs are **plans**, not completed handoffs.

```text
                        ChirpStack LLP / Warble Cloud
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          │                         │                         │
   Commercial product          Open source flock        Foundation wings
   warblecloud.com             warbleoss.org              (planned / forming)
          │                         │                         │
   Starling EX (SaaS)        Starling EX Community    ┌───────┴───────┐
   Warble Cloud agents       Heron Ops · ShrikeOps    │               │
   Enterprise support        Starling Lite · Army     ▼               ▼
                                                   CNCF path      Linux Foundation
                                                   Starling EX    Frakma-X
                                                   Community      (LF wing)
```

## Entities

| Name | Role | Host today | Long-term intent |
|------|------|------------|------------------|
| **Warble Cloud** / ChirpStack LLP | Commercial product, trademarks, enterprise | warblecloud.com | Stays commercial |
| **Warble Army** / warbleoss.org | Umbrella for OSS birds (docs, charts, brand) | GitHub `warble-tech/*` | Neutral OSS home |
| **Starling EX Community** | Self-host agent ops CE (MCP flock) | warble-army + public repos | **Plan: donate to CNCF** |
| **Starling EX** (product) | Full commercial / SaaS agent platform | private core + release channel | Warble Cloud |
| **FRAKMA** | MLOps/LLMOps platform brand | frakma.io | Product / platform |
| **Frakma-X** | **Linux Foundation wing** for open platform work | *forming* | **LF project / program home** |

## Starling EX Community → CNCF (planned)

**Goal:** donate **Starling EX Community** (the self-host CE: Heron Ops + ShrikeOps integration + Starling Lite + army packaging) to the **CNCF** so cluster agent ops stays vendor-neutral and community-governed.

### In scope for a CNCF-bound project (intent)

- Heron Ops (cluster MCP)
- ShrikeOps scanner (or close collaboration if already independent)
- Starling Lite MCP facade + tool registry contracts
- Helm chart `starling-ex-ce` and install docs
- Governance docs, CoC, DCO/CLA as required by CNCF

### Out of scope for donation (stays Warble)

- Starling Cloud multi-tenant control plane
- Billing, SSO product, hosted score history
- Proprietary Brain / private `warble-core` product paths
- Warble trademarks and commercial brand sites (unless separately licensed)

### Maturity checklist (before sandbox / incubation proposal)

- [x] **GitHub repos public** (Wave 1: warbleoss.org, shrikeops-demo, shrikeops, heron-ops, warble-army, community) — *opened 2026-07-31*
- [x] Apache-2.0 (or equivalent) LICENSE files on primary components
- [x] Clear product rename (Heron Ops, ShrikeOps)
- [x] Live demo MCP (`starling-ex.frakma.io/lite/mcp`)
- [x] CE Helm chart
- [x] Documented maintainers + CoC + security policy → [docs/cncf/](cncf/)
- [x] Adopters list (seed) + sandbox application **draft** → [cncf/SANDBOX_APPLICATION.md](cncf/SANDBOX_APPLICATION.md)
- [ ] Multi-org maintainers / real external adopters
- [ ] CNCF TAG presentation / Day 0 GTR
- [ ] Trademark and name clearance (“Starling EX Community” vs commercial Starling)
- [ ] File Sandbox issue on `cncf/sandbox` + Contribution Agreement

**Portfolio open order + LFX crowdfunding path:** monorepo `OSS_PORTFOLIO_ALIGNMENT.md` (warble-opensource workspace).

Until donation completes, **Warble / ChirpStack** remains copyright holder and steward; language on sites must say **“planning CNCF donation”**, not “CNCF project”.

## Frakma-X → Linux Foundation wing

**Frakma-X** is Warble’s **Linux Foundation wing**: the home for open, foundation-aligned work around the **FRAKMA** stack (OpenFaaS, Ray, Argo, KServe, MLflow, Airflow patterns) and related platform standards—not the commercial FRAKMA managed service.

| | **FRAKMA** (product) | **Frakma-X** (LF wing) |
|--|----------------------|------------------------|
| Focus | Deployable MLOps/LLMOps platform, enterprise | Open specs, reference implementations, LF collaboration |
| Site | frakma.io | *TBD under LF / frakma-x branding* |
| Money | Commercial / enterprise | Foundation membership, grants, community |
| Relation to Starling | May host agents *on* FRAKMA | Orthogonal: platform wing vs agent-ops CNCF path |

Frakma-X does **not** replace Starling EX Community’s CNCF path. Different layers:

- **CNCF / Starling EX Community** — agent ↔ cluster ops (MCP, scan, remediate)  
- **LF / Frakma-X** — platform / LLMOps stack openness under Linux Foundation  

## Domain map (with foundations)

| Domain | Audience |
|--------|----------|
| [warbleoss.org](https://warbleoss.org) | OSS flock + CE; future CNCF project landing candidate |
| [warblecloud.com](https://warblecloud.com) | Commercial Starling / Warble |
| [frakma.io](https://frakma.io) | FRAKMA platform + enterprise |
| starling-ex.frakma.io/lite | Live CE-compatible MCP demo |
| *Frakma-X LF site* | Linux Foundation wing (when chartered) |

## Speaking rules (press, README, conferences)

| Say | Don’t say |
|-----|-----------|
| “Starling EX Community is planned for CNCF donation” | “Starling EX is a CNCF project” (until accepted) |
| “Frakma-X is our Linux Foundation wing” | “FRAKMA is an LF project” without Frakma-X distinction |
| “Heron Ops / ShrikeOps are open source under Warble stewardship” | Implying commercial features are open |
| “warbleoss.org hosts the community flock today” | Claiming LF/CNCF owns domains they don’t |

## Related docs

- [cncf/](cncf/) — **CNCF Sandbox application draft + checklist**  
- [BRAND.md](BRAND.md) — product / bird names  
- [STARLING_EX_CE.md](STARLING_EX_CE.md) — CE technical scope  
- [architecture.md](architecture.md) — stack  
