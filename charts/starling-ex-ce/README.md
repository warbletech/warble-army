# starling-ex-ce Helm chart

**Starling EX Community Edition** — deploys Starling Lite MCP (Heron Ops cluster tools + ShrikeOps scan tools) with RBAC.

## Install

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
| `publicDemoUrl` | starling-ex.frakma.io/lite/mcp | Documented Try MCP link |

## Components

- **Starling Lite** container (embeds Heron Ops + ShrikeOps)
- ServiceAccount + ClusterRole (read-only K8s)
- Optional Ingress with `/lite` rewrite

See [warbleoss.org](https://warbleoss.org) and [STARLING_EX_CE.md](../../docs/STARLING_EX_CE.md).
