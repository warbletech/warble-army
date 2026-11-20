# AI / LLMOps incident playbook

**Severity cues:** model latency spike, token cost explosion, wrong remediations, guardrail trips, RAG stale.

## 1. Detect
- [ ] p95/p99 latency + error rate on LLM routes
- [ ] Cost per 1k tokens anomaly
- [ ] Safety / policy rejection rate

## 2. Contain
- [ ] Route to cheaper/safer model
- [ ] Disable high-risk tools
- [ ] Cap concurrent agent loops

## 3. Diagnose
- [ ] Prompt / context bloat
- [ ] Cache miss / KV pressure
- [ ] Bad few-shot or RAG docs

## 4. Remediate
- [ ] Roll back adapter / prompt version
- [ ] Tighten Critic confidence threshold
- [ ] Re-index knowledge if stale

## 5. Verify & learn
- [ ] Eval suite pass (golden incidents)
- [ ] Cost within budget
- [ ] Update agent runbook

*Template — customize for your org.*
