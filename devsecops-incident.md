# DevSecOps incident playbook

**Severity cues:** secret leak, CVE in prod image, admission webhook down, privileged pod, RBAC escalation.

## 1. Detect
- [ ] Alert source (Falco / OPA / scanner / CVE feed)
- [ ] Affected namespaces, images, service accounts
- [ ] Data classification impact (PII / secrets)

## 2. Contain
- [ ] NetworkPolicy / deny egress if exfil risk
- [ ] Revoke tokens / rotate secrets
- [ ] Block image tag / pin digest

## 3. Diagnose
- [ ] Root cause: misconfig vs dependency vs intentional break-glass
- [ ] Blast radius (other clusters / tenants)

## 4. Remediate (Warble Actor → Critic)
- [ ] Propose patch (Rego / Helm / image)
- [ ] Critic: policy + blast-radius gate
- [ ] Executor only after approve (or supervised mode)

## 5. Verify & learn
- [ ] Rescan green
- [ ] Postmortem + runbook update
- [ ] Add regression test to CI

*Template — customize for your org. Not legal advice.*
