package registry

import (
	"encoding/json"
	"fmt"
	"os"
)

// Tool is one MCP tool definition from registry.json.
type Tool struct {
	Name        string         `json:"name"`
	Group       string         `json:"group"`
	Backend     string         `json:"backend"`
	Mode        string         `json:"mode"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// File is the on-disk registry shape.
type File struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Tools   []Tool `json:"tools"`
}

// Load reads and validates a registry JSON file.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry %s: %w", path, err)
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if len(f.Tools) == 0 {
		return nil, fmt.Errorf("registry has no tools")
	}
	return &f, nil
}

// Filtered returns tools allowed for the server mode.
func (f *File) Filtered(mode string) []Tool {
	out := make([]Tool, 0, len(f.Tools))
	for _, t := range f.Tools {
		if t.Mode == "readwrite" && mode != "readwrite" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// ByName looks up a tool definition.
func (f *File) ByName(name string) (Tool, bool) {
	for _, t := range f.Tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}
