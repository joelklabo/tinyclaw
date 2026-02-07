package openclaw

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestGatherEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".openclaw"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestGatherNoDir(t *testing.T) {
	dir := t.TempDir()
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("expected nil, got %v", items)
	}
}

func TestGatherSingleFile(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocDir, "hello.txt"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	want := plugin.ContextItem{Name: "hello.txt", Content: "world", Source: "openclaw"}
	if items[0] != want {
		t.Fatalf("got %+v, want %+v", items[0], want)
	}
}

func TestGatherSortedOrder(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"c.txt", "a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(ocDir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	wantOrder := []string{"a.txt", "b.txt", "c.txt"}
	for i, want := range wantOrder {
		if items[i].Name != want {
			t.Fatalf("item[%d].Name = %q, want %q", i, items[i].Name, want)
		}
	}
}

func TestGatherTruncation(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bigContent := strings.Repeat("x", 200)
	if err := os.WriteFile(filepath.Join(ocDir, "big.txt"), []byte(bigContent), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{MaxCharsPerFile: 50})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].Content) != 50 {
		t.Fatalf("expected content length 50, got %d", len(items[0].Content))
	}
}

func TestGatherDefaultMaxChars(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	short := "hello"
	if err := os.WriteFile(filepath.Join(ocDir, "short.txt"), []byte(short), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Content != short {
		t.Fatalf("content = %q, want %q", items[0].Content, short)
	}
}

func TestGatherMissingFileMarker(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "missing.txt\n"
	if err := os.WriteFile(filepath.Join(ocDir, "manifest.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	// manifest.txt itself + missing.txt marker
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// sorted: manifest.txt, missing.txt
	if items[1].Name != "missing.txt" {
		t.Fatalf("expected missing.txt, got %q", items[1].Name)
	}
	if items[1].Content != "[missing file: missing.txt]" {
		t.Fatalf("unexpected content: %q", items[1].Content)
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
	if err := os.WriteFile(filepath.Join(ocDir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "file.txt" {
		t.Fatalf("expected file.txt, got %q", items[0].Name)
	}
}

func TestGatherContextCancelled(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New(Options{})
	_, err := p.Gather(ctx, dir)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestGatherReadDirError(t *testing.T) {
	dir := t.TempDir()
	// Create .openclaw as a file, not a directory, so ReadDir fails.
	if err := os.WriteFile(filepath.Join(dir, ".openclaw"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	_, err := p.Gather(context.Background(), dir)
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
	fpath := filepath.Join(ocDir, "secret.txt")
	if err := os.WriteFile(fpath, []byte("data"), 0o000); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	_, err := p.Gather(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
	if !strings.Contains(err.Error(), "secret.txt") {
		t.Fatalf("error should mention file name, got: %v", err)
	}
}

func TestGatherManifestEmptyLines(t *testing.T) {
	dir := t.TempDir()
	ocDir := filepath.Join(dir, ".openclaw")
	if err := os.Mkdir(ocDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Manifest with empty lines and whitespace-only lines.
	manifest := "\n  \nactual.txt\n\n  \n"
	if err := os.WriteFile(filepath.Join(ocDir, "manifest.txt"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(Options{})
	items, err := p.Gather(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	// manifest.txt itself + actual.txt (missing marker). Empty/whitespace lines skipped.
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// sorted: actual.txt, manifest.txt
	if items[0].Name != "actual.txt" {
		t.Fatalf("expected actual.txt first, got %q", items[0].Name)
	}
	if items[0].Content != "[missing file: actual.txt]" {
		t.Fatalf("unexpected content: %q", items[0].Content)
	}
}
