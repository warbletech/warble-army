package remediate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// demo manifests from the shrikeops-demo submodule (warble-army/shrikeops-demo).
func demo(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "shrikeops-demo", "k8s", rel))
	if err != nil {
		t.Skipf("shrikeops-demo submodule not checked out: %v", err)
	}
	return string(b)
}

func TestProposeNonRoot(t *testing.T) {
	p, err := Propose("Deployment", "api", "payments", "runAsNonRoot")
	if err != nil {
		t.Fatal(err)
	}
	if p.PatchType == "" || p.Patch == "" {
		t.Fatalf("empty proposal: %+v", p)
	}
	if !containsAny(p.Patch, "runAsNonRoot") {
		t.Fatalf("patch missing runAsNonRoot: %s", p.Patch)
	}
}

// Issue #9: no proposal may contain PLACEHOLDER.
func TestNoPlaceholderEver(t *testing.T) {
	kinds := []string{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"}
	issues := []string{"runAsNonRoot", "missing-resource-limits", "privileged-container", "readOnlyRootFilesystem", "something-else"}
	for _, k := range kinds {
		for _, is := range issues {
			for _, c := range []string{"", "app"} {
				p, err := ProposeFor(Request{Kind: k, Name: "x", Namespace: "ns", Issue: is, Container: c})
				if err != nil {
					t.Fatalf("%s/%s: %v", k, is, err)
				}
				if strings.Contains(p.Patch, "PLACEHOLDER") || strings.Contains(p.Patch, "<CONTAINER") {
					t.Fatalf("%s/%s/%q: placeholder in patch:\n%s", k, is, c, p.Patch)
				}
			}
		}
	}
}

func TestProposeWithManifest(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		kind, res  string
		ns         string
		issue      string
		containers []string
		apiVersion string
		path       []string // path to the pod spec in the patch
		wantKey    string   // key expected in each container entry
	}{
		{"pod privileged", "bad/privileged-pod.yaml", "Pod", "privileged-debug", "default", "privileged-container",
			[]string{"debug"}, "v1", []string{"spec"}, "securityContext"},
		{"deployment limits, two containers", "bad/no-limits.yaml", "Deployment", "resource-hog", "default", "missing-resource-limits",
			[]string{"worker", "sidecar"}, "apps/v1", []string{"spec", "template", "spec"}, "resources"},
		{"deployment root", "bad/root-container.yaml", "Deployment", "insecure-api", "default", "runAsRootAllowed",
			[]string{"api"}, "apps/v1", []string{"spec", "template", "spec"}, "securityContext"},
		{"good deployment read-only root", "good/deployment.yaml", "deployment", "api-server", "production", "readOnlyRootFilesystem",
			[]string{"api"}, "apps/v1", []string{"spec", "template", "spec"}, "securityContext"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ProposeFor(Request{Kind: tc.kind, Name: tc.res, Namespace: tc.ns, Issue: tc.issue, Manifest: demo(t, tc.file)})
			if err != nil {
				t.Fatal(err)
			}
			if p.PatchType != "strategic-merge" || len(p.MissingInputs) != 0 {
				t.Fatalf("want strategic-merge with no missing inputs, got %s %v", p.PatchType, p.MissingInputs)
			}
			if strings.Join(p.Containers, ",") != strings.Join(tc.containers, ",") {
				t.Fatalf("containers = %v, want %v", p.Containers, tc.containers)
			}
			var doc map[string]any
			if err := yaml.Unmarshal([]byte(p.Patch), &doc); err != nil {
				t.Fatalf("patch is not YAML: %v\n%s", err, p.Patch)
			}
			if doc["apiVersion"] != tc.apiVersion {
				t.Errorf("apiVersion = %v, want %s", doc["apiVersion"], tc.apiVersion)
			}
			var cur any = doc
			for _, k := range tc.path {
				cur = cur.(map[string]any)[k]
			}
			list := cur.(map[string]any)["containers"].([]any)
			for i, c := range list {
				cm := c.(map[string]any)
				if cm["name"] != tc.containers[i] || cm[tc.wantKey] == nil {
					t.Errorf("container %d = %v", i, cm)
				}
			}
			if !strings.Contains(p.Kubectl, "--dry-run=server") || !strings.Contains(p.Kubectl, "--type strategic") {
				t.Errorf("kubectl hint = %q", p.Kubectl)
			}
		})
	}
}

