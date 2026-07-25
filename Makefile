.PHONY: help hooks submodules status registry starling-lite steadyhelm test-mcp agent-demo clean

ROOT := $(abspath .)
BIN  := $(ROOT)/bin

help:
	@echo "Warble Army targets:"
	@echo "  make hooks           install .githooks"
	@echo "  make submodules      init/update submodules"
	@echo "  make status          submodule + registry status"
	@echo "  make registry        validate mcp/tools registry"
	@echo "  make steadyhelm      build SteadyHelm → bin/"
	@echo "  make starling-lite   build Starling Lite MCP → bin/"
	@echo "  make test-mcp        tools/list smoke (stdio)"
	@echo "  make agent-demo      HTTP MCP + reflexion on bad manifest"
	@echo "  make clean           remove bin/"

hooks:
	./scripts/install-hooks.sh

submodules:
	./scripts/submodules.sh init

status:
	./scripts/submodules.sh status
	./scripts/check-mcp-registry.sh

registry:
	./scripts/check-mcp-registry.sh

steadyhelm:
	mkdir -p $(BIN)
	cd steadyhelm && go build -o $(BIN)/steadyhelm .

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

clean:
	rm -rf $(BIN)
