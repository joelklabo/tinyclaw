// Package context provides the context building system for tinyclaw.
// It defines context items, manifests, and providers that assemble
// workspace context for agent runs.
package context

import (
	"encoding/json"
	"sort"
)

// ContextItem is a single item in a context manifest.
type ContextItem struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Source   string `json:"source"`
	Priority int    `json:"priority"`
}

// Plan describes what files and context to gather.
type Plan struct {
	Providers []string `json:"providers"`
	WorkDir   string   `json:"work_dir"`
}

// Manifest is a deterministic collection of context items.
type Manifest struct {
	Items []ContextItem `json:"items"`
}

// NewManifest creates a Manifest from items, sorted for deterministic output.
func NewManifest(items []ContextItem) Manifest {
	sorted := make([]ContextItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].Name < sorted[j].Name
	})
	return Manifest{Items: sorted}
}

// ToJSON produces deterministic JSON output. Same inputs always yield
// byte-identical output.
func (m Manifest) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
