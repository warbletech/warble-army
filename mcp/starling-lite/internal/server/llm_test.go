package server

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warbletech/warble-army/mcp/starling-lite/internal/config"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/llmfix"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/protocol"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/remediate"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/scanx"
)

const llmManifest = `apiVersion: apps/v1
kind: Deployment
metadata: {name: api, namespace: default}
spec:
  selector: {matchLabels: {app: api}}
  template:
    metadata: {labels: {app: api}}
    spec:
      containers: [{name: api, image: example/api:1.0}]
`

func newTestServer(t *testing.T, mode string) *Server {
	t.Helper()
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "none"))
	s, err := New(config.Config{Mode: mode, RegistryPath: filepath.Join("..", "..", "..", "tools", "registry.json"), Kubeconfig: filepath.Join(t.TempDir(), "none")})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func call(s *Server, name string, args map[string]any) protocol.ToolCallResult {
	return s.dispatch(context.Background(), protocol.ToolCallParams{Name: name, Arguments: args})
}

func text(r protocol.ToolCallResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}

func TestLLMToolReadonlyAndFailsClosed(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "")
	s := newTestServer(t, "readonly")
	def, ok := s.reg.ByName("propose_remediation_llm")
	if !ok || def.Mode != "readonly" || def.Backend != "starling-lite" {
		t.Fatalf("registry entry: %+v %v", def, ok)
	}
	r := call(s, "propose_remediation_llm", map[string]any{"kind": "Deployment", "name": "api", "issue": "POL-linuxHardening", "manifest": llmManifest})
	if !r.IsError || !strings.Contains(text(r), "LLM fallback disabled") {
		t.Fatalf("want fail-closed, got %s", text(r))
	}
	t.Setenv("LLM_PROVIDER", "openai") // no OPENAI_API_KEY
	r = call(s, "propose_remediation_llm", map[string]any{"kind": "Deployment", "name": "api", "issue": "POL-linuxHardening", "manifest": llmManifest})
	if !r.IsError || !strings.Contains(text(r), "OPENAI_API_KEY") {
		t.Fatalf("want missing-key error, got %s", text(r))
	}
	// apply_remediation gates unchanged: hidden/refused in readonly.
	r = call(s, "apply_remediation", map[string]any{"kind": "Deployment", "name": "api", "namespace": "default", "issue": "x"})
	if !r.IsError || !strings.Contains(text(r), "readwrite") {
		t.Fatalf("apply gate changed: %s", text(r))
	}
	if msg := remediate.ApplyGate("readwrite", false, false, "default", nil); msg == "" {
		t.Fatal("apply without confirm must be refused")
	}
}

func TestLLMToolWithFakeProvider(t *testing.T) {
	s := newTestServer(t, "readonly")
	old := llmOptions
	defer func() { llmOptions = old }()
	llmOptions = func(*Server, map[string]any) llmfix.Options {
		return llmfix.Options{
			Provider: &llmfix.Fake{Respond: func(_, _ string) (string, error) {
				return `{"patch":{"spec":{"template":{"spec":{"containers":[{"name":"api","securityContext":{"seccompProfile":{"type":"RuntimeDefault"}}}]}}}},"rationale":"seccomp"}`, nil
			}},
			Scan: func(_ context.Context, m string) (scanx.Report, error) {
				fs := []any{}
				if !strings.Contains(m, "RuntimeDefault") {
					fs = append(fs, map[string]any{"id": "POL-linuxHardening", "resource": "x"})
				}
				return scanx.Report{"score": map[string]any{"overall": 50.0, "complete": true}, "findings": fs}, nil
			},
		}
	}
	r := call(s, "propose_remediation_llm", map[string]any{"kind": "Deployment", "name": "api", "namespace": "default", "finding": map[string]any{"id": "POL-linuxHardening"}, "manifest": llmManifest})
	if r.IsError {
		t.Fatalf("unexpected error: %s", text(r))
	}
	out := text(r)
	for _, want := range []string{`\"generator\": \"llm\"`, `\"dry_run\": true`, `LLM-generated`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in %s", want, out)
		}
	}
}
