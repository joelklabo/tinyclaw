package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Transport != "fake" {
		t.Fatalf("Transport = %q, want fake", cfg.Transport)
	}
	if cfg.Harness != "replay" {
		t.Fatalf("Harness = %q, want replay", cfg.Harness)
	}
	if cfg.Context != "openclaw" {
		t.Fatalf("Context = %q, want openclaw", cfg.Context)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.BundleDir != "bundles" {
		t.Fatalf("BundleDir = %q, want bundles", cfg.BundleDir)
	}
	if cfg.MaxHops != 5 {
		t.Fatalf("MaxHops = %d, want 5", cfg.MaxHops)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg.Transport != "fake" {
		t.Fatal("should return defaults for missing file")
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"transport":"discord","harness":"claudecode","log_level":"debug","max_hops":3}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Transport != "discord" {
		t.Fatalf("Transport = %q, want discord", cfg.Transport)
	}
	if cfg.Harness != "claudecode" {
		t.Fatalf("Harness = %q, want claudecode", cfg.Harness)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.MaxHops != 3 {
		t.Fatalf("MaxHops = %d, want 3", cfg.MaxHops)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0644)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestFromEnv(t *testing.T) {
	cfg := Defaults()

	t.Setenv("TINYCLAW_TRANSPORT", "discord")
	t.Setenv("TINYCLAW_HARNESS", "claudecode")
	t.Setenv("TINYCLAW_CONTEXT", "custom")
	t.Setenv("TINYCLAW_LOG_LEVEL", "debug")
	t.Setenv("TINYCLAW_BUNDLE_DIR", "/tmp/bundles")

	cfg = FromEnv(cfg)
	if cfg.Transport != "discord" {
		t.Fatalf("Transport = %q, want discord", cfg.Transport)
	}
	if cfg.Harness != "claudecode" {
		t.Fatalf("Harness = %q, want claudecode", cfg.Harness)
	}
	if cfg.Context != "custom" {
		t.Fatalf("Context = %q, want custom", cfg.Context)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.BundleDir != "/tmp/bundles" {
		t.Fatalf("BundleDir = %q, want /tmp/bundles", cfg.BundleDir)
	}
}

func TestFromEnvNoOverride(t *testing.T) {
	cfg := Defaults()
	cfg = FromEnv(cfg)
	if cfg.Transport != "fake" {
		t.Fatal("should not override when env vars unset")
	}
}
