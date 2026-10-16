package historyarchive

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectedFilesTarAndChange(t *testing.T) {
	d := archiveTemp(t)
	p := filepath.Join(d, "x")
	os.WriteFile(p, []byte("abc"), 0600)
	s := SelectedFiles{Root: d}
	entries, e := s.Describe(context.Background(), []string{"x"})
	if e != nil {
		t.Fatal(e)
	}
	m := Manifest{Version: 1, Kind: "pair-terminal-events", Source: SourceIdentity{Host: "host", Root: d}, From: entries[0].Mtime, To: entries[0].Mtime, Compression: "zstd -19", Recipients: []string{"0123456789ABCDEF0123456789ABCDEF01234567"}, Files: entries}
	var b bytes.Buffer
	if e = s.WriteTar(context.Background(), &b, m); e != nil {
		t.Fatal(e)
	}
	r := tar.NewReader(&b)
	h, e := r.Next()
	if e != nil || h.Name != "MANIFEST.json" {
		t.Fatalf("manifest: %v %v", h, e)
	}
	h, e = r.Next()
	if e != nil || h.Name != "files/x" || h.Mode != 0600 || h.Uid != 0 {
		t.Fatalf("file: %v %v", h, e)
	}
	got, _ := io.ReadAll(r)
	if string(got) != "abc" {
		t.Fatal(string(got))
	}
	os.WriteFile(p, []byte("xyz"), 0600)
	os.Chtimes(p, time.Now(), entries[0].Mtime)
	if e = s.WriteTar(context.Background(), io.Discard, m); e == nil {
		t.Fatal("accepted changed content")
	}
}
func TestSelectedFilesRefuseDuplicateReplaceAndSymlink(t *testing.T) {
	d := archiveTemp(t)
	p := filepath.Join(d, "x")
	os.WriteFile(p, []byte("abc"), 0600)
	s := SelectedFiles{Root: d}
	if _, e := s.Describe(context.Background(), []string{"x", "x"}); e == nil {
		t.Fatal("duplicate accepted")
	}
	entries, e := s.Describe(context.Background(), []string{"x"})
	if e != nil {
		t.Fatal(e)
	}
	os.Rename(p, p+"old")
	os.WriteFile(p, []byte("abc"), 0600)
	os.Chtimes(p, time.Now(), entries[0].Mtime)
	m := Manifest{Version: 1, Kind: "pair-terminal-events", Source: SourceIdentity{Host: "host", Root: d}, From: entries[0].Mtime, To: entries[0].Mtime, Compression: "zstd -19", Recipients: []string{"0123456789ABCDEF0123456789ABCDEF01234567"}, Files: entries}
	if e := ValidateManifest(m); e != nil {
		t.Fatal(e)
	}
	if e = s.WriteTar(context.Background(), io.Discard, m); e == nil {
		t.Fatal("replacement accepted")
	}
	os.Symlink("x", filepath.Join(d, "link"))
	if _, e = s.Describe(context.Background(), []string{"link"}); e == nil {
		t.Fatal("link accepted")
	}
}
