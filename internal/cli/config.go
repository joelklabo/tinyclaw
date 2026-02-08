package cli

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the tinyclaw runtime configuration.
type Config struct {
	LogLevel     string `yaml:"log_level"`
	BundleDir    string `yaml:"bundle_dir"`
	SystemPrompt string `yaml:"system_prompt"`
}

// Defaults returns a Config with sensible defaults.
func Defaults() Config {
	return Config{
		LogLevel:  "info",
		BundleDir: "bundles",
	}
}

// Load reads config from the given file path, falling back to defaults for missing fields.
func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Defaults(), err
	}
	return cfg, nil
}

// ParseLogLevel converts a string log level to slog.Level.
// Unknown values default to slog.LevelInfo.
func ParseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// FromEnv overlays environment variables onto the config.
func FromEnv(cfg Config) Config {
	if v := os.Getenv("TINYCLAW_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("TINYCLAW_BUNDLE_DIR"); v != "" {
		cfg.BundleDir = v
	}
	return cfg
}