func TestProposeContainerSources(t *testing.T) {
	// explicit container wins
	p, _ := ProposeFor(Request{Kind: "Deployment", Name: "d", Issue: "privileged", Container: "web",
		FindingResource: "Deployment/ns/d/container:other"})
	if strings.Join(p.Containers, ",") != "web" {
		t.Fatalf("containers = %v", p.Containers)
	}
	// ShrikeOps finding resource
	p, _ = ProposeFor(Request{Kind: "Pod", Name: "privileged-debug", Issue: "runAsPrivileged",
		FindingResource: "Pod/default/privileged-debug/container:debug"})
	if strings.Join(p.Containers, ",") != "debug" || p.PatchType != "strategic-merge" {
		t.Fatalf("got %v %s", p.Containers, p.PatchType)
	}
}

func TestProposeWithoutContainerIsJSONPatch(t *testing.T) {
	tests := []struct {
		kind, issue string
		wantPaths   []string
	}{
		{"Pod", "privileged", []string{"/spec/containers/0/securityContext"}},
		{"Deployment", "missing-resource-limits", []string{"/spec/template/spec/containers/0/resources"}},
		{"CronJob", "runAsNonRoot", []string{"/spec/jobTemplate/spec/template/spec/securityContext", "/spec/jobTemplate/spec/template/spec/containers/0/securityContext"}},
	}
	for _, tc := range tests {
		t.Run(tc.kind+"/"+tc.issue, func(t *testing.T) {
			p, err := ProposeFor(Request{Kind: tc.kind, Name: "x", Namespace: "ns", Issue: tc.issue})
			if err != nil {
				t.Fatal(err)
			}
			if p.PatchType != "json" || len(p.MissingInputs) == 0 || len(p.Notes) == 0 {
				t.Fatalf("want json patch with missing_inputs+notes, got %+v", p)
			}
			var ops []struct{ Op, Path string }
			if err := json.Unmarshal([]byte(p.Patch), &ops); err != nil {
				t.Fatalf("not a JSON patch: %v\n%s", err, p.Patch)
			}
			var got []string
			for _, o := range ops {
				if o.Op != "add" {
					t.Errorf("op = %s", o.Op)
				}
				got = append(got, o.Path)
			}
			if strings.Join(got, " ") != strings.Join(tc.wantPaths, " ") {
				t.Errorf("paths = %v, want %v", got, tc.wantPaths)
			}
		})
	}
}

func TestProposeManifestWithoutMatch(t *testing.T) {
	p, err := ProposeFor(Request{Kind: "Deployment", Name: "nope", Issue: "limits", Manifest: demo(t, "bad/no-limits.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if p.PatchType != "json" || !strings.Contains(strings.Join(p.Notes, " "), "no Deployment named") {
		t.Fatalf("want json fallback with note, got %+v", p)
	}
}

func TestProposeErrors(t *testing.T) {
	if _, err := ProposeFor(Request{Kind: "Ingress", Name: "x", Issue: "privileged"}); err == nil {
		t.Error("expected unsupported kind error")
	}
	if _, err := ProposeFor(Request{Kind: "Pod", Issue: "privileged"}); err == nil {
		t.Error("expected missing name error")
	}
	if _, err := ProposeFor(Request{Kind: "Pod", Name: "x", Issue: "privileged", Manifest: "a: [1"}); err == nil {
		t.Error("expected manifest parse error")
	}
}

func TestIssueMatching(t *testing.T) {
	tests := map[string]string{
		"notReadOnlyRootFilesystem":  "readOnlyRootFilesystem",
		"writable-root-filesystem":   "readOnlyRootFilesystem",
		"runAsRootAllowed":           "runAsNonRoot",
		"privilegeEscalationAllowed": "allowPrivilegeEscalation",
		"cpuLimitsMissing":           "limits",
	}
	for issue, want := range tests {
		p, err := ProposeFor(Request{Kind: "Deployment", Name: "d", Issue: issue, Container: "c"})
		if err != nil || !strings.Contains(p.Patch, want) {
			t.Errorf("%s: want %q in patch, got %v\n%s", issue, want, err, p.Patch)
		}
	}
}

func TestApplyGate(t *testing.T) {
	if msg := ApplyGate("readonly", true, false, "ns", nil); msg == "" {
		t.Fatal("expected readonly deny")
	}
	if msg := ApplyGate("readwrite", false, false, "ns", nil); msg == "" {
		t.Fatal("expected confirm required")
	}
	if msg := ApplyGate("readwrite", true, false, "ns", nil); msg != "" {
		t.Fatalf("dry-run should pass: %s", msg)
	}
	allow := func(ns string) bool { return ns == "ok" }
	if msg := ApplyGate("readwrite", false, true, "bad", allow); msg == "" {
		t.Fatal("expected allowlist deny")
	}
}
