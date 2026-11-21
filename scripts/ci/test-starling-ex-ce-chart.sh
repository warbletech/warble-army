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

if helm template t "$CHART" --set image.tag=local --set ingress.enabled=true >/tmp/starling-ex-ce-noauth.txt 2>&1; then
  echo "ingress without ingress.auth.url should fail"
  exit 1
fi
if ! grep -q "ingress.auth.url is required" /tmp/starling-ex-ce-noauth.txt; then
  echo "missing-auth error did not name ingress.auth.url"
  cat /tmp/starling-ex-ce-noauth.txt
  exit 1
fi

if helm template t "$CHART" --set image.tag=local --set ingress.enabled=true --set ingress.auth.enabled=false --set ingress.auth.url=http://oauth2-proxy.ns.svc/oauth2/auth >/tmp/starling-ex-ce-authoff.txt 2>&1; then
  echo "ingress.auth.enabled=false should fail"
  exit 1
fi
if ! grep -q "ingress.auth.enabled must stay true" /tmp/starling-ex-ce-authoff.txt; then
  echo "auth-off error did not name ingress.auth.enabled"
  cat /tmp/starling-ex-ce-authoff.txt
  exit 1
fi

auth="$(helm template t "$CHART" --set image.tag=local --set ingress.enabled=true --set ingress.auth.url=http://oauth2-proxy.ns.svc/oauth2/auth)"
printf '%s\n' "$auth" > /tmp/starling-ex-ce-auth.txt
if ! grep -q 'nginx.ingress.kubernetes.io/auth-url: "http://oauth2-proxy.ns.svc/oauth2/auth"' /tmp/starling-ex-ce-auth.txt; then
  echo "rendered ingress is missing auth-url"
  exit 1
fi

echo "ok starling-ex-ce chart renders ${IMAGE}:<tag>"
