// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package installer

import (
	"github.com/alexript/chezget/internal/runner"
)

// RustInstaller installs Rust crates using `cargo install <spec>`.
type RustInstaller struct {
	Runner runner.Runner
}

// NewRustInstaller returns a RustInstaller backed by r.
func NewRustInstaller(r runner.Runner) *RustInstaller {
	return &RustInstaller{Runner: r}
}

// Name returns "rust", which is also the configuration section that groups
// Rust crate specs.
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
