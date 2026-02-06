package openclaw

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestName(t *testing.T) {
	p := New(Options{})
	if p.Name() != "openclaw" {
		t.Fatalf("expected name %q, got %q", "openclaw", p.Name())
	}
}

func TestGatherEmpty(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestGatherNoOpenclawDir(t *testing.T) {
	dir := t.TempDir()
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items when .openclaw missing, got %d", len(items))
	}
}

func TestGatherSingleFile(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "hello world"
	if err := os.WriteFile(filepath.Join(ocDir, "README.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "README.md" {
		t.Fatalf("expected name %q, got %q", "README.md", items[0].Name)
	}
	if items[0].Content != content {
		t.Fatalf("expected content %q, got %q", content, items[0].Content)
	}
	if items[0].Source != "openclaw" {
		t.Fatalf("expected source %q, got %q", "openclaw", items[0].Source)
	}
}

func TestGatherSortedOrder(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"c.txt", "a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(ocDir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	expected := []string{"a.txt", "b.txt", "c.txt"}
	for i, name := range expected {
		if items[i].Name != name {
			t.Fatalf("item %d: expected %q, got %q", i, name, items[i].Name)
		}
	}
}

func TestGatherTruncation(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(ocDir, "big.txt"), long, 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{MaxCharsPerFile: 50})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].Content) != 50 {
		t.Fatalf("expected truncated to 50, got %d", len(items[0].Content))
	}
}

func TestGatherDefaultMaxChars(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// File under default limit should not be truncated
	content := "short content"
	if err := os.WriteFile(filepath.Join(ocDir, "small.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Content != content {
		t.Fatalf("expected %q, got %q", content, items[0].Content)
	}
}

func TestGatherMissingFileMarker(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a bootstrap file that references a missing file via special format
	// The .openclaw/manifest.txt lists expected files
	manifest := "present.txt\nmissing.txt\n"
	if err := os.WriteFile(filepath.Join(ocDir, "manifest.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "present.txt"), []byte("here"), 0o644); err != nil {
		t.Fatal(err)
	}
	// missing.txt does NOT exist

	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// Should have manifest.txt, present.txt, and a missing-file marker for missing.txt
	found := false
	for _, item := range items {
		if item.Name == "missing.txt" {
			found = true
			if item.Content != "[missing file: missing.txt]" {
				t.Fatalf("expected missing marker, got %q", item.Content)
			}
		}
	}
	if !found {
		t.Fatal("expected missing file marker for missing.txt")
	}
}

func TestGatherSkipsSubdirectories(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(ocDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "file.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item (skipping subdir), got %d", len(items))
	}
}

func TestGatherContextCancelled(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := New(Options{})
	_, err := p.Gather(ctx, plugin.ContextRequest{WorkDir: dir})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGatherReadDirError(t *testing.T) {
	// Point WorkDir at a file (not a dir) to trigger a non-NotExist ReadDir error
	dir := t.TempDir()
	notADir := filepath.Join(dir, ".openclaw")
	if err := os.WriteFile(notADir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	_, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err == nil {
		t.Fatal("expected error when .openclaw is a file")
	}
}

func TestGatherUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(ocDir, "noperm.txt")
	if err := os.WriteFile(f, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Remove read permission
	if err := os.Chmod(f, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(f, 0o644) })

	p := New(Options{})
	_, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestGatherManifestEmptyLines(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Manifest with empty lines and whitespace-only lines
	manifest := "file.txt\n\n  \nother.txt\n"
	if err := os.WriteFile(filepath.Join(ocDir, "manifest.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "file.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), plugin.ContextRequest{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// Should have: file.txt, manifest.txt, other.txt (missing marker)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(items), items)
	}
}
