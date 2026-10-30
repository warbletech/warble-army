# Labels

Source of truth: [`.github/labels.yml`](../.github/labels.yml). Every issue/PR
should carry one `kind/*`, at least one `area/*`, and (after triage) one
`priority/*`.

| Prefix | Meaning | Examples |
|--------|---------|----------|
| `area/*` | Which part of the repo | `area/mcp`, `area/charts`, `area/agents`, `area/submodules`, `area/ci` |
| `kind/*` | Type of work | `kind/bug`, `kind/feature`, `kind/docs`, `kind/dependencies` |
| `priority/*` | Urgency after triage | `priority/critical` … `priority/low` |
| `triage/needed` | Added by issue forms; removed once triaged | |
| `good first issue` / `help wanted` | Newcomer-friendly work; ideas in [GOOD_FIRST_ISSUES.md](GOOD_FIRST_ISSUES.md) | |

## Syncing (maintainers)

Labels are **not** created automatically. After review, a maintainer can
create them once, e.g.:

```bash
python3 - <<'PY' | sh
import yaml, shlex
for l in yaml.safe_load(open(".github/labels.yml")):
    print("gh label create {} --color {} --description {} --force".format(
        shlex.quote(l["name"]), l["color"], shlex.quote(l["description"])))
PY
```

## Seeding good first issues

[GOOD_FIRST_ISSUES.md](GOOD_FIRST_ISSUES.md) lists ideas. Turning them into
real issues (labelled `good first issue` + `help wanted` + one `area/*`) is a
maintainer step — each issue should state the goal, the files involved, and
how to verify (`make …`).
