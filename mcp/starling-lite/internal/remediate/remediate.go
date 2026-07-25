package remediate

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Proposal is a dry-run remediation suggestion.
type Proposal struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Issue     string `json:"issue"`
	PatchType string `json:"patch_type"` // strategic-merge | json | yaml-snippet
	Patch     string `json:"patch"`
	Rationale string `json:"rationale"`
	DryRun    bool   `json:"dry_run"`
}

// Propose builds a patch for known issue ids / rule names.
func Propose(kind, name, namespace, issue string) (*Proposal, error) {
	issueKey := strings.ToLower(strings.TrimSpace(issue))
	issueKey = strings.ReplaceAll(issueKey, "_", "-")

	p := &Proposal{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
		Issue:     issue,
		DryRun:    true,
	}

	switch {
	case containsAny(issueKey, "runasnonroot", "run-as-non-root", "root"):
		p.PatchType = "strategic-merge"
		p.Rationale = "Force non-root user and runAsNonRoot"
		p.Patch = fmt.Sprintf(`apiVersion: apps/v1
kind: %s
metadata:
  name: %s
  namespace: %s
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
      containers:
      - name: PLACEHOLDER
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          allowPrivilegeEscalation: false
`, kind, name, namespace)

	case containsAny(issueKey, "resource-limit", "missing-resource", "resources", "limits"):
		p.PatchType = "strategic-merge"
		p.Rationale = "Add conservative CPU/memory requests and limits"
		p.Patch = fmt.Sprintf(`apiVersion: apps/v1
kind: %s
metadata:
  name: %s
  namespace: %s
spec:
  template:
    spec:
      containers:
      - name: PLACEHOLDER
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 500m
            memory: 256Mi
`, kind, name, namespace)

	case containsAny(issueKey, "privileged"):
		p.PatchType = "strategic-merge"
		p.Rationale = "Disable privileged and privilege escalation"
		p.Patch = fmt.Sprintf(`apiVersion: apps/v1
kind: %s
metadata:
  name: %s
  namespace: %s
spec:
  template:
    spec:
      containers:
      - name: PLACEHOLDER
        securityContext:
          privileged: false
          allowPrivilegeEscalation: false
`, kind, name, namespace)

	case containsAny(issueKey, "readonlyroot", "read-only-root", "readonlyfilesystem"):
		p.PatchType = "strategic-merge"
		p.Rationale = "Mount root filesystem read-only"
		p.Patch = fmt.Sprintf(`apiVersion: apps/v1
kind: %s
metadata:
  name: %s
  namespace: %s
spec:
  template:
    spec:
      containers:
      - name: PLACEHOLDER
        securityContext:
          readOnlyRootFilesystem: true
`, kind, name, namespace)

	default:
		p.PatchType = "yaml-snippet"
		p.Rationale = "Generic remediation hint — refine with ShrikeOps finding.remediation"
		p.Patch = fmt.Sprintf(`# issue: %s
# resource: %s/%s (ns=%s)
# Replace PLACEHOLDER container name and apply via kubectl patch --dry-run=server
`, issue, kind, name, namespace)
	}

	return p, nil
}

// FormatJSON serializes a proposal.
func FormatJSON(p *Proposal) string {
	b, _ := json.MarshalIndent(p, "", "  ")
	return string(b)
}

// ApplyGate validates apply_remediation guards. Returns error message or "".
func ApplyGate(mode string, dryRun, confirm bool, ns string, allow func(string) bool) string {
	if mode != "readwrite" {
		return "apply_remediation requires --mode=readwrite"
	}
	if !dryRun && !confirm {
		return "set confirm=true to apply when dry_run=false (error -32004)"
	}
	if ns != "" && allow != nil && !allow(ns) {
		return fmt.Sprintf("namespace %q not in --allow-namespaces", ns)
	}
	return ""
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
