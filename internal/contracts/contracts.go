// Package contracts loads and validates JSON Schemas from the contracts/ directory.
package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// CompileSchema validates that raw JSON compiles as a JSON Schema draft 2020-12.
func CompileSchema(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	c := jsonschema.NewCompiler()
	c.AddResource("schema.json", raw) //nolint:errcheck // AddResource accepts any JSON value
	_, err := c.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compiling schema: %w", err)
	}
	return nil
}

// LoadAll walks a directory tree and compiles every .json file as a JSON Schema.
// Returns a map from relative path to compiled schema.
func LoadAll(root string) (map[string]*jsonschema.Schema, error) {
	schemas := make(map[string]*jsonschema.Schema)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		rel, _ := filepath.Rel(root, path)
		c := jsonschema.NewCompiler()
		c.AddResource(rel, raw) //nolint:errcheck // AddResource accepts any JSON value
		schema, err := c.Compile(rel)
		if err != nil {
			return fmt.Errorf("compiling %s: %w", rel, err)
		}
		schemas[rel] = schema
		return nil
	})
	if err != nil {
		return nil, err
	}
	return schemas, nil
}
