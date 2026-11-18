# Dependency licence report

This report backs the "external dependencies" and "licence exception" answers in the sandbox application.
It was generated on **16 Nov 2026 (IST)** with [`scripts/licence-report.sh`](../../scripts/licence-report.sh)
at warble-army `8947320`, with submodules heron-ops `7749c00` and shrikeops `b273b2a`. Regenerate it before
filing.

**Bottom line:** all compiled-in Go dependencies use permissive licences (Apache-2.0, MIT, BSD-3-Clause, ISC).
The open items are the ShrikeOps web frontend's npm packages (see [Items to review](#items-to-review)) and the
licence of the out-of-scope `starling` release channel.

## Scope

| Component | Repo | Own licence | Included here |
|-----------|------|-------------|---------------|
| Starling Lite (MCP facade, includes Heron Ops + ShrikeOps scanner) | warble-army `mcp/starling-lite` | Apache-2.0 | Go modules |
| Heron Ops | heron-ops | Apache-2.0 | Go modules |
| ShrikeOps scanner | shrikeops `scanner/` | Apache-2.0 | Go modules |
| ShrikeOps web frontend | shrikeops `frontend/` | Apache-2.0 | npm packages (from `package-lock.json`) |
| Reflexion agent scaffold | warble-army `agents/reflexion` | Apache-2.0 | Python standard library only, no third-party packages |
| Scanners run as external binaries | Pluto, Polaris, kube-score | see below | Upstream licence only (not linked) |
| `starling` (CLI release channel) | starling | **none detected** | **Excluded from the sandbox scope** |

First-party packages (`github.com/warbletech/*`) are left out of the CSVs. They are covered by each repo's
root Apache-2.0 `LICENSE`. `go-licenses` reports them as "Unknown" only because the nested Go modules have no
LICENSE file of their own.

## Go modules (compiled in)

| Module | Third-party packages | Licences |
|--------|---------------------:|----------|
| Starling Lite | 67 | Apache-2.0 31, MIT 17, BSD-3-Clause 18, ISC 1 |
| Heron Ops | 47 | Apache-2.0 21, MIT 6, BSD-3-Clause 19, ISC 1 |
| ShrikeOps scanner | 16 | MIT 11, BSD-3-Clause 5 |

Full lists: [`licenses/go-starling-lite.csv`](licenses/go-starling-lite.csv),
[`licenses/go-heron-ops.csv`](licenses/go-heron-ops.csv),
[`licenses/go-shrikeops-scanner.csv`](licenses/go-shrikeops-scanner.csv) (package, licence URL, SPDX id).

No copyleft licences. The only ISC package is `github.com/davecgh/go-spew` (via client-go).

## External tools (run as separate processes, not linked)

| Tool | Used by | Licence (GitHub API, 16 Nov 2026) |
|------|---------|-----------------------------------|
| [Pluto](https://github.com/FairwindsOps/pluto) 5.24.4 | ShrikeOps `scan_*` | Apache-2.0 |
| [Polaris](https://github.com/FairwindsOps/polaris) 10.2.5 | ShrikeOps `scan_*` | Apache-2.0 |
| [kube-score](https://github.com/zegl/kube-score) 1.20.0 | ShrikeOps `scan_*` | MIT |
| [Helm](https://github.com/helm/helm) | `scan_helm` (`helm template`) | Apache-2.0 |
| [agentgateway](https://github.com/agentgateway/agentgateway) | optional MCP gateway (`deploy/agentgateway`) | Apache-2.0 |
| [OPA](https://github.com/open-policy-agent/opa) | Rego policy bundles in `shrikeops/policies` | Apache-2.0 |

## npm packages: ShrikeOps frontend

From `shrikeops/frontend/package-lock.json` (declared licences). Full list:
[`licenses/npm-shrikeops-frontend.csv`](licenses/npm-shrikeops-frontend.csv).

| Scope | Packages | Licences |
|-------|---------:|----------|
| prod | 20 | MIT 12, ISC 3, Apache-2.0 2, BSD-3-Clause 1, 0BSD 1, **CC-BY-4.0 1** |
| optional (platform-specific) | 38 | Apache-2.0 14, MIT 10, **LGPL-3.0-or-later 10**, **Apache-2.0 AND LGPL-3.0-or-later 4** |
| dev (build only, not shipped) | 371 | mostly MIT / ISC / Apache-2.0; also BSD-2-Clause, BlueOak-1.0.0, CC0-1.0, Python-2.0, MPL-2.0 1 |

## Items to review

1. **`@img/sharp-libvips-*`, `@img/sharp-win32-*`, `@img/sharp-wasm32` (LGPL-3.0-or-later):** optional
   prebuilt binaries for `sharp`, which Next.js uses for image optimisation. They are dynamically loaded,
   not part of our source, but LGPL is not Apache-2.0-compatible for static redistribution. Check them
   against the foundation's allowlist and exception list. If they are not allowed, configure Next.js without
   `sharp` (`images.unoptimized`), or drop the frontend from the sandbox scope.
2. **`caniuse-lite` (CC-BY-4.0):** browser-support data pulled in by Next.js. Check it against the
   foundation's exception list before answering "no licence exception needed".
3. **`starling` repo has no licence:** keep it out of the application (already marked optional) unless a
   licence is added.
4. **Out of scope, flagged for awareness:** `warbletech/warble-linux` is **GPL-2.0** (archiso-based). It is
   not part of this application. If it is ever proposed, it cannot be described as Apache-2.0 and would need
   separate review.

Until items 1 and 2 are resolved, the application should not say "no licence exception needed" for the
ShrikeOps frontend.
