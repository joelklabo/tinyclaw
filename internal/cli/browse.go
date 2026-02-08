package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/klabo/tinyclaw/internal/bundle"
)

// RunBrowse lists or inspects bundles.
func RunBrowse(cmd Command) error {
	bundleDir := cmd.BundleDir
	if bundleDir == "" {
		cfg, err := Load(cmd.ConfigFile)
		if err != nil {
			return err
		}
		cfg = FromEnv(cfg)
		bundleDir = cfg.BundleDir
	}

	if cmd.BundlePath != "" {
		return browseDetail(cmd.BundlePath)
	}
	return browseList(bundleDir)
}

func browseList(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("browse: %w", err)
	}

	type row struct {
		meta bundle.Meta
		dir  string
	}
	var rows []row
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "bundle-") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := bundle.LoadBundle(path)
		if err != nil {
			continue // skip unreadable bundles
		}
		rows = append(rows, row{meta: info.Meta, dir: path})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].meta.StartTime < rows[j].meta.StartTime
	})

	if len(rows) == 0 {
		fmt.Println("No bundles found.")
		return nil
	}

	for _, r := range rows {
		status := strings.ToUpper(r.meta.Status)
		start := formatTime(r.meta.StartTime)
		dur := formatDuration(r.meta.StartTime, r.meta.EndTime)
		scenario := r.meta.Scenario
		id := r.meta.ID
		fmt.Printf("%-6s %-12s %s %6s  %s\n", status, id, start, dur, scenario)
	}
	return nil
}

func browseDetail(path string) error {
	info, err := bundle.LoadBundle(path)
	if err != nil {
		return fmt.Errorf("browse: %w", err)
	}
	fmt.Printf("Bundle: %s\n", info.Meta.ID)
	fmt.Printf("Status: %s\n", info.Meta.Status)
	fmt.Printf("Scenario: %s\n", info.Meta.Scenario)
	fmt.Printf("Start: %s\n", info.Meta.StartTime)
	if info.Meta.EndTime != "" {
		fmt.Printf("End: %s\n", info.Meta.EndTime)
	}
	fmt.Printf("Files:\n")
	for _, f := range info.Files {
		fmt.Printf("  %s\n", f)
	}
	return nil
}

func formatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("15:04:05")
}

func formatDuration(start, end string) string {
	s, err1 := time.Parse(time.RFC3339, start)
	e, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return "-"
	}
	d := e.Sub(s)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
