package main

import (
	"fmt"
	"os"

	"github.com/klabo/tinyclaw/internal/cli"
)

func main() {
	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch cmd.Action {
	case cli.ActionVersion:
		fmt.Println(cli.Version)
	case cli.ActionTest:
		if err := cli.RunTest(cmd.ScenarioFile, cmd.ConfigFile); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case cli.ActionReplay:
		if err := cli.RunReplay(cmd.BundleDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
