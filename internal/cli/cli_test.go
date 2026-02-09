package cli

import (
	"os"
	"testing"
)

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

func TestParseTestWithConfig(t *testing.T) {
	cmd, err := Parse([]string{"test", "--config", "my.yaml", "scenario.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.ConfigFile != "my.yaml" {
		t.Fatalf("expected config %q, got %q", "my.yaml", cmd.ConfigFile)
	}
	if cmd.ScenarioFile != "scenario.yaml" {
		t.Fatalf("expected scenario %q, got %q", "scenario.yaml", cmd.ScenarioFile)
	}
}

func TestParseTestBadFlag(t *testing.T) {
	_, err := Parse([]string{"test", "--bogus"})
	if err == nil {
		t.Fatal("expected error for bad flag")
	}
}

func TestParseRun(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	t.Setenv("NOSTR_RELAYS", "wss://relay.example.com")
	cmd, err := Parse([]string{"run"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionRun {
		t.Fatalf("expected action %q, got %q", ActionRun, cmd.Action)
	}
	if cmd.PrivateKey != "test-key" {
		t.Fatalf("expected private key %q, got %q", "test-key", cmd.PrivateKey)
	}
	if len(cmd.Relays) != 1 || cmd.Relays[0] != "wss://relay.example.com" {
		t.Fatalf("expected relays [wss://relay.example.com], got %v", cmd.Relays)
	}
	if cmd.WorkDir != "." {
		t.Fatalf("expected workdir %q, got %q", ".", cmd.WorkDir)
	}
	if cmd.SessionKey != "default" {
		t.Fatalf("expected session key %q, got %q", "default", cmd.SessionKey)
	}
}

func TestParseRunMultipleRelays(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	t.Setenv("NOSTR_RELAYS", "wss://relay1.example.com, wss://relay2.example.com")
	cmd, err := Parse([]string{"run"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Relays) != 2 {
		t.Fatalf("expected 2 relays, got %d: %v", len(cmd.Relays), cmd.Relays)
	}
}

func TestParseRunCustomWorkDir(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	t.Setenv("NOSTR_RELAYS", "wss://relay.example.com")
	cmd, err := Parse([]string{"run", "--workdir", "/tmp/project"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.WorkDir != "/tmp/project" {
		t.Fatalf("expected workdir %q, got %q", "/tmp/project", cmd.WorkDir)
	}
}

func TestParseRunWithConfig(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	t.Setenv("NOSTR_RELAYS", "wss://relay.example.com")
	cmd, err := Parse([]string{"run", "--config", "my.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.ConfigFile != "my.yaml" {
		t.Fatalf("expected config %q, got %q", "my.yaml", cmd.ConfigFile)
	}
}

func TestParseRunCustomSessionKey(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	t.Setenv("NOSTR_RELAYS", "wss://relay.example.com")
	t.Setenv("NOSTR_SESSION_KEY", "my-session")
	cmd, err := Parse([]string{"run"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.SessionKey != "my-session" {
		t.Fatalf("expected session key %q, got %q", "my-session", cmd.SessionKey)
	}
}

func TestParseRunMissingPrivateKey(t *testing.T) {
	os.Unsetenv("NOSTR_PRIVATE_KEY")
	t.Setenv("NOSTR_RELAYS", "wss://relay.example.com")
	_, err := Parse([]string{"run"})
	if err == nil {
		t.Fatal("expected error for missing NOSTR_PRIVATE_KEY")
	}
}

func TestParseRunMissingRelays(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	os.Unsetenv("NOSTR_RELAYS")
	_, err := Parse([]string{"run"})
	if err == nil {
		t.Fatal("expected error for missing NOSTR_RELAYS")
	}
}

func TestParseRunBadFlag(t *testing.T) {
	t.Setenv("NOSTR_PRIVATE_KEY", "test-key")
	t.Setenv("NOSTR_RELAYS", "wss://relay.example.com")
	_, err := Parse([]string{"run", "--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for bad flag")
	}
}

func TestParseBrowse(t *testing.T) {
	cmd, err := Parse([]string{"browse"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionBrowse {
		t.Fatalf("expected action %q, got %q", ActionBrowse, cmd.Action)
	}
}

func TestParseBrowseWithFlags(t *testing.T) {
	cmd, err := Parse([]string{"browse", "--bundle-dir", "/tmp/bundles", "--bundle", "/tmp/bundles/bundle-1"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionBrowse {
		t.Fatalf("expected action %q, got %q", ActionBrowse, cmd.Action)
	}
	if cmd.BundleDir != "/tmp/bundles" {
		t.Fatalf("expected bundle dir %q, got %q", "/tmp/bundles", cmd.BundleDir)
	}
	if cmd.BundlePath != "/tmp/bundles/bundle-1" {
		t.Fatalf("expected bundle path %q, got %q", "/tmp/bundles/bundle-1", cmd.BundlePath)
	}
}

func TestParseBrowseBadFlag(t *testing.T) {
	_, err := Parse([]string{"browse", "--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for bad flag")
	}
}
