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

# Apply manifests (updates existing Service agentgateway)
kubectl apply -f deploy/agentgateway/k8s/

# Wait
kubectl -n warblecloud-ai rollout status deploy/agentgateway

# Port-forward MCP
kubectl -n warblecloud-ai port-forward svc/agentgateway 3000:3000
# MCP:   http://127.0.0.1:3000/mcp
# Admin: kubectl port-forward svc/agentgateway 15000:15000
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
