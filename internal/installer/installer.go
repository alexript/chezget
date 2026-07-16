// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

// Package installer turns package specifications loaded from the chezget
// configuration file into concrete invocations of the appropriate package
// managers.
//
// Each package manager is implemented as an Installer in its own source file
// (go_installer.go, rust_installer.go, ...). To add support for a new
// manager, create a new file implementing the Installer interface and
// register it in DefaultInstallers.
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
	// "rust"). The name doubles as the configuration file section header
	// that groups specs for this installer.
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

// RunAll runs every installer against the specs recorded for it in specs and
// returns the aggregated results, preserving the order of installers. A nil
// installer is skipped. Specs for an installer are looked up by the
// installer's Name().
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

// DefaultInstallers returns the set of installers built into chezget, wired
// to the given runner. The order determines installation order and the order
// sections are expected in the configuration file. Add new installers here
// when introducing support for a new package manager.
func DefaultInstallers(r runner.Runner) []Installer {
	return []Installer{
		NewGoInstaller(r),
		NewRustInstaller(r),
	}
}

// SectionNames returns the Name() of each installer in order, skipping nil
// entries. It is a convenience for callers (such as the config loader) that
// need the list of recognized section names.
func SectionNames(installers []Installer) []string {
	names := make([]string, 0, len(installers))
	for _, in := range installers {
		if in != nil {
			names = append(names, in.Name())
		}
	}
	return names
}
