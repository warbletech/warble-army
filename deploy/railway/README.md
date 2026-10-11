# Always-on Starling Lite (Railway / Fly / Render)

Run the **public OSS MCP ops server** on a PaaS so it stays up **independent of GKE / FRAKMA / any single cloud cluster**.

```text
  Cursor / Claude / curl
           │  HTTPS
           ▼
   Railway (or Fly/Render)   ← always-on process
   starling-lite :$PORT
      ├── scan_*           (works offline — no cluster)
      ├── propose_*        (dry-run remediation)
      └── cluster_*        (optional; needs KUBECONFIG_B64)
```

Today’s GKE demo (`starling-ex.frakma.io/lite/mcp`) can stay as primary or become secondary.  
**PaaS is the “keep-alive” path** when your cloud project is paused, billed, or rebuilt.

---

## What you get on Railway

| Capability | Without kubeconfig | With `KUBECONFIG_B64` secret |
|------------|--------------------|------------------------------|
| `tools/list`, health | ✅ | ✅ |
| ShrikeOps `scan_*` | ✅ | ✅ |
| Dry-run remediation | ✅ | ✅ |
| Live cluster MCP (`list_pods`, …) | ❌ graceful | ✅ points at **your** cluster API |

For a **true vendor-neutral public demo**, leave kubeconfig unset: only scan tools.  
Cluster tools always depend on *some* Kubernetes API (yours or a small free-tier cluster).

---

## Deploy on Railway (recommended)

### A. One-click from GitHub (UI)

1. https://railway.app → **New Project** → **Deploy from GitHub repo**
2. Select **`warbletech/warble-army`** (public)
3. Settings:
   - **Root Directory:** repository root
   - **Builder:** Dockerfile
   - **Dockerfile path:** `deploy/railway/Dockerfile`
4. Variables (service):

   | Name | Value |
   |------|--------|
   | `STARLING_MODE` | `readonly` |
   | `STARLING_TRANSPORT` | `http` |
   | `STARLING_REGISTRY` | `/etc/warble-army/registry.json` |
   | `PORT` | *(auto — do not hardcode if Railway injects it)* |

5. **Networking** → Generate domain (e.g. `starling-lite-production.up.railway.app`)
6. **Sleep / idle:** on free/trial tiers Railway may sleep. For always-on:
   - Use a plan that allows **no idle sleep**, or
   - Enable “Always On” / remove idle policy in service settings (wording varies by plan)

7. Smoke:

