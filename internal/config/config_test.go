// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const sampleConfig = `# chezget packages
; semicolons also start comments

[go]
github.com/jesseduffield/lazygit@latest
golang.org/x/tools/cmd/goimports

[rust]
ripgrep
kotlin-lsp

[notes]
this section is ignored
should-not-appear
`

func TestParseBasic(t *testing.T) {
	t.Parallel()
	cfg, err := Parse(strings.NewReader(sampleConfig), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := len(cfg.Sections["go"]), 2; got != want {
		t.Fatalf("len(go) = %d, want %d", got, want)
	}
	if cfg.Sections["go"][0] != "github.com/jesseduffield/lazygit@latest" {
		t.Fatalf("go[0] = %q", cfg.Sections["go"][0])
	}
	if got, want := len(cfg.Sections["rust"]), 2; got != want {
		t.Fatalf("len(rust) = %d, want %d", got, want)
	}
	if cfg.Sections["rust"][0] != "ripgrep" {
		t.Fatalf("rust[0] = %q", cfg.Sections["rust"][0])
	}
}

func TestParseEmpty(t *testing.T) {
	t.Parallel()
	cfg, err := Parse(strings.NewReader(""), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Sections) != 0 {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestParseCommentsAndBlankLines(t *testing.T) {
	t.Parallel()
	src := `# header

  # indented comment
;another

[go]

  
github.com/foo/bar
`
	cfg, err := Parse(strings.NewReader(src), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(cfg.Sections["go"]); got != 1 {
		t.Fatalf("len(go) = %d, want 1", got)
	}
	if cfg.Sections["go"][0] != "github.com/foo/bar" {
		t.Fatalf("go[0] = %q", cfg.Sections["go"][0])
	}
}

func TestParseTrimsWhitespace(t *testing.T) {
	t.Parallel()
	src := "[go]\n   github.com/foo/bar   \n\tgolang.org/x/pkga\n"
	cfg, err := Parse(strings.NewReader(src), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for i, pkg := range cfg.Sections["go"] {
		if pkg != strings.TrimSpace(pkg) {
			t.Fatalf("go[%d] = %q has surrounding whitespace", i, pkg)
		}
	}
}

func TestParseUnknownSectionIgnored(t *testing.T) {
	t.Parallel()
	src := `[notes]
ignored
[go]
keep-me
[rust]
keep-rust
`
	cfg, err := Parse(strings.NewReader(src), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(cfg.Sections["go"]); got != 1 || cfg.Sections["go"][0] != "keep-me" {
		t.Fatalf("go = %v", cfg.Sections["go"])
	}
	if got := len(cfg.Sections["rust"]); got != 1 || cfg.Sections["rust"][0] != "keep-rust" {
		t.Fatalf("rust = %v", cfg.Sections["rust"])
	}
	if _, ok := cfg.Sections["notes"]; ok {
		t.Fatalf("notes should not be captured")
	}
}

func TestParsePackagesBeforeAnySection(t *testing.T) {
	t.Parallel()
	src := `orphan-package
[go]
real-package
`
	cfg, err := Parse(strings.NewReader(src), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(cfg.Sections["go"]); got != 1 || cfg.Sections["go"][0] != "real-package" {
		t.Fatalf("go = %v", cfg.Sections["go"])
	}
	if got := len(cfg.Sections["rust"]); got != 0 {
		t.Fatalf("rust = %v, want empty", cfg.Sections["rust"])
	}
}

func TestParseSectionHeaderWithSpaces(t *testing.T) {
	t.Parallel()
	src := "[ go ]\nexample.com/pkg\n"
	cfg, err := Parse(strings.NewReader(src), "go", "rust")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(cfg.Sections["go"]); got != 1 || cfg.Sections["go"][0] != "example.com/pkg" {
		t.Fatalf("go = %v", cfg.Sections["go"])
	}
}

func TestParseRecognizesDynamicSections(t *testing.T) {
	t.Parallel()
	src := `[foo]
pkg-one
pkg-two
[bar]
crate-x
`
	cfg, err := Parse(strings.NewReader(src), "foo", "bar")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(cfg.Sections["foo"]); got != 2 {
		t.Fatalf("len(foo) = %d, want 2", got)
	}
	if cfg.Sections["foo"][0] != "pkg-one" || cfg.Sections["foo"][1] != "pkg-two" {
		t.Fatalf("foo = %v", cfg.Sections["foo"])
	}
	if got := len(cfg.Sections["bar"]); got != 1 || cfg.Sections["bar"][0] != "crate-x" {
		t.Fatalf("bar = %v", cfg.Sections["bar"])
	}
}

func TestParseIgnoresUnrecognizedSection(t *testing.T) {
	t.Parallel()
	src := `[go]
