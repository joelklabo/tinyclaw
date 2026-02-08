// Package cli implements argument parsing for the tinyclaw CLI.
package cli

import (
	"flag"
	"fmt"
	"os"
)

// Action identifies the CLI command to run.
type Action string

const (
	ActionTest    Action = "test"
	ActionReplay  Action = "replay"
	ActionRun     Action = "run"
	ActionVersion Action = "version"
)

// Command holds the parsed CLI command and its parameters.
type Command struct {
	Action       Action
	ScenarioFile string   // test: optional scenario file
	BundleDir    string   // replay: required bundle directory
	ConfigFile   string   // test/run: optional config file path
	Token        string   // run: Discord bot token (from env)
	Channels     []string // run: Discord channel IDs
	WorkDir      string   // run: working directory for Claude Code
}

// Parse parses the given args (os.Args[1:] typically) into a Command.
func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, fmt.Errorf("usage: tinyclaw <test|replay|run|version>")
	}

	switch args[0] {
	case "test":
		return parseTest(args[1:])
	case "replay":
		return parseReplay(args[1:])
	case "run":
		return parseRun(args[1:])
	case "version":
		return Command{Action: ActionVersion}, nil
	default:
		return Command{}, fmt.Errorf("unknown command %q", args[0])
	}
}

func parseTest(args []string) (Command, error) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	config := fs.String("config", "", "config file path")
	if err := fs.Parse(args); err != nil {
		return Command{}, err
	}
	cmd := Command{Action: ActionTest, ConfigFile: *config}
	if fs.NArg() > 0 {
		cmd.ScenarioFile = fs.Arg(0)
	}
	return cmd, nil
}

func parseReplay(args []string) (Command, error) {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	bundle := fs.String("bundle", "", "bundle directory to replay")
	if err := fs.Parse(args); err != nil {
		return Command{}, err
	}
	if *bundle == "" {
		return Command{}, fmt.Errorf("replay requires --bundle <dir>")
	}
	return Command{Action: ActionReplay, BundleDir: *bundle}, nil
}

// channelList is a flag.Value that collects multiple --channel flags.
type channelList []string

func (c *channelList) String() string { return fmt.Sprintf("%v", *c) }
func (c *channelList) Set(v string) error {
	*c = append(*c, v)
	return nil
}

func parseRun(args []string) (Command, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var channels channelList
	fs.Var(&channels, "channel", "Discord channel ID (repeatable)")
	workDir := fs.String("workdir", ".", "working directory for Claude Code")
	config := fs.String("config", "", "config file path")
	if err := fs.Parse(args); err != nil {
		return Command{}, err
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return Command{}, fmt.Errorf("run requires DISCORD_TOKEN environment variable")
	}
	if len(channels) == 0 {
		return Command{}, fmt.Errorf("run requires at least one --channel <id>")
	}

	return Command{
		Action:     ActionRun,
		Token:      token,
		Channels:   []string(channels),
		WorkDir:    *workDir,
		ConfigFile: *config,
	}, nil
}
