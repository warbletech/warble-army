# Starling Lite MCP — Railway / Fly / Render (always-on PaaS)
# Build context: warble-army repository root
#
#   docker build -f deploy/railway/Dockerfile -t starling-lite .
#   docker run -p 8080:8080 -e PORT=8080 starling-lite

FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY mcp/starling-lite ./mcp/starling-lite
COPY heron-ops ./heron-ops
COPY shrikeops ./shrikeops
COPY mcp/tools ./mcp/tools

WORKDIR /src/mcp/starling-lite
RUN go mod tidy \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=0.2.1" -o /out/starling-lite ./cmd/starling-lite

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/starling-lite /usr/local/bin/starling-lite
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
