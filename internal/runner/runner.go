// Package runner provides an abstraction over spawning external commands so
// that callers can be unit-tested without touching the real process tree.
package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Runner executes a command identified by name with the supplied arguments.
// Implementations may forward stdout/stderr to the parent process, capture
// output, or record the invocations for inspection in tests.
type Runner interface {
	// Run executes the command. It returns the combined exit error of the
	// command, if any.
	Run(name string, args ...string) error
}

// RunnerFunc is a convenience type that lets any function satisfy the Runner
// interface.
type RunnerFunc func(name string, args ...string) error

// Run calls the underlying function.
func (f RunnerFunc) Run(name string, args ...string) error {
	return f(name, args...)
}

// ExecRunner runs commands through os/exec, streaming their stdout and stderr
// to the provided writers (which default to the parent process streams when
// nil). It is the production Runner implementation.
type ExecRunner struct {
	Stdout io.Writer
	Stderr io.Writer
}

// NewExecRunner returns an ExecRunner that streams command output to the
// parent process stdout and stderr.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run executes the named command with the given arguments. Stdout and stderr of
// the child process are forwarded to the runner's writers. The returned error
// is non-nil when the command exits with a non-zero status or cannot be
// started.
func (r ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %q: %w", name, err)
	}
	return nil
}
