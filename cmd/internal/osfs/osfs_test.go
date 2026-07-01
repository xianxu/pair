package osfs

import (
	"path/filepath"
	"testing"
)

func TestFSRoundTrip(t *testing.T) {
	var fs FS
	p := filepath.Join(t.TempDir(), "sub", "f.txt") // sub/ doesn't exist -> MkdirAll
	if _, ok := fs.FileSize(p); ok {
		t.Fatal("absent file should report ok=false")
	}
	if fs.Executable(p) {
		t.Fatal("absent file not executable")
	}
	if err := fs.WriteFile(p, "hello"); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if got, err := fs.ReadFile(p); err != nil || got != "hello" {
		t.Fatalf("ReadFile = %q, %v", got, err)
	}
	if n, ok := fs.FileSize(p); !ok || n != 5 {
		t.Fatalf("FileSize = %d, %v", n, ok)
	}
	if err := fs.WriteAtomic(p, "world!!"); err != nil {
		t.Fatalf("WriteAtomic: %v", err)
	}
	if got, _ := fs.ReadFile(p); got != "world!!" {
		t.Fatalf("after atomic = %q", got)
	}
	if _, ok := fs.ModTime(p); !ok {
		t.Fatal("ModTime should resolve")
	}
	fs.Remove(p)
	if _, ok := fs.FileSize(p); ok {
		t.Fatal("removed file should be gone")
	}
}