keep-me
[rust]
should-be-ignored
`
	cfg, err := Parse(strings.NewReader(src), "go")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := len(cfg.Sections["go"]); got != 1 || cfg.Sections["go"][0] != "keep-me" {
		t.Fatalf("go = %v", cfg.Sections["go"])
	}
	if _, ok := cfg.Sections["rust"]; ok {
		t.Fatalf("rust should not be captured when not recognized")
	}
}

func TestParseNoRecognizedSections(t *testing.T) {
	t.Parallel()
	src := `[go]
should-be-ignored
`
	cfg, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(cfg.Sections) != 0 {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestLoadFromMissingFile(t *testing.T) {
	t.Parallel()
	_, err := LoadFrom(filepath.Join(t.TempDir(), "does-not-exist.ini"), "go", "rust")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !IsMissing(err) {
		t.Fatalf("IsMissing(err) = false; err = %v", err)
	}
}

func TestLoadFromEmptyConfig(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "# just comments\n\n[notes]\nfoo\n")
	_, err := LoadFrom(path, "go", "rust")
	if !errors.Is(err, ErrEmpty) {
		t.Fatalf("err = %v, want ErrEmpty", err)
	}
}

func TestLoadFromValid(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, sampleConfig)
	cfg, err := LoadFrom(path, "go", "rust")
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Path != path {
		t.Fatalf("cfg.Path = %q, want %q", cfg.Path, path)
	}
	if len(cfg.Sections["go"]) != 2 || len(cfg.Sections["rust"]) != 2 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestResolvePathOverrideEnv(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "/custom/path/to/config.ini")
	got, err := ResolvePath()
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != "/custom/path/to/config.ini" {
		t.Fatalf("ResolvePath = %q, want /custom/path/to/config.ini", got)
	}
}

func TestResolvePathXDG(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/home")
	got, err := ResolvePath()
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want := filepath.Join("/xdg/home", AppName, ConfigFile)
	if got != want {
		t.Fatalf("ResolvePath = %q, want %q", got, want)
	}
}

func TestResolvePathFallbackHome(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Force UserHomeDir to use HOME on every platform.
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	got, err := ResolvePath()
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want := filepath.Join(home, ".config", AppName, ConfigFile)
	if got != want {
		t.Fatalf("ResolvePath = %q, want %q", got, want)
	}
}

func TestConfigDir(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/home")
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	want := filepath.Join("/xdg/home", AppName)
	if got != want {
		t.Fatalf("ConfigDir = %q, want %q", got, want)
	}
}

func TestConfigDirFallbackHome(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	want := filepath.Join(home, ".config", AppName)
	if got != want {
		t.Fatalf("ConfigDir = %q, want %q", got, want)
	}
}

func TestConfigDirOverrideEnv(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", "/custom/subdir/config.ini")
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if want := "/custom/subdir"; filepath.ToSlash(got) != want {
		t.Fatalf("ConfigDir = %q, want %q", got, want)
	}
}

func TestLoadViaEnvOverride(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", writeTempFile(t, sampleConfig))
	cfg, err := Load("go", "rust")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Sections["go"]) != 2 || len(cfg.Sections["rust"]) != 2 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadMissingViaEnvOverride(t *testing.T) {
	t.Setenv("CHEZGET_CONFIG", filepath.Join(t.TempDir(), "nope.ini"))
	_, err := Load("go", "rust")
	if err == nil || !IsMissing(err) {
		t.Fatalf("err = %v, want missing", err)
	}
}

func TestIsMissingWithUnrelatedError(t *testing.T) {
	t.Parallel()
	if IsMissing(errors.New("nope")) {
		t.Fatal("IsMissing should be false for unrelated errors")
	}
	if IsMissing(nil) {
		t.Fatal("IsMissing(nil) should be false")
	}
}

func TestIsMissingWithFsErrNotExist(t *testing.T) {
	t.Parallel()
	if !IsMissing(fs.ErrNotExist) {
		t.Fatal("IsMissing(fs.ErrNotExist) should be true")
	}
}

// writeTempFile writes content to a new file in a per-test temp directory and
// returns the file path.
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}
