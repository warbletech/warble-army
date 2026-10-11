package remediate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
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
	// Containers the patch targets (resolved from container / finding / manifest).
	Containers []string `json:"containers,omitempty"`
	// MissingInputs lists inputs that would make the patch precise.
	MissingInputs []string `json:"missing_inputs,omitempty"`
	Notes         []string `json:"notes,omitempty"`
	// Kubectl is a suggested server-side dry-run command for the patch.
	Kubectl string `json:"kubectl,omitempty"`
}

// Request is the full input to ProposeFor.
type Request struct {
	Kind      string
	Name      string
	Namespace string
	Issue     string
	// Container name to patch (optional).
	Container string
	// Manifest is the resource YAML (optional, multi-doc ok); used to find container names.
	Manifest string
	// FindingResource is a ShrikeOps finding "resource" (optional); a
	// ".../container:<name>" suffix identifies the container.
	FindingResource string
}

// Propose builds a patch for known issue ids / rule names (no container info).
func Propose(kind, name, namespace, issue string) (*Proposal, error) {
	return ProposeFor(Request{Kind: kind, Name: name, Namespace: namespace, Issue: issue})
}

// fix describes one remediation: container-level and optional pod-level fields.
type fix struct {
	rationale string
	container map[string]any // merged into each container
	pod       map[string]any // merged into the pod spec
}

func fixFor(issueKey string) (fix, bool) {
	// Order matters: "read-only-root" / "writable-root" must not match the run-as-root case.
	switch {
	case containsAny(issueKey, "readonlyroot", "read-only-root", "readonlyfilesystem", "writable-root"):
		return fix{
			rationale: "Mount root filesystem read-only",
			container: map[string]any{"securityContext": map[string]any{"readOnlyRootFilesystem": true}},
		}, true
	case containsAny(issueKey, "privileged", "privilege-escalation", "privilegeescalation"):
		return fix{
			rationale: "Disable privileged and privilege escalation",
			container: map[string]any{"securityContext": map[string]any{"privileged": false, "allowPrivilegeEscalation": false}},
		}, true
	case containsAny(issueKey, "runasnonroot", "run-as-non-root", "root"):
		return fix{
			rationale: "Force non-root user and runAsNonRoot",
			pod:       map[string]any{"securityContext": map[string]any{"runAsNonRoot": true, "runAsUser": 1000}},
			container: map[string]any{"securityContext": map[string]any{"runAsNonRoot": true, "runAsUser": 1000, "allowPrivilegeEscalation": false}},
		}, true
	case containsAny(issueKey, "resource-limit", "missing-resource", "resources", "limits"):
		return fix{
			rationale: "Add conservative CPU/memory requests and limits",
			container: map[string]any{"resources": map[string]any{
				"requests": map[string]any{"cpu": "50m", "memory": "64Mi"},
				"limits":   map[string]any{"cpu": "500m", "memory": "256Mi"},
			}},
		}, true
	}
	return fix{}, false
}

// ProposeFor builds a dry-run patch. When the container name(s) are known it
// returns a strategic-merge patch naming them; otherwise an RFC 6902 JSON
// patch against the first container (no name needed) and lists "container" /
// "manifest" in MissingInputs.
func ProposeFor(r Request) (*Proposal, error) {
	if strings.TrimSpace(r.Kind) == "" || strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.Issue) == "" {
		return nil, fmt.Errorf("kind, name and issue are required")
	}
	issueKey := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(r.Issue)), "_", "-")
	shape, ok := shapeFor(r.Kind)
	if !ok {
		return nil, fmt.Errorf("unsupported kind %q: expected one of Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet, ReplicationController, Job, CronJob", r.Kind)
	}

	p := &Proposal{Kind: shape.kind, Name: r.Name, Namespace: r.Namespace, Issue: r.Issue, DryRun: true}

	f, known := fixFor(issueKey)
	if !known {
		p.PatchType = "yaml-snippet"
		p.Rationale = "Generic remediation hint — refine with the ShrikeOps finding.remediation"
		p.Patch = fmt.Sprintf("# issue: %s\n# resource: %s/%s (ns=%s)\n# No built-in patch for this issue; edit the %s pod spec at %s\n",
			r.Issue, shape.kind, r.Name, r.Namespace, shape.kind, "."+strings.Join(shape.podSpecPath, "."))
		return p, nil
	}
	p.Rationale = f.rationale
	if shape.kind == "Pod" {
		p.Notes = append(p.Notes, "Most Pod spec fields (securityContext, resources) are immutable on a running Pod: apply this to the manifest or owning controller and recreate the Pod; kubectl patch on a live Pod will be rejected.")
	}

	containers, notes, err := resolveContainers(r, shape)
	if err != nil {
		return nil, err
	}
	p.Notes = append(p.Notes, notes...)

	if len(containers) > 0 {
		p.Containers = containers
		p.PatchType = "strategic-merge"
		p.Patch = strategicPatch(shape, r, f, containers)
		p.Kubectl = kubectlCmd(shape, r, "strategic")
		return p, nil
	}

	p.PatchType = "json"
	p.Patch = jsonPatch(shape, f)
	p.MissingInputs = []string{"container", "manifest"}
	p.Notes = append(p.Notes,
		"Container name unknown: this JSON patch targets the FIRST container (index 0). Pass `container`, the resource `manifest`, or the ShrikeOps `finding` to get a strategic-merge patch for named containers.",
		"JSON patch 'add' replaces the whole object at the path (e.g. an existing securityContext) — review before applying.")
	p.Kubectl = kubectlCmd(shape, r, "json")
	return p, nil
}

// kindShape describes where the pod spec lives for a workload kind.
type kindShape struct {
	kind        string
	apiVersion  string
	podSpecPath []string
}

