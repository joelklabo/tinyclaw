package context

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// PinnedProvider returns fixed file contents from a list of paths.
type PinnedProvider struct {
	Paths []string
}

func (p *PinnedProvider) Name() string { return "pinned" }

func (p *PinnedProvider) Gather(_ context.Context, req plugin.ContextRequest) ([]plugin.ContextItem, error) {
	var items []plugin.ContextItem
	for _, path := range p.Paths {
		full := filepath.Join(req.WorkDir, path)
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("pinned: read %s: %w", path, err)
		}
		items = append(items, plugin.ContextItem{
			Name:    path,
			Content: string(data),
		})
	}
	return items, nil
}

// AttachmentProvider extracts attachments from inbound event data.
type AttachmentProvider struct{}

func (p *AttachmentProvider) Name() string { return "attachment" }

func (p *AttachmentProvider) Gather(_ context.Context, req plugin.ContextRequest) ([]plugin.ContextItem, error) {
	attachments, ok := req.Hints["attachments"]
	if !ok {
		return nil, nil
	}
	list, ok := attachments.([]any)
	if !ok {
		return nil, nil
	}
	var items []plugin.ContextItem
	for _, a := range list {
		m, ok := a.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		content, _ := m["content"].(string)
		if name == "" {
			continue
		}
		items = append(items, plugin.ContextItem{
			Name:    name,
			Content: content,
		})
	}
	return items, nil
}

// ExplicitProvider returns user-added files via /add command.
type ExplicitProvider struct {
	Files map[string]string // name -> content
}

func (p *ExplicitProvider) Name() string { return "explicit" }

func (p *ExplicitProvider) Gather(_ context.Context, _ plugin.ContextRequest) ([]plugin.ContextItem, error) {
	var items []plugin.ContextItem
	for name, content := range p.Files {
		items = append(items, plugin.ContextItem{
			Name:    name,
			Content: content,
		})
	}
	return items, nil
}

// HeuristicProvider is a placeholder that returns empty. Off by default.
type HeuristicProvider struct{}

func (p *HeuristicProvider) Name() string { return "heuristic" }

func (p *HeuristicProvider) Gather(_ context.Context, _ plugin.ContextRequest) ([]plugin.ContextItem, error) {
	return nil, nil
}
