# Architecture — Warble Army

## Flock

```text
┌─────────────────────────────────────────────────────────────────┐
│                     warble-army (umbrella)                      │
│  docs · mcp/tools · scripts · .githooks · .gitmodules           │
└────────────┬──────────────┬──────────────┬──────────────────────┘
             │ submodule    │ submodule    │ submodule
             ▼              ▼              ▼
        shrikeops      steadyhelm       starling
        (scanner)      (OSS MCP)      (release only)
             │              │              │
             └──────┬───────┘              │
                    ▼                      │
            mcp/starling-lite              │
         (stable tool registry)            │
                    │                      │
         agents / IDE MCP clients          │
                    │                      │
                    └────── optional ──────┘
                         SaaS Starling
                      (warble-core private)
```

## Data plane

1. **Static path:** YAML/Helm → ShrikeOps → score + findings + remediation hints.
2. **Live path:** kubeconfig / in-cluster → SteadyHelm MCP → structured cluster text for LLMs.
3. **Agent path:** LLM issues `tools/call` → Starling Lite registry routes to scan or cluster backends → optional dry-run patch.

## Control plane (future agents)

```text
Incident / PR / chat
        → Planner
        → Scout   (SteadyHelm read tools)
        → Auditor (ShrikeOps)
        → Healer  (propose_remediation)
        → Critic  (Reflexion-style guardrails)
        → Apply   (only readwrite + allowlist)
```

## OSS vs commercial

| Capability | OSS (this flock) | Commercial |
|------------|------------------|------------|
| MCP cluster read/write | SteadyHelm | Multi-cluster mesh, SSO |
| Manifest scan | ShrikeOps CLI/server | Hosted API, history, GitHub App |
| Tool schemas | `mcp/tools/*` | Same schemas + proprietary tools |
| Auth | kubeconfig / SA | `starling login`, org billing |
| Remediation apply | Local dry-run / opt-in write | Fleet policy marketplace |

## Module identity note

- SteadyHelm Go module path: `github.com/warblecloud/steadyhelm` (historical).
- Git remotes use `github.com/warble-tech/*`.
- Prefer **warble-tech** for new public repos; document renames in MAINTAINING when unified.
