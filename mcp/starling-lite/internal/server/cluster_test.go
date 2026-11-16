package server

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/warbletech/warble-army/mcp/starling-lite/internal/config"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/protocol"
)

// Unreachable API server: building clients never dials, so this attaches a
// "cluster" without network access.
const fakeKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: demo
  cluster: {server: "https://127.0.0.1:1", insecure-skip-tls-verify: true}
users:
- name: reader
  user: {token: not-a-real-token}
contexts:
- name: demo
  context: {cluster: demo, user: reader}
current-context: demo
`

func newClusterServer(t *testing.T) *Server {
	t.Helper()
	kc := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(kc, []byte(fakeKubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", kc)
	s, err := New(config.Config{Mode: "readonly", RegistryPath: filepath.Join("..", "..", "..", "tools", "registry.json"), Kubeconfig: kc})
	if err != nil {
		t.Fatal(err)
	}
	if !s.ClusterAttached() {
		t.Fatalf("cluster not attached: %v", s.clusterE)
	}
	return s
}

func listNames(t *testing.T, s *Server) map[string]bool {
	t.Helper()
	resp := s.handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), "")
	res, ok := resp.Result.(protocol.ToolsListResult)
	if !ok {
		t.Fatalf("tools/list result %T", resp.Result)
	}
	names := map[string]bool{}
	for _, d := range res.Tools {
		names[d.Name] = true
	}
	return names
}

func instructions(t *testing.T, s *Server) string {
	t.Helper()
	resp := s.handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`), "")
	res, ok := resp.Result.(protocol.InitializeResult)
	if !ok {
		t.Fatalf("initialize result %T", resp.Result)
	}
	return res.Instructions
}

var clusterTools = []string{
	"list_namespaces", "list_pods", "get_pod", "get_pod_logs", "list_nodes", "list_events",
	"list_deployments", "list_hpas", "list_pod_metrics", "list_node_metrics", "list_services",
	"get_configmap", "cluster_summary", "scan_cluster_export",
}

func TestNoClusterHidesClusterTools(t *testing.T) {
	s := newTestServer(t, "readonly")
	if s.ClusterAttached() {
		t.Fatal("expected no cluster")
	}
	names := listNames(t, s)
	for _, n := range clusterTools {
		if names[n] {
			t.Errorf("%s listed without a cluster", n)
		}
	}
	for _, n := range []string{"scan_manifest", "scan_helm", "propose_remediation", "propose_remediation_llm", "sandbox_project"} {
		if !names[n] {
			t.Errorf("%s missing from scan-only tools/list", n)
		}
	}
	if got := instructions(t, s); !strings.Contains(got, "scan-only") || !strings.Contains(got, "kind") {
		t.Errorf("instructions do not explain scan-only mode: %q", got)
	}
}

func TestNoClusterCallExplains(t *testing.T) {
	s := newTestServer(t, "readonly")
	for _, n := range []string{"cluster_summary", "list_namespaces", "scan_cluster_export"} {
		res := call(s, n, nil)
		if !res.IsError {
			t.Fatalf("%s: expected error result, got %+v", n, res)
		}
		msg := res.Content[0].Text
		for _, want := range []string{n, "scan-only", "scan_manifest", "kind create cluster", "deploy/demo-cluster"} {
			if !strings.Contains(msg, want) {
				t.Errorf("%s: message missing %q: %s", n, want, msg)
			}
		}
	}
	// Scan tools are unaffected.
	if res := call(s, "scan_manifest", map[string]any{"manifest": llmManifest}); res.IsError {
		t.Fatalf("scan_manifest failed without a cluster: %s", res.Content[0].Text)
	}
}

func fakeKubectl(t *testing.T, found bool) {
	t.Helper()
	orig := lookPath
	t.Cleanup(func() { lookPath = orig })
	lookPath = func(string) (string, error) {
		if found {
			return "/usr/bin/kubectl", nil
		}
		return "", exec.ErrNotFound
	}
}

func TestClusterAttachedWithoutKubectlHidesExport(t *testing.T) {
	fakeKubectl(t, false)
	s := newClusterServer(t)
	names := listNames(t, s)
	if names["scan_cluster_export"] {
		t.Error("scan_cluster_export listed without kubectl")
	}
	if !names["cluster_summary"] {
		t.Error("cluster_summary missing with a cluster attached")
	}
	res := call(s, "scan_cluster_export", nil)
	if !res.IsError || !strings.Contains(res.Content[0].Text, "kubectl") {
		t.Fatalf("scan_cluster_export without kubectl: %+v", res)
	}
}

func TestClusterAttachedListsClusterTools(t *testing.T) {
	fakeKubectl(t, true)
	s := newClusterServer(t)
	names := listNames(t, s)
	for _, n := range clusterTools {
		if !names[n] {
			t.Errorf("%s missing with a cluster attached", n)
		}
	}
	for _, n := range []string{"scale_deployment", "delete_pod", "apply_remediation"} {
		if names[n] {
			t.Errorf("write tool %s listed in readonly mode", n)
		}
	}
	if got := instructions(t, s); strings.Contains(got, "scan-only") {
		t.Errorf("instructions claim scan-only with a cluster: %q", got)
	}
	// The call reaches heron-ops (and fails on the unreachable API server),
	// rather than being refused as scan-only.
	res := call(s, "list_namespaces", nil)
	if res.IsError && strings.Contains(res.Content[0].Text, "scan-only") {
		t.Fatalf("cluster call refused as scan-only: %s", res.Content[0].Text)
	}
}
