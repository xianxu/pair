package couchcore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingCouchNamespaceNeverCreatesOrReinterprets(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ns, err := ExistingCouchNamespace(root)
	if err != nil || ns.Dir() != root {
		t.Fatalf("namespace=%v err=%v", ns, err)
	}
	missing := filepath.Join(root, "missing")
	if _, err := ExistingCouchNamespace(missing); err == nil {
		t.Fatal("accepted missing")
	}
	if _, err := os.Lstat(missing); !os.IsNotExist(err) {
		t.Fatalf("created missing directory: %v", err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"relative", alias, filepath.Join(alias, ".")} {
		if _, err := ExistingCouchNamespace(path); err == nil {
			t.Fatalf("accepted noncanonical %q", path)
		}
	}
}
