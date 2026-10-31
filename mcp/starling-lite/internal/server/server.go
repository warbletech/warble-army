package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/warbletech/warble-army/mcp/starling-lite/internal/config"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/llmfix"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/protocol"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/registry"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/remediate"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/sandbox"
	"github.com/warbletech/warble-army/mcp/starling-lite/internal/scanx"
	"github.com/warbletech/heron-ops/pkg/mcpserver"
)

// Server is the Starling Lite MCP facade.
type Server struct {
	cfg      config.Config
	reg      *registry.File
	cluster  *mcpserver.Server // may be nil if no kubeconfig
	clusterE error
	mu       sync.Mutex
}

// New loads the registry and optionally initializes the SteadyHelm backend.
func New(cfg config.Config) (*Server, error) {
	reg, err := registry.Load(cfg.RegistryPath)
	if err != nil {
		return nil, err
	}
	s := &Server{cfg: cfg, reg: reg}

	cl, err := mcpserver.New(mcpserver.Options{
		Kubeconfig: cfg.Kubeconfig,
		Namespace:  cfg.Namespace,
		Mode:       cfg.Mode,
	})
	if err != nil {
		s.clusterE = err
		log.Printf("[starling-lite] cluster backend unavailable: %v (scan/remediate still work)", err)
	} else {
		s.cluster = cl
	}
	return s, nil
}

// ServeStdio runs newline-delimited JSON-RPC on stdin/stdout.
func (s *Server) ServeStdio() error {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := s.handle(context.Background(), line, "")
		s.mu.Lock()
		_ = enc.Encode(resp)
		s.mu.Unlock()
	}
	return sc.Err()
}

// ServeHTTP exposes POST /mcp and GET /health (also / and /lite/* for PaaS / legacy paths).
// Honors MCP 2026-07-28 headers: MCP-Protocol-Version, Mcp-Method, Mcp-Name (SEP-2243).
func (s *Server) ServeHTTP(addr string) error {
	mux := http.NewServeMux()
	handleMCP := func(w http.ResponseWriter, r *http.Request) {
		// CORS for browser-based MCP testers (safe for public readonly demo)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Protocol-Version, Mcp-Method, Mcp-Name")
		w.Header().Set("Access-Control-Expose-Headers", "MCP-Protocol-Version, Mcp-Method")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			http.Error(w, "bad JSON", http.StatusBadRequest)
			return
		}
		// Header-based method routing (optional; body method still authoritative if set)
		headerVer := r.Header.Get("MCP-Protocol-Version")
		if headerVer == "" {
			headerVer = r.Header.Get("Mcp-Protocol-Version")
		}
		resp := s.handle(r.Context(), raw, headerVer)
		// Echo negotiated version for gateways
		if v, ok := negotiatedVersion(resp); ok {
			w.Header().Set("MCP-Protocol-Version", v)
		}
		if m := r.Header.Get("Mcp-Method"); m != "" {
			w.Header().Set("Mcp-Method", m)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
	handleHealth := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		st := map[string]any{
			"status":             "ok",
			"server":             "starling-lite",
			"version":            s.cfg.Version,
			"revision":           s.cfg.Revision,
			"mode":               s.cfg.Mode,
			"cluster":            s.cluster != nil,
			"mcp":                "/mcp",
			"protocolVersions":   protocol.SupportedProtocolVersions,
			"protocolPreferred":  protocol.LatestProtocolVersion,
		}
		_ = json.NewEncoder(w).Encode(st)
	}
	mux.HandleFunc("/mcp", handleMCP)
	mux.HandleFunc("/lite/mcp", handleMCP)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/lite/health", handleHealth)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handleHealth(w, r)
	})
	return http.ListenAndServe(addr, mux)
}

func negotiatedVersion(resp protocol.Response) (string, bool) {
	if resp.Result == nil {
		return "", false
	}
	switch r := resp.Result.(type) {
	case protocol.InitializeResult:
		return r.ProtocolVersion, r.ProtocolVersion != ""
	case protocol.DiscoverResult:
		return r.ProtocolVersion, r.ProtocolVersion != ""
	case *protocol.InitializeResult:
		if r != nil {
			return r.ProtocolVersion, r.ProtocolVersion != ""
		}
	case *protocol.DiscoverResult:
		if r != nil {
			return r.ProtocolVersion, r.ProtocolVersion != ""
		}
	}
	return "", false
}

