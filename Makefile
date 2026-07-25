.PHONY: help hooks submodules status registry starling-lite steadyhelm test-mcp agent-demo \
	agentgateway-bin agentgateway-local agentgateway-image agentgateway-apply clean

ROOT := $(abspath .)
BIN  := $(ROOT)/bin
AGW_VERSION ?= v1.4.0-beta.1
REGISTRY ?= us-central1-docker.pkg.dev/warble-tech/warble

help:
	@echo "Warble Army targets:"
	@echo "  make hooks              install .githooks"
	@echo "  make submodules         init/update submodules"
	@echo "  make status             submodule + registry status"
	@echo "  make registry           validate mcp/tools registry"
	@echo "  make steadyhelm         build SteadyHelm → bin/"
	@echo "  make starling-lite      build Starling Lite MCP → bin/"
	@echo "  make test-mcp           tools/list smoke (stdio)"
	@echo "  make agent-demo         HTTP MCP + reflexion on bad manifest"
	@echo "  make agentgateway-bin   download agentgateway $(AGW_VERSION)"
	@echo "  make agentgateway-local starling-lite HTTP + agentgateway front door"
	@echo "  make agentgateway-image build/push starling-lite container"
	@echo "  make agentgateway-apply kubectl apply deploy/agentgateway/k8s"
	@echo "  make clean              remove bin/"

hooks:
	./scripts/install-hooks.sh

submodules:
	./scripts/submodules.sh init

status:
	./scripts/submodules.sh status
	./scripts/check-mcp-registry.sh

registry:
	./scripts/check-mcp-registry.sh

steadyhelm heron-ops:
	mkdir -p $(BIN)
	cd heron-ops && go build -o $(BIN)/heron-ops .
	ln -sfn heron-ops $(BIN)/steadyhelm
	@echo "built $(BIN)/heron-ops (alias: $(BIN)/steadyhelm)"

starling-lite: registry
	mkdir -p $(BIN)
	cd mcp/starling-lite && go mod tidy && go build -o $(BIN)/starling-lite ./cmd/starling-lite

test-mcp: starling-lite
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"make","version":"0"},"capabilities":{}}}' | \
	  $(BIN)/starling-lite --mode=readonly --transport=stdio --registry=$(ROOT)/mcp/tools/registry.json | head -c 400; echo
	@printf '%s\n' \
	  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"t","version":"0"},"capabilities":{}}}' \
	  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
	  | $(BIN)/starling-lite --mode=readonly --transport=stdio --registry=$(ROOT)/mcp/tools/registry.json \
	  | tail -n1 | python3 -c "import sys,json; r=json.load(sys.stdin); print('tools', len(r['result']['tools']))"

agent-demo: starling-lite
	@echo "Starting starling-lite on :8822..."
	@$(BIN)/starling-lite --mode=readonly --transport=http --addr=:8822 \
	  --registry=$(ROOT)/mcp/tools/registry.json & echo $$! > /tmp/starling-lite-demo.pid
	@sleep 1
	@python3 agents/reflexion/run.py \
	  --mcp http://127.0.0.1:8822/mcp \
	  --path shrikeops-demo/k8s/bad/root-container.yaml \
	  || true
	@kill $$(cat /tmp/starling-lite-demo.pid) 2>/dev/null || true
	@rm -f /tmp/starling-lite-demo.pid

# ── Agent Gateway (https://agentgateway.dev) ─────────────────────────────────

agentgateway-bin:
	mkdir -p $(BIN)
	@if [ -x $(BIN)/agentgateway ]; then \
	  $(BIN)/agentgateway --version 2>/dev/null || true; \
	  echo "agentgateway already at $(BIN)/agentgateway"; \
	else \
	  curl -fsSL -o $(BIN)/agentgateway \
	    "https://github.com/agentgateway/agentgateway/releases/download/$(AGW_VERSION)/agentgateway-linux-amd64"; \
	  chmod +x $(BIN)/agentgateway; \
	  echo "installed $(AGW_VERSION) → $(BIN)/agentgateway"; \
	fi

# Starts gateway only (expects starling-lite already on :8822)
agentgateway-local: agentgateway-bin
	$(BIN)/agentgateway -f $(ROOT)/deploy/agentgateway/config.local.yaml

agentgateway-image:
	@command -v docker >/dev/null || { echo "docker required for image build"; exit 1; }
	docker build -f deploy/agentgateway/Dockerfile.starling-lite \
	  -t $(REGISTRY)/starling-lite:latest \
	  -t $(REGISTRY)/starling-lite:$(shell git rev-parse --short HEAD 2>/dev/null || echo dev) \
	  .
	docker push $(REGISTRY)/starling-lite:latest

agentgateway-apply:
	kubectl apply -f $(ROOT)/deploy/agentgateway/k8s/
	kubectl -n warblecloud-ai rollout status deploy/agentgateway --timeout=180s || true
	kubectl -n warblecloud-ai get deploy,svc,pods -l app=agentgateway

clean:
	rm -rf $(BIN)
