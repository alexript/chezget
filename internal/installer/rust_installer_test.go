// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package installer

import (
	"testing"
)

func TestRustInstallerCommand(t *testing.T) {
	t.Parallel()
	r := RustInstaller{}
	name, args := r.Command("ripgrep")
	if name != "cargo" {
		t.Fatalf("name = %q, want cargo", name)
	}
	if len(args) != 2 || args[0] != "install" || args[1] != "ripgrep" {
		t.Fatalf("args = %v, want [install ripgrep]", args)
	}
}

func TestRustInstallerName(t *testing.T) {
	t.Parallel()
	r := RustInstaller{}
	if r.Name() != "rust" {
		t.Fatalf("Name = %q, want rust", r.Name())
	}
}

func TestRustInstallerRunsInstallForEverySpec(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	r := NewRustInstaller(rr)
	results := r.Install([]string{"ripgrep", "kotlin-lsp"})
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, spec := range []string{"ripgrep", "kotlin-lsp"} {
		if rr.calls[i].name != "cargo" {
			t.Fatalf("call[%d].name = %q, want cargo", i, rr.calls[i].name)
		}
		if len(rr.calls[i].args) != 2 || rr.calls[i].args[1] != spec {
			t.Fatalf("call[%d].args = %v, want [install %s]", i, rr.calls[i].args, spec)
		}
	}
}

func TestInstallEmptySpecs(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	r := NewRustInstaller(rr)
	results := r.Install(nil)
	if len(results) != 0 {
		t.Fatalf("len(results) = %d, want 0", len(results))
	}
	if len(rr.calls) != 0 {
		t.Fatalf("len(calls) = %d, want 0", len(rr.calls))
	}
}
