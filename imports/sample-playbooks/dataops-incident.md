# DataOps incident playbook

**Severity cues:** pipeline SLA miss, schema break, duplicate/missing partitions, warehouse cost spike.

## 1. Detect
- [ ] Freshness / completeness SLO alerts
- [ ] Downstream consumer errors
- [ ] Cost anomaly on compute/storage

## 2. Contain
- [ ] Pause bad producers
- [ ] Quarantine bad partitions
- [ ] Freeze schema migrations

## 3. Diagnose
- [ ] Last good watermark / checkpoint
- [ ] Upstream schema / contract change
- [ ] Idempotency failure

## 4. Remediate
- [ ] Replay from last good offset
- [ ] Backfill plan with cost estimate
- [ ] Contract tests before unpause

## 5. Verify & learn
- [ ] SLOs green for 2 windows
- [ ] Document lineage gap
- [ ] Add data quality gate

*Template — customize for your org.*
