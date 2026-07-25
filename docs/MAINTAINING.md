# Maintaining Warble Army

How to keep the umbrella, submodules, remotes, and hooks healthy.

## One-time setup (clone)

```bash
git clone --recurse-submodules https://github.com/warble-tech/warble-army.git
cd warble-army
./scripts/install-hooks.sh
./scripts/submodules.sh status
```

`install-hooks.sh` sets:

```bash
git config core.hooksPath .githooks
```

## Submodule map

| Path | URL | Tracked branch |
|------|-----|----------------|
| `shrikeops` | `https://github.com/warble-tech/shrikeops.git` | `master` |
| `shrikeops-demo` | `https://github.com/warble-tech/shrikeops-demo.git` | `master` |
| `starling` | `https://github.com/warble-tech/starling.git` | `main` |
| `heron-ops` | `https://github.com/warble-tech/heron-ops.git` | `master` |

Defined in [`.gitmodules`](../.gitmodules).

## Common operations

### Init / update checkouts

```bash
./scripts/submodules.sh init      # after fresh clone without --recurse
./scripts/submodules.sh update    # fetch + sync to recorded gitlinks
```

### See drift

```bash
./scripts/submodules.sh status
```

Shows dirty worktrees, detached SHAs, and ahead/behind vs remote branch.

### Bump pins to latest remote tips

```bash
./scripts/submodules.sh bump
git add shrikeops shrikeops-demo starling heron-ops
git commit -m "chore(submodules): bump flock"
```

### Re-add remotes (if `origin` missing inside a child)

```bash
./scripts/submodules.sh remote-add
```

This re-writes `origin` from `.gitmodules` for each submodule.

### Work inside a child product

```bash
cd heron-ops
# ... commits on product branch ...
git push origin HEAD
cd ..
# then bump umbrella gitlink
./scripts/submodules.sh status
git add heron-ops
git commit -m "chore(submodules): pin heron-ops to <sha>"
```

**Never** commit product source changes only on the umbrella without pushing the child first — the gitlink would point at a non-public SHA.

## Adding a new flock member (submodule)

```bash
./scripts/submodules.sh add https://github.com/warble-tech/NEW.git path/NEW [branch]
./scripts/install-hooks.sh   # no-op if already set; re-assert hooksPath
git commit -m "chore(submodules): add NEW"
```

Update this doc’s table and the master `README.md` submodule table in the same PR.

## Hooks

| Hook | Job |
|------|-----|
| `.githooks/pre-commit` | Submodules not dirty; no accidental large secrets; MCP JSON valid |
| `.githooks/commit-msg` | Conventional Commits (`feat|fix|docs|chore|refactor|test|ci|perf|style`) |
| `.githooks/pre-push` | Gitlinks resolve; optional `scripts/check-mcp-registry.sh` |

Skip in emergencies only:

```bash
git commit --no-verify   # avoid; explain in PR
```

## Optional: pre-commit framework

If you use [pre-commit](https://pre-commit.com/) globally:

```bash
pip install pre-commit
pre-commit install
pre-commit run --all-files
```

Config: [`.pre-commit-config.yaml`](../.pre-commit-config.yaml). Native `.githooks` still work without Python.

## Publishing this umbrella

```bash
# create empty repo on GitHub: warble-tech/warble-army
git remote add origin https://github.com/warble-tech/warble-army.git
git push -u origin main
```

Consumers clone with `--recurse-submodules`.

## CI suggestion (GitHub Actions)

```yaml
# .github/workflows/submodules.yml
on: [push, pull_request]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive
      - run: ./scripts/submodules.sh status
      - run: ./scripts/check-mcp-registry.sh
```

## Versioning

- Umbrella tags: `army-v0.1.0` pin a known-good set of submodule SHAs.
- Product tags stay in child repos (`shrikeops v0.x`, `heron-ops v0.x`).
