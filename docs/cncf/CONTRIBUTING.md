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
4. Sign off every commit (DCO, below) — `git commit -s`.  
5. Fill in the [PR template](../../.github/PULL_REQUEST_TEMPLATE.md) (submodule-bump and safety checklists).  
6. Forks get green CI without org secrets: `check.yml` runs on GitHub-hosted runners for forks and fork PRs (see [docs/CI.md](../CI.md)).

## Developer Certificate of Origin (DCO)

All contributions are made under the [Developer Certificate of Origin 1.1](https://developercertificate.org/).
By adding a `Signed-off-by` trailer you certify that you wrote the change or
otherwise have the right to submit it under the project's license
(Apache-2.0 unless noted).

```bash
git commit -s -m "fix(charts): drop privileged securityContext"
# trailer added: Signed-off-by: Your Name <you@example.com>
```

The name/email must match the commit author. The `dco` workflow checks every
commit in a PR. Forgot? Fix your branch with
`git rebase --signoff origin/main && git push --force-with-lease`.

## Labels & releases

- Label taxonomy: [docs/LABELS.md](../LABELS.md)
- Release process / changelog: [docs/RELEASING.md](../RELEASING.md), [CHANGELOG.md](../../CHANGELOG.md)

## Security

Report vulnerabilities privately via [SECURITY.md](SECURITY.md) — do not open public issues for exploitable bugs.

## Maintainers

See [MAINTAINERS.md](MAINTAINERS.md).

## CNCF

This flock is **planning a CNCF Sandbox donation**. See [SANDBOX_APPLICATION.md](SANDBOX_APPLICATION.md). Until accepted, Warble/ChirpStack stewards the project; do not claim “CNCF project” status.
