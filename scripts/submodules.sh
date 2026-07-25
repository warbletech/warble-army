#!/usr/bin/env bash
# Maintain warble-army git submodules: init, update, status, bump, remote-add, add.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

die() { echo "error: $*" >&2; exit 1; }

need_git() {
  git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not a git repo: $ROOT"
}

cmd="${1:-}"
shift || true

usage() {
  cat <<'EOF'
Usage: scripts/submodules.sh <command>

  init         git submodule update --init --recursive
  update       fetch + checkout recorded gitlinks (sync)
  status       dirty / ahead-behind / SHA report
  bump         checkout each submodule tracked branch tip (working tree)
  remote-add   ensure origin exists on each submodule from .gitmodules
  add <url> <path> [branch]
               git submodule add (optional -b branch)

Examples:
  ./scripts/submodules.sh init
  ./scripts/submodules.sh bump && git add shrikeops heron-ops && git commit -m "chore(submodules): bump"
  ./scripts/submodules.sh add https://github.com/warble-tech/foo.git foo master
EOF
}

list_paths() {
  git config --file .gitmodules --get-regexp path | awk '{print $2}'
}

url_for() {
  local path="$1"
  git config --file .gitmodules --get "submodule.${path}.url" 2>/dev/null \
    || git config --file .gitmodules --get-regexp "submodule\..*\.path" \
         | while read -r key p; do
             [[ "$p" == "$path" ]] || continue
             local name="${key#submodule.}"
             name="${name%.path}"
             git config --file .gitmodules --get "submodule.${name}.url"
             break
           done
}

branch_for() {
  local path="$1"
  # Prefer submodule.<name>.branch keyed by name matching path
  local name
  name="$(git config --file .gitmodules --get-regexp 'submodule\..*\.path' \
    | awk -v p="$path" '$2==p { sub(/^submodule\./,"",$1); sub(/\.path$/,"",$1); print $1; exit }')"
  if [[ -n "${name:-}" ]]; then
    git config --file .gitmodules --get "submodule.${name}.branch" 2>/dev/null || echo ""
  else
    echo ""
  fi
}

cmd_init() {
  need_git
  git submodule sync --recursive
  git submodule update --init --recursive
  echo "submodules initialized."
}

cmd_update() {
  need_git
  git submodule sync --recursive
  git submodule update --init --recursive --checkout
  echo "submodules at recorded gitlinks."
}

cmd_status() {
  need_git
  echo "=== umbrella ==="
  git status -sb
  echo
  echo "=== submodules ==="
  git submodule status
  echo
  while IFS= read -r path; do
    [[ -d "$path/.git" || -f "$path/.git" ]] || { echo "$path  (not checked out)"; continue; }
    branch="$(branch_for "$path")"
    branch="${branch:-HEAD}"
    echo "--- $path (track: ${branch}) ---"
    git -C "$path" status -sb
    if git -C "$path" rev-parse --abbrev-ref HEAD >/dev/null 2>&1; then
      cur="$(git -C "$path" rev-parse --abbrev-ref HEAD)"
      if [[ "$cur" != "HEAD" ]] && git -C "$path" rev-parse --verify "@{u}" >/dev/null 2>&1; then
        git -C "$path" rev-list --left-right --count HEAD...@{u} \
          | awk -v p="$path" '{printf "  %s ahead=%s behind=%s (vs upstream)\n", p, $1, $2}'
      fi
    fi
    echo "  HEAD $(git -C "$path" rev-parse --short HEAD) $(git -C "$path" log -1 --oneline)"
  done < <(list_paths)
}

cmd_bump() {
  need_git
  while IFS= read -r path; do
    url="$(url_for "$path")"
    branch="$(branch_for "$path")"
    branch="${branch:-master}"
    echo "bump $path → origin/$branch"
    git -C "$path" fetch origin --tags
    if git -C "$path" rev-parse --verify "origin/${branch}" >/dev/null 2>&1; then
      git -C "$path" checkout -B "$branch" "origin/${branch}"
    else
      echo "  warn: origin/${branch} missing for $path (url=$url)" >&2
    fi
  done < <(list_paths)
  echo
  echo "Working trees updated. Review, then:"
  echo "  git add $(list_paths | tr '\n' ' ')"
  echo "  git commit -m \"chore(submodules): bump flock\""
}

cmd_remote_add() {
  need_git
  while IFS= read -r path; do
    url="$(url_for "$path")"
    [[ -n "$url" ]] || die "no url for $path in .gitmodules"
    if ! git -C "$path" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
      echo "skip $path (not a checkout)"
      continue
    fi
    if git -C "$path" remote get-url origin >/dev/null 2>&1; then
      cur="$(git -C "$path" remote get-url origin)"
      if [[ "$cur" != "$url" ]]; then
        echo "set-url origin $path: $cur → $url"
        git -C "$path" remote set-url origin "$url"
      else
        echo "ok origin $path → $url"
      fi
    else
      echo "remote add origin $path → $url"
      git -C "$path" remote add origin "$url"
    fi
  done < <(list_paths)
}

cmd_add() {
  need_git
  local url="${1:-}" path="${2:-}" branch="${3:-}"
  [[ -n "$url" && -n "$path" ]] || die "usage: submodules.sh add <url> <path> [branch]"
  if [[ -n "$branch" ]]; then
    git submodule add -b "$branch" "$url" "$path"
  else
    git submodule add "$url" "$path"
  fi
  echo "added submodule $path"
}

case "$cmd" in
  init)       cmd_init ;;
  update)     cmd_update ;;
  status)     cmd_status ;;
  bump)       cmd_bump ;;
  remote-add) cmd_remote_add ;;
  add)        cmd_add "$@" ;;
  -h|--help|help|"") usage; [[ -n "$cmd" ]] || exit 1 ;;
  *) die "unknown command: $cmd (try --help)" ;;
esac
