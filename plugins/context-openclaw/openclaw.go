// Package openclaw implements a ContextProvider that reads bootstrap files
// from a .openclaw directory.
package openclaw

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/klabo/tinyclaw/internal/plugin"
)

const defaultMaxChars = 10000

// Options configures the openclaw context provider.
type Options struct {
	MaxCharsPerFile int
}

// Provider implements plugin.ContextProvider for .openclaw directories.
type Provider struct {
	maxChars int
}

// New creates a new openclaw ContextProvider.
func New(opts Options) *Provider {
	max := opts.MaxCharsPerFile
	if max <= 0 {
		max = defaultMaxChars
	}
	return &Provider{maxChars: max}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openclaw"
}

// Gather reads files from the .openclaw directory and returns ContextItems.
// Files referenced in manifest.txt but not present get missing-file markers.
// Results are sorted by name for deterministic ordering.
func (p *Provider) Gather(ctx context.Context, req plugin.ContextRequest) ([]plugin.ContextItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ocDir := filepath.Join(req.WorkDir, ".openclaw")
	entries, err := os.ReadDir(ocDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Collect all regular files
	files := make(map[string]string)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		data, err := os.ReadFile(filepath.Join(ocDir, name))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		content := string(data)
		if len(content) > p.maxChars {
			content = content[:p.maxChars]
		}
		files[name] = content
		names = append(names, name)
	}

	// Check manifest.txt for referenced files
	if manifest, ok := files["manifest.txt"]; ok {
		scanner := bufio.NewScanner(strings.NewReader(manifest))
		for scanner.Scan() {
			ref := strings.TrimSpace(scanner.Text())
			if ref == "" {
				continue
			}
			if _, exists := files[ref]; !exists {
				files[ref] = fmt.Sprintf("[missing file: %s]", ref)
				names = append(names, ref)
			}
		}
	}

	sort.Strings(names)

	items := make([]plugin.ContextItem, 0, len(names))
	for _, name := range names {
		items = append(items, plugin.ContextItem{
			Name:    name,
			Content: files[name],
			Source:  "openclaw",
		})
	}
	return items, nil
}
