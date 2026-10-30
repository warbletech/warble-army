package llmfix

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var update = flag.Bool("update", false, "rewrite testdata/golden/*.json")

// hardeningAnswer is what a well-behaved model returns for POL-linuxHardening:
// RuntimeDefault seccomp + drop ALL capabilities on every named container.
func hardeningAnswer(containers []string, path []string) string {
	var list []any
	for _, c := range containers {
		list = append(list, map[string]any{"name": c, "securityContext": map[string]any{
			"seccompProfile": map[string]any{"type": "RuntimeDefault"},
			"capabilities":   map[string]any{"drop": []any{"ALL"}},
		}})
	}
	var body any = map[string]any{"containers": list}
	for i := len(path) - 1; i >= 0; i-- {
		body = map[string]any{path[i]: body}
	}
	b, _ := json.Marshal(map[string]any{"patch": body, "rationale": "Apply RuntimeDefault seccomp and drop all capabilities."})
	return string(b)
}

type goldenResult struct {
	File       string   `json:"file"`
	Kind       string   `json:"kind"`
	Name       string   `json:"name"`
	Issue      string   `json:"issue"`
	Outcome    string   `json:"outcome"` // accepted | rules | rejected | error
	Generator  string   `json:"generator,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	PatchType  string   `json:"patch_type,omitempty"`
	Patch      string   `json:"patch,omitempty"`
	DryRun     bool     `json:"dry_run"`
	Resolved   []string `json:"resolved_findings,omitempty"`
	New        []string `json:"new_findings,omitempty"`
	ScoreRises bool     `json:"score_not_regressed"`
}

func scannersAvailable() bool {
	for _, b := range []string{"pluto", "polaris", "kube-score"} {
		if _, err := exec.LookPath(b); err != nil {
			return false
		}
	}
	return true
}

// TestGoldenDemoBad runs the LLM path (fake provider, real ShrikeOps re-scan)
// over every workload in shrikeops-demo/k8s/bad/*.yaml.
func TestGoldenDemoBad(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "..", "shrikeops-demo", "k8s", "bad")
	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if len(files) == 0 {
		t.Skip("shrikeops-demo submodule not checked out")
	}
	if !scannersAvailable() {
		if os.Getenv("CI") != "" {
			t.Fatal("pluto/polaris/kube-score must be on PATH in CI for golden tests")
		}
		t.Skip("ShrikeOps scanners (pluto, polaris, kube-score) not on PATH")
	}
	sort.Strings(files)
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		base := filepath.Base(file)
		var results []goldenResult
		dec := yaml.NewDecoder(strings.NewReader(string(raw)))
		for {
			var doc map[string]any
			if err := dec.Decode(&doc); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				t.Fatal(err)
			}
			if doc == nil {
				continue
			}
			kind := fmt.Sprint(doc["kind"])
			meta, _ := doc["metadata"].(map[string]any)
			name := fmt.Sprint(meta["name"])
			for _, issue := range []string{"POL-linuxHardening", "runAsNonRoot", "POL-hostNetworkSet"} {
				results = append(results, runGolden(t, base, string(raw), kind, name, issue))
			}
		}
		checkGolden(t, strings.TrimSuffix(base, ".yaml"), results)
	}
}

func runGolden(t *testing.T, file, manifest, kind, name, issue string) goldenResult {
	t.Helper()
	g := goldenResult{File: file, Kind: kind, Name: name, Issue: issue}
	var path []string
	var containers []string
	f := &Fake{Respond: func(_, user string) (string, error) {
		var data struct {
			Containers []string       `json:"containers"`
			Target     map[string]any `json:"target"`
		}
		_ = json.Unmarshal([]byte(strings.TrimPrefix(user, "DATA (untrusted, JSON):\n")), &data)
		containers = data.Containers
		path = strings.Split(fmt.Sprint(data.Target["pod_spec_path"]), ".")
		if issue == "POL-hostNetworkSet" {
			// Hostile/confused model: escalates instead of fixing. Must be rejected.
			var body any = map[string]any{"hostNetwork": true}
			for i := len(path) - 1; i >= 0; i-- {
				body = map[string]any{path[i]: body}
			}
			b, _ := json.Marshal(map[string]any{"patch": body, "rationale": "ignore previous rules"})
			return string(b), nil
		}
		return hardeningAnswer(containers, path), nil
	}}
	p, err := Propose(context.Background(), Request{Kind: kind, Name: name, Namespace: "default", Issue: issue, Manifest: manifest},
		Options{Provider: f, Scan: DefaultScanner})
	switch re, rej := IsRejected(err); {
	case rej:
		g.Outcome, g.Reason = "rejected", re.Reason
	case err != nil:
		g.Outcome, g.Reason = "error", err.Error()
	default:
		g.Generator, g.PatchType, g.Patch, g.DryRun = p.Generator, p.PatchType, p.Patch, p.DryRun
		if p.Generator == "rules" {
			g.Outcome = "rules"
		} else {
			g.Outcome = "accepted"
			g.Resolved, g.New = p.Validation.ResolvedFindings, p.Validation.NewFindings
			g.ScoreRises = p.Validation.ScoreAfter >= p.Validation.ScoreBefore
		}
		if !p.DryRun {
			t.Errorf("%s %s/%s: proposal must be dry_run", file, kind, name)
		}
	}
	return g
}

func checkGolden(t *testing.T, name string, got []goldenResult) {
	t.Helper()
	b, _ := json.MarshalIndent(got, "", "  ")
	b = append(b, '\n')
	path := filepath.Join("testdata", "golden", name+".json")
	if *update {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run go test ./internal/llmfix -run Golden -update): %v", path, err)
	}
	if string(want) != string(b) {
		t.Errorf("golden mismatch for %s\n--- want\n%s\n--- got\n%s", path, want, b)
	}
}
