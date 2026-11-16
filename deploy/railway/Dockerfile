# Starling Lite MCP — Railway / Fly / Render (always-on PaaS)
# Build context: warble-army repository root
#
#   docker build -f deploy/railway/Dockerfile -t starling-lite .
#   docker run -p 8080:8080 -e PORT=8080 starling-lite

# ShrikeOps scanners — without these, scan_* results are reported as incomplete.
# Static Go binaries, so they run on distroless/static. Versions match the
# ShrikeOps README "Requirements" (parsers are tested against them).
FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc AS scanners
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

FROM golang:1.26-bookworm@sha256:dc40a13b61304647ed7e5d581d246b86413dfc29d63c6d018310feb93cd98fc0 AS build
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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
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
