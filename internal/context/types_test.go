package context

import (
	"encoding/json"
	"testing"
)

func TestNewManifestSortsByPriorityThenName(t *testing.T) {
	items := []ContextItem{
		{Name: "z.go", Content: "z", Source: "pinned", Priority: 1},
		{Name: "a.go", Content: "a", Source: "pinned", Priority: 0},
		{Name: "m.go", Content: "m", Source: "pinned", Priority: 1},
		{Name: "b.go", Content: "b", Source: "pinned", Priority: 0},
	}
	m := NewManifest(items)

	want := []string{"a.go", "b.go", "m.go", "z.go"}
	if len(m.Items) != len(want) {
		t.Fatalf("items = %d, want %d", len(m.Items), len(want))
	}
	for i, w := range want {
		if m.Items[i].Name != w {
			t.Errorf("Items[%d].Name = %q, want %q", i, m.Items[i].Name, w)
		}
	}
}

func TestNewManifestDoesNotMutateInput(t *testing.T) {
	items := []ContextItem{
		{Name: "b.go", Priority: 1},
		{Name: "a.go", Priority: 0},
	}
	NewManifest(items)
	if items[0].Name != "b.go" {
		t.Fatal("NewManifest should not mutate the input slice")
	}
}

func TestManifestToJSONDeterministic(t *testing.T) {
	items := []ContextItem{
		{Name: "b.go", Content: "b", Source: "pinned", Priority: 1},
		{Name: "a.go", Content: "a", Source: "explicit", Priority: 0},
	}
	m := NewManifest(items)
	out1, err := m.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	out2, err := m.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) != string(out2) {
		t.Fatal("ToJSON is not deterministic")
	}

	// Verify it's valid JSON.
	var parsed Manifest
	if err := json.Unmarshal(out1, &parsed); err != nil {
		t.Fatalf("ToJSON produced invalid JSON: %v", err)
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("parsed items = %d, want 2", len(parsed.Items))
	}
	// First item should be a.go (priority 0).
	if parsed.Items[0].Name != "a.go" {
		t.Fatalf("first item = %q, want a.go", parsed.Items[0].Name)
	}
}

func TestNewManifestEmpty(t *testing.T) {
	m := NewManifest(nil)
	if len(m.Items) != 0 {
		t.Fatalf("items = %d, want 0", len(m.Items))
	}
	out, err := m.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"items":[]}` {
		t.Fatalf("empty manifest JSON = %s", out)
	}
}

func TestNewManifestNilItems(t *testing.T) {
	m := NewManifest([]ContextItem{})
	out, err := m.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"items":[]}` {
		t.Fatalf("empty manifest JSON = %s", out)
	}
}

func TestContextItemFields(t *testing.T) {
	item := ContextItem{
		Name:     "test.go",
		Content:  "package test",
		Source:   "pinned",
		Priority: 5,
	}
	if item.Name != "test.go" {
		t.Fatal("Name field")
	}
	if item.Content != "package test" {
		t.Fatal("Content field")
	}
	if item.Source != "pinned" {
		t.Fatal("Source field")
	}
	if item.Priority != 5 {
		t.Fatal("Priority field")
	}
}

func TestPlanFields(t *testing.T) {
	p := Plan{
		Providers: []string{"pinned", "explicit"},
		WorkDir:   "/tmp/work",
	}
	if len(p.Providers) != 2 {
		t.Fatal("Providers length")
	}
	if p.WorkDir != "/tmp/work" {
		t.Fatal("WorkDir field")
	}
}
