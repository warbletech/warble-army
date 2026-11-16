# CNCF Sandbox filing checklist — Starling EX Community

Track readiness before opening https://github.com/cncf/sandbox/issues/new?template=application.yml

## Application package

| Item | Status | Location |
|------|--------|----------|
| Sandbox application draft | ✅ draft | [SANDBOX_APPLICATION.md](SANDBOX_APPLICATION.md) |
| CONTRIBUTING | ✅ | [CONTRIBUTING.md](CONTRIBUTING.md) |
| CODE_OF_CONDUCT | ✅ | [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) |
| SECURITY | ✅ | [SECURITY.md](SECURITY.md) |
| MAINTAINERS | ⚠️ draft, TODOs left | [MAINTAINERS.md](MAINTAINERS.md) |
| ADOPTERS | ✅ template | [ADOPTERS.md](ADOPTERS.md) |
| GOVERNANCE (project) | ✅ seed | [PROJECT_GOVERNANCE.md](PROJECT_GOVERNANCE.md) |
| License Apache-2.0 on primary repos | ✅ GitHub detects Apache-2.0 | heron-ops, shrikeops, shrikeops-demo, warble-army |
| Dependency license scan | ⬜ | Attach before filing |
| Trademark legal review (name) | ⬜ | Counsel |
| Contribution Agreement signatory | ⬜ | Warblecloud, LLC (TODO: signatory name and title) |
| TAG presentation / Day 0 GTR | ⬜ | App Delivery / Runtime |
| File issue on cncf/sandbox | ⬜ | After checklist green |

## Pre-flight technical

| Item | Status |
|------|--------|
| Public repos | ✅ heron-ops, shrikeops, warble-army |
| Live demo MCP | ✅ https://mcp.warbleoss.org/mcp (read-only; cluster tools need an attached demo cluster) |
| Helm CE chart | ⚠️ charts/starling-ex-ce exists; anonymous image/chart pull still blocked |
| Speaking rules (not “CNCF project” yet) | ✅ GOVERNANCE.md |
| Adopters with real orgs | ⬜ need ≥1–3 named if possible |

## Public visibility (blocker)

Repos must be **public** before filing or claiming open source. Status checked 16 Nov 2026 (IST) with the GitHub API.

| Repo | Public? |
|------|---------|
| warbleoss.org | ✅ public |
| shrikeops-demo | ✅ public |
| shrikeops | ✅ public |
| heron-ops | ✅ public |
| warble-army | ✅ public |
| community | ✅ public (GitHub does not detect a licence yet) |

## After TOC acceptance (do not start early)

- [ ] Transfer trademarks / domains / GitHub orgs per CNCF  
- [ ] Landscape listing  
- [ ] LFX Insights  
- [ ] Onboarding template from cncf/sandbox  
- [ ] Community meetings + maintainers from multiple orgs  
- [ ] Consider [LFX Crowdfunding](https://docs.linuxfoundation.org/lfx/crowdfunding/apply-for-crowdfunding) (needs 100% OSS, multi-contributor, ~100+ stars)  

## Owners

| Role | Name | Contact |
|------|------|---------|
| Application lead | *TBD* | contact@warblecloud.com |
| Technical lead | *TBD* | |
| Legal / CA signatory | *TBD* | Warblecloud, LLC |
