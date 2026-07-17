// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

// Package main is the entrypoint for the chezget command.
//
// chezget is a companion to chezmoi: it reads a small INI configuration file
// listing Go and Rust packages and runs `go install` / `cargo install` for
// each entry. See the project README for the configuration file format.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/alexript/chezget/internal/app"
)

// version is the application version. It is overridable at build time via
// -ldflags "-X main.version=...".
var version = "dev"

// run is the testable body of the chezget command. It parses flags from args,
// writes diagnostics to stderr and version/output to stdout, and returns the
// process exit code. Keeping the logic here rather than in main() allows
// unit tests to inject args and capture output.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("chezget", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "chezget is a small companion tool for chezmoi.\n")
		fmt.Fprintf(stderr, "While chezmoi manages your dot files, chezget installs\n")
		fmt.Fprintf(stderr, "the Go and Rust applications those dot files depend on,\n")
		fmt.Fprintf(stderr, "running `go install` / `cargo install` for each entry.\n\n")
		fmt.Fprintf(stderr, "Project page: https://github.com/alexript/chezget\n\n")
		fmt.Fprintf(stderr, "Usage of chezget:\n")
		fs.PrintDefaults()
	}

	configPath := fs.String("config", "", "path to the chezget config file (overrides XDG location)")
	showVersion := fs.Bool("version", false, "print the chezget version and exit")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *showVersion {
		fmt.Fprintf(stdout, "chezget %s\n", version)
		return 0
	}

	a := app.New(app.Options{ConfigPath: *configPath, Stdout: stdout, Stderr: stderr})
	return a.Run()
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
