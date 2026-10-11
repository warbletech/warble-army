# Contributing to Starling EX Community

Thanks for contributing to the **Starling EX Community** flock (Warble Army umbrella, Heron Ops, ShrikeOps, Starling Lite).

## Code of Conduct

All participation is governed by [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## Where to contribute

| Area | Repository |
|------|------------|
| Umbrella, MCP registry, Helm CE, docs | [warbletech/warble-army](https://github.com/warbletech/warble-army) |
| Heron Ops (cluster MCP) | [warbletech/heron-ops](https://github.com/warbletech/heron-ops) |
| ShrikeOps scanner | [warbletech/shrikeops](https://github.com/warbletech/shrikeops) |
| Scanner demos | [warbletech/shrikeops-demo](https://github.com/warbletech/shrikeops-demo) |

## Development setup (army)

```bash
git clone --recurse-submodules https://github.com/warbletech/warble-army.git
cd warble-army
./scripts/install-hooks.sh
make heron-ops
make starling-lite
make registry
```

## Pull requests

1. Open an issue for large changes (design first).  
2. Use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`).  
3. Keep PRs focused; include tests where applicable.  
4. Sign commits if required by the target repo.  
5. DCO: by contributing you agree your work is licensed under the project’s stated license (Apache-2.0 unless noted).

## Security

Report vulnerabilities privately via [SECURITY.md](SECURITY.md) — do not open public issues for exploitable bugs.

## Maintainers

See [MAINTAINERS.md](MAINTAINERS.md).

## CNCF

This flock is **planning a CNCF Sandbox donation**. See [SANDBOX_APPLICATION.md](SANDBOX_APPLICATION.md). Until accepted, Warble/ChirpStack stewards the project; do not claim “CNCF project” status.
