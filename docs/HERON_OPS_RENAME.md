# SteadyHelm → Heron Ops rename checklist

**Product name:** Heron Ops  
**CLI (target):** `heron-ops`  
**Legacy alias:** `steadyhelm`  
**Logo:** `brand/heron-ops/heron-ops-logo.jpg`

## Why

SteadyHelm is accurate but opaque. **Heron** matches the Warble Army bird flock (with ShrikeOps + Starling) and reads better on warbleoss.org.

## Steps

### 1. Docs & brand (this PR / army umbrella)

- [x] `docs/BRAND.md` flock table  
- [x] Logo mark  
- [x] Army README product map uses Heron Ops  
- [x] SteadyHelm README states product rename  

### 2. Code (steadyhelm / heron-ops repo)

- [ ] Add `main` package name string `heron-ops` in version output; keep `steadyhelm` binary name temporarily  
- [ ] `go build -o heron-ops` in Makefile; install both names  
- [ ] Module path: either  
  - **A.** `github.com/warble-tech/heron-ops` with `replace` during migration, or  
  - **B.** Keep module path one release, rename module next minor  
- [ ] `pkg/mcpserver` serverInfo name: `heron-ops`  
- [ ] Update Homebrew formula when published  

### 3. GitHub

- [ ] Create `warble-tech/heron-ops` **or** rename `steadyhelm` → `heron-ops`  
- [ ] Redirect old submodule URL in warble-army `.gitmodules`  
- [ ] Topics: `mcp`, `kubernetes`, `heron-ops`, `warbleoss`  

### 4. Army consumers

- [ ] Starling Lite docs: “cluster backend = Heron Ops (SteadyHelm)”  
- [ ] Agent Gateway configs comments  
- [ ] CI binary artifact names  

### 5. warbleoss.org

- [ ] Product card: Heron Ops  
- [ ] Install: `curl … | install heron-ops`  

## Compatibility

Until v1.0 of Heron Ops:

| Interface | Behavior |
|-----------|----------|
| MCP tool names | **Unchanged** (`list_pods`, `cluster_summary`, …) |
| HTTP `/mcp` | Unchanged |
| Binary `steadyhelm` | Still builds; prints deprecation notice once |
| Config flags | Same (`--mode`, `--transport`, `--kubeconfig`) |
