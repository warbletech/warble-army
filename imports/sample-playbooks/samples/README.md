# Sample templates

Ready-to-copy artifacts for operators, trainers, and Avirka fine-tunes.

| File | Purpose |
|------|---------|
| [incident.alert.sample.json](./incident.alert.sample.json) | Canonical alert payload |
| [actor.proposal.sample.json](./actor.proposal.sample.json) | ActorProposal (approved path) |
| [critic.verdict.sample.json](./critic.verdict.sample.json) | Critic approve |
| [critic.reject.sample.json](./critic.reject.sample.json) | Critic hard reject |
| [healing.recipe.sample.yaml](./healing.recipe.sample.yaml) | HealingRecipe CR-style |
| [postmortem.sample.md](./postmortem.sample.md) | Blameless postmortem |
| [status-page.sample.md](./status-page.sample.md) | External status updates |
| [devsecops.incident.sample.json](./devsecops.incident.sample.json) | Domain sample |
| [dataops.incident.sample.json](./dataops.incident.sample.json) | Domain sample |
| [aiops.incident.sample.json](./aiops.incident.sample.json) | Domain sample |
| [noops.incident.sample.json](./noops.incident.sample.json) | Domain sample |
| [sharegpt.actor.sample.jsonl](./sharegpt.actor.sample.jsonl) | Training line (Actor) |
| [sharegpt.critic.sample.jsonl](./sharegpt.critic.sample.jsonl) | Training line (Critic) |

Parent playbooks: `../` (DevSecOps → NoOps).  
Train: `warble-brain/data/synthetic/playbooks_to_sharegpt.py`
