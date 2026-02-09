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
	ActionBrowse  Action = "browse"
	ActionVersion Action = "version"
)

// Command holds the parsed CLI command and its parameters.
type Command struct {
	Action       Action
	ScenarioFile string // test: optional scenario file
	BundleDir    string // replay/browse: bundle directory
	BundlePath   string // browse: specific bundle to inspect
	ConfigFile   string // test/run: optional config file path
	PrivateKey   string // run: Nostr private key (from env)
	SessionKey   string // run: Nostr session key (from env)
	Relays       []string // run: Nostr relay URLs
	WorkDir      string // run: working directory for Claude Code
}

// Parse parses the given args (os.Args[1:] typically) into a Command.
func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, fmt.Errorf("usage: tinyclaw <test|replay|run|browse|version>")
	}

	switch args[0] {
	case "test":
		return parseTest(args[1:])
	case "replay":
		return parseReplay(args[1:])
	case "run":
		return parseRun(args[1:])
	case "browse":
		return parseBrowse(args[1:])
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

func parseRun(args []string) (Command, error) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	workDir := fs.String("workdir", ".", "working directory for Claude Code")
	config := fs.String("config", "", "config file path")
	if err := fs.Parse(args); err != nil {
		return Command{}, err
	}

	privateKey := os.Getenv("NOSTR_PRIVATE_KEY")
	if privateKey == "" {
		return Command{}, fmt.Errorf("run requires NOSTR_PRIVATE_KEY environment variable")
	}
	relaysEnv := os.Getenv("NOSTR_RELAYS")
	if relaysEnv == "" {
		return Command{}, fmt.Errorf("run requires NOSTR_RELAYS environment variable")
	}
	sessionKey := os.Getenv("NOSTR_SESSION_KEY")
	if sessionKey == "" {
		sessionKey = "default"
	}

	return Command{
		Action:     ActionRun,
		PrivateKey: privateKey,
		SessionKey: sessionKey,
		Relays:     parseRelayList(relaysEnv),
		WorkDir:    *workDir,
		ConfigFile: *config,
	}, nil
}

func parseBrowse(args []string) (Command, error) {
	fs := flag.NewFlagSet("browse", flag.ContinueOnError)
	bundleDir := fs.String("bundle-dir", "", "bundle directory")
	bundlePath := fs.String("bundle", "", "specific bundle to inspect")
	if err := fs.Parse(args); err != nil {
		return Command{}, err
	}
	return Command{
		Action:     ActionBrowse,
		BundleDir:  *bundleDir,
		BundlePath: *bundlePath,
	}, nil
}
