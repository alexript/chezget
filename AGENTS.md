# AGENTS.md

Guidance for AI agents working on this repository. Keep this file in sync with
the project whenever its structure, conventions, or tooling change.

## Project purpose

`chezget` is a companion to [chezmoi](https://github.com/twpayne/chezmoi). It
reads an INI configuration file listing Go and Rust packages and runs
`go install` / `cargo install` for each entry. See `README.md` for user-facing
documentation.

## Build & test commands

| Task               | Command                              |
| ------------------ | ------------------------------------ |
| Build the binary   | `make build` (or `go build -o chezget ./cmd/chezget`) |
| Run unit tests     | `make test` (or `go test ./...`)     |
| Coverage report   | `make cover` or `make cover-html`    |
| Lint              | `make vet` (or `go vet ./...`)       |
| Format            | `make fmt` (or `gofmt -s -w .`)      |
| Cross-compile     | `make cross`                         |
| Install           | `make install`                       |

The binary artifact must be named `chezget`.

> **Note:** A bare `go build` from the project root fails because the
> entrypoint lives in `cmd/chezget/`. Always use `go build ./cmd/chezget`
> (or `make build`) instead. This is intentional and follows the standard
> Go project layout.

## Architecture

The code is split into small, single-responsibility packages so that each one
can be unit-tested in isolation:

- `cmd/chezget/main.go` - the CLI entrypoint. It parses flags and delegates to
  `internal/app`. Keep it thin: any new logic belongs in a package so it can be
  tested.
- `internal/app` - wires the config loader together with the installers and
  owns the run loop. `App.Run` returns the process exit code. Tests inject a
  recording `Runner` via `Options.Runner` so no real commands are spawned.
- `internal/config` - INI parser and XDG path resolution. The parser is
  deliberately minimal and dependency-free; see the package doc for the grammar.
  Config is read from `$XDG_CONFIG_HOME/chezget/config.ini` by default,
  overridable via `CHEZGET_CONFIG` or the `--config` flag.
- `internal/installer` - `GoInstaller` and `RustInstaller` implement the
  `Installer` interface. Each spec is installed in its own process invocation
  so a single failure does not abort the run.
- `internal/runner` - abstracts command execution. `ExecRunner` is the
  production implementation; tests use recording runners to assert the argv
  without spawning processes.

### Key types & flow

```
main.App.Run
  -> config.Load (ResolvePath -> LoadFrom -> Parse)
  -> installer.RunAll([GoInstaller, RustInstaller], specs)
       -> each Installer.Install -> Runner.Run("go"|"cargo", "install", spec)
```

## Coding conventions

- Go 1.26 module path: `github.com/alexript/chezget`. Do not change it.
- Every `.go` file starts with the SPDX/MIT header:
  ```go
  // SPDX-License-Identifier: MIT
  // Copyright (c) 2026 Alex 'Ript' Malyshev
  ```
  Keep this header on any new source file.
- Follow standard Go project layout: `cmd/<binary>/main.go` for entrypoints,
  everything else under `internal/`.
- No external runtime dependencies. The INI parser is hand-written to keep the
  module dependency-free and easy to audit. New features should prefer the
  standard library; only add external deps if their license is MIT-compatible
  and the benefit clearly outweighs the cost.
- Match the surrounding style: `gofmt -s`, `go vet` clean, table-driven tests
  where appropriate, `t.Helper()` in test helpers, `t.Setenv` for env-dependent
  tests (do **not** use `t.Parallel()` in those tests - `t.Setenv` forbids it).
- Comments document **why**, not what. Public packages and exported symbols
  have doc comments.
- Do not add comments unless they explain intent.

## Testing requirements

- Overall coverage target: 60%+; critical packages (`installer`, `config`,
  `runner`) and the `cmd/chezget` entrypoint aim for 80%+. Current coverage:
  `installer` 100%, `runner` 100%, `config` 91.1%, `app` 93.8%,
  `cmd/chezget` 93.3% (overall ~93%).
- Tests must pass via `go test ./...` with no network access and no real
  `go`/`cargo` invocations. Use recording runners / string readers, not the
  production exec runner.
- After any change, run `make test` and `make vet` before declaring done.

## Configuration file format

INI dialect. Recognized sections: `[go]`, `[rust]`. Comments start with `#` or
`;`. Each non-empty line under a section is a package spec passed verbatim to
the package manager (last arg of `go install` / `cargo install`). Unknown
sections are ignored.

## Cross-platform support

Target platforms: linux, darwin, windows, freebsd x amd64, arm64. Avoid
platform-specific APIs in `internal/`; use the standard library's
`os.UserHomeDir` and `os/exec` which handle platform differences. The `Makefile`
`cross` target builds all combinations.

## When you change the project

- Update `AGENTS.md` if you add/rename packages, change the config format, or
  alter build/test commands.
- Keep `README.md` accurate for user-facing behavior.
- Do not commit the `chezget` binary, `coverage.out`, or `coverage.html`.
- License is MIT; keep the LICENSE file intact and the copyright header current.