# Security policy — Starling EX Community

## Supported versions

| Component | Supported |
|-----------|-----------|
| Starling Lite / starling-ex-ce chart (latest main) | ✅ |
| Heron Ops latest release / main | ✅ |
| ShrikeOps latest release / main | ✅ |
| Older tags | Best effort only |

## Reporting a vulnerability

**Do not** open a public GitHub issue for security vulnerabilities.

Email: **security@warblecloud.com**  

Please include:

- Affected component and version/commit  
- Description and impact  
- Reproduction steps or PoC (if safe)  
- Whether you are available for follow-up  

We aim to acknowledge within **3 business days** and provide a status update within **10 business days**.

## Disclosure

We prefer coordinated disclosure. Fix timelines depend on severity. Credits for reporters are welcome unless anonymity is requested.

## Scope notes

- Heron Ops **readwrite** mode can mutate clusters — treat as high impact.  
- Public demo MCP (`starling-ex.frakma.io/lite/mcp`) is **readonly** and demo-grade; report abuse to the same alias.  
- Supply-chain / dependency issues in scanners (Pluto, Polaris, etc.) may need upstream coordination.

## Security-related configuration

- Prefer `--mode=readonly` for agents by default.  
- Use namespace allowlists and dry-run remediation.  
- Run under least-privilege ServiceAccounts (see Helm chart RBAC).
