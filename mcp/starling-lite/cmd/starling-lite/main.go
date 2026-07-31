package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/warble-tech/warble-army/mcp/starling-lite/internal/config"
	"github.com/warble-tech/warble-army/mcp/starling-lite/internal/server"
)

// version is overridden at build time: -ldflags "-X main.version=..."
var version = "0.2.1"

func main() {
	var (
		mode       = flag.String("mode", envOr("STARLING_MODE", "readonly"), "Server mode: readonly | readwrite (env STARLING_MODE)")
		transport  = flag.String("transport", envOr("STARLING_TRANSPORT", "stdio"), "MCP transport: stdio | http (env STARLING_TRANSPORT)")
		addr       = flag.String("addr", defaultAddr(), "HTTP listen address (env PORT or STARLING_ADDR; PaaS uses PORT)")
		kubeconfig = flag.String("kubeconfig", envOr("KUBECONFIG", ""), "Path to kubeconfig (default: env KUBECONFIG / ~/.kube/config)")
		namespace  = flag.String("namespace", envOr("STARLING_NAMESPACE", ""), "Restrict cluster tools to one namespace")
		registry   = flag.String("registry", envOr("STARLING_REGISTRY", ""), "Path to mcp/tools/registry.json (default: auto-discover)")
		allowNS    = flag.String("allow-namespaces", envOr("STARLING_ALLOW_NAMESPACES", ""), "Comma-separated write allowlist (empty = all when readwrite)")
		ver        = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	// Support: starling-lite mcp --stdio  (Starling product-compatible argv)
	args := flag.Args()
	if len(args) > 0 && args[0] == "mcp" {
		// remaining flags already parsed if before; ignore subcommand
	}

	if *ver {
		fmt.Printf("starling-lite %s\n", version)
		os.Exit(0)
	}

	// Optional: KUBECONFIG_B64 for Railway/Fly secrets (no file mount needed)
	if *kubeconfig == "" {
		if path, err := materializeKubeconfigB64(); err != nil {
			log.Printf("[starling-lite] KUBECONFIG_B64 ignored: %v", err)
		} else if path != "" {
			*kubeconfig = path
		}
	}

	regPath := *registry
	if regPath == "" {
		regPath = discoverRegistry()
	}

	cfg := config.Config{
		Mode:            *mode,
		Transport:       *transport,
		Addr:            *addr,
		Kubeconfig:      *kubeconfig,
		Namespace:       *namespace,
		RegistryPath:    regPath,
		AllowNamespaces: config.ParseCSV(*allowNS),
		Version:         version,
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("[starling-lite] init: %v", err)
	}

	log.Printf("[starling-lite] version=%s mode=%s transport=%s registry=%s",
		version, cfg.Mode, cfg.Transport, cfg.RegistryPath)

	switch cfg.Transport {
	case "stdio":
		if err := srv.ServeStdio(); err != nil {
			log.Fatalf("[starling-lite] stdio: %v", err)
		}
	case "http":
		log.Printf("[starling-lite] listening on %s", cfg.Addr)
		if err := srv.ServeHTTP(cfg.Addr); err != nil {
			log.Fatalf("[starling-lite] http: %v", err)
		}
	default:
		log.Fatalf("[starling-lite] unknown transport %q", cfg.Transport)
	}
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// defaultAddr prefers PaaS PORT (Railway, Fly, Render, Heroku-style).
func defaultAddr() string {
	if v := strings.TrimSpace(os.Getenv("STARLING_ADDR")); v != "" {
		if strings.HasPrefix(v, ":") || strings.Contains(v, ":") {
			return v
		}
		return ":" + v
	}
	if p := strings.TrimSpace(os.Getenv("PORT")); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return ":8822"
}

// materializeKubeconfigB64 writes KUBECONFIG_B64 to a temp file for cluster tools.
// Leave unset on public demos so only scan/remediate work (no cloud dependency).
func materializeKubeconfigB64() (string, error) {
	raw := strings.TrimSpace(os.Getenv("KUBECONFIG_B64"))
	if raw == "" {
		return "", nil
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		// allow raw kubeconfig pasted without base64 (small files)
		if strings.Contains(raw, "apiVersion:") {
			b = []byte(raw)
		} else {
			return "", fmt.Errorf("decode: %w", err)
		}
	}
	dir := filepath.Join(os.TempDir(), "starling-lite")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "kubeconfig")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func discoverRegistry() string {
	candidates := []string{
		"mcp/tools/registry.json",
		"../tools/registry.json",
		"../../tools/registry.json",
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "registry.json"),
			filepath.Join(dir, "..", "tools", "registry.json"),
			filepath.Join(dir, "..", "..", "tools", "registry.json"),
		)
	}
	// walk up from cwd
	wd, _ := os.Getwd()
	for i := 0; i < 6 && wd != ""; i++ {
		candidates = append(candidates, filepath.Join(wd, "mcp", "tools", "registry.json"))
		wd = filepath.Dir(wd)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "mcp/tools/registry.json"
}
