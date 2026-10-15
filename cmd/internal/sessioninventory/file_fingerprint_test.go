package sessioninventory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintFileInfoDetectsSameSizeRewrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "evidence")
	if err := os.WriteFile(p, []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	a, err := FingerprintFileInfo(p, before)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := FingerprintFileInfo(p, after)
	if err != nil {
		t.Fatal(err)
	}
	if equalFingerprint(a, b) {
		t.Fatal("same-size rewrite with restored mtime reused evidence")
	}
	if _, err := FingerprintFileInfo(filepath.Dir(p), mustFingerprintStat(t, filepath.Dir(p))); err == nil {
		t.Fatal("directory accepted as evidence file")
	}
}
func mustFingerprintStat(t *testing.T, p string) os.FileInfo {
	t.Helper()
	i, e := os.Lstat(p)
	if e != nil {
		t.Fatal(e)
	}
	return i
}
