package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BundleInfo describes the structure and contents of a bundle.
type BundleInfo struct {
	Dir   string
	Meta  Meta
	Files []string
}

// requiredFiles lists the files that should exist in a valid bundle.
var requiredFiles = []string{
	"run.json",
}

// LoadBundle reads a bundle directory and returns its metadata and file listing.
func LoadBundle(dir string) (*BundleInfo, error) {
	metaPath := filepath.Join(dir, "run.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("bundle: read run.json: %w", err)
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("bundle: parse run.json: %w", err)
	}

	if meta.ID == "" {
		return nil, fmt.Errorf("bundle: run.json missing id")
	}

	files := listBundleFiles(dir)

	return &BundleInfo{
		Dir:   dir,
		Meta:  meta,
		Files: files,
	}, nil
}

func listBundleFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() {
			found = append(found, e.Name())
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
			return fmt.Errorf("bundle: missing required file %s", req)
		}
	}
	if info.Meta.Status == "fail" || info.Meta.Status == "error" {
		if !fileSet["FAIL.md"] {
			return fmt.Errorf("bundle: failed run missing FAIL.md")
		}
	}
	return nil
}
