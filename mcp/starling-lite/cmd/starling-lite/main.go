package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/warble-tech/warble-army/mcp/starling-lite/internal/config"
	"github.com/warble-tech/warble-army/mcp/starling-lite/internal/server"
)

const version = "0.2.0"

func main() {
	var (
		mode       = flag.String("mode", "readonly", "Server mode: readonly | readwrite")
		transport  = flag.String("transport", "stdio", "MCP transport: stdio | http")
		addr       = flag.String("addr", ":8822", "HTTP listen address (transport=http)")
		kubeconfig = flag.String("kubeconfig", "", "Path to kubeconfig (default: env / ~/.kube/config)")
		namespace  = flag.String("namespace", "", "Restrict cluster tools to one namespace")
		registry   = flag.String("registry", "", "Path to mcp/tools/registry.json (default: auto-discover)")
		allowNS    = flag.String("allow-namespaces", "", "Comma-separated write allowlist (empty = all when readwrite)")
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
