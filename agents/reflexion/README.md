# Reflexion agent (Phase 3 scaffold)

Minimal **Actor → Critic → propose** loop over Starling Lite MCP (HTTP).

```text
Incident / path
    → Actor:   call scan_manifest + cluster_summary
    → Critic:  reject if blast radius high / missing dry_run
    → Healer:  propose_remediation for top findings
    → Audit:   write JSONL trace
```

## Prerequisites

```bash
# from warble-army root
make starling-lite
./bin/starling-lite --transport=http --addr=:8822 --mode=readonly
```

## Run

```bash
# Python 3.10+
pip install -r agents/reflexion/requirements.txt   # empty / stdlib only

# Scan a bad manifest through the loop
python3 agents/reflexion/run.py \
  --mcp http://127.0.0.1:8822/mcp \
  --path shrikeops-demo/k8s/bad/root-container.yaml
```

## Output

- stdout: summary table (score, findings, proposals)
- `agents/reflexion/traces/trace-*.jsonl` — training / audit data

## Roadmap

- [x] HTTP MCP client (`tools/list`, `tools/call`)
- [x] Actor scan + propose
- [x] Critic gates (critical count, write denial)
- [ ] Wire real LLMs (OpenAI-compatible / Ollama) for free-form reasoning
- [ ] Avirka webhook intake
- [ ] Apply path behind dual approval
