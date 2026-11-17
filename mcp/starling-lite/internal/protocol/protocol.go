package protocol

import "encoding/json"

// Supported MCP protocol versions (newest first).
// 2026-07-28 = MCP "stateless core" (often called MCP 2.0 in community shorthand).
// We still accept 2025-03-26 and 2024-11-05 for older clients.
var SupportedProtocolVersions = []string{
	"2026-07-28",
	"2025-03-26",
	"2024-11-05",
}

const LatestProtocolVersion = "2026-07-28"

// Negotiate picks the best mutual protocol version.
// Empty client request → latest. Unknown → fall back to 2024-11-05 if nothing matches.
func Negotiate(clientVersion string) string {
	if clientVersion == "" {
		return LatestProtocolVersion
	}
	for _, v := range SupportedProtocolVersions {
		if v == clientVersion {
			return v
		}
	}
	// Client asked for something we don't know — offer latest and let them decide
	return LatestProtocolVersion
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func ErrResponse(id json.RawMessage, code int, msg string) Response {
	return Response{JSONRPC: "2.0", ID: id, Error: &RPCError{Code: code, Message: msg}}
}

type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ClientInfo      ClientInfo     `json:"clientInfo"`
	Capabilities    map[string]any `json:"capabilities"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
	Instructions    string         `json:"instructions,omitempty"`
}

// DiscoverResult is the 2026-07-28 server/discover payload (optional capability probe).
type DiscoverResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
	Instructions    string         `json:"instructions,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolsListResult includes 2026-07-28 cache hints (SEP-2549); older clients ignore extras.
type ToolsListResult struct {
	Tools      []ToolDefinition `json:"tools"`
	TtlMs      int64            `json:"ttlMs,omitempty"`
	CacheScope string           `json:"cacheScope,omitempty"`
}

type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	// Meta carries client identity on 2026-07-28 stateless requests.
	Meta map[string]any `json:"_meta,omitempty"`
}

type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func TextResult(text string) ToolCallResult {
	return ToolCallResult{Content: []ContentBlock{{Type: "text", Text: text}}}
}

func ErrorResult(text string) ToolCallResult {
	return ToolCallResult{IsError: true, Content: []ContentBlock{{Type: "text", Text: text}}}
}

// DefaultCapabilities for Starling Lite (tools-only, list cacheable).
func DefaultCapabilities() map[string]any {
	return map[string]any{
		"tools": map[string]any{
			"listChanged": false, // static registry in CE demo
		},
	}
}

func DefaultInstructions() string {
	return Instructions(true)
}

// Instructions describes the server to the model. Without an attached cluster
// the live cluster tools are not listed, so say so instead of pointing at them.
func Instructions(clusterAttached bool) string {
	if clusterAttached {
		return "Starling Lite: Kubernetes agent ops MCP. Use scan_* for manifests/Helm, " +
			"cluster tools (cluster_summary, list_*, get_*) for the attached cluster, " +
			"propose_remediation for dry-run fixes. " +
			"Prefer readonly mode. Protocol: 2026-07-28 preferred; 2024-11-05 still supported."
	}
	return "Starling Lite: Kubernetes agent ops MCP. No Kubernetes cluster is attached to this " +
		"server, so it is scan-only: pass YAML to scan_manifest (manifest=...) and use " +
		"propose_remediation for dry-run fixes. Live cluster tools (cluster_summary, list_*, get_*) " +
		"appear when the server has a kubeconfig; run starling-lite locally against a kind cluster " +
		"to use them. Protocol: 2026-07-28 preferred; 2024-11-05 still supported."
}