func (s *Server) handle(ctx context.Context, raw []byte, headerProtocol string) protocol.Response {
	var req protocol.Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return protocol.ErrResponse(nil, -32700, "parse error")
	}

	switch req.Method {
	case "initialize":
		var params protocol.InitializeParams
		_ = json.Unmarshal(req.Params, &params)
		ver := protocol.Negotiate(firstNonEmpty(params.ProtocolVersion, headerProtocol))
		return protocol.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.InitializeResult{
				ProtocolVersion: ver,
				ServerInfo:      protocol.ServerInfo{Name: "starling-lite", Version: s.cfg.Version},
				Capabilities:    protocol.DefaultCapabilities(),
				Instructions:    protocol.DefaultInstructions(),
			},
		}
	case "server/discover":
		// MCP 2026-07-28 optional capability probe (no session required)
		ver := protocol.Negotiate(headerProtocol)
		return protocol.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.DiscoverResult{
				ProtocolVersion: ver,
				ServerInfo:      protocol.ServerInfo{Name: "starling-lite", Version: s.cfg.Version},
				Capabilities:    protocol.DefaultCapabilities(),
				Instructions:    protocol.DefaultInstructions(),
			},
		}
	case "notifications/initialized":
		// Legacy handshake ack — accepted for 2024/2025 clients; no-op on 2026-07-28
		return protocol.Response{}
	case "tools/list":
		tools := s.reg.Filtered(s.cfg.Mode)
		defs := make([]protocol.ToolDefinition, 0, len(tools))
		for _, t := range tools {
			defs = append(defs, protocol.ToolDefinition{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
		return protocol.Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: protocol.ToolsListResult{
				Tools:      defs,
				TtlMs:      60_000, // cacheable list (SEP-2549)
				CacheScope: "server",
			},
		}
	case "tools/call":
		var params protocol.ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.ErrResponse(req.ID, -32602, "invalid params")
		}
		result := s.dispatch(ctx, params)
		return protocol.Response{JSONRPC: "2.0", ID: req.ID, Result: result}
	case "ping":
		return protocol.Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	default:
		return protocol.ErrResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func (s *Server) dispatch(ctx context.Context, params protocol.ToolCallParams) protocol.ToolCallResult {
	def, ok := s.reg.ByName(params.Name)
	if !ok {
		return protocol.ErrorResult(fmt.Sprintf("unknown tool: %q", params.Name))
	}
	if def.Mode == "readwrite" && s.cfg.Mode != "readwrite" {
		return protocol.ErrorResult(fmt.Sprintf("tool %q requires --mode=readwrite", params.Name))
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}

	switch def.Backend {
	case "steadyhelm":
		return s.callCluster(ctx, params.Name, params.Arguments)
	case "shrikeops":
		return s.callScan(params.Name, params.Arguments)
	case "starling-lite":
		if params.Name == "propose_remediation_llm" {
			return s.callRemediateLLM(ctx, params.Arguments)
		}
		return s.callRemediate(params.Name, params.Arguments)
	default:
		return protocol.ErrorResult(fmt.Sprintf("unhandled backend %q for %s", def.Backend, params.Name))
	}
}

func (s *Server) callCluster(ctx context.Context, name string, args map[string]any) protocol.ToolCallResult {
	if s.cluster == nil {
		msg := "cluster backend unavailable"
		if s.clusterE != nil {
			msg = s.clusterE.Error()
		}
		return protocol.ErrorResult(msg + " (set KUBECONFIG or --kubeconfig)")
	}
	if name == "scale_deployment" || name == "delete_pod" {
		ns, _ := args["namespace"].(string)
		if ns != "" && !s.cfg.NamespaceAllowed(ns) {
			return protocol.ErrorResult(fmt.Sprintf("namespace %q not in --allow-namespaces", ns))
		}
	}
	text, isErr := s.cluster.CallTool(ctx, name, args)
	if isErr {
		return protocol.ErrorResult(text)
	}
	return protocol.TextResult(text)
}

func (s *Server) callScan(name string, args map[string]any) protocol.ToolCallResult {
	switch name {
	case "scan_manifest":
		path, _ := args["path"].(string)
		manifest, _ := args["manifest"].(string)
		source, _ := args["source"].(string)
		out, err := scanx.Manifest(path, manifest, source)
		if err != nil {
			return protocol.ErrorResult(err.Error())
		}
		return protocol.TextResult(out)
	case "scan_helm":
		chart, _ := args["chart_path"].(string)
		release, _ := args["release_name"].(string)
		var values []string
		if vf, ok := args["values_files"].([]any); ok {
			for _, v := range vf {
				if s, ok := v.(string); ok {
					values = append(values, s)
				}
			}
		}
		set := map[string]string{}
		if m, ok := args["set"].(map[string]any); ok {
			for k, v := range m {
				set[k] = fmt.Sprint(v)
			}
		}
		out, err := scanx.Helm(chart, values, set, release)
		if err != nil {
			return protocol.ErrorResult(err.Error())
		}
		return protocol.TextResult(out)
	case "scan_cluster_export":
		ns, _ := args["namespace"].(string)
		var kinds []string
		if k, ok := args["kinds"].([]any); ok {
			for _, v := range k {
				if s, ok := v.(string); ok {
					kinds = append(kinds, s)
				}
			}
		}
		out, err := scanx.ClusterExport(ns, kinds)
		if err != nil {
			return protocol.ErrorResult(err.Error())
		}
		return protocol.TextResult(out)
	default:
		return protocol.ErrorResult("unknown scan tool: " + name)
	}
}

func (s *Server) callRemediate(name string, args map[string]any) protocol.ToolCallResult {
	kind, _ := args["kind"].(string)
	n, _ := args["name"].(string)
	ns, _ := args["namespace"].(string)
	issue, _ := args["issue"].(string)

	req := remediate.Request{Kind: kind, Name: n, Namespace: ns, Issue: issue}
	req.Container, _ = args["container"].(string)
	req.Manifest, _ = args["manifest"].(string)
	if f, ok := args["finding"].(map[string]any); ok {
		req.FindingResource, _ = f["resource"].(string)
		if req.Issue == "" {
			req.Issue, _ = f["id"].(string)
		}
	}

	switch name {
	case "sandbox_project":
		return s.callSandbox(args)
	case "propose_remediation":
		p, err := remediate.ProposeFor(req)
		if err != nil {
			return protocol.ErrorResult(err.Error())
		}
		return protocol.TextResult(remediate.FormatJSON(p))
	case "apply_remediation":
		dryRun := true
		if v, ok := args["dry_run"].(bool); ok {
			dryRun = v
		}
		confirm, _ := args["confirm"].(bool)
		if msg := remediate.ApplyGate(s.cfg.Mode, dryRun, confirm, ns, s.cfg.NamespaceAllowed); msg != "" {
			return protocol.ErrorResult(msg)
		}
		patch, _ := args["patch"].(string)
		if patch == "" {
			p, err := remediate.ProposeFor(req)
			if err != nil {
				return protocol.ErrorResult(err.Error())
			}
			patch = p.Patch
		}
		// Phase 2: dry-run only via kubectl if present; never silent apply without user confirm
		if dryRun {
			return protocol.TextResult(fmt.Sprintf(
				"DRY-RUN apply_remediation for %s/%s ns=%s issue=%s\n\n--- patch ---\n%s\n\n"+
					"To apply for real: --mode=readwrite dry_run=false confirm=true\n"+
					"(kubectl apply --dry-run=server -f -  recommended first)",
				kind, n, ns, issue, patch,
			))
		}
		return protocol.TextResult(fmt.Sprintf(
			"APPLY REQUESTED (confirm=true) for %s/%s ns=%s\n\n%s\n\n"+
				"Starling Lite Phase 2 does not auto-apply to the API server yet. "+
				"Save the patch and run: kubectl apply -f patch.yaml\n"+
				"Phase 3 Reflexion executor will gate real applies.",
			kind, n, ns, patch,
		))
	default:
		return protocol.ErrorResult("unknown remediate tool: " + name)
	}
}

func strList(v any) []string {
	var out []string
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if str, ok := x.(string); ok && str != "" {
				out = append(out, str)
			}
		}
	case string:
		out = config.ParseCSV(t)
	}
	return out
}

