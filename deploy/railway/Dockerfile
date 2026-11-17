# Starling Lite MCP — Railway / Fly / Render (always-on PaaS)
# Build context: warble-army repository root
#
#   docker build -f deploy/railway/Dockerfile -t starling-lite .
#   docker run -p 8080:8080 -e PORT=8080 starling-lite

# ShrikeOps scanners — without these, scan_* results are reported as incomplete.
# Static Go binaries, so they run on distroless/static. Versions match the
# ShrikeOps README "Requirements" (parsers are tested against them).
FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS scanners
ARG TARGETARCH=amd64
ARG PLUTO_VERSION=5.24.4
ARG POLARIS_VERSION=10.2.5
ARG KUBE_SCORE_VERSION=1.20.0
RUN apk add --no-cache curl tar \
 && mkdir -p /out \
 && curl -fsSL "https://github.com/FairwindsOps/pluto/releases/download/v${PLUTO_VERSION}/pluto_${PLUTO_VERSION}_linux_${TARGETARCH}.tar.gz" | tar -xz -C /out pluto \
 && curl -fsSL "https://github.com/FairwindsOps/polaris/releases/download/v${POLARIS_VERSION}/polaris_${POLARIS_VERSION}_linux_${TARGETARCH}.tar.gz" | tar -xz -C /out polaris \
 && curl -fsSL "https://github.com/zegl/kube-score/releases/download/v${KUBE_SCORE_VERSION}/kube-score_${KUBE_SCORE_VERSION}_linux_${TARGETARCH}.tar.gz" | tar -xz -C /out kube-score \
 && /out/pluto version && /out/polaris version && /out/kube-score version

FROM golang:1.27-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS build
WORKDIR /src

COPY mcp/starling-lite ./mcp/starling-lite
COPY heron-ops ./heron-ops
COPY shrikeops ./shrikeops
COPY mcp/tools ./mcp/tools
# Git commit stamped by CD (.github/workflows/deploy-starling-lite.yml); "dev" in git.
COPY deploy/railway/REVISION /src/REVISION

WORKDIR /src/mcp/starling-lite
RUN go mod tidy \
 && REV="$(tr -d '[:space:]' < /src/REVISION)" \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=0.3.0 -X main.revision=${REV:-dev}" -o /out/starling-lite ./cmd/starling-lite \
 && mkdir -p /out/scan-root

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
WORKDIR /
COPY --from=build /out/starling-lite /usr/local/bin/starling-lite
COPY --from=scanners /out/pluto /out/polaris /out/kube-score /usr/local/bin/
COPY --from=build /src/mcp/tools/registry.json /etc/warble-army/registry.json
# Empty, read-only scan root: path= / chart_path= arguments on the public server
# resolve inside it, never against "/" (KUBECONFIG_B64 is written under /tmp).
COPY --from=build --chown=0:0 /out/scan-root /srv/scan

# Railway injects PORT; default 8080 is common on PaaS
ENV PORT=8080 \
    STARLING_MODE=readonly \
    STARLING_TRANSPORT=http \
    STARLING_REGISTRY=/etc/warble-army/registry.json \
    STARLING_SCAN_ROOT=/srv/scan \
    STARLING_CHART_ROOT=/srv/scan

USER nonroot:nonroot
EXPOSE 8080

# Distroless has no shell — use the binary as entrypoint; PORT resolved at runtime in Go
ENTRYPOINT ["/usr/local/bin/starling-lite"]
CMD ["--mode=readonly", "--transport=http", "--registry=/etc/warble-army/registry.json"]
