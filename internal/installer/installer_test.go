package installer

import (
	"errors"
	"testing"

	"github.com/alexript/chezget/internal/runner"
)

// recordingRunner is a tiny runner that records invocations and can be
// instructed to fail for specific specs. It mirrors the one in the runner
// tests but is local so installer tests stay self-contained.
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

func TestGoInstallerCommand(t *testing.T) {
	t.Parallel()
	g := GoInstaller{}
	name, args := g.Command("github.com/foo/bar@latest")
	if name != "go" {
		t.Fatalf("name = %q, want go", name)
	}
	if len(args) != 2 || args[0] != "install" || args[1] != "github.com/foo/bar@latest" {
		t.Fatalf("args = %v, want [install github.com/foo/bar@latest]", args)
	}
}

func TestGoInstallerName(t *testing.T) {
	t.Parallel()
	g := GoInstaller{}
	if g.Name() != "go" {
		t.Fatalf("Name = %q, want go", g.Name())
	}
}

func TestGoInstallerRunsInstallForEverySpec(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	g := NewGoInstaller(rr)
	results := g.Install([]string{
		"github.com/jesseduffield/lazygit@latest",
		"golang.org/x/tools/cmd/goimports",
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if len(rr.calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2", len(rr.calls))
	}
	for i, want := range []string{"install", "install"} {
		if rr.calls[i].name != "go" {
			t.Fatalf("call[%d].name = %q, want go", i, rr.calls[i].name)
		}
		if len(rr.calls[i].args) != 2 || rr.calls[i].args[0] != want {
			t.Fatalf("call[%d].args = %v", i, rr.calls[i].args)
		}
	}
	for _, r := range results {
		if r.Failed() {
			t.Fatalf("unexpected failure: %+v", r)
		}
		if r.Installer != "go" {
			t.Fatalf("result.Installer = %q, want go", r.Installer)
		}
	}
}

func TestGoInstallerContinuesAfterFailure(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	boom := errors.New("network down")
	rr.failFor["bad-pkg"] = boom
	g := NewGoInstaller(rr)

	results := g.Install([]string{"good-pkg", "bad-pkg", "another-pkg"})
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if len(rr.calls) != 3 {
		t.Fatalf("expected 3 invocations, got %d", len(rr.calls))
	}
	if results[1].Err == nil || !errors.Is(results[1].Err, boom) {
		t.Fatalf("results[1].Err = %v, want boom", results[1].Err)
	}
	if results[0].Failed() || results[2].Failed() {
		t.Fatalf("non-failing specs should succeed: %+v", results)
	}
}

func TestRustInstallerCommand(t *testing.T) {
	t.Parallel()
	r := RustInstaller{}
	name, args := r.Command("ripgrep")
	if name != "cargo" {
		t.Fatalf("name = %q, want cargo", name)
	}
	if len(args) != 2 || args[0] != "install" || args[1] != "ripgrep" {
		t.Fatalf("args = %v, want [install ripgrep]", args)
	}
}

func TestRustInstallerName(t *testing.T) {
	t.Parallel()
	r := RustInstaller{}
	if r.Name() != "rust" {
		t.Fatalf("Name = %q, want rust", r.Name())
	}
}

func TestRustInstallerRunsInstallForEverySpec(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	r := NewRustInstaller(rr)
	results := r.Install([]string{"ripgrep", "kotlin-lsp"})
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, spec := range []string{"ripgrep", "kotlin-lsp"} {
		if rr.calls[i].name != "cargo" {
			t.Fatalf("call[%d].name = %q, want cargo", i, rr.calls[i].name)
		}
		if len(rr.calls[i].args) != 2 || rr.calls[i].args[1] != spec {
			t.Fatalf("call[%d].args = %v, want [install %s]", i, rr.calls[i].args, spec)
		}
	}
}

func TestInstallEmptySpecs(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	r := NewRustInstaller(rr)
	results := r.Install(nil)
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
	if len(rr.calls) != 0 {
		t.Fatalf("len(calls) = %d, want 0", len(rr.calls))
	}
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
	// Just verify the type assertions pass; we don't actually run anything.
	var _ runner.Runner = runner.NewExecRunner()
	var _ Installer = NewGoInstaller(runner.NewExecRunner())
	var _ Installer = NewRustInstaller(runner.NewExecRunner())
}
