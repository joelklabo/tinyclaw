package claudecode

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestExecRunnerCompileCheck(t *testing.T) {
	var _ Runner = (*ExecRunner)(nil)
}

func TestNewExecRunner(t *testing.T) {
	r := NewExecRunner("/tmp")
	if r.WorkDir != "/tmp" {
		t.Fatalf("got workdir %q, want %q", r.WorkDir, "/tmp")
	}
	if r.Command != "claude" {
		t.Fatalf("got command %q, want %q", r.Command, "claude")
	}
	if len(r.Args) != 4 {
		t.Fatalf("got %d args, want 4", len(r.Args))
	}
}

func TestExecRunnerRunEcho(t *testing.T) {
	r := &ExecRunner{
		WorkDir: t.TempDir(),
		Command: "echo",
		Args:    []string{},
	}
	rc, err := r.Run(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != "-p hello world" {
		t.Fatalf("got %q, want %q", got, "-p hello world")
	}
}

func TestExecRunnerRunMissingBinary(t *testing.T) {
	r := &ExecRunner{
		WorkDir: t.TempDir(),
		Command: "nonexistent-binary-that-does-not-exist",
		Args:    []string{},
	}
	_, err := r.Run(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestExecRunnerCmdReaderClose(t *testing.T) {
	r := &ExecRunner{
		WorkDir: t.TempDir(),
		Command: "echo",
		Args:    []string{},
	}
	rc, err := r.Run(context.Background(), "test")
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	// Read all then close.
	_, _ = io.ReadAll(rc)
	if err := rc.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}
}

func TestExecRunnerContextCancel(t *testing.T) {
	r := &ExecRunner{
		WorkDir: t.TempDir(),
		Command: "sleep",
		Args:    []string{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	rc, err := r.Run(ctx, "10")
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	cancel()
	_, _ = io.ReadAll(rc)
	// Close should return an error since the process was killed.
	_ = rc.Close()
}

func TestExecRunnerLive(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to run live Claude Code exec tests")
	}
	r := NewExecRunner(".")
	rc, err := r.Run(context.Background(), "Say hello in one word")
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	_ = rc.Close()
	t.Logf("output: %s", data)
}
