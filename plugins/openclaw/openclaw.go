// Package openclaw reads bootstrap context files from a .openclaw directory.
package openclaw

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

// Provider reads .openclaw directories for bootstrap context.
type Provider struct {
	maxChars int
}

// New creates a new openclaw Provider.
func New(opts Options) *Provider {
	max := opts.MaxCharsPerFile
	if max <= 0 {
		max = defaultMaxChars
	}
	return &Provider{maxChars: max}
}

// Gather reads files from the .openclaw directory and returns ContextItems.
func (p *Provider) Gather(ctx context.Context, workDir string) ([]plugin.ContextItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ocDir := filepath.Join(workDir, ".openclaw")
	entries, err := os.ReadDir(ocDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	files := make(map[string]string)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		f, err := os.Open(filepath.Join(ocDir, name))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		data, err := io.ReadAll(io.LimitReader(f, int64(p.maxChars)))
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		content := string(data)
		files[name] = content
		names = append(names, name)
	}

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
