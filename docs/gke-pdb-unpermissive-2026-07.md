# GKE DiagnosisRecommender: PDB_UNPERMISSIVE

**Project:** `warble-tech` (`295728087951`)  
**Cluster:** `warble-cluster` (`us-central1-a`)  
**Recommendation ID:** `8cdf4c22-8b9a-4b87-9963-b1ba44e5b3f9`  
**Subtype:** `PDB_UNPERMISSIVE` (P2 reliability / service disruption)  
**Fixed:** 2026-07-25

## Problem

GKE Active Assist flagged PodDisruptionBudgets where `minAvailable` equals the only healthy pod (`disruptionsAllowed: 0`). Node upgrades can block up to ~1h.

| Namespace | PDB | Before |
|-----------|-----|--------|
| `kourier-system` | `3scale-kourier-gateway-pdb` | `minAvailable: 80%` with 1 pod |
| `knative-serving` | `webhook-pdb` | `minAvailable: 1` with 1 pod |
| `knative-serving` | `activator-pdb` | `minAvailable: 1` with 1 pod |
| `cnpg-system` | `postgres-cluster-primary` | `minAvailable: 1` with 1 primary (operator-owned) |

## Fix applied

### Knative + Kourier

1. Scaled deployments to **2 replicas**: `activator`, `webhook`, `3scale-kourier-gateway`.
2. Patched PDBs: removed `minAvailable`, set **`maxUnavailable: 1`**.

Result: `disruptionsAllowed: 1` while 2 pods healthy — GKE drains can proceed.

### CloudNativePG (`postgres-cluster`)

1. Scaled `spec.instances` **1 → 2** (HA replica; operator-managed).
2. Left primary PDB at `minAvailable: 1` — **intentional** for single-primary role (CNPG owns the PDB via ownerReference). With 2 instances, failover can complete during voluntary disruption.

If Active Assist re-flags only the primary PDB after refresh, options are:

- Keep HA (recommended) and dismiss/mark succeeded again, or
- `spec.enablePDB: false` on the Cluster (not recommended for multi-instance prod).

## Verify

```bash
export PATH="$HOME/google-cloud-sdk/bin:$HOME/.local/bin:$PATH"
export CLOUDSDK_CORE_ACCOUNT=contact@warblecloud.com
gcloud container clusters get-credentials warble-cluster --zone us-central1-a --project warble-tech

kubectl get pdb -A -o custom-columns=\
'NS:.metadata.namespace,NAME:.metadata.name,MIN:.spec.minAvailable,MAXU:.spec.maxUnavailable,ALLOWED:.status.disruptionsAllowed,HEALTHY:.status.currentHealthy'

kubectl get clusters.postgresql.cnpg.io postgres-cluster -n cnpg-system
```

## Note: related but not in this recommendation

- `istio-system/istio-ingressgateway` PDB: `ALLOWED=0` with **0 healthy** pods — separate availability issue.
- `warblecloud-ai/reflexion-engine` PDB: single replica + `minAvailable: 1` — scale or relax if upgrades block.
