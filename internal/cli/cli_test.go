package cli

import "testing"

func TestParseTest(t *testing.T) {
	cmd, err := Parse([]string{"test", "scenario.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionTest {
		t.Fatalf("expected action %q, got %q", ActionTest, cmd.Action)
	}
	if cmd.ScenarioFile != "scenario.yaml" {
		t.Fatalf("expected scenario file %q, got %q", "scenario.yaml", cmd.ScenarioFile)
	}
}

func TestParseTestNoFile(t *testing.T) {
	cmd, err := Parse([]string{"test"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionTest {
		t.Fatalf("expected action %q, got %q", ActionTest, cmd.Action)
	}
	if cmd.ScenarioFile != "" {
		t.Fatalf("expected empty scenario file, got %q", cmd.ScenarioFile)
	}
}

func TestParseReplay(t *testing.T) {
	cmd, err := Parse([]string{"replay", "--bundle", "/tmp/bundle-123"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionReplay {
		t.Fatalf("expected action %q, got %q", ActionReplay, cmd.Action)
	}
	if cmd.BundleDir != "/tmp/bundle-123" {
		t.Fatalf("expected bundle dir %q, got %q", "/tmp/bundle-123", cmd.BundleDir)
	}
}

func TestParseReplayMissingBundle(t *testing.T) {
	_, err := Parse([]string{"replay"})
	if err == nil {
		t.Fatal("expected error for missing --bundle")
	}
}

func TestParseVersion(t *testing.T) {
	cmd, err := Parse([]string{"version"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionVersion {
		t.Fatalf("expected action %q, got %q", ActionVersion, cmd.Action)
	}
}

func TestParseNoArgs(t *testing.T) {
	_, err := Parse([]string{})
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestParseUnknownCommand(t *testing.T) {
	_, err := Parse([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestParseReplayEmptyBundle(t *testing.T) {
	_, err := Parse([]string{"replay", "--bundle", ""})
	if err == nil {
		t.Fatal("expected error for empty --bundle value")
	}
}

func TestParseReplayBadFlag(t *testing.T) {
	_, err := Parse([]string{"replay", "--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for bad flag")
	}
}
