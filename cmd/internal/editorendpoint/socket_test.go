package editorendpoint

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestEndpointSocketRequiresPrivatePhysicalOwner(t *testing.T) {
	root := endpointTestRuntime(t)
	path := filepath.Join(root, "e.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := ValidateEndpointSocket(path); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias.sock")
	if err := os.Symlink(path, alias); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEndpointSocket(alias); err == nil {
		t.Fatal("socket symlink accepted")
	}
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(root, 0700)
	if err := ValidateEndpointSocket(path); err == nil {
		t.Fatal("public runtime directory accepted")
	}
}

func endpointTestRuntime(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("", "ee-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(physical, 0700); err != nil {
		t.Fatal(err)
	}
	return physical
}
