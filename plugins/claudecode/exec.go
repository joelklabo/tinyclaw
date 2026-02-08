package claudecode

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// ExecRunner implements Runner by spawning a Claude Code subprocess.
type ExecRunner struct {
	WorkDir string
	Command string
	Args    []string
}

// compile-time check
var _ Runner = (*ExecRunner)(nil)

// NewExecRunner creates an ExecRunner that runs claude in the given directory.
func NewExecRunner(workDir string) *ExecRunner {
	return &ExecRunner{
		WorkDir: workDir,
		Command: "claude",
		Args:    []string{"--output-format", "stream-json", "--print"},
	}
}

// Run spawns the claude process with the given prompt and returns its stdout.
func (r *ExecRunner) Run(ctx context.Context, prompt string) (io.ReadCloser, error) {
	args := append(r.Args, "-p", prompt)
	cmd := exec.CommandContext(ctx, r.Command, args...)
	cmd.Dir = r.WorkDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claudecode: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("claudecode: start: %w", err)
	}
	return &cmdReader{pipe: stdout, cmd: cmd}, nil
}

// cmdReader wraps a stdout pipe and waits for the process on Close.
type cmdReader struct {
	pipe io.ReadCloser
	cmd  *exec.Cmd
}

func (r *cmdReader) Read(p []byte) (int, error) {
	return r.pipe.Read(p)
}

func (r *cmdReader) Close() error {
	_ = r.pipe.Close()
	return r.cmd.Wait()
}
