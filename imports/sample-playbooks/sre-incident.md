# SRE reliability incident playbook

**Severity cues:** multi-service 5xx, error budget burn, customer-impacting latency, cascading timeouts.

## 1. Detect
- [ ] SLO burn rate (fast/slow windows)
- [ ] Synthetic + real-user signals
- [ ] Dependency map (upstream/downstream)

## 2. Contain
- [ ] Feature flags / traffic shed
- [ ] Failover / degrade gracefully
- [ ] Comms: status page draft

## 3. Diagnose
- [ ] Correlation IDs across services
- [ ] Recent deploy window
- [ ] Resource vs logic fault

## 4. Remediate
- [ ] Roll back or fix-forward with canary
- [ ] Warble Critic gate for high-blast actions
- [ ] Keep incident commander role clear

## 5. Verify & learn
- [ ] SLOs recovered N consecutive windows
- [ ] Blameless postmortem
- [ ] Action items tracked

*Template — customize for your org.*
