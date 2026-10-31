<!--
PR title must follow Conventional Commits, e.g.
  feat(starling-lite): add scan_helm tool
  fix(charts): drop privileged securityContext
  docs: clarify readonly mode
  chore(deps): bump heron-ops submodule
Every commit must be signed off (DCO): git commit -s
-->

## What & why

<!-- Short summary. Link the issue: "Fixes #123" or "Refs #123". -->

## Checklist

- [ ] Title follows [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, `chore:`, `ci:` …)
- [ ] All commits are signed off (`git commit -s`, see [DCO](../docs/cncf/CONTRIBUTING.md#developer-certificate-of-origin-dco))
- [ ] `make registry && make starling-lite && make test-mcp` pass locally
- [ ] Docs / CHANGELOG updated if user-visible

### Submodule bump (delete if not applicable)

- [ ] Gitlink points to a commit that is **merged on the submodule's default branch** (not a fork / unmerged branch)
- [ ] `./scripts/submodules.sh status` is clean
- [ ] Linked the upstream changes (compare URL or release notes) in this PR
- [ ] Only one submodule bumped per PR, unless the bumps depend on each other

### Safety (from [docs/safety.md](../docs/safety.md)) — delete if no runtime/MCP change

- [ ] MCP default stays **readonly**; new write tools are hidden unless `mode=readwrite`
- [ ] Remediations are **dry-run** first (`propose_*` never mutates; `apply_*` needs readwrite + allowlist + `confirm=true`)
- [ ] Write tools honour the **namespace allowlist**
- [ ] Tool calls remain **audited** (JSONL) in agent loops
- [ ] RBAC samples remain least-privilege
- [ ] No kubeconfigs, cloud tokens, `.env` or private keys committed
