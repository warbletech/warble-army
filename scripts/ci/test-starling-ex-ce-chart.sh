#!/usr/bin/env bash
# Render checks for charts/starling-ex-ce. Does not pull from a registry.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CHART="$ROOT/charts/starling-ex-ce"
IMAGE="ghcr.io/warbletech/warble-army/starling-lite"

command -v helm >/dev/null || { echo "helm required"; exit 1; }

if helm template t "$CHART" >/tmp/starling-ex-ce-notag.txt 2>&1; then
  echo "helm template without image.tag should fail"
  exit 1
fi
if ! grep -q "image.tag must be the full git commit SHA" /tmp/starling-ex-ce-notag.txt; then
  echo "missing-tag error did not name the GHCR tag rule"
  cat /tmp/starling-ex-ce-notag.txt
  exit 1
fi

out="$(helm template t "$CHART" --set image.tag=local)"
printf '%s\n' "$out" > /tmp/starling-ex-ce-local.txt

if ! grep -q "image: \"${IMAGE}:local\"" /tmp/starling-ex-ce-local.txt; then
  echo "rendered image is not ${IMAGE}:local"
  exit 1
fi
if grep -q "us-central1-docker.pkg.dev" /tmp/starling-ex-ce-local.txt; then
  echo "rendered chart still references Artifact Registry"
  exit 1
fi

echo "ok starling-ex-ce chart renders ${IMAGE}:<tag>"
