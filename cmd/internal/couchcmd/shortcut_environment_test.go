package couchcmd

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortcutSocketRootIsPrivateShortAndCleaned(t *testing.T) {
	ambient := t.TempDir()
	t.Setenv("TMPDIR", filepath.Join(ambient, strings.Repeat("long", 25)))
	t.Setenv("ZELLIJ_SOCKET_DIR", ambient)
	var sockets string
	t.Run("fixture", func(t *testing.T) {
		setupShortcutSocketRoot(t)
		sockets = os.Getenv("ZELLIJ_SOCKET_DIR")
		if sockets == ambient {
			t.Fatal("inherited ambient socket root")
		}
		resolved, err := filepath.EvalSymlinks(sockets)
		if err != nil {
			t.Fatal(err)
		}
		version := filepath.Join(resolved, "0.45.1")
		if err := os.Mkdir(version, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(version, "pair-keys-"+strings.Repeat("f", 20))
		if len(path) > 103 {
			t.Fatalf("socket path exceeds Darwin budget: %d", len(path))
		}
		listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
	})
	if _, err := os.Stat(sockets); !os.IsNotExist(err) {
		t.Fatalf("socket root survived cleanup: %v", err)
	}
	if os.Getenv("ZELLIJ_SOCKET_DIR") != ambient {
		t.Fatal("fixture did not restore caller environment")
	}
}
