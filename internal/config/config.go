// Package config loads and parses the chezget configuration file.
//
// The configuration uses a small INI dialect with two recognized sections:
//
//	[go]
//	github.com/jesseduffield/lazygit@latest
//	golang.org/x/tools/cmd/goimports
//
//	[rust]
//	ripgrep
//	kotlin-lsp
//
// Each non-empty, non-comment line under a section is a package specification
// interpreted by the package manager for that section (go install for the
// "go" section, cargo install for the "rust" section).
//
// The configuration file is resolved following the XDG Base Directory
// Specification: $XDG_CONFIG_HOME/chezget/config.ini, falling back to
// ~/.config/chezget/config.ini when XDG_CONFIG_HOME is unset or empty.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AppName is the application directory used under the XDG config root.
const AppName = "chezget"

// ConfigFile is the configuration file name placed inside the application
// config directory.
const ConfigFile = "config.ini"

// Section names recognized in the configuration file.
const (
	SectionGo   = "go"
	SectionRust = "rust"
)

// Config holds the parsed contents of the chezget configuration file. The
// Path field records where the configuration was loaded from so callers can
// surface it in diagnostics.
type Config struct {
	Path string
	Go   []string
	Rust []string
}

// ErrEmpty is returned by Load when the configuration file exists but
// contains no package entries at all.
var ErrEmpty = errors.New("configuration file contains no packages")

// Load reads and parses the configuration from the default XDG location. The
// location is determined by ResolvePath and can be overridden through the
// CHEZGET_CONFIG environment variable, which is useful for tests and ad-hoc
// invocations.
func Load() (Config, error) {
	path, err := ResolvePath()
	if err != nil {
		return Config{}, err
	}
	return LoadFrom(path)
}

// LoadFrom parses the configuration file at path. An empty result is reported
// as ErrEmpty to distinguish "file missing" from "file present but unused".
func LoadFrom(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	cfg, err := Parse(f)
	if err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.Path = path
	if len(cfg.Go) == 0 && len(cfg.Rust) == 0 {
		return cfg, ErrEmpty
	}
	return cfg, nil
}

// Parse reads an INI-style configuration from r and returns the resulting
// Config. Parse does not touch the filesystem, which makes it convenient to
// unit-test with string readers.
//
// The grammar is intentionally minimal:
//
//   - Lines starting with '#' or ';' are comments and ignored.
//   - A line of the form "[section]" begins a new section.
//   - Any other non-blank line is treated as a package specification under the
//     current section.
//   - Leading and trailing whitespace is trimmed from both section headers
//     and package specifications.
//   - Unknown sections are ignored so the file can carry additional metadata
//     without breaking the parser.
func Parse(r io.Reader) (Config, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var cfg Config
	section := ""
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			switch section {
			case SectionGo, SectionRust:
				// recognized section
			default:
				section = "" // ignore contents of unknown sections
			}
			continue
		}

		switch section {
		case SectionGo:
			cfg.Go = append(cfg.Go, line)
		case SectionRust:
			cfg.Rust = append(cfg.Rust, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("scan: %w", err)
	}
	return cfg, nil
}

// ResolvePath returns the absolute path of the chezget configuration file
// according to the XDG Base Directory Specification. The CHEZGET_CONFIG
// environment variable, when set and non-empty, overrides the default
// location.
func ResolvePath() (string, error) {
	if v := os.Getenv("CHEZGET_CONFIG"); v != "" {
		return v, nil
	}

	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}

	return filepath.Join(configDir, AppName, ConfigFile), nil
}

// ConfigDir returns the directory that should contain the chezget
// configuration file. It mirrors ResolvePath's resolution rules but stops at
// the directory level; callers can use it to create the directory on first
// run.
func ConfigDir() (string, error) {
	if v := os.Getenv("CHEZGET_CONFIG"); v != "" {
		return filepath.Dir(v), nil
	}
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, AppName), nil
}

// IsMissing reports whether err is the error returned when the configuration
// file does not exist. Callers can branch on "first run" scenarios without
// importing io/fs themselves.
func IsMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
