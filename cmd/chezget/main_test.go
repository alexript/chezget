// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	code := run([]string{"--version"}, &out, &errw)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	want := "chezget " + version + "\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

func TestRunVersionShortOutput(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	code := run([]string{"--version"}, &out, &errw)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.HasPrefix(out.String(), "chezget ") {
		t.Fatalf("stdout = %q, want prefix 'chezget '", out.String())
	}
}

func TestRunConfigFlag(t *testing.T) {
	// Cannot use t.Parallel — t.Setenv forbids it.
	t.Setenv("CHEZGET_CONFIG", "")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(cfgPath, []byte("[go]\nexample.com/pkg\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var out, errw bytes.Buffer
	code := run([]string{"--config", cfgPath}, &out, &errw)
	if code != 1 {
		t.Fatalf("code = %d, want 1 (no real go binary in test); stderr=%q", code, errw.String())
	}
	// The config was found and parsed (not "no configuration found"), so the
	// installer attempted to run. With a real go/cargo missing, Run returns 1.
	if strings.Contains(errw.String(), "no configuration found") {
		t.Fatalf("config should have been loaded; stderr=%q", errw.String())
	}
}

func TestRunConfigFlagMissingFile(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	missing := filepath.Join(t.TempDir(), "does-not-exist.ini")
	code := run([]string{"--config", missing}, &out, &errw)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(errw.String(), "no configuration found") {
		t.Fatalf("stderr = %q, want 'no configuration found'", errw.String())
	}
}

func TestRunNoArgs(t *testing.T) {
	// Cannot use t.Parallel — t.Setenv forbids it.
	t.Setenv("CHEZGET_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var out, errw bytes.Buffer
	code := run(nil, &out, &errw)
	if code != 1 {
		t.Fatalf("code = %d, want 1 (no config present)", code)
	}
	if !strings.Contains(errw.String(), "no configuration found") {
		t.Fatalf("stderr = %q, want 'no configuration found'", errw.String())
	}
}

func TestRunUnknownFlag(t *testing.T) {
	t.Parallel()
	var out, errw bytes.Buffer
	code := run([]string{"--bogus"}, &out, &errw)
	if code != 1 {
		t.Fatalf("code = %d, want 1 for unknown flag", code)
	}
	if !strings.Contains(errw.String(), "flag provided but not defined") {
		t.Fatalf("stderr = %q, want 'flag provided but not defined'", errw.String())
	}
}
