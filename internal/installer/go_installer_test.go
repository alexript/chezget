// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Alex 'Ript' Malyshev

package installer

import (
	"errors"
	"testing"
)

func TestGoInstallerCommand(t *testing.T) {
	t.Parallel()
	g := GoInstaller{}
	name, args := g.Command("github.com/foo/bar@latest")
	if name != "go" {
		t.Fatalf("name = %q, want go", name)
	}
	if len(args) != 2 || args[0] != "install" || args[1] != "github.com/foo/bar@latest" {
		t.Fatalf("args = %v, want [install github.com/foo/bar@latest]", args)
	}
}

func TestGoInstallerName(t *testing.T) {
	t.Parallel()
	g := GoInstaller{}
	if g.Name() != "go" {
		t.Fatalf("Name = %q, want go", g.Name())
	}
}

func TestGoInstallerRunsInstallForEverySpec(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	g := NewGoInstaller(rr)
	results := g.Install([]string{
		"github.com/jesseduffield/lazygit@latest",
		"golang.org/x/tools/cmd/goimports",
	})

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if len(rr.calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2", len(rr.calls))
	}
	for i, want := range []string{"install", "install"} {
		if rr.calls[i].name != "go" {
			t.Fatalf("call[%d].name = %q, want go", i, rr.calls[i].name)
		}
		if len(rr.calls[i].args) != 2 || rr.calls[i].args[0] != want {
			t.Fatalf("call[%d].args = %v", i, rr.calls[i].args)
		}
	}
	for _, r := range results {
		if r.Failed() {
			t.Fatalf("unexpected failure: %+v", r)
		}
		if r.Installer != "go" {
			t.Fatalf("result.Installer = %q, want go", r.Installer)
		}
	}
}

func TestGoInstallerContinuesAfterFailure(t *testing.T) {
	t.Parallel()
	rr := newRecordingRunner()
	boom := errors.New("network down")
	rr.failFor["bad-pkg"] = boom
	g := NewGoInstaller(rr)

	results := g.Install([]string{"good-pkg", "bad-pkg", "another-pkg"})
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if len(rr.calls) != 3 {
		t.Fatalf("expected 3 invocations, got %d", len(rr.calls))
	}
	if results[1].Err == nil || !errors.Is(results[1].Err, boom) {
		t.Fatalf("results[1].Err = %v, want boom", results[1].Err)
	}
	if results[0].Failed() || results[2].Failed() {
		t.Fatalf("non-failing specs should succeed: %+v", results)
	}
}
