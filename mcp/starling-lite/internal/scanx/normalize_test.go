package scanx

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// Output of the live server before the fix (ShrikeOps 0.1.0, no tools installed).
const legacyNoTools = `{"scan_id":"kop-1","source":"inline.yaml",
 "score":{"security":100,"api_stability":100,"reliability":100,"lint_clean":100,"overall":100,"grade":"A"},
 "findings":[
  {"id":"PLUTO-NOTFOUND","category":"deprecated_api","severity":"info","resource":"scanner","message":"pluto binary not found"},
  {"id":"POLARIS-NOTFOUND","category":"security_policy","severity":"info","resource":"scanner","message":"polaris binary not found"},
  {"id":"KUBESCORE-NOTFOUND","category":"reliability","severity":"info","resource":"scanner","message":"kube-score binary not found"}],
 "summary":{"critical":0,"high":0,"medium":0,"low":0,"info":3,"total":3}}`

func TestNormalize(t *testing.T) {
	tests := []struct {
		name           string
		in             string
		wantIncomplete bool
		wantGrade      string
		wantOverall    float64
		wantSkipped    int
		wantSecurity   float64
	}{
		{"legacy, all tools missing", legacyNoTools, true, "N/A", 100, 3, 0},
		{"legacy, only polaris missing",
			`{"score":{"security":100,"api_stability":70,"reliability":40,"lint_clean":100,"overall":76,"grade":"B"},
			  "findings":[{"id":"POLARIS-NOTFOUND"}]}`,
			// (70*.2 + 40*.25 + 100*.15) / .6 = 65
			true, "N/A", 65, 1, 0},
		{"new shrikeops, incomplete",
			`{"score":{"security":0,"api_stability":100,"reliability":51,"lint_clean":100,"overall":79.6,"grade":"N/A","complete":false,
			  "skipped_checks":[{"tool":"polaris","category":"security_policy","reason":"not found on PATH"}]},
			  "findings":[{"id":"POLARIS-NOTRUN"}]}`,
			true, "N/A", 79.6, 1, 0},
		{"complete legacy report untouched",
			`{"score":{"security":52,"api_stability":70,"reliability":55,"lint_clean":100,"overall":63.6,"grade":"C"},"findings":[]}`,
			false, "C", 63.6, 0, 52},
		{"complete new report untouched",
			`{"score":{"security":97,"api_stability":100,"reliability":62,"lint_clean":100,"overall":89.3,"grade":"B","complete":true},"findings":[]}`,
			false, "B", 89.3, 0, 97},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Normalize([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			s := r["score"].(map[string]any)
			if r["incomplete"] != tc.wantIncomplete || s["grade"] != tc.wantGrade || s["overall"] != tc.wantOverall || s["security"] != tc.wantSecurity {
				t.Fatalf("incomplete=%v grade=%v overall=%v security=%v", r["incomplete"], s["grade"], s["overall"], s["security"])
			}
			sk, _ := s["skipped_checks"].([]any)
			if len(sk) != tc.wantSkipped {
				t.Fatalf("skipped_checks = %v", sk)
			}
			w, _ := r["warning"].(string)
			if tc.wantIncomplete != strings.Contains(w, "INCOMPLETE SCAN") {
				t.Fatalf("warning = %q", w)
			}
			if tc.wantIncomplete && s["complete"] != false {
				t.Fatalf("complete = %v", s["complete"])
			}
		})
	}
}

func TestNormalizeInvalid(t *testing.T) {
	if _, err := Normalize([]byte("not json")); err == nil {
		t.Fatal("expected error")
	}
}

// scan_manifest on a server without the scanner binaries must never report a grade.
func TestManifestWithoutToolsIsIncomplete(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	path := filepath.Join("..", "..", "..", "..", "shrikeops-demo", "k8s", "bad", "privileged-pod.yaml")
	out, err := Manifest(path, "", "")
	if err != nil {
		t.Skipf("shrikeops-demo submodule not checked out: %v", err)
	}
	var r map[string]any
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatal(err)
	}
	s := r["score"].(map[string]any)
	if r["incomplete"] != true || s["grade"] != "N/A" || s["complete"] != false {
		t.Fatalf("expected incomplete report, got incomplete=%v score=%v", r["incomplete"], s)
	}
}
