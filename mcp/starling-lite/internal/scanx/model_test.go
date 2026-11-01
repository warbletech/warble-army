package scanx

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanModelFixtures(t *testing.T) {
	cases := map[string]struct {
		format, verdict string
		dangerous       string
		rule            string
	}{
		"evil-proto0.pkl":      {"pickle", "unsafe", "posix.system", RuleModelPickleRCE},
		"evil-proto4.bin":      {"pickle", "unsafe", "posix.system", RuleModelPickleRCE},
		"evil-eval.pkl":        {"pickle", "unsafe", "eval", RuleModelPickleRCE},
		"evil-torch.pt":        {"torch-zip", "unsafe", "posix.system", RuleModelPickleRCE},
		"safe-ordereddict.pkl": {"pickle", "pickle-allowlisted", "", RuleModelPickle},
		"safe-plain.pkl":       {"pickle", "pickle-allowlisted", "", RuleModelPickle},
		"unknown-global.pkl":   {"pickle", "suspicious", "", RuleModelPickleUnknown},
		"model.safetensors":    {"safetensors", "safe", "", ""},
		"truncated.pkl":        {"pickle", "unparseable", "", RuleModelPickleUnknown},
	}
	for name, want := range cases {
		b, err := os.ReadFile(filepath.Join("testdata", "models", name))
		if err != nil {
			t.Fatal(err)
		}
		r := ScanModelBytes(b, name)
		if r.Format != want.format || r.Verdict != want.verdict {
			t.Errorf("%s: format=%s verdict=%s, want %s/%s (globals %v, notes %v)", name, r.Format, r.Verdict, want.format, want.verdict, r.Globals, r.Notes)
		}
		if want.dangerous != "" && !strings.Contains(strings.Join(r.Dangerous, ","), want.dangerous) {
			t.Errorf("%s: dangerous=%v, want %s", name, r.Dangerous, want.dangerous)
		}
		ids := ""
		for _, f := range r.Findings {
			ids += str(f["id"]) + ","
		}
		if want.rule == "" && ids != "" {
			t.Errorf("%s: unexpected findings %s", name, ids)
		}
		if want.rule != "" && !strings.Contains(ids, want.rule+",") {
			t.Errorf("%s: findings %s missing %s", name, ids, want.rule)
		}
	}
}

func TestScanModelInputs(t *testing.T) {
	b, _ := os.ReadFile("testdata/models/evil-eval.pkl")
	out, err := ScanModel("", base64.StdEncoding.EncodeToString(b), "")
	if err != nil {
		t.Fatal(err)
	}
	var r ModelReport
	if err := json.Unmarshal([]byte(out), &r); err != nil || r.Verdict != "unsafe" {
		t.Fatalf("base64 input: %v %+v", err, r)
	}
	t.Setenv("STARLING_SCAN_ROOT", "testdata")
	if out, err = ScanModel("models/evil-torch.pt", "", ""); err != nil || !strings.Contains(out, `"unsafe"`) {
		t.Fatalf("path input: %v %s", err, out)
	}
	if _, err = ScanModel("../model_test.go", "", ""); err == nil {
		t.Fatal("path outside scan root must be refused")
	}
	t.Setenv("STARLING_MODEL_MAX_BYTES", "16")
	if _, err = ScanModel("models/evil-torch.pt", "", ""); err == nil {
		t.Fatal("size cap not enforced")
	}
	if _, err = ScanModel("", "", ""); err == nil {
		t.Fatal("empty input must error")
	}
}

func TestPickleParserNeverPanicsOnGarbage(t *testing.T) {
	inputs := [][]byte{{}, {0x80}, {0x80, 0x04, 0x95}, {'c'}, {'c', 'o', 's', '\n'}, {0x8d, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, []byte("PK\x03\x04garbage")}
	for _, in := range inputs {
		_ = ScanModelBytes(in, "garbage")
	}
}
