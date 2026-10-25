package clipcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/zellijquery"
)

// Explicit commands retain support for legacy endpoints while excluding live creators.
func TestNativeMutationWaitsForNamespaceLease(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("TMPDIR", root)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(root, "native"))
	called := filepath.Join(root, "called")
	t.Setenv("NATIVE_CALLED", called)
	if err := os.WriteFile(filepath.Join(root, "zellij"), []byte("#!/bin/sh\nprintf called > \"$NATIVE_CALLED\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	lease, err := zellijquery.AcquireProbe(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	// A legacy/unqualified endpoint must not disable explicit controls.
	dir := filepath.Join(root, "native", "contract_version_1")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unknown"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- (OSRuntime{}).SetPaneColor("7", "#ffffff") }()
	select {
	case err := <-done:
		t.Fatalf("command crossed held creator lease: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if _, err := os.Stat(called); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("native invoked while creator held exclusion: %v", err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("explicit command after release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("explicit command did not resume after release")
	}
	if _, err := os.Stat(called); err != nil {
		t.Fatalf("explicit command never invoked: %v", err)
	}
}

func TestNativeClipboardMutationRetainsCallerDeadline(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("TMPDIR", root)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(root, "native"))
	lease, err := zellijquery.AcquireProbe(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	for _, run := range []func(context.Context) error{
		func(ctx context.Context) error { return (OSRuntime{}).FocusPaneContext(ctx, "7") },
		func(ctx context.Context) error { return (OSRuntime{}).WritePaneKeyContext(ctx, "7", 13) },
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		started := time.Now()
		err := run(ctx)
		elapsed := time.Since(started)
		cancel()
		if elapsed > time.Second {
			t.Fatalf("mutation exceeded caller budget: %v", elapsed)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v, want caller deadline", err)
		}
	}
}
