// Package cli implements argument parsing for the tinyclaw CLI.
package cli

import (
	"flag"
	"fmt"
)

// Action identifies the CLI command to run.
type Action string

const (
	ActionTest    Action = "test"
	ActionReplay  Action = "replay"
	ActionVersion Action = "version"
)

// Command holds the parsed CLI command and its parameters.
type Command struct {
	Action       Action
	ScenarioFile string // test: optional scenario file
	BundleDir    string // replay: required bundle directory
}

// Parse parses the given args (os.Args[1:] typically) into a Command.
func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{}, fmt.Errorf("usage: tinyclaw <test|replay|version>")
	}

	switch args[0] {
	case "test":
		return parseTest(args[1:])
	case "replay":
		return parseReplay(args[1:])
	case "version":
		return Command{Action: ActionVersion}, nil
	default:
		return Command{}, fmt.Errorf("unknown command %q", args[0])
	}
}

func parseTest(args []string) (Command, error) {
	cmd := Command{Action: ActionTest}
	if len(args) > 0 {
		cmd.ScenarioFile = args[0]
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
