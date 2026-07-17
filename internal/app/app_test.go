// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alexript/chezget/internal/config"
)

// recordingRunner records invocations and can fail per spec (last arg).
type recordingRunner struct {
	calls   []recCall
	failFor map[string]error
}

type recCall struct {
	name string
	args []string
}

func newRecordingRunner() *recordingRunner {
	return &recordingRunner{failFor: make(map[string]error)}
}

func (r *recordingRunner) Run(name string, args ...string) error {
	r.calls = append(r.calls, recCall{name: name, args: append([]string(nil), args...)})
	if len(args) > 0 {
		if err, ok := r.failFor[args[len(args)-1]]; ok {
			return err
		}
	}
	return nil
}

func TestRunMissingConfigReturns1(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errw bytes.Buffer
	a := New(Options{Stdout: &out, Stderr: &errw})
	code := a.Run()
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errw.String(), "no configuration found") {
		t.Fatalf("stderr = %q", errw.String())
	}
}

func TestRunConfigPathMissingFile(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	a := New(Options{
		ConfigPath: filepath.Join(t.TempDir(), "missing.ini"),
		Stdout:     &out,
		Stderr:     &errw,
	})
	if code := a.Run(); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestRunEmptyConfigReturns1(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, "# empty\n")
	var out, errw bytes.Buffer
	a := New(Options{ConfigPath: path, Stdout: &out, Stderr: &errw})
	if code := a.Run(); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestRunSuccessWithRecordingRunner(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `[go]
github.com/foo/bar@latest
golang.org/x/tools/cmd/goimports

[rust]
ripgrep
kotlin-lsp
`)
	rr := newRecordingRunner()
	var out, errw bytes.Buffer
	a := New(Options{
		ConfigPath: path,
		OS:         "linux",
		Runner:     rr,
		Stdout:     &out,
		Stderr:     &errw,
	})
	if code := a.Run(); code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, errw.String())
	}
	if len(rr.calls) != 4 {
		t.Fatalf("calls = %d, want 4", len(rr.calls))
	}
	// Order: go installs first, then cargo.
	if rr.calls[0].name != "go" || rr.calls[0].args[1] != "github.com/foo/bar@latest" {
		t.Fatalf("call[0] = %+v", rr.calls[0])
	}
	if rr.calls[1].name != "go" || rr.calls[1].args[1] != "golang.org/x/tools/cmd/goimports" {
		t.Fatalf("call[1] = %+v", rr.calls[1])
	}
	if rr.calls[2].name != "cargo" || rr.calls[2].args[1] != "ripgrep" {
		t.Fatalf("call[2] = %+v", rr.calls[2])
	}
	if rr.calls[3].name != "cargo" || rr.calls[3].args[1] != "kotlin-lsp" {
		t.Fatalf("call[3] = %+v", rr.calls[3])
	}
	if !strings.Contains(out.String(), "installed 4 package(s)") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRunFailureReturns1(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `[rust]
bad-crate
`)
	rr := newRecordingRunner()
	rr.failFor["bad-crate"] = errors.New("install failed")
	var out, errw bytes.Buffer
	a := New(Options{
		ConfigPath: path,
		OS:         "linux",
		Runner:     rr,
		Stdout:     &out,
		Stderr:     &errw,
	})
	if code := a.Run(); code != 1 {
		t.Fatalf("code = %d, want 1; stderr=%q", code, errw.String())
	}
	if !strings.Contains(errw.String(), "bad-crate") {
		t.Fatalf("stderr = %q", errw.String())
	}
}

func TestRunOnlyGoSection(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `[go]
example.com/pkg
`)
	rr := newRecordingRunner()
	a := New(Options{ConfigPath: path, OS: "linux", Runner: rr, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if code := a.Run(); code != 0 {
		t.Fatal("expected success")
	}
	if len(rr.calls) != 1 || rr.calls[0].name != "go" {
		t.Fatalf("calls = %+v", rr.calls)
	}
}

func TestRunOnlyRustSection(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `[rust]
ripgrep
`)
	rr := newRecordingRunner()
	a := New(Options{ConfigPath: path, OS: "linux", Runner: rr, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if code := a.Run(); code != 0 {
		t.Fatal("expected success")
	}
	if len(rr.calls) != 1 || rr.calls[0].name != "cargo" {
		t.Fatalf("calls = %+v", rr.calls)
	}
}

func TestRunOSSpecificSectionsMerge(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `[go]
common-pkg
[go windows]
windows-pkg
[go linux]
linux-pkg
`)
	rr := newRecordingRunner()
	a := New(Options{ConfigPath: path, OS: "windows", Runner: rr, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if code := a.Run(); code != 0 {
		t.Fatal("expected success")
	}
	if len(rr.calls) != 2 {
		t.Fatalf("calls = %+v, want 2", rr.calls)
	}
	if rr.calls[0].args[1] != "common-pkg" {
		t.Fatalf("call[0] = %+v", rr.calls[0])
	}
	if rr.calls[1].args[1] != "windows-pkg" {
		t.Fatalf("call[1] = %+v", rr.calls[1])
	}
}

func TestRunOSSpecificSectionNonMatchReturns1(t *testing.T) {
	t.Parallel()
	path := writeConfig(t, `[go windows]
windows-only
`)
	rr := newRecordingRunner()
	var out, errw bytes.Buffer
	a := New(Options{ConfigPath: path, OS: "linux", Runner: rr, Stdout: &out, Stderr: &errw})
	if code := a.Run(); code != 1 {
		t.Fatalf("code = %d, want 1; stderr=%q", code, errw.String())
	}
	if len(rr.calls) != 0 {
		t.Fatalf("calls = %+v, want none", rr.calls)
	}
	if !strings.Contains(errw.String(), "no packages") && !strings.Contains(errw.String(), "contains no packages") {
		t.Fatalf("stderr = %q", errw.String())
	}
}

func TestRunOSDefaultsToRuntimeGOOS(t *testing.T) {
	t.Parallel()
	// Section targets the host OS under its runtime.GOOS name; with the
	// default (empty Options.OS) the run should pick it up.
	path := writeConfig(t, "[go "+runtime.GOOS+"]\nhost-only\n")
	rr := newRecordingRunner()
	a := New(Options{ConfigPath: path, Runner: rr, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if code := a.Run(); code != 0 {
		t.Fatal("expected success on host OS")
	}
	if len(rr.calls) != 1 || rr.calls[0].args[1] != "host-only" {
		t.Fatalf("calls = %+v", rr.calls)
	}
}

func TestNewFillsDefaultWriters(t *testing.T) {
	t.Parallel()
	a := New(Options{})
	if a.out != os.Stdout {
		t.Fatal("default stdout should be os.Stdout")
	}
	if a.err != os.Stderr {
		t.Fatal("default stderr should be os.Stderr")
	}
}

func TestHintPath(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/home")
	if got := hintPath(); got != filepath.Join("/xdg/home", "chezget", "config.ini") {
		t.Fatalf("hintPath = %q", got)
	}
}

func TestHintPathFallbackOnResolveError(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	_, err := config.ResolvePath()
	if err == nil {
		t.Skip("os.UserHomeDir() does not fail with empty HOME on this platform")
	}

	got := hintPath()
	want := "$XDG_CONFIG_HOME/chezget/config.ini"
	if got != want {
		t.Fatalf("hintPath = %q, want %q", got, want)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}
