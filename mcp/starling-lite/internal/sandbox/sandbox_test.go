package sandbox

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func kinds(t *testing.T, objs []obj) map[string]obj {
	t.Helper()
	m := map[string]obj{}
	for _, o := range objs {
		md := o["metadata"].(obj)
		m[o["kind"].(string)+"/"+md["name"].(string)] = o
	}
	return m
}

func TestDefaultsAndHardening(t *testing.T) {
	objs, err := Objects(Spec{Project: "payments", Image: "ghcr.io/warbletech/shrikeops:v1.2.3", Command: []string{"shrikeops", "scan"}})
	if err != nil {
		t.Fatal(err)
	}
	k := kinds(t, objs)
	ns := k["Namespace/sbx-payments"]
	if ns == nil {
		t.Fatal("namespace missing")
	}
	if ns["metadata"].(obj)["labels"].(obj)["pod-security.kubernetes.io/enforce"] != "restricted" {
		t.Error("PSA restricted not enforced")
	}
	if k["ServiceAccount/sandbox-runner"]["automountServiceAccountToken"] != false {
		t.Error("SA token must not automount")
	}
	deny := k["NetworkPolicy/default-deny-all"]["spec"].(obj)
	if len(deny["policyTypes"].([]any)) != 2 || deny["ingress"] != nil || deny["egress"] != nil {
		t.Error("default deny must cover ingress+egress with no rules")
	}
	if k["ResourceQuota/sandbox-quota"] == nil || k["LimitRange/sandbox-limits"] == nil {
		t.Error("quota/limitrange missing")
	}
	if _, ok := k["NetworkPolicy/allow-egress-cidrs"]; ok {
		t.Error("no extra egress expected by default")
	}
	ps := k["Job/sandbox-task"]["spec"].(obj)["template"].(obj)["spec"].(obj)
	if ps["automountServiceAccountToken"] != false {
		t.Error("job must not mount SA token")
	}
	c := ps["containers"].([]any)[0].(obj)["securityContext"].(obj)
	if c["readOnlyRootFilesystem"] != true || c["allowPrivilegeEscalation"] != false {
		t.Error("container not hardened")
	}
}

func TestEgressCIDRs(t *testing.T) {
	objs, err := Objects(Spec{Project: "p1", EgressCIDRs: []string{"10.0.0.0/8"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := kinds(t, objs)["NetworkPolicy/allow-egress-cidrs"]; !ok {
		t.Error("expected egress policy")
	}
	if _, ok := kinds(t, objs)["Job/sandbox-task"]; ok {
		t.Error("no job without image")
	}
}

func TestValidate(t *testing.T) {
	bad := []Spec{
		{},
		{Project: "Bad_Name"},
		{Project: "ok", EgressCIDRs: []string{"0.0.0.0/0"}},
		{Project: "ok", EgressCIDRs: []string{"nope"}},
		{Project: "ok", Image: "busybox:latest"},
		{Project: "ok", Pods: 500},
	}
	for _, s := range bad {
		if _, err := Objects(s); err == nil {
			t.Errorf("expected error for %+v", s)
		}
	}
}

func TestRenderYAML(t *testing.T) {
	out, err := Render(Spec{Project: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	docs := strings.Split(out, "\n---\n")
	if len(docs) != 6 {
		t.Fatalf("want 6 docs, got %d", len(docs))
	}
	for _, d := range docs {
		var v map[string]any
		if err := yaml.Unmarshal([]byte(d), &v); err != nil || v["kind"] == nil {
			t.Fatalf("bad doc: %v", err)
		}
	}
}
