package runner

import (
	"errors"
	"strings"
	"testing"
)

// recordingRunner captures every invocation instead of running commands.
type recordingRunner struct {
	calls   []call
	failOn  map[string]error
	stdoutW *strings.Builder
	stderrW *strings.Builder
}

type call struct {
	name string
	args []string
}

func newRecordingRunner() *recordingRunner {
	return &recordingRunner{
		failOn:  make(map[string]error),
		stdoutW: &strings.Builder{},
		stderrW: &strings.Builder{},
	}
}

func (r *recordingRunner) Run(name string, args ...string) error {
	r.calls = append(r.calls, call{name: name, args: append([]string(nil), args...)})
	if err, ok := r.failOn[name]; ok {
		return err
	}
	return nil
}

func TestRunnerFunc(t *testing.T) {
	t.Parallel()
	called := false
	f := RunnerFunc(func(name string, args ...string) error {
		called = true
		if name != "echo" {
			t.Fatalf("name = %q, want %q", name, "echo")
		}
		if len(args) != 1 || args[0] != "hi" {
			t.Fatalf("args = %v, want [hi]", args)
		}
		return nil
	})
	if err := f.Run("echo", "hi"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("RunnerFunc was not invoked")
	}
}

func TestRunnerFuncPropagatesError(t *testing.T) {
	t.Parallel()
	want := errors.New("boom")
	f := RunnerFunc(func(name string, args ...string) error { return want })
	if err := f.Run("anything"); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestExecRunnerBuildsCommand(t *testing.T) {
	t.Parallel()
	// Use a command that exists on every test platform: "go" with "env".
	r := ExecRunner{}
	if err := r.Run("go", "env", "GOPATH"); err != nil {
		t.Fatalf("unexpected error running go env: %v", err)
	}
}

func TestExecRunnerReportsMissingBinary(t *testing.T) {
	t.Parallel()
	r := ExecRunner{}
	err := r.Run("this-binary-definitely-does-not-exist-chezget")
	if err == nil {
		t.Fatal("expected error for missing binary, got nil")
	}
}

func TestNewExecRunnerDefaultsToOsStreams(t *testing.T) {
	t.Parallel()
	r := NewExecRunner()
	if r.Stdout == nil {
		t.Fatal("Stdout should default to os.Stdout")
	}
	if r.Stderr == nil {
		t.Fatal("Stderr should default to os.Stderr")
	}
}

func TestRecordingRunnerCapturesCalls(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	_ = rr.Run("cargo", "install", "ripgrep")
	_ = rr.Run("go", "install", "example.com/pkg@latest")

	if len(rr.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(rr.calls))
	}
	if rr.calls[0].name != "cargo" {
		t.Fatalf("call[0].name = %q, want %q", rr.calls[0].name, "cargo")
	}
	if got, want := strings.Join(rr.calls[0].args, " "), "install ripgrep"; got != want {
		t.Fatalf("call[0].args = %q, want %q", got, want)
	}
	if rr.calls[1].name != "go" {
		t.Fatalf("call[1].name = %q, want %q", rr.calls[1].name, "go")
	}
}

func TestRecordingRunnerFailsOnDemand(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	boom := errors.New("install failed")
	rr.failOn["cargo"] = boom
	if err := rr.Run("cargo", "install", "x"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
	// subsequent successful command still recorded
	if err := rr.Run("go", "install", "y"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rr.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(rr.calls))
	}
}
