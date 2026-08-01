# starling-ex-ce Helm chart

**Starling EX Community Edition** — deploys Starling Lite MCP (Heron Ops cluster tools + ShrikeOps scan tools) with RBAC.

## Local preview first (recommended)

```bash
# from warble-army root
make ce-preview          # helm template only — no cluster write
make ce-up               # local binary MCP on :8822
make ce-smoke            # health + tools/list
make ce-down             # stop

# or Docker
make ce-docker
```

Installer: [`scripts/install.sh`](../../scripts/install.sh)

```bash
./scripts/install.sh help
./scripts/install.sh local
./scripts/install.sh docker
./scripts/install.sh helm-template
# helm install only when you intend to use the current kube context:
# CE_FORCE=1 ./scripts/install.sh helm   # required if context is GKE/prod-looking
```

Always-on public demo (no install): https://mcp.warbleoss.org/mcp

## Install (Helm — cluster)

```bash
helm upgrade --install starling-ex-ce ./charts/starling-ex-ce \
  --namespace warble-army --create-namespace

kubectl -n warble-army port-forward svc/starling-ex-ce 8822:8822
curl -sS http://127.0.0.1:8822/health
```

With public Ingress (example: frakma demo host):

```bash
helm upgrade --install starling-ex-ce ./charts/starling-ex-ce \
  --namespace warblecloud-ai \
  --set ingress.enabled=true \
  --set ingress.host=starling-ex.frakma.io \
  --set image.tag=6c6505f
```

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `mode` | `readonly` | MCP mode |
| `image.repository` | AR starling-lite | Container image |
| `serviceAccount.clusterRole` | `true` | Heron Ops read ClusterRole |
| `ingress.enabled` | `false` | Public Ingress |
| `publicDemoUrl` | mcp.warbleoss.org/mcp | Documented Try MCP link |

## Components

- **Starling Lite** container (embeds Heron Ops + ShrikeOps)
- ServiceAccount + ClusterRole (read-only K8s)
- Optional Ingress with `/lite` rewrite

See [warbleoss.org](https://warbleoss.org) and [STARLING_EX_CE.md](../../docs/STARLING_EX_CE.md).
