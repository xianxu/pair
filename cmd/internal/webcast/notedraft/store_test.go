package notedraft

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func storePNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestDiskStoreDurability(t *testing.T) {
	root := filepath.Join(t.TempDir(), "captures")
	s, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	p := storePNG(t)
	if err = s.put("abc", []byte(`{"v":1}`), p); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, filepath.Join(root, "abc"), filepath.Join(root, "abc", "metadata.json"), filepath.Join(root, "abc", "capture.png")} {
		st, e := os.Stat(path)
		if e != nil {
			t.Fatal(e)
		}
		want := os.FileMode(0600)
		if st.IsDir() {
			want = 0700
		}
		if st.Mode().Perm() != want {
			t.Fatalf("%s mode %o", path, st.Mode().Perm())
		}
	}
	s, err = newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.update("abc", []byte(`{"v":2}`)); err != nil {
		t.Fatal(err)
	}
	m, got, err := s.read("abc")
	if err != nil || string(m) != `{"v":2}` || !bytes.Equal(p, got) {
		t.Fatalf("read %s %v", m, err)
	}
	n, b, err := s.usage()
	if err != nil || n != 1 || b != int64(len(m)+len(p)) {
		t.Fatalf("usage %d %d %v", n, b, err)
	}
	if err = s.put("abc", m, append(p, 0)); err == nil {
		t.Fatal("changed PNG accepted")
	}
	if err = s.remove("abc"); err != nil {
		t.Fatal(err)
	}
	ids, err := s.list()
	if err != nil || len(ids) != 0 {
		t.Fatalf("list %v %v", ids, err)
	}
}
func TestDiskStoreRejectsUnsafeAndOversize(t *testing.T) {
	s, err := newDiskStore(filepath.Join(t.TempDir(), "captures"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "../escape", "a/b", ".", strings.Repeat("a", 129)} {
		if err = s.put(id, []byte(`{}`), nil); err == nil {
			t.Fatalf("accepted %q", id)
		}
		if _, _, err = s.read(id); err == nil {
			t.Fatalf("read accepted %q", id)
		}
		if err = s.remove(id); err == nil {
			t.Fatalf("remove accepted %q", id)
		}
	}
	for _, m := range [][]byte{[]byte("not json"), []byte(`"` + strings.Repeat("a", 65536) + `"`)} {
		if err = s.put("bad", m, nil); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	if err = s.put("bad", []byte(`{}`), []byte("not png")); err == nil {
		t.Fatal("invalid PNG accepted")
	}
	outside := t.TempDir()
	if err = os.Symlink(outside, filepath.Join(s.root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.read("link"); err == nil {
		t.Fatal("symlink read accepted")
	}
	if err = s.remove("link"); err == nil {
		t.Fatal("symlink remove accepted")
	}
	if err = s.put("safe", []byte(`{}`), storePNG(t)); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(s.root, "safe", "metadata.json")); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(filepath.Join(outside, "meta"), filepath.Join(s.root, "safe", "metadata.json")); err != nil {
		t.Fatal(err)
	}
	if err = s.update("safe", []byte(`{}`)); err == nil {
		t.Fatal("metadata symlink accepted")
	}
}
func TestDiskStoreIgnoresForeignEntriesAndBoundsRecords(t *testing.T) {
	s, err := newDiskStore(filepath.Join(t.TempDir(), "captures"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.root, ".DS_Store"), []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxRecords; i++ {
		if err = s.put(fmt.Sprint(i), []byte(`{}`), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.put("overflow", []byte(`{}`), nil); err == nil {
		t.Fatal("record quota not enforced")
	}
	n, b, err := s.usage()
	if err != nil || n != MaxRecords || b != 2*MaxRecords {
		t.Fatalf("usage %d %d %v", n, b, err)
	}
}

func TestDiskStoreFirstCaptureAndFailedUpdate(t *testing.T) {
	s, err := newDiskStore(filepath.Join(t.TempDir(), "captures"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.put("record", []byte(`{"phase":"preparing"}`), nil); err != nil {
		t.Fatal(err)
	}
	p := storePNG(t)
	m := []byte(`{"phase":"ready"}`)
	if err = s.put("record", m, p); err != nil {
		t.Fatal(err)
	}
	if err = s.update("record", []byte(`"`+strings.Repeat("x", 65536)+`"`)); err == nil {
		t.Fatal("oversize update accepted")
	}
	got, pixels, err := s.read("record")
	if err != nil || !bytes.Equal(got, m) || !bytes.Equal(pixels, p) {
		t.Fatalf("failed update changed record: %s %v", got, err)
	}
	if err = s.put("record", m, nil); err == nil {
		t.Fatal("PNG removal accepted")
	}
	if err = s.put("record", m, p); err != nil {
		t.Fatalf("identical retry: %v", err)
	}
}
func TestDiskStoreByteQuotaAccountsActualFiles(t *testing.T) {
	s, err := newDiskStore(filepath.Join(t.TempDir(), "captures"))
	if err != nil {
		t.Fatal(err)
	}
	// Sparse files exercise real disk accounting without allocating 256 MiB in RAM.
	for i := 0; i < 32; i++ {
		id := fmt.Sprint(i)
		if err = s.put(id, []byte(`{}`), nil); err != nil {
			t.Fatal(err)
		}
		f, e := os.OpenFile(filepath.Join(s.root, id, "capture.png"), os.O_CREATE|os.O_WRONLY, 0600)
		if e != nil {
			t.Fatal(e)
		}
		e = f.Truncate(MaxPNGBytes - 2)
		_ = f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	n, size, err := s.usage()
	if err != nil || n != 32 || size != MaxStoreBytes {
		t.Fatalf("usage %d %d %v", n, size, err)
	}
	if err = s.put("overflow", []byte(`{}`), nil); err == nil || err.Error() != StoreFull {
		t.Fatalf("byte quota: %v", err)
	}
}
func TestDiskStoreRejectsPNGAndRootSymlinks(t *testing.T) {
	root := filepath.Join(t.TempDir(), "captures")
	s, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.put("safe", []byte(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("metadata.json", filepath.Join(root, "safe", "capture.png")); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.read("safe"); err == nil {
		t.Fatal("PNG symlink read")
	}
	if _, _, err = s.usage(); err == nil {
		t.Fatal("PNG symlink accounting")
	}
	if err = s.remove("safe"); err == nil {
		t.Fatal("PNG symlink deletion")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err = os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if _, err = newDiskStore(alias); err == nil {
		t.Fatal("root symlink accepted")
	}
}

func TestDiskStoreReopenCleansInterruptedStaging(t *testing.T) {
	root := filepath.Join(t.TempDir(), "captures")
	s, err := newDiskStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.put("keep", []byte(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	stage := ".stage-0123456789abcdef0123456789abcdef"
	if err = os.Mkdir(filepath.Join(root, stage), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, stage, "metadata.json"), filepath.Join(root, stage, "capture.png"), filepath.Join(root, "keep", stage)} {
		if err = os.WriteFile(path, []byte("interrupted"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = newDiskStore(root); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, stage), filepath.Join(root, "keep", stage)} {
		if _, err = os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("staging retained: %s %v", path, err)
		}
	}
	if _, _, err = s.read("keep"); err != nil {
		t.Fatal(err)
	}
}

func TestDiskStoreRemovalRefusesForeignChildrenWithoutLosingEvidence(t *testing.T) {
	s, err := newDiskStore(filepath.Join(t.TempDir(), "captures"))
	if err != nil {
		t.Fatal(err)
	}
	p := storePNG(t)
	if err = s.put("record", []byte(`{}`), p); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(s.root, "record", "foreign"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = s.remove("record"); err == nil {
		t.Fatal("foreign child deletion accepted")
	}
	_, got, err := s.read("record")
	if err != nil || !bytes.Equal(got, p) {
		t.Fatalf("failed deletion damaged evidence: %v", err)
	}
}
