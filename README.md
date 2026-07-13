# chezget

`chezget` is a small companion tool for
[chezmoi](https://github.com/twpayne/chezmoi). While chezmoi manages your dot
files, `chezget` installs the **Go** and **Rust** applications that those dot
files depend on. It reads a single INI configuration file listing packages and
runs the appropriate package manager (`go install` or `cargo install`) for each
entry.

## Configuration

`chezget` follows the
[XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/).
The configuration file is expected at:

```
$XDG_CONFIG_HOME/chezget/config.ini
```

falling back to `~/.config/chezget/config.ini` when `XDG_CONFIG_HOME` is unset.
The path can also be overridden with the `CHEZGET_CONFIG` environment variable or
the `--config` flag.

### Format

The file is a minimal INI dialect with two recognized sections, `[go]` and
`[rust]`. Each non-empty, non-comment line under a section is a package
specification understood by the corresponding package manager. Lines starting
with `#` or `;` are comments.

```ini
# Go packages (installed via `go install <spec>`)
[go]
github.com/jesseduffield/lazygit@latest
golang.org/x/tools/cmd/goimports

# Rust crates (installed via `cargo install <spec>`)
[rust]
ripgrep
kotlin-lsp
```

For the example above, `chezget` runs:

```
go install github.com/jesseduffield/lazygit@latest
go install golang.org/x/tools/cmd/goimports
cargo install ripgrep
cargo install kotlin-lsp
```

Unknown sections are ignored so the file can carry extra notes without breaking
the parser.

## Installation

### From source

```sh
go install github.com/alexript/chezget/cmd/chezget@latest
```

### Build locally

```sh
git clone https://github.com/alexript/chezget.git
cd chezget
make build      # produces a ./chezget binary
make install    # installs into $(go env GOPATH)/bin
```

### Cross-platform builds

`chezget` builds and runs on **Linux, macOS, Windows, and FreeBSD** for both
**amd64** and **arm64**. To produce binaries for every supported platform:

```sh
make cross      # writes dist/chezget-<os>-<arch>[.exe]
```

## Usage

```sh
chezget                 # install everything listed in the config file
chezget --config FILE   # use a specific config file
chezget --version       # print the version
```

Exit status is `0` when every package installs successfully, and `1` when the
configuration cannot be loaded or at least one installation fails. A failure on
one package does not stop the remaining packages from being installed.

## Development

```sh
make test        # run the unit tests
make cover       # run tests with coverage and print a summary
make cover-html  # generate an HTML coverage report in coverage.html
make vet         # run go vet
make fmt         # format the source tree
```

The project has no external runtime dependencies: the configuration parser is
self-contained and the only commands it shells out to are `go` and `cargo`,
which must be installed and available on `PATH` for `chezget` to do its job.

## Project layout

```
cmd/chezget/      entrypoint
internal/app/     CLI orchestration (loads config, runs installers)
internal/config/  INI parser and XDG path resolution
internal/installer/  go/cargo installer implementations
internal/runner/  command-execution abstraction (for testability)
```

## License

[MIT](LICENSE) © Alex 'Ript' Malyshev