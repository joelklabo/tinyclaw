package cli

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsReturnsSensibleValues(t *testing.T) {
	cfg := Defaults()
	if cfg.BundleDir != "bundles" {
		t.Fatalf("expected BundleDir %q, got %q", "bundles", cfg.BundleDir)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("expected LogLevel %q, got %q", "info", cfg.LogLevel)
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "log_level: debug\nbundle_dir: /tmp/out\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected LogLevel %q, got %q", "debug", cfg.LogLevel)
	}
	if cfg.BundleDir != "/tmp/out" {
		t.Fatalf("expected BundleDir %q, got %q", "/tmp/out", cfg.BundleDir)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	defaults := Defaults()
	if cfg != defaults {
		t.Fatalf("expected defaults %+v, got %+v", defaults, cfg)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(":::bad yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("log_level: debug\n"), 0644); err != nil {
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

func TestFromEnvOverlays(t *testing.T) {
	t.Setenv("TINYCLAW_BUNDLE_DIR", "/env/bundles")
	t.Setenv("TINYCLAW_LOG_LEVEL", "debug")

	cfg := FromEnv(Defaults())
	if cfg.BundleDir != "/env/bundles" {
		t.Fatalf("expected BundleDir %q, got %q", "/env/bundles", cfg.BundleDir)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected LogLevel %q, got %q", "debug", cfg.LogLevel)
	}
}

func TestFromEnvNoOverrideWhenUnset(t *testing.T) {
	t.Setenv("TINYCLAW_BUNDLE_DIR", "")
	t.Setenv("TINYCLAW_LOG_LEVEL", "")

	defaults := Defaults()
	cfg := FromEnv(defaults)
	if cfg.BundleDir != defaults.BundleDir {
		t.Fatalf("expected BundleDir %q, got %q", defaults.BundleDir, cfg.BundleDir)
	}
	if cfg.LogLevel != defaults.LogLevel {
		t.Fatalf("expected LogLevel %q, got %q", defaults.LogLevel, cfg.LogLevel)
	}
}

func TestLoadConfigWithSystemPrompt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "log_level: info\nbundle_dir: bundles\nsystem_prompt: \"You are a helpful bot.\"\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SystemPrompt != "You are a helpful bot." {
		t.Fatalf("expected SystemPrompt %q, got %q", "You are a helpful bot.", cfg.SystemPrompt)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}
	for _, tt := range tests {
		got := ParseLogLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
