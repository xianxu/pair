package couchmessage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSocketPathExplicitIsolationIgnoresAmbient(t *testing.T) {
	root, err := filepath.EvalSymlinks(filepath.Dir(transportSocket(t)))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("COUCH_ISOLATED_ROOT", filepath.Join(root, "foreign"))
	ns := filepath.Join(root, "store")
	got, err := SocketPathForIsolation(ns, "editor-endpoint", root)
	want, _ := selectMessageSocket(ns, "editor-endpoint", filepath.Join(root, "m"), true)
	if err != nil || got != want {
		t.Fatalf("explicit selection = %q, %v; want %q", got, err, want)
	}
	if _, err := SocketPathForIsolation(filepath.Dir(root), "editor-endpoint", root); err == nil {
		t.Fatal("escaped namespace accepted")
	}
	if os.Getenv("COUCH_ISOLATED_ROOT") != filepath.Join(root, "foreign") {
		t.Fatal("ambient environment mutated")
	}
}