// callSandbox renders an isolated project sandbox (dry-run manifests only).
func (s *Server) callSandbox(args map[string]any) protocol.ToolCallResult {
	spec := sandbox.Spec{}
	spec.Project, _ = args["project"].(string)
	spec.Namespace, _ = args["namespace"].(string)
	spec.CPU, _ = args["cpu"].(string)
	spec.Memory, _ = args["memory"].(string)
	spec.Image, _ = args["image"].(string)
	if p, ok := args["pods"].(float64); ok {
		spec.Pods = int(p)
	}
	spec.EgressCIDRs = strList(args["egress_cidrs"])
	spec.Command = strList(args["command"])
	out, err := sandbox.Render(spec)
	if err != nil {
		return protocol.ErrorResult(err.Error())
	}
	return protocol.TextResult("# DRY-RUN project sandbox (not applied). Review, then:\n" +
		"#   kubectl apply --dry-run=server -f sandbox.yaml && kubectl apply -f sandbox.yaml\n" + out)
}

// llmOptions is overridable in tests (fake provider / scanner).
var llmOptions = func(s *Server, args map[string]any) llmfix.Options {
	o := llmfix.Options{Scan: llmfix.DefaultScanner}
	o.Provider, o.ProviderErr = llmfix.FromEnv(os.Getenv)
	useDryRun := true
	if v, ok := args["server_dry_run"].(bool); ok {
		useDryRun = v
	}
	if useDryRun && s.cluster != nil {
		if kubectl, err := exec.LookPath("kubectl"); err == nil {
			kubeconfig := s.cfg.Kubeconfig
			o.DryRun = func(ctx context.Context, manifest string) error {
				cmdArgs := []string{"apply", "--dry-run=server", "-o", "name", "-f", "-"}
				if kubeconfig != "" {
					cmdArgs = append([]string{"--kubeconfig", kubeconfig}, cmdArgs...)
				}
				cmd := exec.CommandContext(ctx, kubectl, cmdArgs...)
				cmd.Stdin = strings.NewReader(manifest)
				if out, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
				}
				return nil
			}
		}
	}
	return o
}

