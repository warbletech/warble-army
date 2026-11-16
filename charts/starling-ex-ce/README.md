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

From a clone, render does not pull an image. `image.tag=local` is only a render placeholder:

```bash
./scripts/install.sh helm-template
```

A cluster install refuses to run until you set `CE_IMAGE_TAG` to a full git SHA that [Publish starling-lite image](../../.github/workflows/publish-starling-lite.yml) has already pushed. No such tag is published yet.

```bash
CE_IMAGE_TAG=<published-git-sha> CE_FORCE=1 ./scripts/install.sh helm
```

Intended outsider install from GHCR, once both objects exist and an org owner has made them public. This is the target flow. It is not an anonymous-pull claim:

```bash
helm pull oci://ghcr.io/warbletech/charts/starling-ex-ce --version 0.1.1

helm upgrade --install starling-ex-ce \
  oci://ghcr.io/warbletech/charts/starling-ex-ce \
  --version 0.1.1 \
  --namespace warble-army --create-namespace \
  --set image.repository=ghcr.io/warbletech/warble-army/starling-lite \
  --set image.tag=<published-git-sha>
```

Chart `0.1.1` is this tree's package version. It is not in GHCR until `Publish Helm charts` runs on `main`. Chart `0.1.0` is an older private package and still points at Artifact Registry. Do not install `0.1.0` for this image.

The image name is `ghcr.io/warbletech/warble-army/starling-lite`. The only tag the publish workflow pushes is the full git commit SHA of that run. `:latest` is not published. `<published-git-sha>` above is a placeholder, not a tag that exists today.

Anonymous `helm pull` of `0.1.0` and anonymous manifest GET of the Artifact Registry image returned 401 on 16 Nov 2026. Those checks have not been repeated against `0.1.1` or against `ghcr.io/warbletech/warble-army/starling-lite`, because neither has been pushed yet.

```bash
kubectl -n warble-army port-forward svc/starling-ex-ce 8822:8822
curl -sS http://127.0.0.1:8822/health
```

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `mode` | `readonly` | MCP mode |
| `image.repository` | `ghcr.io/warbletech/warble-army/starling-lite` | Chart image. Not anonymously verified. |
| `image.tag` | empty | Required. Full git SHA already pushed by the image workflow. |
| `serviceAccount.clusterRole` | `true` | Heron Ops read ClusterRole |
| `ingress.enabled` | `false` | Public Ingress |
| `publicDemoUrl` | mcp.warbleoss.org/mcp | Documented Try MCP link |

## Components

- **Starling Lite** container (embeds Heron Ops + ShrikeOps)
- ServiceAccount + ClusterRole (read-only K8s)
- Optional Ingress with `/lite` rewrite

See [warbleoss.org](https://warbleoss.org) and [STARLING_EX_CE.md](../../docs/STARLING_EX_CE.md).
