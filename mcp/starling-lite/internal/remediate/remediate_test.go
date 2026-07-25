package remediate

import "testing"

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
