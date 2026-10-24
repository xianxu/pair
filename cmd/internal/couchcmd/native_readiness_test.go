package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// An unqualified namespace must refuse before launching even a named mutation.
func TestNativeMutationRefusesUnqualifiedNamespace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("TMPDIR", root)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(root, "native"))
	dir := filepath.Join(root, "native", "contract_version_1")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unknown"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	called := filepath.Join(root, "called")
	t.Setenv("NATIVE_CALLED", called)
	if err := os.WriteFile(filepath.Join(root, "zellij"), []byte("#!/bin/sh\nprintf called > \"$NATIVE_CALLED\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	if err := (&osCouchTitles{}).RenamePane(context.Background(), couchcore.TitleTarget{Session: "pair-test"}, "terminal_7", "title"); !errors.Is(err, zellijquery.ErrReadinessUnknown) {
		t.Errorf("call 0: error=%v, want readiness unknown", err)
	}
	if _, err := os.Stat(called); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("native invoked before qualification: %v", err)
	}
}
