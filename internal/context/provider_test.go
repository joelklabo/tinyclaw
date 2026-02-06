package context

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestPinnedProviderName(t *testing.T) {
	p := &PinnedProvider{}
	if p.Name() != "pinned" {
		t.Fatalf("Name() = %q, want pinned", p.Name())
	}
}

func TestPinnedProviderGather(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bbb"), 0644)

	p := &PinnedProvider{Paths: []string{"a.txt", "b.txt"}}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Content != "aaa" {
		t.Fatalf("items[0].Content = %q, want aaa", items[0].Content)
	}
	if items[1].Name != "b.txt" {
		t.Fatalf("items[1].Name = %q, want b.txt", items[1].Name)
	}
}

func TestPinnedProviderGatherMissingFile(t *testing.T) {
	p := &PinnedProvider{Paths: []string{"missing.txt"}}
	_, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPinnedProviderGatherEmpty(t *testing.T) {
	p := &PinnedProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestAttachmentProviderName(t *testing.T) {
	p := &AttachmentProvider{}
	if p.Name() != "attachment" {
		t.Fatalf("Name() = %q, want attachment", p.Name())
	}
}

func TestAttachmentProviderGather(t *testing.T) {
	p := &AttachmentProvider{}
	req := plugin.ContextRequest{
		Hints: map[string]any{
			"attachments": []any{
				map[string]any{"name": "file.txt", "content": "hello"},
				map[string]any{"name": "img.png", "content": "data"},
			},
		},
	}
	items, err := p.Gather(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Name != "file.txt" {
		t.Fatalf("items[0].Name = %q, want file.txt", items[0].Name)
	}
}

func TestAttachmentProviderNoAttachments(t *testing.T) {
	p := &AttachmentProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{
		Hints: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("items = %v, want nil", items)
	}
}

func TestAttachmentProviderBadType(t *testing.T) {
	p := &AttachmentProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{
		Hints: map[string]any{
			"attachments": "not a list",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("items = %v, want nil", items)
	}
}

func TestAttachmentProviderBadElement(t *testing.T) {
	p := &AttachmentProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{
		Hints: map[string]any{
			"attachments": []any{
				"not a map",
				map[string]any{"name": "ok.txt", "content": "yes"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
}

func TestAttachmentProviderEmptyName(t *testing.T) {
	p := &AttachmentProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{
		Hints: map[string]any{
			"attachments": []any{
				map[string]any{"name": "", "content": "data"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestExplicitProviderName(t *testing.T) {
	p := &ExplicitProvider{}
	if p.Name() != "explicit" {
		t.Fatalf("Name() = %q, want explicit", p.Name())
	}
}

func TestExplicitProviderGather(t *testing.T) {
	p := &ExplicitProvider{
		Files: map[string]string{
			"main.go": "package main",
		},
	}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Name != "main.go" {
		t.Fatalf("items[0].Name = %q, want main.go", items[0].Name)
	}
}

func TestExplicitProviderGatherEmpty(t *testing.T) {
	p := &ExplicitProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestHeuristicProviderName(t *testing.T) {
	p := &HeuristicProvider{}
	if p.Name() != "heuristic" {
		t.Fatalf("Name() = %q, want heuristic", p.Name())
	}
}

func TestHeuristicProviderGather(t *testing.T) {
	p := &HeuristicProvider{}
	items, err := p.Gather(context.Background(), plugin.ContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("items = %v, want nil", items)
	}
}
