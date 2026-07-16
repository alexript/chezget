// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package installer

import (
	"github.com/alexript/chezget/internal/runner"
)

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

// Name returns "go", which is also the configuration section that groups Go
// package specs.
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
