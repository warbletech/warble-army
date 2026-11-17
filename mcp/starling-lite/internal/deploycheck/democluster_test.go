package deploycheck

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

type rule struct {
	APIGroups []string `yaml:"apiGroups"`
	Resources []string `yaml:"resources"`
	Verbs     []string `yaml:"verbs"`
}

type object struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Rules   []rule `yaml:"rules"`
	RoleRef struct {
		Kind string `yaml:"kind"`
		Name string `yaml:"name"`
	} `yaml:"roleRef"`
}

func loadRBAC(t *testing.T) []object {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "deploy", "demo-cluster", "rbac.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var out []object
	dec := yaml.NewDecoder(bytes.NewReader(b))
	for {
		var o object
		if err := dec.Decode(&o); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if o.Kind != "" {
			out = append(out, o)
		}
	}
	return out
}

// The public demo's identity may only read, never secrets, never wildcards;
// ConfigMap values and pod logs only in the demo namespaces.
func TestDemoClusterRBACIsReadOnly(t *testing.T) {
	objs := loadRBAC(t)
	readVerbs := []string{"get", "list", "watch"}
	contentOnly := []string{"configmaps", "pods/log"}
	roles := map[string]object{}
	for _, o := range objs {
		switch o.Kind {
		case "Role":
			t.Errorf("use ClusterRoles bound per namespace, found Role %s", o.Metadata.Name)
		case "ClusterRole":
			roles[o.Metadata.Name] = o
			for _, r := range o.Rules {
				for _, v := range r.Verbs {
					if !slices.Contains(readVerbs, v) {
						t.Errorf("%s: verb %q is not read-only", o.Metadata.Name, v)
					}
				}
				for _, res := range r.Resources {
					if res == "*" || res == "secrets" || res == "serviceaccounts/token" {
						t.Errorf("%s: resource %q must not be readable", o.Metadata.Name, res)
					}
				}
				if slices.Contains(r.APIGroups, "*") {
					t.Errorf("%s: wildcard apiGroup", o.Metadata.Name)
				}
			}
		}
	}
	if len(roles) == 0 {
		t.Fatal("no ClusterRoles found")
	}
	nsBound := map[string]bool{}
	for _, o := range objs {
		switch o.Kind {
		case "ClusterRoleBinding":
			role, ok := roles[o.RoleRef.Name]
			if !ok {
				t.Fatalf("%s binds unknown role %s", o.Metadata.Name, o.RoleRef.Name)
			}
			for _, r := range role.Rules {
				for _, res := range r.Resources {
					if slices.Contains(contentOnly, res) {
						t.Errorf("%s grants %s cluster-wide; bind it per namespace", o.Metadata.Name, res)
					}
				}
			}
		case "RoleBinding":
			nsBound[o.Metadata.Namespace] = true
		}
	}
	for _, ns := range []string{"demo", "broken-config", "broken-readiness", "broken-scheduling", "broken-hpa"} {
		if !nsBound[ns] {
			t.Errorf("demo namespace %s has no RoleBinding", ns)
		}
	}
}
