#!/usr/bin/env bash
# Sync WARBLE_CI_PAT from GCP Secret Manager → GitHub Actions secrets.
#
# Source of truth:
#   https://console.cloud.google.com/security/secret-manager/secret/WARBLE_CI_PAT/versions?project=warble-tech
#   gcloud: projects/warble-tech/secrets/WARBLE_CI_PAT
#
# Usage:
#   ./scripts/sync-warble-ci-pat-from-gcp.sh
#   ./scripts/sync-warble-ci-pat-from-gcp.sh warble-army warble-linux
#
# Token requirements (GitHub fine-grained or classic):
#   - Access to warbletech private repos used as submodules
#   - Contents: Read (and Write recommended — fine-grained Read-only often fails git clone with 403)
#   - Metadata: Read
#
set -euo pipefail

PROJECT="${GCP_PROJECT:-warble-tech}"
SECRET_NAME="${GCP_SECRET_NAME:-WARBLE_CI_PAT}"
if [[ $# -gt 0 ]]; then
  REPOS=("$@")
else
  REPOS=(warble-army warble-linux shrikeops heron-ops shrikeops-demo)
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

echo "==> reading projects/${PROJECT}/secrets/${SECRET_NAME}"
gcloud secrets versions access latest --secret="$SECRET_NAME" --project="$PROJECT" \
  | tr -d '\r\n' >"$TMP"

[[ -s "$TMP" ]] || { echo "error: empty secret $SECRET_NAME" >&2; exit 1; }

TOKEN="$(cat "$TMP")"
# Basic validation (never print token)
case "$TOKEN" in
  github_pat_*|ghp_*|gho_*) ;;
  *) echo "warn: token does not look like a GitHub PAT (prefix unexpected)" >&2 ;;
esac

echo "==> testing git access to private repos (Contents permission)"
fail=0
for repo in warble-army heron-ops shrikeops; do
  if git -c "http.extraHeader=Authorization: Basic $(printf 'x-access-token:%s' "$TOKEN" | base64 -w0)" \
      ls-remote "https://github.com/warbletech/${repo}.git" HEAD >/dev/null 2>&1; then
    echo "  OK  git  warbletech/$repo"
  else
    echo "  FAIL git  warbletech/$repo  (403 Write access not granted? fix PAT Contents permission)"
    fail=1
  fi
done

if [[ "$fail" -ne 0 ]]; then
  cat >&2 <<'EOF'

PAT from GCP cannot git-clone private repos. Actions checkout with submodules will fail.

Fix in GitHub → Settings → Developer settings → Fine-grained tokens:
  - Resource owner: warbletech
  - Repository access: All warbletech private repos (or select army + heron-ops + shrikeops + shrikeops-demo + starling)
  - Permissions → Repository → Contents: Read and write
  - Permissions → Repository → Metadata: Read

Then add a new version of the secret in GCP Secret Manager (WARBLE_CI_PAT) and re-run this script.
EOF
  exit 2
fi

for r in "${REPOS[@]}"; do
  gh secret set WARBLE_CI_PAT -R "warbletech/$r" <"$TMP"
  echo "  updated GitHub Actions secret warbletech/$r WARBLE_CI_PAT"
done

echo "==> done (source of truth remains GCP Secret Manager)"