func shapeFor(kind string) (kindShape, bool) {
	tmpl := []string{"spec", "template", "spec"}
	switch strings.ToLower(kind) {
	case "pod":
		return kindShape{"Pod", "v1", []string{"spec"}}, true
	case "deployment":
		return kindShape{"Deployment", "apps/v1", tmpl}, true
	case "statefulset":
		return kindShape{"StatefulSet", "apps/v1", tmpl}, true
	case "daemonset":
		return kindShape{"DaemonSet", "apps/v1", tmpl}, true
	case "replicaset":
		return kindShape{"ReplicaSet", "apps/v1", tmpl}, true
	case "replicationcontroller":
		return kindShape{"ReplicationController", "v1", tmpl}, true
	case "job":
		return kindShape{"Job", "batch/v1", tmpl}, true
	case "cronjob":
		return kindShape{"CronJob", "batch/v1", []string{"spec", "jobTemplate", "spec", "template", "spec"}}, true
	}
	return kindShape{}, false
}

// resolveContainers finds container names from (in order) Container,
// FindingResource and Manifest.
func resolveContainers(r Request, shape kindShape) ([]string, []string, error) {
	if c := strings.TrimSpace(r.Container); c != "" {
		return []string{c}, nil, nil
	}
	if i := strings.LastIndex(r.FindingResource, "container:"); i >= 0 {
		if c := strings.TrimSpace(r.FindingResource[i+len("container:"):]); c != "" {
			return []string{c}, []string{"container taken from finding.resource"}, nil
		}
	}
	if strings.TrimSpace(r.Manifest) == "" {
		return nil, nil, nil
	}
	names, found, err := containersFromManifest(r.Manifest, shape, r.Name, r.Namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("manifest: %w", err)
	}
	if !found {
		return nil, []string{fmt.Sprintf("manifest has no %s named %q; falling back to a first-container JSON patch", shape.kind, r.Name)}, nil
	}
	return names, []string{"containers taken from manifest"}, nil
}

func containersFromManifest(manifest string, shape kindShape, name, namespace string) ([]string, bool, error) {
	dec := yaml.NewDecoder(strings.NewReader(manifest))
	for {
		var doc map[string]any
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, false, nil
			}
			return nil, false, err
		}
		if doc == nil || !strings.EqualFold(fmt.Sprint(doc["kind"]), shape.kind) {
			continue
		}
		meta, _ := doc["metadata"].(map[string]any)
		if meta == nil || fmt.Sprint(meta["name"]) != name {
			continue
		}
		if ns, ok := meta["namespace"].(string); ok && namespace != "" && ns != namespace {
			continue
		}
		var cur any = doc
		for _, k := range shape.podSpecPath {
			m, _ := cur.(map[string]any)
			cur = m[k]
		}
		spec, _ := cur.(map[string]any)
		list, _ := spec["containers"].([]any)
		var names []string
		for _, c := range list {
			if cm, ok := c.(map[string]any); ok {
				if n, ok := cm["name"].(string); ok && n != "" {
					names = append(names, n)
				}
			}
		}
		return names, len(names) > 0, nil
	}
}

func strategicPatch(shape kindShape, r Request, f fix, containers []string) string {
	podSpec := map[string]any{}
	for k, v := range f.pod {
		podSpec[k] = v
	}
	var list []any
	for _, c := range containers {
		entry := map[string]any{"name": c}
		for k, v := range f.container {
			entry[k] = v
		}
		list = append(list, entry)
	}
	podSpec["containers"] = list

	// Build nested spec from the pod-spec path, innermost first.
	var body any = podSpec
	for i := len(shape.podSpecPath) - 1; i >= 0; i-- {
		body = map[string]any{shape.podSpecPath[i]: body}
	}
	meta := map[string]any{"name": r.Name}
	if r.Namespace != "" {
		meta["namespace"] = r.Namespace
	}
	out := map[string]any{
		"apiVersion": shape.apiVersion,
		"kind":       shape.kind,
		"metadata":   meta,
	}
	for k, v := range body.(map[string]any) {
		out[k] = v
	}
	return marshalOrdered(out)
}

// marshalOrdered emits apiVersion, kind, metadata, spec in conventional order.
func marshalOrdered(m map[string]any) string {
	var b strings.Builder
	for _, k := range []string{"apiVersion", "kind", "metadata", "spec"} {
		v, ok := m[k]
		if !ok {
			continue
		}
		enc := yaml.NewEncoder(&b)
		enc.SetIndent(2)
		_ = enc.Encode(map[string]any{k: v})
		_ = enc.Close()
	}
	return b.String()
}

func jsonPatch(shape kindShape, f fix) string {
	base := "/" + strings.Join(shape.podSpecPath, "/")
	type op struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value any    `json:"value"`
	}
	var ops []op
	for _, k := range sortedKeys(f.pod) {
		ops = append(ops, op{"add", base + "/" + k, f.pod[k]})
	}
	for _, k := range sortedKeys(f.container) {
		ops = append(ops, op{"add", base + "/containers/0/" + k, f.container[k]})
	}
	b, _ := json.MarshalIndent(ops, "", "  ")
	return string(b)
}

func kubectlCmd(shape kindShape, r Request, typ string) string {
	ns := ""
	if r.Namespace != "" {
		ns = " -n " + r.Namespace
	}
	file := "patch.yaml"
	if typ == "json" {
		file = "patch.json"
	}
	return fmt.Sprintf("kubectl patch %s %s%s --type %s --patch-file %s --dry-run=server -o yaml",
		strings.ToLower(shape.kind), r.Name, ns, typ, file)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
