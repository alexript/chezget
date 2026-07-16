// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package installer

import (
	"errors"
	"testing"

	"github.com/alexript/chezget/internal/runner"
)

// recordingRunner is a tiny runner that records invocations and can be
// instructed to fail for specific specs. It is shared across all installer
// test files in this package.
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
	// key the failure on the last arg (the spec) for convenience.
	if len(args) > 0 {
		if err, ok := r.failFor[args[len(args)-1]]; ok {
			return err
		}
	}
	return nil
}

func TestRunAllOrder(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	installers := []Installer{
		NewGoInstaller(rr),
		NewRustInstaller(rr),
	}
	specs := map[string][]string{
		"go":   {"pkgA"},
		"rust": {"crateB"},
	}
	results := RunAll(installers, specs)
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Installer != "go" || results[0].Spec != "pkgA" {
		t.Fatalf("results[0] = %+v, want go/pkgA", results[0])
	}
	if results[1].Installer != "rust" || results[1].Spec != "crateB" {
		t.Fatalf("results[1] = %+v, want rust/crateB", results[1])
	}
}

func TestRunAllSkipsNilInstaller(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	installers := []Installer{nil, NewGoInstaller(rr)}
	specs := map[string][]string{"go": {"pkgA"}}
	results := RunAll(installers, specs)
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
}

func TestRunAllEmptySpecs(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	installers := []Installer{NewGoInstaller(rr), NewRustInstaller(rr)}
	results := RunAll(installers, nil)
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
}

func TestDefaultInstallers(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	installers := DefaultInstallers(rr)
	if len(installers) != 2 {
		t.Fatalf("len(installers) = %d, want 2", len(installers))
	}
	if installers[0].Name() != "go" {
		t.Fatalf("installers[0].Name() = %q, want go", installers[0].Name())
	}
	if installers[1].Name() != "rust" {
		t.Fatalf("installers[1].Name() = %q, want rust", installers[1].Name())
	}
}

func TestSectionNames(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	installers := DefaultInstallers(rr)
	got := SectionNames(installers)
	want := []string{"go", "rust"}
	if len(got) != len(want) {
		t.Fatalf("SectionNames = %v, want %v", got, want)
	}
	for i, name := range got {
		if name != want[i] {
			t.Fatalf("SectionNames[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestSectionNamesSkipsNil(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	installers := []Installer{nil, NewGoInstaller(rr), nil, NewRustInstaller(rr)}
	got := SectionNames(installers)
	want := []string{"go", "rust"}
	if len(got) != len(want) {
		t.Fatalf("SectionNames = %v, want %v", got, want)
	}
}

func TestSummaryNoPackages(t *testing.T) {
	t.Parallel()
	if got := Summary(nil); got != "no packages to install" {
		t.Fatalf("Summary() = %q", got)
	}
}

func TestSummaryAllOk(t *testing.T) {
	t.Parallel()
	results := []Result{
		{Installer: "go", Spec: "a"},
		{Installer: "go", Spec: "b"},
	}
	if got := Summary(results); got != "installed 2 package(s)" {
		t.Fatalf("Summary() = %q", got)
	}
}

func TestSummaryMixed(t *testing.T) {
	t.Parallel()
	results := []Result{
		{Installer: "go", Spec: "a"},
		{Installer: "rust", Spec: "b", Err: errors.New("nope")},
	}
	got := Summary(results)
	if got != "installed 1 package(s), 1 failed" {
		t.Fatalf("Summary() = %q", got)
	}
}

func TestResultFailed(t *testing.T) {
	t.Parallel()
	if (Result{}).Failed() {
		t.Fatal("zero Result should not be Failed")
	}
	if !(Result{Err: errors.New("x")}).Failed() {
		t.Fatal("Result with Err should be Failed")
	}
}

// Ensure ExecRunner satisfies the Runner interface contract via the installer.
func TestInstallerAcceptsExecRunner(t *testing.T) {
	t.Parallel()
	var _ runner.Runner = runner.NewExecRunner()
	var _ Installer = NewGoInstaller(runner.NewExecRunner())
	var _ Installer = NewRustInstaller(runner.NewExecRunner())
}
