# CI

| Workflow | Trigger | Runner | Secrets |
|----------|---------|--------|---------|
| `check.yml` | push to main/master, every PR, manual | `warble-ci` (self-hosted) for warbletech pushes + same-repo PRs; `ubuntu-latest` for fork PRs and forks | none required (`WARBLE_CI_PAT` opt-in via repo variable `CI_USE_PAT=true`) |
| `dco.yml` | every PR | same rule as `check.yml` | none |
| `scorecard.yml` | push to main, weekly, manual | `ubuntu-latest`, **only while the repo is public** | none |
| CodeQL | GitHub **default setup** (repo setting) | GitHub-hosted | none |
| `release-please.yml` | manual | `ubuntu-latest` | none |

## Forks

1. Fork, enable Actions on your fork.
2. `make starling-lite` locally (Go 1.25+, Python 3, GNU make).
3. Push / open a PR — `check.yml` runs on GitHub-hosted runners in your
   fork (free for public repos) and fork PRs to warbletech also run on hosted
   runners. All submodules are public, so no secrets are required.

The existing `WARBLE_CI_PAT` secret is no longer used by default (it caused
checkout to fail with `could not read Username`); set the repository variable
`CI_USE_PAT=true` only if a submodule becomes private and the PAT is valid.

Untrusted fork code is **never** scheduled on the self-hosted `warble-ci`
runner: the runner expression checks
`github.event.pull_request.head.repo.full_name == github.repository`.
Do not add `pull_request_target` workflows that check out PR code.

## Cost notes

- warble-army is public: GitHub-hosted minutes, code scanning SARIF upload
  (Scorecard) and CodeQL default setup are free. If the repo ever goes
  private, `scorecard.yml` skips itself; CodeQL would need GitHub Advanced
  Security (paid) and should be disabled.
- Scorecard `publish_results` requires GitHub-hosted runners, so it stays on
  `ubuntu-latest`.
- CodeQL uses *default setup* (already active). Do **not** add an advanced
  `codeql.yml` while default setup is on — the two conflict.
