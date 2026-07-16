// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

// Package config loads and parses the chezget configuration file.
//
// The configuration uses a small INI dialect. Each section corresponds to an
// installer (identified by its Name), and each non-empty, non-comment line
// under a section is a package specification interpreted by that installer's
// package manager:
//
//	[go]
//	github.com/jesseduffield/lazygit@latest
//	golang.org/x/tools/cmd/goimports
//
//	[rust]
//	ripgrep
//	kotlin-lsp
//
// The set of recognized sections is not hardcoded: callers pass the section
// names (typically derived from the registered installers via
// [github.com/alexript/chezget/internal/installer].SectionNames) to Parse,
// Load, or LoadFrom. Sections not in that set are ignored so the file can
// carry additional metadata without breaking the parser.
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

// Config holds the parsed contents of the chezget configuration file. The
// Path field records where the configuration was loaded from so callers can
// surface it in diagnostics. Sections maps each recognized section name to
// the list of package specs found under it.
type Config struct {
	Path     string
	Sections map[string][]string
}

// ErrEmpty is returned by Load when the configuration file exists but
// contains no package entries at all.
var ErrEmpty = errors.New("configuration file contains no packages")

// Load reads and parses the configuration from the default XDG location. The
// location is determined by ResolvePath and can be overridden through the
// CHEZGET_CONFIG environment variable, which is useful for tests and ad-hoc
// invocations. sections lists the section names to recognize (typically the
// Name() of each registered installer).
func Load(sections ...string) (Config, error) {
	path, err := ResolvePath()
	if err != nil {
		return Config{}, err
	}
	return LoadFrom(path, sections...)
}

// LoadFrom parses the configuration file at path. An empty result is reported
// as ErrEmpty to distinguish "file missing" from "file present but unused".
// sections lists the section names to recognize.
func LoadFrom(path string, sections ...string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config %q: %w", path, err)
	}
	defer f.Close()

	cfg, err := Parse(f, sections...)
	if err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.Path = path
	if len(cfg.Sections) == 0 {
		return cfg, ErrEmpty
	}
	return cfg, nil
}

// Parse reads an INI-style configuration from r and returns the resulting
// Config. Parse does not touch the filesystem, which makes it convenient to
// unit-test with string readers. sections lists the section names to
// recognize; any other section header in the input is ignored.
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
func Parse(r io.Reader, sections ...string) (Config, error) {
	recognized := make(map[string]struct{}, len(sections))
	for _, s := range sections {
		recognized[s] = struct{}{}
	}

	var cfg Config
	cfg.Sections = make(map[string][]string)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	section := ""
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if _, ok := recognized[section]; !ok {
				section = "" // ignore contents of unrecognized sections
			}
			continue
		}

		if section != "" {
			cfg.Sections[section] = append(cfg.Sections[section], line)
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
