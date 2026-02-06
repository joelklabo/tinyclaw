package main

import (
	"fmt"
	"os"

	"github.com/klabo/tinyclaw/internal/cli"
	"github.com/klabo/tinyclaw/internal/version"
)

func main() {
	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch cmd.Action {
	case cli.ActionVersion:
		fmt.Println(version.Version)
	case cli.ActionTest:
		fmt.Fprintln(os.Stderr, "test command not yet implemented")
		os.Exit(1)
	case cli.ActionReplay:
		fmt.Fprintln(os.Stderr, "replay command not yet implemented")
		os.Exit(1)
	}
}
