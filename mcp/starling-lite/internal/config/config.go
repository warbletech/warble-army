package config

import "strings"

// Config is runtime configuration for Starling Lite.
type Config struct {
	Mode            string // readonly | readwrite
	Transport       string // stdio | http
	Addr            string
	Kubeconfig      string
	Namespace       string
	RegistryPath    string
	AllowNamespaces []string
	Version         string
	Revision        string // git commit of the build ("dev" when unknown)
}

// ParseCSV splits a comma-separated list, trimming empties.
func ParseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// NamespaceAllowed returns true if writes may target ns.
func (c Config) NamespaceAllowed(ns string) bool {
	if len(c.AllowNamespaces) == 0 {
		return true
	}
	for _, a := range c.AllowNamespaces {
		if a == ns || a == "*" {
			return true
		}
	}
	return false
}
