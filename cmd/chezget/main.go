// Package main is the entrypoint for the chezget command.
//
// chezget is a companion to chezmoi: it reads a small INI configuration file
// listing Go and Rust packages and runs `go install` / `cargo install` for
// each entry. See the project README for the configuration file format.
package main

import (
	"flag"
	"os"

	"github.com/alexript/chezget/internal/app"
)

// version is the application version. It is overridable at build time via
// -ldflags "-X main.version=...".
var version = "dev"

func main() {
	configPath := flag.String("config", "", "path to the chezget config file (overrides XDG location)")
	showVersion := flag.Bool("version", false, "print the chezget version and exit")
	flag.Parse()

	if *showVersion {
		os.Stdout.WriteString("chezget " + version + "\n")
		return
	}

	a := app.New(app.Options{ConfigPath: *configPath})
	os.Exit(a.Run())
}
