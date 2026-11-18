package scanx

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func aiIDs(t *testing.T, file string) []string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, f := range AIFindings(string(b)) {
		ids = append(ids, str(f["id"])+"@"+str(f["resource"]))
	}
	sort.Strings(ids)
	return ids
}

func TestAIRulesBadFixtures(t *testing.T) {
	want := map[string][]string{
		"ai-model-weights.yaml": {
			"AI-GPU-NO-LIMIT@Deployment/ml/llm-server/container:server",
			"AI-HOSTPATH-MODEL-CACHE@Deployment/ml/llm-server/container:fetch-weights",
			"AI-HOSTPATH-MODEL-CACHE@Deployment/ml/llm-server/container:server",
			"AI-MODEL-PICKLE@Deployment/ml/llm-server/container:fetch-weights",
			"AI-MODEL-TRUST-REMOTE-CODE@Deployment/ml/llm-server/container:server",
			"AI-MODEL-UNPINNED@Deployment/ml/llm-server/container:fetch-weights",
			"AI-MODEL-UNPINNED@Deployment/ml/llm-server/container:server",
		},
		"ai-mcp-exposed.yaml": {
			"AI-MCP-INGRESS-NOAUTH@Deployment/agents/docs-mcp",
			"AI-MCP-READWRITE-EXPOSED@Deployment/agents/ops-mcp",
		},
		"ai-agent-config.yaml": {
			"AI-AGENT-LETHAL-TRIFECTA@ConfigMap/agents/agent-tools",
			"AI-TOOL-INSTRUCTION-TEXT@ConfigMap/agents/agent-tools",
		},
		"ai-gpu-device-plugin.yaml": {
			"AI-GPU-NO-LIMIT@Job/ml/finetune/container:train",
			"AI-PRIVILEGED-DEVICE-PLUGIN@DaemonSet/kube-system/nvidia-device-plugin/container:nvidia-device-plugin-ctr",
		},
	}
	files, _ := filepath.Glob("testdata/ai/bad/ai-*.yaml")
	if len(files) != len(want) {
		t.Fatalf("fixtures: got %d files, want %d", len(files), len(want))
	}
	for _, f := range files {
		got := aiIDs(t, f)
		w := want[filepath.Base(f)]
		if strings.Join(got, "\n") != strings.Join(w, "\n") {
			t.Errorf("%s:\n got  %v\n want %v", f, got, w)
		}
	}
}

func TestAIRulesGoodFixturesClean(t *testing.T) {
	files, _ := filepath.Glob("testdata/ai/good/ai-*.yaml")
	if len(files) == 0 {
		t.Fatal("no good fixtures")
	}
	for _, f := range files {
		if got := aiIDs(t, f); len(got) != 0 {
			t.Errorf("%s: false positives %v", f, got)
		}
	}
}

func TestAIRulesDemoBadUnaffected(t *testing.T) {
	// The existing non-AI demo fixtures must not trigger ai-* rules.
	files, _ := filepath.Glob(filepath.Join("..", "..", "..", "..", "shrikeops-demo", "k8s", "*", "*.yaml"))
	if len(files) == 0 {
		t.Skip("shrikeops-demo not checked out")
	}
	for _, f := range files {
		if strings.HasPrefix(filepath.Base(f), "ai-") {
			continue
		}
		if got := aiIDs(t, f); len(got) != 0 {
			t.Errorf("%s: unexpected ai-* findings %v", f, got)
		}
	}
}

func TestAIFindingShapeAndCatalogue(t *testing.T) {
	b, _ := os.ReadFile("testdata/ai/bad/ai-mcp-exposed.yaml")
	for _, f := range AIFindings(string(b)) {
		for _, k := range []string{"id", "category", "severity", "resource", "message", "remediation", "docs_url"} {
			if str(f[k]) == "" {
				t.Fatalf("finding missing %s: %v", k, f)
			}
		}
	}
	for id, r := range AIRules {
		if r.ID != id || severityPenalty[r.Severity] == 0 || r.Remediation == "" {
			t.Errorf("bad catalogue entry %s: %+v", id, r)
		}
	}
}

func TestMergeAIScoring(t *testing.T) {
	b, _ := os.ReadFile("testdata/ai/bad/ai-mcp-exposed.yaml")
	r := Report{
		"score":    map[string]any{"security": 100.0, "api_stability": 100.0, "reliability": 100.0, "lint_clean": 100.0, "overall": 100.0, "grade": "A", "complete": true},
		"summary":  map[string]any{"critical": 0.0, "medium": 0.0, "total": 0.0},
		"findings": []any{},
	}
	r = MergeAI(r, string(b))
	score := r["score"].(map[string]any)
	// critical 25 + medium 7 = 32 off security (weight 0.40) -> 87.2
	if score["security"] != 68.0 || score["overall"] != 87.2 || score["grade"] != "B" {
		t.Fatalf("score: %v", score)
	}
	sum := r["summary"].(map[string]any)
	if sum["critical"] != 1.0 || sum["medium"] != 1.0 || sum["total"] != 2.0 || r["ai_findings"] != 2 {
		t.Fatalf("summary: %v ai=%v", sum, r["ai_findings"])
	}
	// incomplete reports keep N/A
	r2 := Report{"score": map[string]any{"security": 100.0, "overall": 100.0, "grade": "N/A", "complete": false}, "findings": []any{}}
	r2 = MergeAI(r2, string(b))
	if r2["score"].(map[string]any)["grade"] != "N/A" {
		t.Fatal("incomplete report must keep grade N/A")
	}
}

func TestInstructionTextFalsePositivesOnRegistry(t *testing.T) {
	// Our own MCP tool registry must not look like tool poisoning.
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "tools", "registry.json"))
	if err != nil {
		t.Skip(err)
	}
	for _, d := range toolDescriptions(string(b)) {
		if m := instructionRe.FindString(d); m != "" {
			t.Errorf("registry description flagged (%q): %s", m, d)
		}
	}
}

// Repo's own deploy/ manifests and charts: report findings (not asserted clean;
// see PR description). Charts are rendered with helm when available.
func TestAIRulesRepoManifests(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	dirs, _ := filepath.Glob(filepath.Join(root, "deploy", "*", "k8s"))
	var all []string
	for _, d := range dirs {
		files, _ := filepath.Glob(filepath.Join(d, "*.yaml"))
		var docs []string
		for _, f := range files {
			b, _ := os.ReadFile(f)
			docs = append(docs, string(b))
		}
		// one bundle per directory so Ingress -> Service -> Deployment links resolve
		for _, x := range AIFindings(strings.Join(docs, "\n---\n")) {
			all = append(all, d+": "+str(x["id"])+" "+str(x["resource"]))
		}
	}
	if _, err := exec.LookPath("helm"); err == nil {
		for _, c := range []string{"starling-ex-ce", "project-sandbox"} {
			args := []string{"template", "t", filepath.Join(root, "charts", c), "--set", "ingress.enabled=true"}
			if _, err := os.Stat(filepath.Join(root, "charts", c, "ci-values.yaml")); err == nil {
				args = append(args, "-f", filepath.Join(root, "charts", c, "ci-values.yaml"))
			}
			out, err := exec.Command("helm", args...).Output()
			if err != nil {
				t.Logf("helm template %s: %v", c, err)
				continue
			}
			for _, x := range AIFindings(string(out)) {
				all = append(all, c+": "+str(x["id"])+" "+str(x["resource"]))
			}
		}
	}
	for _, l := range all {
		t.Log(l)
	}
}
