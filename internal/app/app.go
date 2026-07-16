// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

// Package app wires together the config loader and the installers and is the
// single place where the CLI-level orchestration lives. Keeping the logic out
// of package main makes it straightforward to unit-test the end-to-end flow.
package app

import (
	"fmt"
	"io"
	"os"

	"github.com/alexript/chezget/internal/config"
	"github.com/alexript/chezget/internal/installer"
	"github.com/alexript/chezget/internal/runner"
)

// Options configures an App invocation.
type Options struct {
	// ConfigPath, when non-empty, overrides the default XDG config location.
	ConfigPath string
	// Stdout and Stderr receive run output. When nil, os.Stdout/os.Stderr
	// are used.
	Stdout io.Writer
	Stderr io.Writer
	// Runner is used to invoke the package managers. When nil, a
	// runner.NewExecRunner() is used; tests can pass a recording runner to
	// avoid spawning real processes.
	Runner runner.Runner
}

// App holds the dependencies of a single chezget run. Construct one with New
// and call Run to execute.
type App struct {
	opts     Options
	out, err io.Writer
}

// New returns an App configured with opts. Default writers are filled in
// from os.Stdout and os.Stderr when opts leaves them unset.
func New(opts Options) *App {
	out, errw := opts.Stdout, opts.Stderr
	if out == nil {
		out = os.Stdout
	}
	if errw == nil {
		errw = os.Stderr
	}
	return &App{opts: opts, out: out, err: errw}
}

// Run performs the install: load configuration, then run the registered
// installers for the listed specs. It returns a non-zero exit code (1) when
// the configuration cannot be loaded or when at least one installation
// fails, and 0 on success.
func (a *App) Run() int {
	execRunner := a.opts.Runner
	if execRunner == nil {
		execRunner = runner.NewExecRunner()
	}
	installers := installer.DefaultInstallers(execRunner)

	cfg, err := a.loadConfig(installer.SectionNames(installers)...)
	if err != nil {
		fmt.Fprintf(a.err, "chezget: %v\n", err)
		if config.IsMissing(err) {
			fmt.Fprintf(a.err, "chezget: no configuration found; create %s\n", hintPath())
		}
		return 1
	}

	results := installer.RunAll(installers, cfg.Sections)
	for _, r := range results {
		if r.Failed() {
			fmt.Fprintf(a.err, "chezget: %s: %s: %v\n", r.Installer, r.Spec, r.Err)
		}
	}
	fmt.Fprintf(a.out, "chezget: %s\n", installer.Summary(results))

	for _, r := range results {
		if r.Failed() {
			return 1
		}
	}
	return 0
}

// loadConfig resolves the configuration path (honoring Options.ConfigPath and
// the CHEZGET_CONFIG environment variable) and parses it. sections lists the
// section names the parser should recognize.
func (a *App) loadConfig(sections ...string) (config.Config, error) {
	if a.opts.ConfigPath != "" {
		return config.LoadFrom(a.opts.ConfigPath, sections...)
	}
	return config.Load(sections...)
}

// hintPath returns the default configuration file path to suggest to the user
// when no configuration is found.
func hintPath() string {
	if p, err := config.ResolvePath(); err == nil {
		return p
	}
	return "$XDG_CONFIG_HOME/chezget/config.ini"
}
