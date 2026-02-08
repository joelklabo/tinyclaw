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
	t.Setenv("DISCORD_TOKEN", "test-token")
	cmd, err := Parse([]string{"run", "--channel", "123456"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Action != ActionRun {
		t.Fatalf("expected action %q, got %q", ActionRun, cmd.Action)
	}
	if cmd.Token != "test-token" {
		t.Fatalf("expected token %q, got %q", "test-token", cmd.Token)
	}
	if len(cmd.Channels) != 1 || cmd.Channels[0] != "123456" {
		t.Fatalf("expected channels [123456], got %v", cmd.Channels)
	}
	if cmd.WorkDir != "." {
		t.Fatalf("expected workdir %q, got %q", ".", cmd.WorkDir)
	}
}

func TestParseRunMultipleChannels(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	cmd, err := Parse([]string{"run", "--channel", "111", "--channel", "222"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(cmd.Channels))
	}
}

func TestParseRunCustomWorkDir(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	cmd, err := Parse([]string{"run", "--channel", "123", "--workdir", "/tmp/project"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.WorkDir != "/tmp/project" {
		t.Fatalf("expected workdir %q, got %q", "/tmp/project", cmd.WorkDir)
	}
}

func TestParseRunWithConfig(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	cmd, err := Parse([]string{"run", "--channel", "123", "--config", "my.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.ConfigFile != "my.yaml" {
		t.Fatalf("expected config %q, got %q", "my.yaml", cmd.ConfigFile)
	}
}

func TestParseRunMissingToken(t *testing.T) {
	os.Unsetenv("DISCORD_TOKEN")
	_, err := Parse([]string{"run", "--channel", "123"})
	if err == nil {
		t.Fatal("expected error for missing DISCORD_TOKEN")
	}
}

func TestParseRunMissingChannel(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	_, err := Parse([]string{"run"})
	if err == nil {
		t.Fatal("expected error for missing --channel")
	}
}

func TestParseRunBadFlag(t *testing.T) {
	t.Setenv("DISCORD_TOKEN", "test-token")
	_, err := Parse([]string{"run", "--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for bad flag")
	}
}
