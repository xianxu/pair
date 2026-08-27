package couchcore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCouchNamespaceResolvesRelativeStoreOnceAgainstStartupCWD(t *testing.T) {
	startup := t.TempDir()

	ns, err := ResolveCouchNamespace(filepath.Join("state", "couch"), startup)
	if err != nil {
		t.Fatalf("ResolveCouchNamespace: %v", err)
	}
	want, err := filepath.EvalSymlinks(filepath.Join(startup, "state", "couch"))
	if err != nil {
		t.Fatalf("EvalSymlinks(want): %v", err)
	}
	if ns.Dir() != want {
		t.Fatalf("Dir = %q, want %q", ns.Dir(), want)
	}
	info, err := os.Stat(ns.Dir())
	if err != nil {
		t.Fatalf("resolved namespace was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("resolved namespace mode = %v, want directory", info.Mode())
	}
}

func TestCouchNamespaceCollapsesPhysicalAliases(t *testing.T) {
	realParent := t.TempDir()
	aliasParent := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realParent, aliasParent); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	throughAlias, err := ResolveCouchNamespace(filepath.Join(aliasParent, "couch"), "/unused")
	if err != nil {
		t.Fatalf("ResolveCouchNamespace(alias): %v", err)
	}
	physical, err := ResolveCouchNamespace(filepath.Join(realParent, "couch"), "/unused")
	if err != nil {
		t.Fatalf("ResolveCouchNamespace(physical): %v", err)
	}
	if throughAlias != physical {
		t.Fatalf("alias namespace = %#v, physical namespace = %#v", throughAlias, physical)
	}
}

func TestCouchNamespaceRejectsEmptyStorePath(t *testing.T) {
	if _, err := ResolveCouchNamespace("", t.TempDir()); err == nil {
		t.Fatal("empty store path must not silently select the startup directory")
	}
}
