#!/usr/bin/env bash
# Sync WARBLE_CI_PAT from GCP Secret Manager → GitHub Actions secrets.
# Secret: projects/warble-tech/secrets/WARBLE_CI_PAT
# Usage: ./scripts/sync-warble-ci-pat-from-gcp.sh [repo...]
set -euo pipefail
PROJECT="${GCP_PROJECT:-warble-tech}"
SECRET_NAME="${GCP_SECRET_NAME:-WARBLE_CI_PAT}"
REPOS=("${@:-warble-army warble-linux shrikeops heron-ops}")
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
gcloud secrets versions access latest --secret="$SECRET_NAME" --project="$PROJECT" \
  | tr -d '\r\n' >"$TMP"
[[ -s "$TMP" ]] || { echo "empty secret $SECRET_NAME" >&2; exit 1; }
for r in "${REPOS[@]}"; do
  gh secret set WARBLE_CI_PAT -R "warble-tech/$r" <"$TMP"
  echo "updated warble-tech/$r WARBLE_CI_PAT"
done