// callRemediateLLM: deterministic rules first, then the guarded LLM fallback.
// Dry-run only; never mutates.
func (s *Server) callRemediateLLM(ctx context.Context, args map[string]any) protocol.ToolCallResult {
	r := llmfix.Request{}
	r.Kind, _ = args["kind"].(string)
	r.Name, _ = args["name"].(string)
	r.Namespace, _ = args["namespace"].(string)
	r.Issue, _ = args["issue"].(string)
	r.Container, _ = args["container"].(string)
	r.Manifest, _ = args["manifest"].(string)
	r.Finding, _ = args["finding"].(map[string]any)
	if r.Issue == "" && r.Finding != nil {
		r.Issue, _ = r.Finding["id"].(string)
	}
	switch cc := args["cluster_context"].(type) {
	case string:
		r.ClusterContext = cc
	case nil:
	default:
		b, _ := json.Marshal(cc)
		r.ClusterContext = string(b)
	}
	ctx, cancel := context.WithTimeout(ctx, llmfix.Timeout(os.Getenv)+30*time.Second)
	defer cancel()
	p, err := llmfix.Propose(ctx, r, llmOptions(s, args))
	if err != nil {
		if re, ok := llmfix.IsRejected(err); ok {
			b, _ := json.MarshalIndent(map[string]any{
				"rejected": true, "provider": re.Provider, "reason": re.Reason, "validation": re.Detail,
			}, "", "  ")
			return protocol.ErrorResult(string(b))
		}
		return protocol.ErrorResult(err.Error())
	}
	return protocol.TextResult(remediate.FormatJSON(p))
}
