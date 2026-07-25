# CNCF Sandbox Application — Starling EX Community

**Status:** DRAFT for review (not yet filed on `cncf/sandbox`)  
**Target form:** [CNCF Sandbox application](https://github.com/cncf/sandbox/issues/new?template=application.yml)  
**Issue title when filing:** `[Sandbox] Starling EX Community`  
**Steward today:** ChirpStack LLP (Warble Cloud)  
**Related:** [GOVERNANCE.md](../GOVERNANCE.md) · [STARLING_EX_CE.md](../STARLING_EX_CE.md)

> **Caution (CNCF):** Do not represent this project as “donated” or “a CNCF project” until the TOC votes to accept **and** a Contribution Agreement is signed. This document is an internal draft for filing.

---

## How to file

1. Complete remaining gaps in [SANDBOX_CHECKLIST.md](SANDBOX_CHECKLIST.md) (CoC, SECURITY, MAINTAINERS, adopters).  
2. Review trademarks / name (“Starling EX Community” vs commercial Starling EX).  
3. Open: https://github.com/cncf/sandbox/issues/new?template=application.yml  
4. Paste answers below into the form fields.  
5. Sign Contribution Agreement when CNCF sends it (before TOC vote can complete).

**Suggested TAG:** App Delivery and/or Runtime (agent/tooling ↔ Kubernetes).  
**Suggested landscape category:** Observability / Continuous Integration & Delivery / Scheduling & Orchestration (confirm with TAG).

---

## Form answers (ready to paste)

### Project summary

> Open-source Kubernetes agent ops: MCP tools to scan, observe, and remediate cluster workloads with guardrails.

### Project description

Starling EX Community is a self-hosted, open flock of tools that let AI agents operate Kubernetes safely:

- **ShrikeOps** — scan manifests and Helm charts for misconfigurations, deprecated APIs, reliability and security findings with a scored report.  
- **Heron Ops** (formerly SteadyHelm) — Model Context Protocol (MCP) server exposing live cluster state (pods, logs, events, metrics) with readonly/readwrite modes and RBAC.  
- **Starling Lite** — unified MCP facade and tool registry (scan + cluster + dry-run remediation).  
- **starling-ex-ce** Helm chart — one-install Community Edition packaging.

Cloud-native platforms increasingly use LLM agents for Day-2 ops, but tool access is fragmented (`kubectl` hacks, proprietary SaaS, or ad-hoc scripts). Starling EX Community provides **vendor-neutral, MCP-native** contracts so any agent IDE or runtime (Claude, Cursor, custom) can inspect and improve clusters without lock-in.

The commercial Starling Cloud product (multi-tenant SaaS, billing, hosted history) remains outside this donation. Only the open agent-ops flock is proposed for CNCF.

### Org repo URL

`N/A` (multi-repo under `github.com/warble-tech`, not all org repos in scope)

### Project repo URL in scope of application

Primary umbrella:  
https://github.com/warble-tech/warble-army

### Additional repos in scope of the application

- https://github.com/warble-tech/heron-ops — Heron Ops (Kubernetes MCP server)  
- https://github.com/warble-tech/shrikeops — ShrikeOps scanner  
- https://github.com/warble-tech/shrikeops-demo — scanner demos / golden manifests  
- https://github.com/warble-tech/starling — public release channel (binaries; product docs) — *optional; confirm before filing*  
- Helm chart lives in `warble-army/charts/starling-ex-ce`

**Explicitly out of scope:** private `warble-core`, Warble Cloud SaaS, commercial Starling EX control plane, Frakma-X / FRAKMA commercial platform (Frakma-X is a separate Linux Foundation wing).

### Website URL

https://warble-tech.github.io/warbleoss.org/  
(Custom domain https://warbleoss.org pending DNS)

Live MCP demo: https://starling-ex.frakma.io/lite/mcp  

### Roadmap

https://github.com/warble-tech/warble-army/blob/main/docs/STARLING_EX_CE.md  

### Roadmap context

Near-term: harden CE install (Helm + install script), complete community governance files, expand remediation dry-run quality, Agent Gateway federation docs. Mid-term: plugin/tool registry stability, SARIF/GitHub Checks for ShrikeOps, multi-cluster readonly Heron Ops. CNCF path: sandbox onboarding, TAG engagement, adopter growth, then incubation criteria.

### Contributing guide

https://github.com/warble-tech/warble-army/blob/main/docs/cncf/CONTRIBUTING.md  

### Code of Conduct (CoC)

https://github.com/warble-tech/warble-army/blob/main/docs/cncf/CODE_OF_CONDUCT.md  

### Adopters

https://github.com/warble-tech/warble-army/blob/main/docs/cncf/ADOPTERS.md  

### Maintainers file

https://github.com/warble-tech/warble-army/blob/main/docs/cncf/MAINTAINERS.md  

### Security policy file

https://github.com/warble-tech/warble-army/blob/main/docs/cncf/SECURITY.md  

### Standard or specification?

Not a formal standard. Includes a **stable MCP tool registry contract** (`mcp/tools/registry.json`) describing names/schemas for scan, cluster, and remediate tools so agents and gateways interoperate. Aligns with the open [Model Context Protocol](https://modelcontextprotocol.io) rather than defining a competing protocol.

### Business product or service to project separation

ChirpStack LLP (Warble Cloud) offers **Starling EX** commercial/SaaS offerings and **FRAKMA** enterprise platform services.

**Starling EX Community** is the open upstream for agent ↔ cluster ops (Heron Ops, ShrikeOps, Starling Lite, CE chart). Commercial products may consume Community as upstream dependencies but:

- SaaS multi-tenancy, billing, hosted dashboards, and private control planes are **not** in this application.  
- Development of Community components happens in public Apache-2.0 repos.  
- Maintainership and roadmap for Community will move to CNCF governance if accepted; Warble remains one contributor among others.  
- **Frakma-X** (Linux Foundation wing for platform openness) is **orthogonal** and not part of this sandbox application.

### Why CNCF?

Agent-driven Kubernetes operations is emerging infrastructure. Placing Starling EX Community in CNCF:

- Keeps MCP tool contracts and scanners **vendor-neutral**  
- Aligns with CNCF’s Kubernetes-centric landscape and existing security/observability projects  
- Gives adopters a foundation-backed home instead of a single-vendor open core  
- Enables TAG collaboration (App Delivery / Runtime / Security) for safe agent write paths  

### Benefit to the landscape

Bridges **AI agents** and **Kubernetes** with production-minded defaults (readonly MCP, dry-run remediation, RBAC). Complements static policy engines and scanners by adding an **agent-facing MCP surface** and a unified flock install. Fills the gap between “run kubectl in a loop” and proprietary AIOps SaaS.

### Cloud native 'fit'

- Runs on and targets CNCF Kubernetes  
- Delivers cluster observability and change workflows as **cloud-native APIs** (HTTP/stdio MCP, Helm install)  
- Designed for multi-environment (any CNCF-conformant cluster)  
- Separates data plane tools from proprietary control planes  

### Cloud native 'integration'

- **Kubernetes** — primary API via client-go (Heron Ops)  
- **Helm** — chart install; ShrikeOps scans rendered charts  
- **OPA/Rego** — ShrikeOps policy bundles  
- **cert-manager / Ingress nginx** — reference deployments  
- **agentgateway** (upstream OSS) — optional MCP gateway  
- Metrics via **metrics-server** (optional)  

### Cloud native overlap

- **Kyverno / OPA Gatekeeper / Polaris** — policy and config best practices (ShrikeOps *composes* several scanners rather than replacing admission controllers)  
- **kubectl / k9s / headlamp** — human UIs; we target **agent/MCP** interfaces  
- **Keptn / CloudEvents-based remediation** — different event models; we focus on MCP tool calls  
- No single CNCF project currently ships this exact MCP flock packaging  

### Similar projects

| Project | Relation |
|---------|----------|
| Kubernetes MCP experiments / community MCP servers | Overlap on live cluster tools; Heron Ops is production-oriented with mode gates |
| kube-score, Polaris, Pluto | Used *inside* ShrikeOps scoring |
| Portkey / LiteLLM / agent gateways | LLM/MCP routing; we focus on K8s ops tools |
| Proprietary AIOps (various) | Closed; we open the tool contracts |

### Landscape

Not yet listed on https://landscape.cncf.io/ — will apply for listing if accepted to Sandbox.

### Insights

Not yet listed on LFX Insights — will onboard if accepted.

### Trademark and accounts

☐ If accepted, we agree to donate project trademarks and accounts to the CNCF  
*(Must be checked on the GitHub form after legal review.)*

### IP policy

☐ If accepted, the project will follow the CNCF IP Policy  
*(Must be checked on the GitHub form.)*

### Will the project require a license exception?

N/A — core components target Apache-2.0. Dependencies are intended to comply with the CNCF Allowlist; a dependency scan will be attached before filing if any exception is needed.

### Project "Domain Technical Review"

Not yet. Plan: present to relevant TAG (App Delivery and/or Runtime) and complete Day 0 of the General Technical Review questionnaire before or during TOC review.

### Application contact email(s)

`contact@warblecloud.com`, `avinash@warblecloud.com`  
*(Add additional maintainers before filing.)*

### Contributing or sponsoring entity signatory information

| Name | Address | Type | Signatory name and title | Email address |
|------|---------|------|--------------------------|---------------|
| ChirpStack LLP | *legal address TBD* | *entity type TBD* | *signatory TBD* | contact@warblecloud.com |

### CNCF contacts

*None yet — to be filled after TAG outreach.*

### Additional information

- Public MCP demo: https://starling-ex.frakma.io/lite/mcp  
- Flock brand map: https://github.com/warble-tech/warble-army/blob/main/docs/BRAND.md  
- Separation of **Frakma-X** (Linux Foundation wing for platform) from this CNCF agent-ops path is intentional.  
- Commercial site: https://warblecloud.com (not in scope)

---

## Filing checklist (internal)

See [SANDBOX_CHECKLIST.md](SANDBOX_CHECKLIST.md).
