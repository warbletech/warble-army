# Security policy

This is the short version for the Warble Flock (`warble-army`) repository. The full policy,
with supported versions and scope notes, is in
[docs/cncf/SECURITY.md](docs/cncf/SECURITY.md).

## Reporting a vulnerability

Please do **not** report security vulnerabilities in public GitHub issues,
discussions or pull requests.

Report privately in either of these ways:

- GitHub private vulnerability reporting:
  <https://github.com/warbletech/warble-army/security/advisories/new>
- Email: [security@warblecloud.com](mailto:security@warblecloud.com)

Include the affected component and version or commit, a description of the
vulnerability and its impact, reproduction steps or a proof of concept (if
safe), and whether you are available for follow-up.

## What to expect

- We acknowledge a report within **3 business days**.
- We send a status update with our assessment within **10 business days**.
- We aim to release a fix within **90 days** of the report, sooner for
  critical issues. If a fix needs more time, we agree a timeline with you.

## Disclosure

We follow coordinated disclosure. Once a fix is available we publish a GitHub
security advisory (and request a CVE where appropriate), and credit the
reporter unless they ask to stay anonymous. Please give us the time above
before disclosing the vulnerability publicly.

## Scope

In scope: code in this repository and its submodules (Starling Lite, Heron
Ops, ShrikeOps), the Helm charts in `charts/`, and the container images built
from `deploy/`. Vulnerabilities in third-party scanners (Pluto, Polaris,
kube-score) should also be reported upstream; we help coordinate.
