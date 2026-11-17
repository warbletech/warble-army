# Rename: Warble Army → Warble Flock

**Status:** proposed. *Warble Flock* is on the shortlist in
[warble-marketing#70](https://github.com/warbletech/warble-marketing/issues/70)
(with SkeinOps and Warble Chorus). It is **not final** until the Linux
Foundation / AAIF teams and the maintainers sign off. If a different name is
picked, repeat the same steps with that name.

This page lists what has already been renamed in this repository, what was left
on purpose, and the exact steps an org admin needs to finish the rename.

## Done in this repository (text only)

User-facing prose now says **Warble Flock**: README title and naming note,
BRAND, GOVERNANCE, MAINTAINING, architecture, `docs/cncf/*` (CONTRIBUTING,
CODE_OF_CONDUCT, MAINTAINERS), the warbleoss.org landing draft, MCP docs, the
Agent Gateway README, the `project-sandbox` chart description, the bug report
form, the CHANGELOG header, `make help`, and the tool registry description.

Nothing that a build, deploy, install or link depends on was changed.

## Left as `warble-army` on purpose

These would break before the GitHub repository and packages are renamed, or
would change the behaviour of running installs. Keep them until the admin steps
below are done.

| What | Where | Why it stays |
|------|-------|--------------|
| Repo URL / slug `warbletech/warble-army` | README badges and clone commands, issue forms, docs links, chart `sources`, `NOTES.txt`, `railway.toml` comments | Only valid after the repo rename (GitHub then redirects the old URL) |
| Clone directory name `warble-army/` | "from warble-army root", `/path/to/warble-army/...` in examples | Matches what `git clone` creates today |
| Go module `github.com/warbletech/warble-army/mcp/starling-lite` | `mcp/starling-lite/go.mod` and every import | Must match the repo path |
| Workflow guards `github.repository == 'warbletech/warble-army'` | `release-please.yml`, `scorecard.yml` | Jobs would silently stop running |
| CI artifact `warble-army-bin` | `check.yml` | Workflow file (also touched by #46) |
| release-please component `warble-army` / tag `warble-army-vX.Y.Z` | `release-please-config.json`, `docs/RELEASING.md` | Open release PR #45 pins `warble-army-v0.1.0` |
| Image and chart paths `ghcr.io/warbletech/warble-army/...` | docs, release workflow (#45) | Package names on GHCR |
| Registry path `/etc/warble-army/registry.json` | Dockerfiles, Helm chart, k8s manifests, Agent Gateway configs, Railway `STARLING_REGISTRY`, ops scripts | Railway and running pods read this path |
| Server name `warble-army-mcp` | `mcp/tools/registry.json` `name` | Clients may key on it |
| MCP client key `"warble-army"` | README, `mcp/config/starling-lite.example.json`, Agent Gateway configs | Changes the tool prefix users already have configured |
| K8s names and labels (`warble-army-mcp`, `warble-army-mcp-readonly`, `app.kubernetes.io/part-of: warble-army`, namespace `warble-army`) | `deploy/agentgateway/k8s/*`, `charts/starling-ex-ce`, `scripts/install.sh` | Renaming creates new objects next to the old ones on existing clusters |
| Old umbrella tag naming `army-v0.1.0` | `docs/MAINTAINING.md` | Describes existing tag history |
| `docs/RELEASING.md`, `docs/CI.md`, Railway / starling-lite Dockerfiles, `deploy/railway/README.md` | — | Being edited by open PRs #45 / #46; rename after they merge to avoid conflicts |

## Admin checklist (needs org owner / admin)

Do these only after the name is signed off. Steps 1–3 need an org admin.

1. **Rename the repo** `warbletech/warble-army` → `warbletech/warble-flock`
   (Settings → General). GitHub redirects old web, clone and API URLs, but do
   **not** create a new repo called `warble-army` afterwards, or the redirect
   breaks. Update the repo description and topics.
2. **GHCR packages.** Publish new packages under
   `ghcr.io/warbletech/warble-flock/...` (image `starling-lite`) and set them
   public (same blocker as warble-army#26 / warble-marketing#73). GHCR does not
   redirect old names: keep pushing (or leave) the old `warble-army` packages
   for one release with a deprecation note. Check the chart OCI path used by
   `publish-charts.yml` (`oci://ghcr.io/warbletech/charts`).
3. **Railway / mcp.warbleoss.org.** Point the Railway service at the renamed
   repo (it may need reconnecting), rename the service if wanted, and keep
   `STARLING_REGISTRY=/etc/warble-army/registry.json` until step 6 lands.
4. **Go module path.** In one PR change `go.mod` to
   `github.com/warbletech/warble-flock/mcp/starling-lite` and update every
   import; run `go build ./... && go test ./...`.
5. **Workflows.** Change the `github.repository == 'warbletech/warble-army'`
   guards in `release-please.yml` and `scorecard.yml`, and the
   `warble-army-bin` artifact in `check.yml`. Re-check the Scorecard badge URL.
6. **Runtime paths and names.** Move `/etc/warble-army/registry.json` →
   `/etc/warble-flock/registry.json` in both Dockerfiles, the chart, the k8s
   manifests, the Agent Gateway configs and the ops scripts, and update the
   Railway variable in the same deploy. Optionally rename `warble-army-mcp`,
   the RBAC names and the `part-of` label (delete the old objects on existing
   clusters), and the default install namespace in `scripts/install.sh`.
7. **Releases.** Existing tags stay as they are. To switch the tag prefix
   (`warble-army-v0.1.0` → `warble-flock-v0.1.0`), change the release-please
   component to `warble-flock` and the tags in `docs/RELEASING.md`. Coordinate
   with release PR #45: either cut `warble-army-v0.1.0` first and switch for
   the next release, or change the component in #45 before the first tag.
   Do not retag old releases.
8. **Links in this repo.** Replace remaining `warbletech/warble-army` URLs,
   clone commands, the MCP client key and `/path/to/warble-army` examples, and
   remove the naming note in the README.
9. **Other repos** (found with code search on 17 Nov 2026; private repos may
   have more):
   - `warbletech/warble-marketing`: `lfx-submission/warble-army-sandbox-plan.md`
     (PR #91: title, heading, repo URLs, application fields), milestone and
     issue titles that say Warble Army.
   - `warbletech/warbleoss.org`: `.gitmodules`, `index.html`, `README.md`,
     `CNCF.md`, `SECURITY.md`.
   - `warbletech/heron-ops`, `warbletech/shrikeops`, `warbletech/shrikeops-demo`,
     `warbletech/warble-harness`: README, CONTRIBUTING, CODE_OF_CONDUCT,
     SECURITY links.
   - `warbletech/community`: README, SECURITY.
   - `warbletech/reflexion-ai`: `docs/marketing/WARBLE_ARMY_MASCOTS.md`,
     `frontend/src/components/army/*` (WarbleArmyRoster).
   - `warbletech/warble-core`: README, `apps/frontend/src/data/showcase.ts`.
   - `warbletech/warble-oss`, `warbletech/warble-labs` (`agentgateway.yaml`),
     `warbletech/warble-docs`, `warbletech/warble-gcp-v3.0` docs.
10. **Public surfaces.** warbleoss.org, mcp.warbleoss.org docs, the CNCF / AAIF
    drafts in `docs/cncf/`, and any social or pitch material.

The separate *Starling EX* → new-name work in #70 is not covered here.