```bash
export MCP=https://YOUR-APP.up.railway.app
curl -sS "$MCP/health"
curl -sS -X POST "$MCP/mcp" \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### B. Railway CLI

```bash
# https://docs.railway.app/guides/cli
npm i -g @railway/cli   # or: brew install railway
railway login
cd /path/to/warble-army
railway init             # link project
railway up               # uses deploy/railway/railway.toml if present at root
# or:
railway up --dockerfile deploy/railway/Dockerfile
railway domain
```

Copy `deploy/railway/railway.toml` to repo root if your Railway project expects root-level config, **or** set Dockerfile path in the dashboard (recommended).

### Optional: live cluster tools

**Do not** put production cluster admin kubeconfigs on a public demo.

For a private Railway service (auth in front / non-public URL):

```bash
# base64 of a least-privilege readonly kubeconfig
base64 -w0 ~/.kube/readonly-demo.yaml | railway variables set KUBECONFIG_B64=@/dev/stdin
```

Use a ServiceAccount that is **namespace-scoped + get/list/watch only**.

---

## Fly.io alternative

```bash
# flyctl launch --dockerfile deploy/railway/Dockerfile --no-deploy
# fly.toml:
#   [http_service]
#     internal_port = 8080
#     auto_stop_machines = false
#     min_machines_running = 1
fly deploy --dockerfile deploy/railway/Dockerfile
```

`min_machines_running = 1` + `auto_stop_machines = false` ≈ always-on.

## Render alternative

- New **Web Service** → Docker
- Dockerfile path: `deploy/railway/Dockerfile`
- Health check: `/health`
- Instance: paid “always on” (free tier spins down)

---

## MCP client config

```json
{
  "mcpServers": {
    "warble-army": {
      "url": "https://YOUR-APP.up.railway.app/mcp"
    }
  }
}
```

Same shape as the GKE demo:

```json
{ "url": "https://starling-ex.frakma.io/lite/mcp" }
```

(`/lite/mcp` is also served for path compatibility.)

---

## Helper scripts (Linux + PowerShell)

From repo root:

```bash
# Linux / macOS
./scripts/starling-lite-ops.sh help
./scripts/starling-lite-ops.sh smoke --url https://starling-lite-production.up.railway.app
./scripts/starling-lite-ops.sh railway-up
./scripts/starling-lite-ops.sh cloudflare-dns --host mcp --zone warbleoss.org
```

```powershell
# Windows PowerShell
.\scripts\starling-lite-ops.ps1 help
.\scripts\starling-lite-ops.ps1 smoke -Url https://starling-lite-production.up.railway.app
.\scripts\starling-lite-ops.ps1 railway-up
.\scripts\starling-lite-ops.ps1 cloudflare-dns -HostName mcp -Zone warbleoss.org
```

Cloudflare needs `CLOUDFLARE_API_TOKEN` + `CLOUDFLARE_ZONE_ID` (or gcloud secrets
`cloudflare-api-token` / `cloudflare-zone-id` on Linux).

## Local Docker (parity with Railway)

```bash
# from warble-army root
docker build -f deploy/railway/Dockerfile -t starling-lite:local .
docker run --rm -p 8080:8080 -e PORT=8080 starling-lite:local

curl -s localhost:8080/health
```

---

## Env reference

| Variable | Default | Meaning |
|----------|---------|---------|
| `PORT` | `8080` in image / PaaS injects | HTTP listen port |
| `STARLING_ADDR` | derived from `PORT` | Override listen addr (`:8822`) |
| `STARLING_MODE` | `readonly` | `readonly` \| `readwrite` |
| `STARLING_TRANSPORT` | set via CMD `http` on PaaS | Must be `http` on Railway |
| `STARLING_REGISTRY` | `/etc/warble-army/registry.json` | Tool registry path |
| `STARLING_NAMESPACE` | empty | Restrict cluster tools |
| `STARLING_ALLOW_NAMESPACES` | empty | Write allowlist |
| `KUBECONFIG` | empty | Path inside container |
| `KUBECONFIG_B64` | empty | Base64 kubeconfig for optional cluster tools |

---

## Architecture choice

| Host | Use when |
|------|----------|
| **Railway / Fly / Render** | Public always-on MCP demo; OSS marketing; independent of GKE |
| **GKE + Ingress** (`deploy/agentgateway/k8s`) | In-cluster Agent Gateway, FRAKMA co-location, private mesh |
| **Both** | Railway primary for docs; GKE secondary for dogfood |

Recommended for warbleoss.org:

1. Deploy Railway **scan-only** public MCP.  
2. Point landing “Try MCP” at Railway URL (or keep GKE until Railway is live).  
3. Keep GKE for internal FRAKMA demos with real cluster tools.

---

## Security

- Public service: **always** `STARLING_MODE=readonly`.
- Never commit kubeconfigs.
- Prefer Railway **shared variables** / secrets for `KUBECONFIG_B64`.
- Rate-limit at Cloudflare / Railway if abuse appears.
- Report issues: security@warblecloud.com

## Related

- [deploy/agentgateway/README.md](../agentgateway/README.md) — GKE path  
- [mcp/starling-lite/README.md](../../mcp/starling-lite/README.md) — binary flags  
- [docs/STARLING_EX_CE.md](../../docs/STARLING_EX_CE.md) — CE product scope  
