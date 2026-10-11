package sessionsync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func identity(p string) string { return p }

func TestOwedExcludesBeforeSinceFoldedAndOtherFilesSubmits(t *testing.T) {
	rows := []Entry{
		{ID: "old", From: "pair:1", At: t0.Add(-time.Minute)},
		{ID: "late", From: "pair:2", At: t0.Add(2 * time.Minute)},
		{ID: "sub", From: "ops:0", At: t0.Add(time.Minute), Head: SubmitPrefix + "/w/tl.md"},
		{ID: "other", From: "ops:0", At: t0, Head: SubmitPrefix + "/w/else.md"},
		{ID: "rel", From: "ops:0", At: t0, Head: SubmitPrefix + "tl.md"},
		{ID: "done", From: "pair:3", At: t0},
		{ID: "late", From: "pair:2", At: t0.Add(2 * time.Minute)},
		{ID: "maybe", From: "pair:4", At: t0, Uncertain: true},
	}
	got := Owed(rows, State{Since: t0, Folded: []string{"done"}}, "/w/tl.md", identity)
	want := []OwedEvent{
		{ID: "rel", Kind: KindDelivery, From: "ops:0", At: t0},
		{ID: "maybe", Kind: KindDelivery, From: "pair:4", At: t0, Uncertain: true},
		{ID: "sub", Kind: KindSubmit, From: "ops:0", At: t0.Add(time.Minute)},
		{ID: "late", Kind: KindDelivery, From: "pair:2", At: t0.Add(2 * time.Minute)},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestOwedIsAnEmptyArrayWhenNothingIsOwed(t *testing.T) {
	if got := Owed(nil, State{}, "/f", identity); got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestOwedMatchesASubmitThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	os.Mkdir(real, 0o700)
	os.WriteFile(filepath.Join(real, "tl.md"), nil, 0o600)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	rows := []Entry{{ID: "s", At: t0, Head: SubmitPrefix + filepath.Join(link, "tl.md")}}
	got := Owed(rows, State{}, filepath.Join(real, "tl.md"), Canonical)
	if len(got) != 1 || got[0].Kind != KindSubmit {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadStateFallsBackToTheShadowMtime(t *testing.T) {
	dir := t.TempDir()
	p := PathsOf(filepath.Join(dir, "tl.md"))
	if st, err := LoadState(p); err != nil || !st.Since.IsZero() {
		t.Fatalf("no shadow: %+v %v", st, err)
	}
	os.WriteFile(p.Shadow, []byte("x"), 0o600)
	os.Chtimes(p.Shadow, t0, t0)
	if st, _ := LoadState(p); !st.Since.Equal(t0) {
		t.Fatalf("got %v", st.Since)
	}
	if err := saveState(p, State{Since: t0.Add(time.Hour), Folded: []string{"a"}}); err != nil {
		t.Fatal(err)
	}
	if st, _ := LoadState(p); !st.Since.Equal(t0.Add(time.Hour)) || st.Folded[0] != "a" {
		t.Fatalf("got %+v", st)
	}
}

func TestFoldPrunesIDsThatLeftTheLedger(t *testing.T) {
	st := State{Folded: []string{"gone", "a"}}
	st.Fold([]string{"b", "a", "typo"}, []Entry{{ID: "a"}, {ID: "b"}})
	if len(st.Folded) != 2 || st.Folded[0] != "a" || st.Folded[1] != "b" {
		t.Fatalf("got %v", st.Folded)
	}
}
