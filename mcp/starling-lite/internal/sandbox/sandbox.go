// Package sandbox renders an isolated, least-privilege Kubernetes namespace
// ("project sandbox") in which agent / remediation tasks for one target
// project can run. It only renders manifests (dry-run); it never mutates a
// cluster. See docs/sandbox.md.
package sandbox

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec describes one project sandbox.
type Spec struct {
	Project string // target project slug (DNS-1123 label)
	// Namespace defaults to "sbx-<project>".
	Namespace string
	// EgressCIDRs are extra allowed egress destinations (DNS is always allowed).
	EgressCIDRs []string
	// Quota (defaults applied when empty).
	CPU, Memory string
	Pods        int
	// Optional task: when Image is set a Job is rendered that runs in the sandbox.
	Image   string
	Command []string
	// TTLSeconds after a finished Job (default 600).
	TTLSeconds int
	// ActiveDeadlineSeconds caps the Job runtime (default 900).
	DeadlineSeconds int
}

var dns1123 = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// Defaults fills unset fields.
func (s *Spec) Defaults() {
	if s.Namespace == "" {
		s.Namespace = "sbx-" + s.Project
	}
	if s.CPU == "" {
		s.CPU = "2"
	}
	if s.Memory == "" {
		s.Memory = "2Gi"
	}
	if s.Pods == 0 {
		s.Pods = 5
	}
	if s.TTLSeconds == 0 {
		s.TTLSeconds = 600
	}
	if s.DeadlineSeconds == 0 {
		s.DeadlineSeconds = 900
	}
}

// Validate checks the spec after Defaults.
func (s Spec) Validate() error {
	if s.Project == "" {
		return errors.New("project is required")
	}
	if !dns1123.MatchString(s.Project) || len(s.Namespace) > 63 || !dns1123.MatchString(s.Namespace) {
		return fmt.Errorf("project/namespace must be a DNS-1123 label (got %q / %q)", s.Project, s.Namespace)
	}
	for _, c := range s.EgressCIDRs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return fmt.Errorf("invalid egress CIDR %q", c)
		}
		if ones, _ := n.Mask.Size(); ones == 0 {
			return fmt.Errorf("egress CIDR %q opens all egress; not allowed in a sandbox", c)
		}
	}
	if s.Pods < 1 || s.Pods > 50 {
		return errors.New("pods quota must be 1..50")
	}
	if s.Image != "" && strings.HasSuffix(s.Image, ":latest") {
		return errors.New("sandbox image must be pinned (tag or digest), not :latest")
	}
	return nil
}

type obj = map[string]any

func meta(name, ns string, labels obj) obj {
	m := obj{"name": name, "labels": labels}
	if ns != "" {
		m["namespace"] = ns
	}
	return m
}

