// Package config manages tinyclaw configuration from files, environment, and defaults.
package config

import (
	"encoding/json"
	"os"
)

// Config holds the tinyclaw runtime configuration.
type Config struct {
	Transport string `json:"transport"` // transport plugin name
	Harness   string `json:"harness"`   // harness plugin name
	Context   string `json:"context"`   // context strategy name
	LogLevel  string `json:"log_level"` // log level (debug, info, warn, error)
	BundleDir string `json:"bundle_dir"`
	MaxHops   int    `json:"max_hops"` // delegation loop prevention
}

// Defaults returns a Config with sensible defaults.
func Defaults() Config {
	return Config{
		Transport: "fake",
		Harness:   "replay",
		Context:   "openclaw",
		LogLevel:  "info",
		BundleDir: "bundles",
		MaxHops:   5,
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
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Defaults(), err
	}
	return cfg, nil
}

// FromEnv overlays environment variables onto the config.
func FromEnv(cfg Config) Config {
	if v := os.Getenv("TINYCLAW_TRANSPORT"); v != "" {
		cfg.Transport = v
	}
	if v := os.Getenv("TINYCLAW_HARNESS"); v != "" {
		cfg.Harness = v
	}
	if v := os.Getenv("TINYCLAW_CONTEXT"); v != "" {
		cfg.Context = v
	}
	if v := os.Getenv("TINYCLAW_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("TINYCLAW_BUNDLE_DIR"); v != "" {
		cfg.BundleDir = v
	}
	return cfg
}
