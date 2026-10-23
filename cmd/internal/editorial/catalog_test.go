package editorial

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xianxu/ariadne/pkg/vocab"
)

func TestCatalogProcessingPrefixes(t *testing.T) {
	c, err := DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	modes := c.Modes()
	if len(modes) != 6 {
		t.Fatalf("got %d modes", len(modes))
	}
	for i, m := range modes {
		if m.Order != i+1 || m.Brief == "" {
			t.Errorf("invalid mode: %+v", m)
		}
		if got, ok := c.Lookup(m.Name); !ok || got != m {
			t.Errorf("Lookup(%q)=%+v %v", m.Name, got, ok)
		}
		for n := 1; n <= len(m.Name); n++ {
			prefix := m.Name[:n]
			matches := 0
			for _, other := range modes {
				if strings.HasPrefix(other.Name, prefix) {
					matches++
				}
			}
			got, err := c.ResolveProcessing(prefix)
			if matches == 1 && (err != nil || got != m) {
				t.Errorf("%q: %+v %v", prefix, got, err)
			}
			if matches > 1 && err == nil {
				t.Errorf("ambiguous prefix %q accepted", prefix)
			}
		}
	}
	for _, s := range []string{"", "unknown", " copy", "copy-editing ", "[copy-editing]", "copy-editing,proofreading", "COPY", "\x00", "🤖"} {
		if _, err := c.ResolveProcessing(s); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
	modes[0].Name = "mutated"
	if c.Modes()[0].Name == "mutated" {
		t.Fatal("Modes exposes state")
	}
}

func TestCatalogMalformedMetadata(t *testing.T) {
	good := "---\nname: sample\norder: 1\nscope: whole-doc\nfrontier: off\n---\nBrief.\n"
	for _, tc := range []struct{ name, raw string }{
		{"missing delimiter", strings.TrimPrefix(good, "---\n")},
		{"missing closing", strings.Replace(good, "\n---\nBrief.", "\nBrief.", 1)},
		{"bad order", strings.Replace(good, "order: 1", "order: x", 1)},
		{"zero order", strings.Replace(good, "order: 1", "order: 0", 1)},
		{"bad scope", strings.Replace(good, "whole-doc", "other", 1)},
		{"bad frontier", strings.Replace(good, "off", "maybe", 1)},
		{"missing name", strings.Replace(good, "name: sample\n", "", 1)},
		{"duplicate key", strings.Replace(good, "name: sample", "name: sample\nname: other", 1)},
		{"unknown key", strings.Replace(good, "name: sample", "extra: yes\nname: sample", 1)},
		{"list name", strings.Replace(good, "sample", "[sample]", 1)},
		{"empty brief", strings.Replace(good, "Brief.\n", "", 1)},
		{"nested", strings.Replace(good, "order: 1", " order: 1", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewCatalog(fstest.MapFS{"modes/sample.md": {Data: []byte(tc.raw)}}); err == nil {
				t.Fatal("accepted malformed metadata")
			}
		})
	}
	for _, dup := range []string{good, strings.Replace(good, "sample", "other", 1)} {
		if _, err := NewCatalog(fstest.MapFS{"modes/a.md": {Data: []byte(good)}, "modes/b.md": {Data: []byte(dup)}}); err == nil {
			t.Fatal("accepted duplicate name/order")
		}
	}
	if _, err := NewCatalog(fstest.MapFS{}); err == nil {
		t.Fatal("accepted empty catalog")
	}
}

func TestCatalogAxisResolvers(t *testing.T) {
	for _, s := range vocab.CollaborationModes {
		if got, err := ResolveCollaboration(s); err != nil || got != s {
			t.Fatalf("%s: %s %v", s, got, err)
		}
	}
	for _, s := range []string{"direct", "highlight", "propose"} {
		if got, err := ResolvePresentation(s); err != nil || got != s {
			t.Fatalf("%s: %s %v", s, got, err)
		}
	}
	for _, s := range []string{"", "unknown", "dir", "direct ", "[chat]"} {
		if _, err := ResolvePresentation(s); err == nil {
			t.Errorf("presentation accepted %q", s)
		}
		if _, err := ResolveCollaboration(s); err == nil {
			t.Errorf("collaboration accepted %q", s)
		}
	}
}

func FuzzCatalogProcessing(f *testing.F) {
	for _, s := range []string{"", "f", "copy", "[copy-editing] fix", "copy-editing,proofreading", "\xff\x00", "🤖[]"} {
		f.Add(s)
	}
	c, err := DefaultCatalog()
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got, err := c.ResolveProcessing(s)
		if err == nil {
			if s == "" || !strings.HasPrefix(got.Name, s) {
				t.Fatalf("invalid resolution %q: %+v", s, got)
			}
		}
	})
}

func TestCatalogMetadataAndExactNamePriority(t *testing.T) {
	source := fstest.MapFS{
		"modes/z.md": {Data: []byte("---\nname: sample-more\norder: 2\nscope: markers-only\nfrontier: on\n---\nSecond.\n")},
		"modes/a.md": {Data: []byte("---\nname: sample\norder: 1\nscope: whole-doc\nfrontier: off\n---\nFirst.\n")},
	}
	c, err := NewCatalog(source)
	if err != nil {
		t.Fatal(err)
	}
	modes := c.Modes()
	if modes[0] != (Mode{Name: "sample", Order: 1, Scope: WholeDocument, Frontier: false, Brief: "First.\n"}) || modes[1] != (Mode{Name: "sample-more", Order: 2, Scope: MarkersOnly, Frontier: true, Brief: "Second.\n"}) {
		t.Fatalf("metadata not preserved: %+v", modes)
	}
	if got, err := c.ResolveProcessing("sample"); err != nil || got != modes[0] {
		t.Fatalf("exact name must win: %+v %v", got, err)
	}
	if _, ok := c.Lookup("sam"); ok {
		t.Fatal("Lookup accepted prefix")
	}
}

func TestCatalogInvalidProcessingListsChoices(t *testing.T) {
	c, err := DefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ResolveProcessing("invalid")
	if err == nil {
		t.Fatal("accepted invalid mode")
	}
	for _, m := range c.Modes() {
		if !strings.Contains(err.Error(), m.Name) {
			t.Errorf("diagnostic omits %s: %v", m.Name, err)
		}
	}
}
