# Quickstart: evaluate Warble Army in 15 minutes

This guide takes you from a clean laptop to a **dry-run fix proposal for a broken workload**. Nothing in it
changes your cluster: Starling Lite runs in `--mode=readonly`, and every fix is a proposal
(`dry_run: true`) that you can read and throw away.

| Path | Time | Needs |
|------|------|-------|
| **A. Public demo, no install** | 2 min | `curl` |
| **B. Laptop with kind** (recommended) | ~15 min | Git, Go 1.26+, Docker, [kind](https://kind.sigs.k8s.io/), `kubectl` |

---

## A. Public demo (no install)

The read-only demo at `https://mcp.warbleoss.org/mcp` serves the scan and fix tools. Cluster tools
(`cluster_summary`, `list_pods`, …) return an error there until a demo cluster is attached.

```bash
curl -s https://mcp.warbleoss.org/health

curl -s -X POST https://mcp.warbleoss.org/mcp \
  -H 'content-type: application/json' -H 'accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | head -c 400; echo
```

To use it from an MCP client, point the client at `https://mcp.warbleoss.org/mcp` (HTTP transport).

---

## B. Laptop with kind

### 1. Clone with submodules (1 min)

```bash
git clone --recurse-submodules https://github.com/warbletech/warble-army.git
cd warble-army
```

### 2. Build Starling Lite (2–3 min)

Starling Lite is the MCP server. It loads `mcp/tools/registry.json` and embeds Heron Ops (cluster tools)
and ShrikeOps (scanner).

```bash
make starling-lite
# without make:
#   ./scripts/check-mcp-registry.sh
#   (cd mcp/starling-lite && go build -o ../../bin/starling-lite ./cmd/starling-lite)
```

### 3. Install the three scanners (1 min)

`scan_*` shells out to `pluto`, `polaris` and `kube-score`. If any is missing, results come back
`"incomplete": true` with grade `N/A`. That means *not scanned*, not a pass. These are the versions pinned in
`deploy/railway/Dockerfile` (Linux amd64 shown; use `darwin` / `arm64` as needed):

```bash
mkdir -p ~/.local/bin && cd ~/.local/bin
OS=linux ARCH=amd64
curl -fsSL "https://github.com/FairwindsOps/pluto/releases/download/v5.24.4/pluto_5.24.4_${OS}_${ARCH}.tar.gz" | tar -xz pluto
curl -fsSL "https://github.com/FairwindsOps/polaris/releases/download/v10.2.5/polaris_10.2.5_${OS}_${ARCH}.tar.gz" | tar -xz polaris
curl -fsSL "https://github.com/zegl/kube-score/releases/download/v1.20.0/kube-score_1.20.0_${OS}_${ARCH}.tar.gz" | tar -xz kube-score
export PATH="$HOME/.local/bin:$PATH"
cd -
```

### 4. Check scan and fix without a cluster (1 min)

```bash
cat > /tmp/mcp-call.sh <<'SH'
#!/usr/bin/env bash
# usage: mcp-call.sh <tool> '<json arguments>'
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"quickstart","version":"0"},"capabilities":{}}}' \
  "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"$1\",\"arguments\":$2}}" \
  | ./bin/starling-lite --mode=readonly --transport=stdio --registry=mcp/tools/registry.json 2>/dev/null \
  | tail -1 | jq -r '.result.content[0].text'
SH
chmod +x /tmp/mcp-call.sh

M=$(jq -Rs . < shrikeops-demo/k8s/bad/root-container.yaml)
/tmp/mcp-call.sh scan_manifest "{\"manifest\":$M}" | jq '{score: .score.overall, grade: .score.grade, findings: (.findings|length)}'
# expect roughly: score 65.5, grade "C", 14 findings

/tmp/mcp-call.sh propose_remediation \
  "{\"kind\":\"Deployment\",\"name\":\"insecure-api\",\"issue\":\"POL-runAsRootAllowed\",\"manifest\":$M}"
# expect a strategic-merge patch with runAsNonRoot: true, "dry_run": true and a kubectl --dry-run=server hint
```

### 5. Create a kind cluster and break some workloads (3–4 min)

```bash
kind create cluster --name warble-eval
kubectl config use-context kind-warble-eval   # make sure you are NOT pointed at a real cluster

kubectl apply -f shrikeops-demo/k8s/broken/namespace.yaml
kubectl apply -f shrikeops-demo/k8s/broken/
kubectl -n demo get pods -w   # within ~1 minute: CrashLoopBackOff, OOMKilled, ImagePullBackOff
```

The samples are a crash loop (`broken-api`), an out-of-memory kill (`memory-hog`) and a bad image tag
(`bad-image`). See `shrikeops-demo/k8s/broken/README.md`.

> If `shrikeops-demo/k8s/broken/` is missing, your submodule pin is older than the samples. Run
> `git -C shrikeops-demo pull origin master`.

### 6. Connect your IDE in read-only mode (2 min)

Cursor (`~/.cursor/mcp.json`), Claude Desktop, or any MCP client that runs a stdio server. Use absolute paths:

```json
{
  "mcpServers": {
    "warble-army": {
      "command": "/ABS/PATH/warble-army/bin/starling-lite",
      "args": ["--mode=readonly", "--transport=stdio",
               "--registry", "/ABS/PATH/warble-army/mcp/tools/registry.json"],
      "env": {
        "KUBECONFIG": "/home/you/.kube/config",
        "PATH": "/home/you/.local/bin:/usr/local/bin:/usr/bin:/bin"
      }
    }
  }
}
```

`--mode=readonly` hides the write tools (`scale_deployment`, `delete_pod`). `apply_remediation` stays dry-run.

### 7. Ask the question (2 min)

> What is unhealthy in namespace demo? Propose a fix without changing anything.

A good answer uses `cluster_summary` / `list_pods` / `get_pod_logs` / `list_events` to find the three
failures, then `propose_remediation` for dry-run patches. For example, it raises the memory limit for
`memory-hog` and fixes the image tag for `bad-image`. Nothing is applied.

### 8. Clean up

```bash
kind delete cluster --name warble-eval
```

---

## What to look at next

- Safety model: [`docs/safety.md`](docs/safety.md) (read-only default, dry-run, gated writes)
- Guarded LLM fallback for fixes: [`mcp/starling-lite/README.md`](mcp/starling-lite/README.md)
- Per-project sandbox namespaces: [`docs/sandbox.md`](docs/sandbox.md)
- Helm install (`charts/starling-ex-ce`): the image and chart are **not anonymously pullable yet**, so outside
  evaluators should use path B for now.

Found a problem with this guide? Open an issue. Security reports go to security@warblecloud.com
(see [`SECURITY.md`](SECURITY.md)).
