# CNCF Sandbox application (draft): warble-flock

**Status:** DRAFT. **Not filed.** Pre-Sandbox. We plan to file on the [CNCF Sandbox form](https://github.com/cncf/sandbox/issues/new?template=application.yml) in **late January 2027**, after this repository is six months old (it was created 25 July 2026).

**Name:** the project was renamed from *Warble Army* to **warble-flock**. The GitHub repository is still `warbletech/warble-army`. A vendor-neutral rename is planned before filing, and every name and URL below will be updated when it lands.

> **Speaking rule:** don't describe this project as "donated", "contributed", or "a CNCF project" until the CNCF TOC votes to accept it **and** the Contribution Agreement is signed. Until then it is "a project preparing a CNCF Sandbox application".

**Last refreshed:** 21 November 2026. Facts come from this repository, the public demo at https://mcp.warbleoss.org, and the current CNCF Sandbox form.

---

## How the application will be filed

1. Follow the [CNCF Sandbox README](https://github.com/cncf/sandbox/blob/main/README.md) and the [Project Lifecycle & Process](https://github.com/cncf/toc/blob/main/process/README.md). Respect the [Allowlist License Policy](https://github.com/cncf/foundation/blob/main/policies-guidance/allowed-third-party-license-policy.md).
2. Close the open items in [§ Readiness](#readiness-not-yet-ready-to-file).
3. Open the [application issue form](https://github.com/cncf/sandbox/issues/new?template=application.yml) ([form source](https://github.com/cncf/sandbox/blob/main/.github/ISSUE_TEMPLATE/application.yml)) and paste the answers below.
4. Tick the trademark, IP-policy and naming checkboxes only after legal review.
5. Sign the Contribution Agreement when CNCF sends it. The TOC can't complete a vote without it.

---

## Form answers (draft)

### Project summary

Open-source Kubernetes tools for AI agents over MCP: scan, observe, and propose remediations for cluster workloads, with read-only and dry-run defaults.

### Project description

warble-flock (formerly Warble Army) is a self-hosted, Apache-2.0 set of tools that lets AI agents work with Kubernetes safely:

- **ShrikeOps**: scans manifests and Helm charts for misconfigurations, deprecated APIs, and reliability and security findings, and produces a scored report. It composes Pluto, Polaris and kube-score.
- **Heron Ops** (formerly SteadyHelm): a Model Context Protocol (MCP) server that exposes live cluster state (pods, logs, events, metrics) with read-only/read-write modes and RBAC.
- **The read-only MCP server (Starling Lite)**: a unified MCP facade and tool registry for scanning, cluster reads, and dry-run remediation proposals.
- **`charts/starling-ex-ce`**: a Helm chart for one-step self-hosted installation.

Cloud native platforms increasingly use LLM agents for Day-2 operations, but tool access is fragmented across `kubectl` scripts, proprietary SaaS and ad-hoc wrappers. warble-flock provides vendor-neutral, MCP-native tool contracts so any agent runtime or IDE can inspect a cluster and propose fixes, with safe defaults: read-only mode, namespace allowlists, and dry-run patches as the write path ([docs/safety.md](../safety.md)).

Warble Cloud's commercial products (multi-tenant SaaS, billing, hosted history, FRAKMA product code) are not part of this application.

### Project vs Reference Architecture/Implementation

**Check:** This is a reusable open source project designed for broad adoption, not a reference architecture, reference implementation, or demonstration of patterns.

(The project ships MCP servers, a scanner, a tool registry, and a Helm chart that anyone can install on their own cluster. The demo manifests and the public read-only endpoint are examples, not the product.)

### Org repo URL

N/A. Multiple repositories under https://github.com/warbletech are in scope, but not every repository in the org.

### Project repo URL in scope of application

https://github.com/warbletech/warble-army (to be updated after the planned rename)

### Additional repos in scope of the application

- https://github.com/warbletech/heron-ops: Heron Ops (Kubernetes MCP server)
- https://github.com/warbletech/shrikeops: ShrikeOps scanner
- https://github.com/warbletech/shrikeops-demo: scanner demos and golden/bad manifests
- The Helm chart lives in this repository under `charts/starling-ex-ce`.

Out of scope: the private `warble-core` repository, Warble Cloud SaaS, and commercial control planes and product code.

### Website URL

https://warbleoss.org/

### Roadmap

Currently the README's [Leverage roadmap (OSS)](../../README.md#leverage-roadmap-oss) table. A dedicated `ROADMAP.md` will be added before filing.

### Roadmap context

Shipped: the umbrella repository with submodules and hooks, the Heron Ops embed API, the Starling Lite facade, and the public read-only MCP endpoint. In progress: Community Edition Helm packaging and the warbleoss.org site. Planned: a plugin protocol, and SARIF / GitHub Checks output for ShrikeOps.

Before filing we also plan to attach a read-only demo cluster to the public MCP endpoint, publish a 15-minute evaluator quickstart, and cut signed releases. After Sandbox acceptance the focus is adopters and maintainers from more than one organization.

### Contributing guide

https://github.com/warbletech/warble-army/blob/main/docs/cncf/CONTRIBUTING.md

### Code of Conduct (CoC)

https://github.com/warbletech/warble-army/blob/main/docs/cncf/CODE_OF_CONDUCT.md (Contributor Covenant 2.1)

### Adopters

https://github.com/warbletech/warble-army/blob/main/docs/cncf/ADOPTERS.md

Currently the only listed adopter is the steward (Warble Cloud). There are no external production adopters yet.

### Maintainers file

https://github.com/warbletech/warble-army/blob/main/docs/cncf/MAINTAINERS.md

| Maintainer | GitHub ID | Company/Organization |
| ---------- | --------- | -------------------- |
| Avinash Patil | @avinashetrx | Warblecloud, LLC |
| Yaksha | @yaksha-warble | Warblecloud, LLC |
| Akshat Dwivedi | @akshat-warble | Warblecloud, LLC |

All current maintainers are from one company. We're actively looking for outside reviewers and maintainers. Avinash Patil's commits in this repository are mostly under the `@warble-tech` account.

### Security policy file

https://github.com/warbletech/warble-army/blob/main/docs/cncf/SECURITY.md

### Standard or specification?

N/A. This project isn't a formal standard. It includes a tool registry (`mcp/tools/registry.json`) that describes tool names and schemas so agents and gateways can interoperate, and it builds on the open [Model Context Protocol](https://modelcontextprotocol.io) rather than defining a competing protocol.

### Business product or service to project separation

Warblecloud, LLC (Warble Cloud) offers commercial SaaS and enterprise platform products. warble-flock is the open upstream for agent-to-cluster operations (ShrikeOps, Heron Ops, Starling Lite, the CE chart):

- Multi-tenant SaaS, billing, hosted dashboards and private control planes are **not** part of this application.
- Development of the in-scope components happens in public Apache-2.0 repositories.
- If accepted, maintainership and roadmap move to CNCF governance, and Warble becomes one contributor among others.
- The in-scope code is and stays Apache-2.0.

### Dependency on paid cloud, community, and SaaS services or subscriptions

The tools can be built and used locally with a kubeconfig (stdio or local HTTP MCP) with no paid subscription. The steward hosts a public website (https://warbleoss.org/) and a public read-only MCP endpoint (https://mcp.warbleoss.org) so evaluators can try the project. CI runs on GitHub Actions public runners. The hosting cost and a post-acceptance hosting plan will be stated in the filed application.

### Why CNCF?

Agent-driven Kubernetes operations is emerging infrastructure. A CNCF home would:

- keep MCP tool contracts and safety defaults for clusters vendor-neutral rather than inside one company's open core
- connect the project with CNCF work on AI agents and cluster operations (see overlap below), including the TOC's Distributed Agentic Systems initiative
- give adopters a neutral home and a clear Apache-2.0 IP policy
- enable TAG collaboration on safe agent write paths (read-only defaults, dry-run remediation, RBAC)

### Benefit to the landscape

It bridges AI agents and Kubernetes with production-minded defaults (read-only MCP, dry-run remediation, RBAC). It complements policy engines, scanners and agent runtimes by adding an agent-facing MCP tool layer, without requiring a particular agent framework.

### Cloud native 'fit'

- Kubernetes is the system of record (client-go in Heron Ops; manifest and Helm scanning in ShrikeOps).
- Cluster observation and change proposals are delivered as cloud native APIs (HTTP/stdio MCP, Helm install).
- Designed for any conformant Kubernetes cluster, not a single vendor's control plane.
- Safety defaults follow common operations practice: least privilege, dry-run first, namespace allowlists.

### Cloud native 'integration'

- **Kubernetes**: primary API via client-go (Heron Ops)
- **Helm**: CE chart install; ShrikeOps scans rendered charts
- **OPA/Rego**: ShrikeOps policy bundles
- **agentgateway** (https://agentgateway.dev): optional MCP gateway
- **metrics-server**: optional live metrics
- **Pluto, Polaris, kube-score**: composed inside ShrikeOps
- **MCP**: agent runtimes such as kagent can call these tools instead of wrapping `kubectl`

### Cloud native overlap

- **kagent** (CNCF project; Sandbox May 2025, Incubation application open): a Kubernetes-native *agent runtime*. warble-flock isn't an agent framework. It's a set of MCP tools that a runtime such as kagent, or an IDE, can call.
- **K8sGPT** (CNCF Sandbox, December 2023): AI-assisted Kubernetes diagnosis. Overlaps on "explain my cluster". warble-flock differs in its MCP-native tool contracts, composed scanner report, and dry-run patches rather than a standalone SRE assistant.
- **HolmesGPT** (CNCF Sandbox; TOC vote passed, application closed 8 October 2025): an AI SRE agent that investigates Kubernetes alerts. It's the closest overlap in purpose. warble-flock is a tool layer (scan, read, propose dry-run patches) that agents like it could call, not an investigation agent.
- **CAIPE** (CNCF Sandbox; TOC vote passed, application closed 9 November 2026): a shared platform for building and governing agents. It works at a different layer (agent platform vs cluster MCP tools).
- **Kyverno / OPA Gatekeeper / Polaris**: admission and configuration policy. ShrikeOps composes scanners and doesn't replace admission controllers.
- **kubectl / k9s / Headlamp**: human-facing tools. warble-flock targets agent and MCP interfaces.

### Similar projects

| Project | Relation |
|---------|----------|
| kagent (CNCF) | Agent runtime; a complementary MCP consumer, not a substitute |
| K8sGPT (CNCF Sandbox) | AI cluster diagnosis; different interface and packaging |
| HolmesGPT (CNCF Sandbox) | AI SRE investigation agent; could consume these tools |
| CAIPE (CNCF Sandbox) | Agent platform; different layer |
| kubernetes-mcp-server, mcp-server-kubernetes, kubectl-mcp-server | Broad Kubernetes-over-MCP tool sets. Heron Ops is a narrower, mode-gated cluster MCP, combined with scanning and dry-run remediation |
| kube-score, Polaris, Pluto | Used *inside* ShrikeOps scoring |
| agentgateway and other MCP gateways | Optional front door; this project focuses on Kubernetes ops tools |

This space is moving quickly. The landscape will be re-checked right before filing.

### Landscape

Not listed on https://landscape.cncf.io/ yet.

### Trademark and accounts

To be ticked only after legal review. A vendor-neutral rename is planned before filing.

### IP policy

To be ticked only after legal review.

### Project naming

To be ticked only after the planned rename and a trademark review.

### Does the project name use "kube", "k8s", or another existing project's prefix/suffix?

No.

### Will the project require a license exception?

N/A. The in-scope repositories are Apache-2.0.

### Does the project utilize any dependencies with licenses not on the allowlist or an approved exception?

Go code in the Starling Lite image, Heron Ops and the ShrikeOps scanner uses only Apache-2.0, MIT, BSD-3-Clause and ISC dependencies (a go-licenses report will be attached to the filed application). The bundled scanners Pluto and Polaris are Apache-2.0, and kube-score is MIT. The optional ShrikeOps web UI (Next.js) pulls in `sharp-libvips` (LGPL; covered by the CNCF blanket exception for Next.js, unmodified, build/install-time) and `caniuse-lite` (CC-BY-4.0, browser-compatibility data pulled in transitively by Next.js). We'd welcome CNCF guidance on caniuse-lite. Dev-only tooling includes BlueOak-1.0.0 and CC0 packages that aren't distributed. The scan will be re-run right before filing.

### Project "Domain Technical Review"

Not yet. Plan: engage **TAG Operational Resilience** (primary: Day-2 operations and troubleshooting) and the **Distributed Agentic Systems initiative** under **TAG Workloads Foundation** (MCP for clusters, and governance/policy for autonomous agents). We'll also complete Day 0 of the [General Technical Review](https://github.com/cncf/toc/blob/main/operations/project-reviews/general-technical-questions.md). Links to notes or recordings will be added here.

### Application contact email(s)

contact@warblecloud.com (more maintainer contacts will be added at filing)

### Contributing or sponsoring entity signatory information

| Name | Address | Type | Signatory name and title | Email address |
|------|---------|------|--------------------------|---------------|
| Warblecloud, LLC | to be provided at filing | Delaware limited liability company | to be provided at filing | contact@warblecloud.com |

### CNCF contacts

None yet.

### Additional information

- Project name: **warble-flock** (formerly Warble Army). The repository is still `warbletech/warble-army`, and a vendor-neutral rename is planned before filing.
- Public read-only MCP endpoint: https://mcp.warbleoss.org. On 21 November 2026, `/health` reported `"mode":"readonly"`, `"tools":5` and `"cluster":false` (Starling Lite 0.3.0): five read-only/dry-run tools, with no demo cluster attached yet.
- Safety defaults: [docs/safety.md](../safety.md)
- OpenSSF Scorecard: https://scorecard.dev/viewer/?uri=github.com/warbletech/warble-army
- Evaluator quickstart: a 15-minute path is proposed in https://github.com/warbletech/warble-army/pull/41 (not merged yet).
- Repository age: created 25 July 2026, so we'll file after 25 January 2027 rather than before the six-month minimum.

---

## Readiness (not yet ready to file)

| Item | Status |
|------|--------|
| Apache-2.0 licence on in-scope repos | ✅ |
| Repository six months old | ⏳ after 25 January 2027 |
| Maintainers file with a Company/Organization column | ⚠️ the file exists; the header still says `Company` |
| Maintainers from more than one organization | ❌ all from one company today; recruiting outside reviewers |
| Vendor-neutral rename and trademark review | ⏳ planned before filing |
| Dependency licence scan | ✅ done; re-run before filing |
| Dedicated `ROADMAP.md` | ❌ to add |
| 15-minute quickstart on `main` | ❌ PR #41 open |
| Read-only demo cluster on the public endpoint | ❌ `"cluster": false` |
| GitHub releases | ❌ none on this repository yet |
| TAG engagement / Day 0 GTR | ⏳ planned |
| External adopters | ❌ none yet |

---

## Sources

- Application form: https://github.com/cncf/sandbox/blob/main/.github/ISSUE_TEMPLATE/application.yml
- Sandbox README (process and auto-close rules): https://github.com/cncf/sandbox/blob/main/README.md
- TAGs and initiatives: https://github.com/cncf/toc/tree/main/tags and https://github.com/cncf/toc/blob/main/tags/tag-workloads-foundation/initiatives/distributed-agentic-systems/README.md
- HolmesGPT application: https://github.com/cncf/sandbox/issues/392 · CAIPE application: https://github.com/cncf/sandbox/issues/528
- Public demo: https://mcp.warbleoss.org/health
