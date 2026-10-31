# Releasing

warble-army uses [release-please](https://github.com/googleapis/release-please)
driven by Conventional Commit PR titles (enforced by the PR template / review).

| Component | Path | Tag | Changelog |
|-----------|------|-----|-----------|
| warble-army (umbrella) | `.` | `warble-army-vX.Y.Z` | [CHANGELOG.md](../CHANGELOG.md) |
| Starling Lite | `mcp/starling-lite` | `starling-lite-vX.Y.Z` | [mcp/starling-lite/CHANGELOG.md](../mcp/starling-lite/CHANGELOG.md) |

Config: [`release-please-config.json`](../release-please-config.json),
[`.release-please-manifest.json`](../.release-please-manifest.json).

## Cutting a release

1. Merge PRs with Conventional Commit titles (`feat:` → minor, `fix:` → patch,
   `feat!:` / `BREAKING CHANGE:` → major; `0.x` stays minor).
2. Run **Actions → release-please → Run workflow** on `main`. It opens/updates a
   release PR with the version bump and changelog.
3. A maintainer reviews and merges the release PR; re-run the workflow to tag
   and create the GitHub Release.

The workflow is manual (`workflow_dispatch`) until maintainers decide to run it
on every push to `main`. It needs the repo setting *Allow GitHub Actions to
create and approve pull requests*. Binary artifacts (goreleaser) are a possible
follow-up for `starling-lite`; today users build with `make starling-lite`.
