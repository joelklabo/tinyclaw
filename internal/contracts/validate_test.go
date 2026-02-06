package contracts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllSchemasCompile(t *testing.T) {
	root := contractsRoot(t)
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking contracts dir: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no JSON schema files found in contracts/")
	}
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("reading %s: %v", f, err)
			}
			if err := CompileSchema(data); err != nil {
				t.Fatalf("compiling %s: %v", rel, err)
			}
		})
	}
}

func TestCompileSchemaValid(t *testing.T) {
	schema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`
	if err := CompileSchema([]byte(schema)); err != nil {
		t.Fatalf("expected valid schema to compile: %v", err)
	}
}

func TestCompileSchemaInvalidJSON(t *testing.T) {
	if err := CompileSchema([]byte("not json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCompileSchemaInvalidSchema(t *testing.T) {
	bad := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"not-a-type"}`
	if err := CompileSchema([]byte(bad)); err == nil {
		t.Fatal("expected error for invalid schema type")
	}
}

func TestLoadAll(t *testing.T) {
	root := contractsRoot(t)
	schemas, err := LoadAll(root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(schemas) == 0 {
		t.Fatal("expected at least one schema")
	}
	for name, s := range schemas {
		if s == nil {
			t.Errorf("schema %q is nil", name)
		}
	}
}

func TestLoadAllBadDir(t *testing.T) {
	_, err := LoadAll("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestLoadAllBadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAll(dir)
	if err == nil {
		t.Fatal("expected error for bad JSON file")
	}
}

func TestLoadAllInvalidSchema(t *testing.T) {
	dir := t.TempDir()
	bad := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"not-a-type"}`
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(bad), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadAll(dir)
	if err == nil {
		t.Fatal("expected error for invalid schema")
	}
}

func TestLoadAllUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unreadable.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0644)
	_, err := LoadAll(dir)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestLoadAllSkipsNonJSON(t *testing.T) {
	dir := t.TempDir()
	// Write a non-JSON file and a valid schema
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	schema := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`
	if err := os.WriteFile(filepath.Join(dir, "valid.json"), []byte(schema), 0644); err != nil {
		t.Fatal(err)
	}
	schemas, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected 1 schema, got %d", len(schemas))
	}
}

func TestLoadAllEmptyDir(t *testing.T) {
	dir := t.TempDir()
	schemas, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(schemas) != 0 {
		t.Fatalf("expected 0 schemas, got %d", len(schemas))
	}
}

// contractsRoot finds the contracts/ directory relative to the repo root.
func contractsRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "contracts")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod)")
		}
		dir = parent
	}
}