// Objects returns the sandbox manifests in apply order.
func Objects(s Spec) ([]obj, error) {
	s.Defaults()
	if err := s.Validate(); err != nil {
		return nil, err
	}
	labels := obj{
		"app.kubernetes.io/managed-by": "starling-lite",
		"warbleoss.org/sandbox":        "true",
		"warbleoss.org/project":        s.Project,
	}
	nsLabels := obj{
		"pod-security.kubernetes.io/enforce":         "restricted",
		"pod-security.kubernetes.io/enforce-version": "latest",
		"pod-security.kubernetes.io/audit":           "restricted",
		"pod-security.kubernetes.io/warn":            "restricted",
	}
	for k, v := range labels {
		nsLabels[k] = v
	}
	ns := s.Namespace
	out := []obj{
		{"apiVersion": "v1", "kind": "Namespace", "metadata": meta(ns, "", nsLabels)},
		{"apiVersion": "v1", "kind": "ServiceAccount", "metadata": meta("sandbox-runner", ns, labels),
			"automountServiceAccountToken": false},
		{"apiVersion": "v1", "kind": "ResourceQuota", "metadata": meta("sandbox-quota", ns, labels),
			"spec": obj{"hard": obj{
				"requests.cpu": s.CPU, "limits.cpu": s.CPU,
				"requests.memory": s.Memory, "limits.memory": s.Memory,
				"pods": fmt.Sprint(s.Pods), "services": "0", "services.loadbalancers": "0",
				"services.nodeports": "0", "persistentvolumeclaims": "0", "secrets": "5",
			}}},
		{"apiVersion": "v1", "kind": "LimitRange", "metadata": meta("sandbox-limits", ns, labels),
			"spec": obj{"limits": []any{obj{"type": "Container",
				"default":        obj{"cpu": "500m", "memory": "512Mi"},
				"defaultRequest": obj{"cpu": "100m", "memory": "128Mi"},
				"max":            obj{"cpu": s.CPU, "memory": s.Memory},
			}}}},
		{"apiVersion": "networking.k8s.io/v1", "kind": "NetworkPolicy", "metadata": meta("default-deny-all", ns, labels),
			"spec": obj{"podSelector": obj{}, "policyTypes": []any{"Ingress", "Egress"}}},
		{"apiVersion": "networking.k8s.io/v1", "kind": "NetworkPolicy", "metadata": meta("allow-dns", ns, labels),
			"spec": obj{"podSelector": obj{}, "policyTypes": []any{"Egress"},
				"egress": []any{obj{
					"to": []any{obj{"namespaceSelector": obj{"matchLabels": obj{"kubernetes.io/metadata.name": "kube-system"}},
						"podSelector": obj{"matchLabels": obj{"k8s-app": "kube-dns"}}}},
					"ports": []any{obj{"protocol": "UDP", "port": 53}, obj{"protocol": "TCP", "port": 53}},
				}}}},
	}
	if len(s.EgressCIDRs) > 0 {
		var to []any
		for _, c := range s.EgressCIDRs {
			to = append(to, obj{"ipBlock": obj{"cidr": c}})
		}
		out = append(out, obj{"apiVersion": "networking.k8s.io/v1", "kind": "NetworkPolicy",
			"metadata": meta("allow-egress-cidrs", ns, labels),
			"spec":     obj{"podSelector": obj{}, "policyTypes": []any{"Egress"}, "egress": []any{obj{"to": to}}}})
	}
	if s.Image != "" {
		out = append(out, job(s, labels))
	}
	return out, nil
}

// PodSecurity returns the hardened pod/container security contexts used by sandbox Jobs.
func PodSecurity() (pod, container obj) {
	pod = obj{"runAsNonRoot": true, "runAsUser": 65532, "runAsGroup": 65532, "fsGroup": 65532,
		"seccompProfile": obj{"type": "RuntimeDefault"}}
	container = obj{"allowPrivilegeEscalation": false, "readOnlyRootFilesystem": true, "privileged": false,
		"runAsNonRoot": true, "capabilities": obj{"drop": []any{"ALL"}},
		"seccompProfile": obj{"type": "RuntimeDefault"}}
	return
}

func job(s Spec, labels obj) obj {
	psc, csc := PodSecurity()
	c := obj{"name": "task", "image": s.Image, "securityContext": csc,
		"resources":    obj{"requests": obj{"cpu": "100m", "memory": "128Mi"}, "limits": obj{"cpu": "1", "memory": "1Gi"}},
		"volumeMounts": []any{obj{"name": "tmp", "mountPath": "/tmp"}, obj{"name": "work", "mountPath": "/work"}},
		"workingDir":   "/work"}
	if len(s.Command) > 0 {
		cmd := make([]any, len(s.Command))
		for i, v := range s.Command {
			cmd[i] = v
		}
		c["command"] = cmd
	}
	return obj{"apiVersion": "batch/v1", "kind": "Job", "metadata": meta("sandbox-task", s.Namespace, labels),
		"spec": obj{"backoffLimit": 0, "ttlSecondsAfterFinished": s.TTLSeconds,
			"activeDeadlineSeconds": s.DeadlineSeconds,
			"template": obj{"metadata": obj{"labels": labels}, "spec": obj{
				"serviceAccountName": "sandbox-runner", "automountServiceAccountToken": false,
				"enableServiceLinks": false, "hostNetwork": false, "hostPID": false, "hostIPC": false,
				"restartPolicy": "Never", "securityContext": psc, "containers": []any{c},
				"volumes": []any{obj{"name": "tmp", "emptyDir": obj{"sizeLimit": "256Mi"}},
					obj{"name": "work", "emptyDir": obj{"sizeLimit": "1Gi"}}},
			}}}}
}

// Render returns the multi-document YAML for the sandbox.
func Render(s Spec) (string, error) {
	objs, err := Objects(s)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i, o := range objs {
		if i > 0 {
			b.WriteString("---\n")
		}
		y, err := yaml.Marshal(o)
		if err != nil {
			return "", err
		}
		b.Write(y)
	}
	return b.String(), nil
}
