package couchcore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CouchNamespace is the durable identity of one couch store. Its directory is
// resolved once to an absolute physical path and is safe to compare directly.
type CouchNamespace struct {
	dir string
}

func (n CouchNamespace) Dir() string { return n.dir }

// ResolveCouchNamespace creates and canonicalizes storeDir. Relative values
// are interpreted exactly once against startupCWD, before any child can change
// cwd and reinterpret the namespace.
func ResolveCouchNamespace(storeDir, startupCWD string) (CouchNamespace, error) {
	if strings.TrimSpace(storeDir) == "" {
		return CouchNamespace{}, fmt.Errorf("resolve couch namespace: empty store path")
	}
	if !filepath.IsAbs(storeDir) {
		if strings.TrimSpace(startupCWD) == "" {
			return CouchNamespace{}, fmt.Errorf("resolve couch namespace: empty startup cwd")
		}
		storeDir = filepath.Join(startupCWD, storeDir)
	}
	abs, err := filepath.Abs(filepath.Clean(storeDir))
	if err != nil {
		return CouchNamespace{}, fmt.Errorf("resolve couch namespace %q: %w", storeDir, err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return CouchNamespace{}, fmt.Errorf("create couch namespace %q: %w", abs, err)
	}
	physical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return CouchNamespace{}, fmt.Errorf("resolve physical couch namespace %q: %w", abs, err)
	}
	physical, err = filepath.Abs(filepath.Clean(physical))
	if err != nil {
		return CouchNamespace{}, fmt.Errorf("absolutize physical couch namespace %q: %w", physical, err)
	}
	return CouchNamespace{dir: physical}, nil
}

// ExistingCouchNamespace validates an already registered physical namespace
// without creating a directory or silently following a changed registration.
func ExistingCouchNamespace(path string) (CouchNamespace, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return CouchNamespace{}, fmt.Errorf("noncanonical couch namespace %q", path)
	}
	physical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return CouchNamespace{}, err
	}
	if physical != path {
		return CouchNamespace{}, fmt.Errorf("couch namespace changed physical identity: %q", path)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return CouchNamespace{}, err
	}
	if !info.IsDir() {
		return CouchNamespace{}, fmt.Errorf("couch namespace is not a directory: %q", path)
	}
	return CouchNamespace{dir: path}, nil
}
