# Starling Lite MCP — Railway / Fly / Render (always-on PaaS)
# Build context: warble-army repository root
#
#   docker build -f deploy/railway/Dockerfile -t starling-lite .
#   docker run -p 8080:8080 -e PORT=8080 starling-lite

# ShrikeOps scanners — without these, scan_* results are reported as incomplete.
# Static Go binaries, so they run on distroless/static. Versions match the
# ShrikeOps README "Requirements" (parsers are tested against them).
FROM alpine:3.20 AS scanners
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

FROM golang:1.25-bookworm AS build
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
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=0.3.0 -X main.revision=${REV:-dev}" -o /out/starling-lite ./cmd/starling-lite

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/starling-lite /usr/local/bin/starling-lite
COPY --from=scanners /out/pluto /out/polaris /out/kube-score /usr/local/bin/
COPY --from=build /src/mcp/tools/registry.json /etc/warble-army/registry.json

# Railway injects PORT; default 8080 is common on PaaS
ENV PORT=8080 \
    STARLING_MODE=readonly \
    STARLING_TRANSPORT=http \
    STARLING_REGISTRY=/etc/warble-army/registry.json

USER nonroot:nonroot
EXPOSE 8080

# Distroless has no shell — use the binary as entrypoint; PORT resolved at runtime in Go
ENTRYPOINT ["/usr/local/bin/starling-lite"]
CMD ["--mode=readonly", "--transport=http", "--registry=/etc/warble-army/registry.json"]
