package titlepoller

import (
	"os"
	"path/filepath"
	"testing"
)

// Legacy namespace entries do not deny a bounded native mutation.
func TestNativeMutationPreservesLegacyNamespace(t *testing.T) {
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
	if err := (OSRuntime{}).RenamePane("pair-test", "terminal_7", "title"); err != nil {
		t.Errorf("legacy rename refused: %v", err)
	}
	if data, err := os.ReadFile(called); err != nil || string(data) != "called" {
		t.Fatalf("native invocation absent: %q %v", data, err)
	}
}
