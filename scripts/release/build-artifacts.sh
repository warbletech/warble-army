#!/usr/bin/env bash
# Build the release assets for one component into an output directory.
#
#   scripts/release/build-artifacts.sh starling-lite 0.4.0 dist
#   scripts/release/build-artifacts.sh warble-army   0.1.0 dist
#
# starling-lite: static binaries (CGO off) for linux, darwin and windows on
#   amd64 and arm64, one archive each, an SPDX SBOM per archive (if `syft`
#   is on PATH), and the starling-ex-ce Helm chart pointing at the release
#   image (if `helm` and `yq` are on PATH).
# warble-army: a source archive of the tagged tree plus its SBOM.
# Both: a SHA-256 checksums file that the release workflow signs.
#
# Run from the repo root with submodules checked out (heron-ops, shrikeops).
# Nothing is pushed or published. Used by .github/workflows/release.yml and
# for local dry runs (see docs/RELEASING.md).
set -euo pipefail

component=${1:?component: starling-lite | warble-army}
version=${2:?version without the leading v, e.g. 0.4.0}
out=${3:-dist}
version=${version#v}
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || {
  echo "not a semver version: $version" >&2; exit 2; }

root=$(git rev-parse --show-toplevel)
cd "$root"
revision=$(git rev-parse HEAD)
# Reproducible archive timestamps: the commit time, not the build time.
export SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH:-$(git log -1 --format=%ct)}
image=${IMAGE:-ghcr.io/warbletech/warble-army/starling-lite}

rm -rf "$out"
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

tar_reproducible() { # <archive.tar.gz> <dir> <entries...>
  local archive=$1 dir=$2; shift 2
  tar --sort=name --owner=0 --group=0 --numeric-owner \
      --mtime="@${SOURCE_DATE_EPOCH}" -C "$dir" -cf - "$@" | gzip -n -9 > "$archive"
}

zip_reproducible() { # <archive.zip> <dir> <entry>
  python3 - "$@" <<'PY'
import os, sys, time, zipfile
archive, base, entry = sys.argv[1:4]
stamp = time.gmtime(max(int(os.environ["SOURCE_DATE_EPOCH"]), 315532800))[:6]
with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as zf:
    for dirpath, dirs, files in os.walk(os.path.join(base, entry)):
        dirs.sort()
        for f in sorted(files):
            full = os.path.join(dirpath, f)
            info = zipfile.ZipInfo(os.path.relpath(full, base), stamp)
            info.external_attr = 0o755 << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            with open(full, "rb") as fh:
                zf.writestr(info, fh.read())
PY
}

sbom() { # <path> <output.spdx.json>
  if command -v syft >/dev/null 2>&1; then
    syft scan --quiet "$1" -o "spdx-json=$2"
  else
    echo "syft not found; skipping SBOM for $1" >&2
  fi
}

case "$component" in
starling-lite)
  for f in heron-ops/go.mod shrikeops/scanner/go.mod; do
    [ -f "$f" ] || { echo "missing $f: run git submodule update --init heron-ops shrikeops" >&2; exit 1; }
  done
  ldflags="-s -w -buildid= -X main.version=${version} -X main.revision=${revision}"
  for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
    goos=${target%/*}; goarch=${target#*/}
    name="starling-lite_${version}_${goos}_${goarch}"
    ext=""; [ "$goos" = windows ] && ext=".exe"
    mkdir -p "$stage/$name"
    (cd mcp/starling-lite && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch \
      go build -trimpath -buildvcs=false -ldflags "$ldflags" \
        -o "$stage/$name/starling-lite${ext}" ./cmd/starling-lite)
    cp LICENSE mcp/starling-lite/README.md mcp/tools/registry.json "$stage/$name/"
    if [ "$goos" = windows ]; then
      zip_reproducible "$out/$name.zip" "$stage" "$name"
      sbom "$out/$name.zip" "$out/$name.zip.spdx.json"
    else
      tar_reproducible "$out/$name.tar.gz" "$stage" "$name"
      sbom "$out/$name.tar.gz" "$out/$name.tar.gz.spdx.json"
    fi
    echo "built $name"
  done

  chart=charts/starling-ex-ce
  if command -v helm >/dev/null 2>&1 && command -v yq >/dev/null 2>&1; then
    cp -r "$chart" "$stage/chart"
    cp LICENSE "$stage/chart/LICENSE"
    IMAGE="$image" VERSION="$version" \
      yq -i '.image.repository = strenv(IMAGE) | .image.tag = strenv(VERSION)' "$stage/chart/values.yaml"
    helm lint "$stage/chart" >/dev/null
    helm package "$stage/chart" --app-version "$version" --destination "$out" >/dev/null
    echo "packaged $(basename "$out"/starling-ex-ce-*.tgz) (image ${image}:${version})"
  else
    echo "helm or yq not found; skipping the chart" >&2
  fi
  ;;
warble-army)
  # git archive leaves submodules out; they are separate repos with their own releases.
  name="warble-army_${version}_source"
  git archive --format=tar --prefix="${name}/" HEAD | gzip -n -9 > "$out/${name}.tar.gz"
  tar -xzf "$out/${name}.tar.gz" -C "$stage"
  sbom "dir:$stage/${name}" "$out/${name}.tar.gz.spdx.json"
  echo "built ${name}.tar.gz"
  ;;
*)
  echo "unknown component: $component" >&2; exit 2 ;;
esac

checksums="${component}_${version}_checksums.txt"
(cd "$out" && find . -maxdepth 1 -type f -printf '%f\n' | LC_ALL=C sort \
  | xargs sha256sum > "$stage/checksums.txt")
mv "$stage/checksums.txt" "$out/$checksums"
echo "wrote $checksums"
