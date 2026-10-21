package mockserve

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanLinearWork(t *testing.T) {
	for _, n := range []int{20, 40} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			p, d := testProject(t)
			for i := 0; i < n; i++ {
				writeProjectFile(t, d, fmt.Sprintf("assets/d%d/a.css", i), "12345")
			}
			visits := 0
			var reads int64
			p.scanWork = func(entries int, bytes int64) { visits += entries; reads += bytes }
			s, e := p.Scan(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			if visits != 4+2*n || reads != int64(5+5*n) || s.Entries != visits || s.Bytes != reads {
				t.Fatalf("snapshot %+v; observed entry visits=%d bytes read=%d", s, visits, reads)
			}
		})
	}
}
func TestScanChangesAndBounds(t *testing.T) {
	t.Run("changes", func(t *testing.T) {
		p, d := testProject(t)
		old, e := p.Scan(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if again, e := p.Scan(context.Background()); e != nil || again.Digest != old.Digest {
			t.Fatal(again, e)
		}
		for _, change := range []func(){func() {
			f := filepath.Join(d, "mock-1/index.html")
			st, _ := os.Stat(f)
			writeProjectFile(t, d, "mock-1/index.html", "two")
			os.Chtimes(f, st.ModTime(), st.ModTime())
		}, func() { writeProjectFile(t, d, "a.css", "a") }, func() { writeProjectFile(t, d, "b.js", "b") }, func() { writeProjectFile(t, d, "fixtures.json", "[]") }, func() { os.Remove(filepath.Join(d, "a.css")) }, func() { os.Mkdir(filepath.Join(d, "empty"), 0700) }, func() { writeProjectFile(t, d, "mock-2/index.html", "two") }} {
			change()
			next, e := p.Scan(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			if next.Digest == old.Digest {
				t.Fatal("missed change")
			}
			old = next
		}
		if old.Version != "mock-2" {
			t.Fatal(old)
		}
	})
	for _, kind := range []string{"empty-dirs", "files", "html", "bytes", "json", "symlink", "selection", "no-index"} {
		t.Run(kind, func(t *testing.T) {
			p, d := testProject(t)
			switch kind {
			case "empty-dirs":
				for i := 0; i < MaxEntries; i++ {
					os.Mkdir(filepath.Join(d, fmt.Sprint(i)), 0700)
				}
			case "files":
				os.Mkdir(filepath.Join(d, "many"), 0700)
				for i := 0; i < MaxEntries; i++ {
					writeProjectFile(t, d, fmt.Sprintf("many/%d", i), "")
				}
			case "html":
				writeProjectFile(t, d, "mock-1/index.html", strings.Repeat("x", int(MaxHTMLBytes)+1))
			case "bytes":
				f, e := os.Create(filepath.Join(d, "large.bin"))
				if e != nil {
					t.Fatal(e)
				}
				f.Truncate(MaxBytes + 1)
				f.Close()
			case "json":
				writeProjectFile(t, d, "fixtures.json", "{")
			case "symlink":
				os.Symlink("fixtures.json", filepath.Join(d, "link"))
			case "selection":
				writeProjectFile(t, d, ".mock-current", "mock-99\n")
			case "no-index":
				os.Remove(filepath.Join(d, "mock-1/index.html"))
			}
			if _, e := p.Scan(context.Background()); e == nil {
				t.Fatal("accepted invalid tree")
			}
			if kind == "json" {
				writeProjectFile(t, d, "fixtures.json", "{}")
				if _, e := p.Scan(context.Background()); e != nil {
					t.Fatal("repair failed", e)
				}
			}
		})
	}
	t.Run("cancel", func(t *testing.T) {
		p, _ := testProject(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, e := p.Scan(ctx); e != context.Canceled {
			t.Fatal(e)
		}
	})
}

func TestScanStrictSelection(t *testing.T) {
	for _, selection := range []string{"", "mock-1\n\n", " mock-1", "mock-1\r\n", "mock-01", strings.Repeat("x", 257)} {
		t.Run(fmt.Sprintf("%q", selection), func(t *testing.T) {
			p, d := testProject(t)
			writeProjectFile(t, d, ".mock-current", selection)
			if _, e := p.Scan(context.Background()); e == nil {
				t.Fatal("accepted malformed selection")
			}
		})
	}
	p, d := testProject(t)
	writeProjectFile(t, d, "mock-2/index.html", "two")
	if e := p.Use("mock-1"); e != nil {
		t.Fatal(e)
	}
	s, e := p.Scan(context.Background())
	if e != nil || s.Version != "mock-1" {
		t.Fatal(s, e)
	}
	if e := os.Remove(filepath.Join(d, "fixtures.json")); e != nil {
		t.Fatal(e)
	}
	if _, e := p.Scan(context.Background()); e == nil {
		t.Fatal("missing fixtures accepted")
	}
}
