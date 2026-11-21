# Agent Gateway + Warble Army

[Agentgateway](https://agentgateway.dev) is the **front door** for agent traffic:

| Path | Backend |
|------|---------|
| `/mcp` | **Starling Lite** (scan / cluster / remediate tools) |
| `/v1/*` (local config) | Optional LLM providers (Gemini, …) |
| `:15000/ui/` | Agentgateway admin UI |

```text
Cursor / Claude / custom agents
        │  MCP streamable HTTP  /mcp
        ▼
  agentgateway  (cr.agentgateway.dev)
        │  multiplex / auth / CORS / metrics
        ▼
  starling-lite :8822
        ├── ShrikeOps  (scan_*)
        └── SteadyHelm (cluster_*)
```

The in-cluster Deployment previously named `agentgateway` was an **nginx stub**  
(`warblecloud.ai/parked`). These manifests replace it with the real gateway + Starling Lite sidecar.

## Local (no cluster)

```bash
# 1. Build Starling Lite
make starling-lite

# 2. Terminal A — MCP backend
./bin/starling-lite --mode=readonly --transport=http --addr=:8822 \
  --registry=mcp/tools/registry.json

# 3. Terminal B — Agent Gateway
make agentgateway-bin   # downloads linux/amd64 binary if missing
./bin/agentgateway -f deploy/agentgateway/config.local.yaml

# 4. Point clients at http://127.0.0.1:3000/mcp
#    or run the Reflexion agent:
python3 agents/reflexion/run.py --mcp http://127.0.0.1:3000/mcp \
  --path shrikeops-demo/k8s/bad/root-container.yaml
```

**stdio mode** (gateway spawns Starling Lite):

```bash
export PATH="$PWD/bin:$PATH"
./bin/agentgateway -f deploy/agentgateway/config.stdio.yaml
```

## Kubernetes (`warblecloud-ai`)

```bash
# Build & push Starling Lite image (Artifact Registry)
export REGISTRY=us-central1-docker.pkg.dev/warble-tech/warble
make agentgateway-image

# Apply the in-cluster gateway only. 50-ingress.yaml and 51-oauth2-proxy.yaml
# are not in this command. They change https://starling-ex.frakma.io/lite and
# must wait until the oauth2-proxy Secret exists and that Deployment is Ready.
kubectl apply \
  -f deploy/agentgateway/k8s/00-namespace.yaml \
  -f deploy/agentgateway/k8s/10-configmap.yaml \
  -f deploy/agentgateway/k8s/20-rbac.yaml \
  -f deploy/agentgateway/k8s/30-deployment.yaml \
  -f deploy/agentgateway/k8s/40-service.yaml

# Wait
kubectl -n warblecloud-ai rollout status deploy/agentgateway

# Port-forward MCP
kubectl -n warblecloud-ai port-forward svc/agentgateway 3000:3000
# MCP:   http://127.0.0.1:3000/mcp
# Admin: kubectl port-forward svc/agentgateway 15000:15000
```

### Public URL (production)

| URL | Backend |
|-----|---------|
| **https://starling-ex.frakma.io/lite/mcp** | Starling Lite MCP on GKE (JSON-RPC HTTP) |
| https://starling-ex.frakma.io/lite/health | Health probe |
| https://starling-ex.frakma.io/ | Marketing site (unchanged) |
| https://starling-ex.frakma.io/api | License API (unchanged) |

**Always-on alternative (no GKE):** deploy Starling Lite to Railway / Fly / Render so the public MCP stays up when cloud infra is down or rebuilt — see [`deploy/railway/README.md`](../railway/README.md).

In-cluster federation still uses **Agent Gateway** (`svc/agentgateway`). The public `/lite` path hits Starling Lite directly so clients get simple MCP JSON-RPC without streamable-HTTP session friction.

Ingress: `deploy/agentgateway/k8s/50-ingress.yaml` (namespace `warblecloud-ai`).

`/lite` requires oauth2-proxy (`k8s/51-oauth2-proxy.yaml`). nginx `auth-url` runs before the MCP backend. The proxy Ingress is only `/oauth2`, so the sign-in redirect is not itself behind the MCP auth check. Apply the proxy and wait until it is Ready before applying `50-ingress.yaml`. Applying the MCP Ingress first fails closed: unauthenticated calls do not reach Starling Lite, and authenticated calls fail until the proxy is up. The OIDC client secret stays in the `oauth2-proxy` Secret and is not in this repo. https://mcp.warbleoss.org is the Railway front door and is not this Ingress.

```json
{
  "mcpServers": {
    "warble-army": {
      "url": "https://starling-ex.frakma.io/lite/mcp"
    }
  }
}
```

### Cluster clients

```json
{
  "mcpServers": {
    "warble-army": {
      "url": "http://agentgateway.warblecloud-ai.svc.cluster.local/mcp"
    }
  }
}
```

(For HTTP MCP clients that support URL transport; otherwise port-forward and use streamable HTTP.)

## Config files

| File | Use |
|------|-----|
| `config.local.yaml` | Laptop: HTTP → `127.0.0.1:8822` |
| `config.stdio.yaml` | Laptop: gateway spawns `starling-lite` |
| `config.k8s.yaml` | Reference; live config is in ConfigMap |
| `k8s/*` | Deploy to GKE `warblecloud-ai` |

## Relation to `warble-labs/agentgateway.yaml`

Labs config routes `/mcp/*` and `/v1/chat/completions` for the IoT/marketing stack.  
Army config is the **same product**, specialized for K8s ops tools. Keep both; don’t merge blindly.

## Versions

- Agentgateway image/binary: `v1.4.0-beta.1` (`cr.agentgateway.dev/agentgateway`)
- Schema: https://agentgateway.dev/schema/config
