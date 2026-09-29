package couchcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/storagegc"
)

func TestListWithMissingAuxiliaryStoreUsesIsolatedRoots(t *testing.T) {
	operatorRoot := t.TempDir()
	operatorRegistry := filepath.Join(operatorRoot, "stores.json")
	sentinel := []byte("operator registry sentinel")
	if err := os.WriteFile(operatorRegistry, sentinel, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", operatorRoot)
	t.Setenv("COUCH_STORE_DIR", operatorRoot)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "PAIR_") || strings.HasPrefix(key, "COUCH_") {
			t.Setenv(key, "")
		}
	}
	home, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", data)
	t.Chdir(t.TempDir())
	root := launcher.ResolveDataDir(home, data)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PAIR_DATA_DIR", root)
	t.Setenv("COUCH_STORE_DIR", filepath.Join(root, "couch"))
	c, err := storagegc.NewCoordinator(root)
	if err != nil {
		t.Fatal(err)
	}
	auxiliary := t.TempDir()
	if err := c.RegisterStore(context.Background(), auxiliary); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(auxiliary); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := Run([]string{"--list"}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("production --list rejected intact store: %d %s", code, stderr.String())
	}
	after, err := os.ReadFile(operatorRegistry)
	if err != nil || !bytes.Equal(after, sentinel) {
		t.Fatalf("ambient registry modified: %q %v", after, err)
	}
	if _, err := c.ReadRegistry(); err == nil {
		t.Fatal("listing silently removed missing registration")
	}
}
