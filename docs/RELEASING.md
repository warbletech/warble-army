# Releasing

warble-army uses [release-please](https://github.com/googleapis/release-please)
driven by Conventional Commit PR titles (enforced by the PR template / review),
and [`release.yml`](../.github/workflows/release.yml) to build, sign and
upload the release assets.

| Component | Path | Tag | Changelog | Assets |
|-----------|------|-----|-----------|--------|
| warble-army (umbrella) | `.` | `warble-army-vX.Y.Z` | [CHANGELOG.md](../CHANGELOG.md) | source archive, SBOM |
| Starling Lite | `mcp/starling-lite` | `starling-lite-vX.Y.Z` | [mcp/starling-lite/CHANGELOG.md](../mcp/starling-lite/CHANGELOG.md) | binaries, SBOMs, chart, image |

Config: [`release-please-config.json`](../release-please-config.json),
[`.release-please-manifest.json`](../.release-please-manifest.json).

## What a release contains

Every release has `<component>_<version>_checksums.txt` (SHA-256 of every
other asset), a keyless [cosign](https://docs.sigstore.dev/) bundle for it
(`…_checksums.txt.sigstore.json`) and SLSA build provenance
(`<component>_<version>.intoto.jsonl`, plus the full Sigstore bundle in
`….provenance.sigstore.json`). The provenance is also stored as a GitHub
artifact attestation. Signing uses the workflow's GitHub OIDC identity: no
keys or secrets to manage.

Starling Lite (`starling-lite-vX.Y.Z`) also has:

- `starling-lite_X.Y.Z_<os>_<arch>.tar.gz` (`.zip` on Windows) for linux,
  darwin and windows on amd64 and arm64: a static binary (`--version` prints
  `X.Y.Z` and the commit), `LICENSE`, `README.md` and `registry.json`.
  Each archive has an SPDX SBOM next to it (`….spdx.json`).
- `starling-ex-ce-<chart version>.tgz`: the Helm chart with `appVersion` and
  `image.tag` set to `X.Y.Z` and `image.repository` set to the GHCR image.
- Image `ghcr.io/warbletech/warble-army/starling-lite:X.Y.Z` for
  linux/amd64 and linux/arm64, signed with cosign, with an SPDX SBOM
  attestation and GitHub build provenance.

The chart is attached to the release only. Pushing charts to
`oci://ghcr.io/warbletech/charts` stays with `publish-charts.yml`.

### First release

`release-as` in `release-please-config.json` pins the first versions:

- `warble-army-v0.1.0` (manifest is `0.0.0`, nothing tagged before).
- `starling-lite-v0.4.0`. The binary already reports `0.3.0`
  (`cmd/starling-lite/main.go`, deployed at mcp.warbleoss.org) and the chart
  `appVersion` is `0.2.0`, so `0.1.0` would make the version go backwards.
  To start at `0.1.0` anyway, change `release-as` and the manifest entry before
  running release-please.

**Remove both `release-as` lines after the first release** (normal PR),
otherwise every later release PR proposes the same version again.

## Cutting a release (normal path)

1. Merge PRs with Conventional Commit titles (`feat:` → minor, `fix:` → patch,
   `feat!:` / `BREAKING CHANGE:` → major; `0.x` stays minor).
2. **Actions → release-please → Run workflow** on `main`. It opens or updates
   one release PR with the version bumps and changelogs.
3. A maintainer reviews and merges the release PR.
4. **Run release-please again** on `main`. It creates the tags and GitHub
   Releases, then calls `release.yml` once per released component. The assets
   appear on the release about 15–30 minutes later (the arm64 image is built
   under QEMU). The release page is visible before its assets are uploaded.

The workflow is manual (`workflow_dispatch`) until maintainers decide to run it
on every push to `main`. It needs the repo setting *Allow GitHub Actions to
create and approve pull requests* (Settings → Actions → General).

## Fallback: push a tag by hand

If release-please is not used (or step 4 failed after tagging):

```sh
git fetch origin && git checkout origin/main   # the commit to release
git tag -s starling-lite-v0.4.0 -m "starling-lite v0.4.0"   # -a if you do not sign tags
git push origin starling-lite-v0.4.0
```

A pushed `starling-lite-v*` or `warble-army-v*` tag starts `release.yml`. If no
GitHub Release exists for the tag it creates one with generated notes, then
uploads the assets. To rebuild and re-upload for an existing tag, use
**Actions → release → Run workflow** with the tag (assets are replaced).

Update the version in `cmd/starling-lite/main.go` and the manifest in the same
PR if you bypass release-please, so the next release PR starts from the right
version.

## Dry runs

`release.yml` also runs on pull requests that touch the release files: it builds
all binaries, SBOMs, the chart and both image architectures with version
`0.0.0-dryrun`. Nothing is pushed, signed or uploaded.

Locally (Go 1.26+, submodules `heron-ops` and `shrikeops` checked out; `syft`,
`helm` and `yq` are optional and skipped if missing):

```sh
scripts/release/build-artifacts.sh starling-lite 0.4.0 dist
scripts/release/build-artifacts.sh warble-army 0.1.0 dist-army
```

## Verifying a release

Download the assets you need plus the checksums file and its bundle, e.g. with
`gh release download starling-lite-v0.4.0 -R warbletech/warble-army`.

```sh
V=0.4.0
ID='^https://github\.com/warbletech/warble-army/\.github/workflows/release\.yml@refs/(heads/main|tags/)'
ISSUER=https://token.actions.githubusercontent.com

# 1. The checksums file was signed by this repo's release workflow.
cosign verify-blob \
  --bundle starling-lite_${V}_checksums.txt.sigstore.json \
  --certificate-identity-regexp "$ID" \
  --certificate-oidc-issuer "$ISSUER" \
  starling-lite_${V}_checksums.txt

# 2. The archives match the signed checksums.
sha256sum --ignore-missing -c starling-lite_${V}_checksums.txt

# 3. Build provenance (GitHub artifact attestation).
gh attestation verify starling-lite_${V}_linux_amd64.tar.gz -R warbletech/warble-army

# 4. Image signature, SBOM attestation and provenance.
IMG=ghcr.io/warbletech/warble-army/starling-lite:${V}
cosign verify --certificate-identity-regexp "$ID" --certificate-oidc-issuer "$ISSUER" "$IMG"
cosign verify-attestation --type spdxjson \
  --certificate-identity-regexp "$ID" --certificate-oidc-issuer "$ISSUER" "$IMG" >/dev/null
gh attestation verify "oci://$IMG" -R warbletech/warble-army
```

The certificate identity is always `release.yml` (the signing workflow, also
when release-please calls it): `@refs/heads/main` when release-please or a
manual run on `main` built it, `@refs/tags/<tag>` for a pushed tag.

## Needs an org owner / repo admin

These are settings, not code, and are not changed by any workflow here:

1. **Package visibility** ([#26](https://github.com/warbletech/warble-army/issues/26)):
   the first push creates `ghcr.io/warbletech/warble-army/starling-lite` as a
   **private** package. Make it public (org → Packages → starling-lite →
   Package settings → Change visibility) and link it to this repo, otherwise
   outsiders get 401 on `docker pull` / `helm install`. Same for the chart
   packages under `charts/`.
2. **Actions settings**: *Allow GitHub Actions to create and approve pull
   requests* for release-please. Org Actions policy must allow the actions used
   in `release.yml` (all pinned by SHA).
3. **Tag protection / rulesets**: optionally restrict who can create
   `starling-lite-v*` and `warble-army-v*` tags (Settings → Rules → Rulesets →
   tag ruleset), and protect `main` (Scorecard Branch-Protection).
4. **Do not enable immutable releases** yet. release-please publishes the
   release first and `release.yml` uploads the assets a few minutes later;
   an immutable release would reject that upload.

## Scorecard

Signed `*.sigstore.json` and `*.intoto.jsonl` assets satisfy OpenSSF Scorecard
**Signed-Releases** (it checks the last 5 releases), and publishing the image
from a workflow satisfies **Packaging**. Both stay "not applicable" until the
first release exists.
