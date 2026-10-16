package procutil

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"
)

// The weekly/manual macOS conformance lane runs this through the same lsof
// boundary as native admission, using only this test process and its temp files.
func TestLsofWritableNamesLiveConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH") != "1" {
		t.Skip("set PAIR_LIVE_COUCH=1 to compare writable-file evidence with real lsof")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	readPath, writePath := filepath.Join(dir, "read-only.txt"), filepath.Join(dir, "writable.txt")
	if err := os.WriteFile(readPath, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	readOnly, err := os.Open(readPath)
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	writable, err := os.OpenFile(writePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer writable.Close()
	observe := func() []string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		paths, err := LsofWritableNames(ctx, strconv.Itoa(os.Getpid()))
		if err != nil {
			t.Fatalf("real writable-file inspection: %v", err)
		}
		if slices.Contains(paths, readPath) {
			t.Fatal("read-only descriptor was reported writable")
		}
		return paths
	}
	if paths := observe(); !slices.Contains(paths, writePath) {
		t.Fatal("open writable descriptor was absent")
	}
	if err := writable.Close(); err != nil {
		t.Fatal(err)
	}
	if paths := observe(); slices.Contains(paths, writePath) {
		t.Fatal("closed writable descriptor remained present")
	}
	// Reopen read-only: file existence and earlier write access are not authority.
	reopened, err := os.Open(writePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if paths := observe(); slices.Contains(paths, writePath) {
		t.Fatal("reopened read-only descriptor was reported writable")
	}
}
