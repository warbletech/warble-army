# NoOps autonomous healing playbook

**Severity cues:** autopilot executing without human; need bounds for Actor→Critic→Executor.

## 1. Detect
- [ ] Incident opened by Observer
- [ ] Autonomy level (observe / suggest / supervised / autopilot)
- [ ] Policy pack version

## 2. Contain (always)
- [ ] Max actions per incident
- [ ] Namespace / cluster allowlist
- [ ] Kill-switch: set autonomy → observe

## 3. Diagnose (agents)
- [ ] Actor hypotheses + confidence
- [ ] Critic objections (SLO, blast radius)
- [ ] Similar past incidents (learning)

## 4. Remediate bounds
- [ ] Autopilot only for pre-approved action classes
- [ ] Secrets / IAM changes always supervised
- [ ] Dry-run first when uncertain

## 5. Verify & learn
- [ ] Healing action status audited
- [ ] Human review sample of autopilot runs
- [ ] Promote successful remediations to recipes

*Template — customize for your org. Autonomy is a product decision, not a default.*
