// Package installer turns package specifications loaded from the chezget
// configuration file into concrete invocations of the appropriate package
// managers (go install for Go packages, cargo install for Rust crates).
//
// Installers depend on a [github.com/alexript/chezget/internal/runner].Runner
// so they can be unit-tested with a recording runner instead of spawning real
// processes.
package installer

import (
	"fmt"
	"strings"

	"github.com/alexript/chezget/internal/runner"
)

// Installer installs a batch of package specifications using a single,
// language-specific package manager.
type Installer interface {
	// Name returns the human-readable name of the installer (e.g. "go",
	// "rust").
	Name() string
	// Install installs every entry in specs, in order. Implementations should
	// attempt each spec independently so a single failure does not prevent
	// the remaining packages from being installed.
	Install(specs []string) []Result
}

// Result reports the outcome of installing a single package spec.
type Result struct {
	Installer string
	Spec      string
	Err       error
}

// Failed reports whether the installation ended in an error.
func (r Result) Failed() bool { return r.Err != nil }

// Summary converts a slice of results into a human-readable report. It is
// primarily intended for end-of-run output in the CLI.
func Summary(results []Result) string {
	if len(results) == 0 {
		return "no packages to install"
	}
	var failed, ok int
	for _, r := range results {
		if r.Failed() {
			failed++
		} else {
			ok++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "installed %d package(s)", ok)
	if failed > 0 {
		fmt.Fprintf(&b, ", %d failed", failed)
	}
	return b.String()
}

// GoInstaller installs Go packages using `go install <spec>`.
type GoInstaller struct {
	Runner runner.Runner
}

// NewGoInstaller returns a GoInstaller backed by r. If r is nil, the returned
// installer's Install panics; callers should pass a non-nil runner such as
// runner.NewExecRunner().
func NewGoInstaller(r runner.Runner) *GoInstaller {
	return &GoInstaller{Runner: r}
}

// Name returns "go".
func (GoInstaller) Name() string { return "go" }

// Command returns the argv that would be used to install spec, without running
// it. It is exposed mainly for testability and inspection.
func (GoInstaller) Command(spec string) (string, []string) {
	return "go", []string{"install", spec}
}

// Install runs `go install <spec>` for every entry in specs. Each spec is
// installed in its own process invocation so a failure on one does not block
// the others.
func (g *GoInstaller) Install(specs []string) []Result {
	results := make([]Result, 0, len(specs))
	for _, spec := range specs {
		name, args := g.Command(spec)
		err := g.Runner.Run(name, args...)
		results = append(results, Result{Installer: g.Name(), Spec: spec, Err: err})
	}
	return results
}

// RustInstaller installs Rust crates using `cargo install <spec>`.
type RustInstaller struct {
	Runner runner.Runner
}

// NewRustInstaller returns a RustInstaller backed by r.
func NewRustInstaller(r runner.Runner) *RustInstaller {
	return &RustInstaller{Runner: r}
}

// Name returns "rust".
func (RustInstaller) Name() string { return "rust" }

// Command returns the argv used to install spec.
func (RustInstaller) Command(spec string) (string, []string) {
	return "cargo", []string{"install", spec}
}

// Install runs `cargo install <spec>` for every entry in specs.
func (r *RustInstaller) Install(specs []string) []Result {
	results := make([]Result, 0, len(specs))
	for _, spec := range specs {
		name, args := r.Command(spec)
		err := r.Runner.Run(name, args...)
		results = append(results, Result{Installer: r.Name(), Spec: spec, Err: err})
	}
	return results
}

// RunAll runs every installer against the specs recorded for it in specs and
// returns the aggregated results, preserving installation order: all Go
// packages first, then all Rust packages. A nil installer is skipped.
func RunAll(installers []Installer, specs map[string][]string) []Result {
	var out []Result
	for _, in := range installers {
		if in == nil {
			continue
		}
		out = append(out, in.Install(specs[in.Name()])...)
	}
	return out
}
