package testpair

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ShortRoot owns a canonical private root that leaves room for Couch's Unix
// socket names, including a ten-digit PID. Long inherited temporary paths
// cannot be used for these tests.
// Cleanup is registered before returning, so child/listener cleanups run first.
func ShortRoot(t testing.TB) string {
	t.Helper()
	root, err := NewShortRoot()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Errorf("remove owned short root: %v", err)
		}
	})
	return root
}

// NewShortRoot provides the same allocator to TestMain owners without a testing.TB.
// The caller owns removal after all children and listeners have joined.
func NewShortRoot() (string, error) {
	// The fallback must fit AF_UNIX after physical canonicalization.
	parents := []string{"/Volumes/TestScratch", os.TempDir(), "/tmp"}
	for _, parent := range parents {
		physical, err := filepath.EvalSymlinks(parent)
		if err != nil || len(filepath.Join(physical, "pc-1234567890")) > 36 {
			continue
		}
		root, err := os.MkdirTemp(physical, "pc-")
		if err != nil {
			continue
		}
		return root, nil
	}
	return "", errors.New("cannot allocate a private short root for Unix sockets")
}
