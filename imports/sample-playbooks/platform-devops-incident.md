# Platform / DevOps incident playbook

**Severity cues:** node NotReady, CNI flake, API server latency, etcd pressure, disk full, DNS failures.

## 1. Detect
- [ ] Cluster health + control-plane metrics
- [ ] Pending pods / scheduling failures
- [ ] Autoscaler / quota limits

## 2. Contain
- [ ] Cordon bad nodes
- [ ] Scale critical Deployments carefully
- [ ] Protect kube-system / data plane

## 3. Diagnose
- [ ] Resource pressure (CPU/mem/disk/ephemeral)
- [ ] Network path (kube-proxy / CNI / LB)
- [ ] Recent deploys / Helm upgrades

## 4. Remediate
- [ ] Drain + replace nodes
- [ ] Roll back bad release
- [ ] Raise pool max only with cost awareness

## 5. Verify & learn
- [ ] Workloads Ready
- [ ] Error budgets recovering
- [ ] Capacity / HPA notes

*Template — customize for your org.*
