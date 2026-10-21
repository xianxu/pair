package mockserve

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func writeProjectFile(t *testing.T, dir, name, data string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func testProject(t *testing.T) (*Project, string) {
	t.Helper()
	d := t.TempDir()
	writeProjectFile(t, d, "fixtures.json", "{}")
	writeProjectFile(t, d, "mock-1/index.html", "one")
	p, e := OpenProject(d)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { p.Close() })
	return p, d
}
func TestVersionSelection(t *testing.T) {
	for _, n := range []string{"mock-1", "mock-2", "mock-7a", "mock-9999999999999999999999"} {
		if _, e := ParseVersion(n); e != nil {
			t.Errorf("%s: %v", n, e)
		}
	}
	for _, n := range []string{"mock-0", "mock-01", "mock-1A", "mock-1ab", "../mock-1", "mock-1\n", "mock--1"} {
		if _, e := ParseVersion(n); e == nil {
			t.Errorf("accepted %q", n)
		}
	}
	names := []string{"mock-2", "mock-10", "mock-10a", "mock-9z", "unrelated"}
	if got, e := SelectVersion(names, ""); e != nil || got != "mock-10a" {
		t.Fatalf("%q %v", got, e)
	}
	for _, n := range []string{"mock-3", "../mock-2"} {
		if _, e := SelectVersion(names, n); e == nil {
			t.Fatalf("accepted missing/invalid %s", n)
		}
	}
	if got, e := SelectVersion(names, "mock-2"); e != nil || got != "mock-2" {
		t.Fatal(got, e)
	}
}
func TestUseAtomicSelection(t *testing.T) {
	p, d := testProject(t)
	writeProjectFile(t, d, "mock-2/index.html", "two")
	if e := p.Use("mock-1"); e != nil {
		t.Fatal(e)
	}
	for _, v := range []string{"mock-9", "../mock-1"} {
		if e := p.Use(v); e == nil {
			t.Fatal("accepted", v)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := "mock-1"
			if i%2 == 0 {
				v = "mock-2"
			}
			if e := p.Use(v); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	q, e := OpenProject(d)
	if e != nil {
		t.Fatal(e)
	}
	defer q.Close()
	s, e := q.Scan(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if s.Version != "mock-1" && s.Version != "mock-2" {
		t.Fatal(s)
	}
	entries, e := os.ReadDir(d)
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) != 4 {
		t.Fatalf("temp residue: %v", entries)
	}
	// Rename failure must remove the temporary file.
	if e := os.Remove(filepath.Join(d, ".mock-current")); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(filepath.Join(d, ".mock-current"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := p.Use("mock-1"); e == nil {
		t.Fatal("expected rename failure")
	}
	entries, _ = os.ReadDir(d)
	if len(entries) != 4 {
		t.Fatal("temp residue", entries)
	}
}
func TestProjectRootConfinement(t *testing.T) {
	p, d := testProject(t)
	outside := t.TempDir()
	writeProjectFile(t, outside, "secret", "secret")
	if e := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(d, "escape")); e != nil {
		t.Fatal(e)
	}
	if _, e := p.readBounded("escape", 128); e == nil {
		t.Fatal("outside symlink read")
	}
	if _, e := p.readBounded("../secret", 128); e == nil {
		t.Fatal("traversal read")
	}
	if p.existsScreen("mock-1", "../fixtures.json") {
		t.Fatal("invalid screen")
	}
	if !p.existsScreen("mock-1", "index.html") {
		t.Fatal("missing screen")
	}
}

func TestUseAtomicSelectionReaders(t *testing.T) {
	p, d := testProject(t)
	writeProjectFile(t, d, "mock-2/index.html", "two")
	if e := p.Use("mock-1"); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			v := "mock-1"
			if i%2 == 0 {
				v = "mock-2"
			}
			if e := p.Use(v); e != nil {
				t.Error(e)
				return
			}
		}
	}()
	for i := 0; i < 200; i++ {
		b, e := p.readBounded(".mock-current", selectionLimit)
		if e != nil {
			t.Fatal(e)
		}
		if string(b) != "mock-1\n" && string(b) != "mock-2\n" {
			t.Fatalf("partial selection %q", b)
		}
	}
	wg.Wait()
}
func TestProjectBoundedReads(t *testing.T) {
	p, d := testProject(t)
	if _, e := p.readBounded("mock-1/index.html", 2); e == nil {
		t.Fatal("exceeded read bound")
	}
	if _, e := p.readBounded("mock-1", 100); e == nil {
		t.Fatal("accepted directory")
	}
	outside := t.TempDir()
	writeProjectFile(t, outside, "index.html", "outside secret")
	// Replacing a validated version with an outside link cannot escape the root.
	if e := os.Rename(filepath.Join(d, "mock-1"), filepath.Join(d, "saved")); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(d, "mock-1")); e != nil {
		t.Fatal(e)
	}
	if _, e := p.readBounded("mock-1/index.html", 100); e == nil {
		t.Fatal("outside ancestor symlink read")
	}
	if e := p.Use("mock-1"); e == nil {
		t.Fatal("outside version accepted")
	}
}

func TestProjectRejectsReplacementDuringOpen(t *testing.T) {
	for _, kind := range []string{"directory", "file", "file-in-parent", "scanner-directory", "scanner-file", "scanner-file-in-parent"} {
		t.Run(kind, func(t *testing.T) {
			p, dir := testProject(t)
			writeProjectFile(t, dir, "secret.html", "PRIVATE_ROOT_SECRET")
			writeProjectFile(t, dir, "index.html", "PRIVATE_ROOT_SECRET")
			writeProjectFile(t, dir, "mock-1/alternate.html", "different public file")
			matched := false
			p.beforeOpen = func(name string) {
				target := "mock-1"
				if strings.Contains(kind, "file") {
					target = "mock-1/index.html"
				}
				if name != target || matched {
					return
				}
				matched = true
				saved := "saved"
				destination := "."
				if strings.Contains(kind, "file") {
					saved = "mock-1/saved.html"
					destination = "../secret.html"
					if strings.HasSuffix(kind, "in-parent") {
						destination = "alternate.html"
					}
				}
				if err := os.Rename(filepath.Join(dir, target), filepath.Join(dir, saved)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(destination, filepath.Join(dir, target)); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			if strings.HasPrefix(kind, "scanner-") {
				_, err = p.Scan(t.Context())
			} else {
				_, err = p.readBounded("mock-1/index.html", MaxHTMLBytes)
			}
			if !matched || err == nil {
				t.Fatalf("replacement boundary not refused: matched=%v err=%v", matched, err)
			}
		})
	}
}
