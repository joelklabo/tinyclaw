// Package replay loads and validates bundle structures for deterministic replay.
package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BundleMeta is the metadata from run.json.
type BundleMeta struct {
	ID        string `json:"id"`
	StartTime string `json:"start_time"`
	Scenario  string `json:"scenario"`
	Status    string `json:"status"`
	EndTime   string `json:"end_time,omitempty"`
}

// BundleInfo describes the structure and contents of a bundle.
type BundleInfo struct {
	Dir   string
	Meta  BundleMeta
	Files []string
}

// requiredFiles lists the files that should exist in a valid bundle (excluding FAIL.md).
var requiredFiles = []string{
	"run.json",
}

// optionalFiles lists files that may exist in a bundle.
var optionalFiles = []string{
	"frames.jsonl",
	"events.jsonl",
	"transitions.jsonl",
	"ctx.json",
	"transport.jsonl",
	"logs.jsonl",
	"FAIL.md",
}

// LoadBundle reads a bundle directory and returns its metadata and file listing.
func LoadBundle(dir string) (*BundleInfo, error) {
	metaPath := filepath.Join(dir, "run.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("replay: read run.json: %w", err)
	}

	var meta BundleMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("replay: parse run.json: %w", err)
	}

	if meta.ID == "" {
		return nil, fmt.Errorf("replay: run.json missing id")
	}

	files := listBundleFiles(dir)

	return &BundleInfo{
		Dir:   dir,
		Meta:  meta,
		Files: files,
	}, nil
}

func listBundleFiles(dir string) []string {
	var found []string
	all := append(requiredFiles, optionalFiles...)
	for _, name := range all {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			found = append(found, name)
		}
	}
	return found
}

// Validate checks that a bundle has the minimum required files.
func Validate(info *BundleInfo) error {
	fileSet := make(map[string]bool, len(info.Files))
	for _, f := range info.Files {
		fileSet[f] = true
	}
	for _, req := range requiredFiles {
		if !fileSet[req] {
			return fmt.Errorf("replay: missing required file %s", req)
		}
	}
	if info.Meta.Status == "fail" || info.Meta.Status == "error" {
		if !fileSet["FAIL.md"] {
			return fmt.Errorf("replay: failed run missing FAIL.md")
		}
	}
	return nil
}
