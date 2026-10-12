package query

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestQuerierMeetsTheSurfaceContract(t *testing.T) {
	RunSurfaceContract(t, func(recs []recording.Record) Surface { return New(recs, notes.FakeModel{}) })
}

// Every query answers on every fixture, at every pinned moment.
func TestQuerierOnFixtures(t *testing.T) {
	for _, name := range recording.Fixtures {
		r, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		q := New(r.Records, notes.FakeModel{})
		for _, at := range recording.CanonicalTimes(r) {
			page, err := q.DOMAt(at, "")
			if err != nil {
				// A pin at a pause has no page only when nothing was live yet.
				t.Errorf("%s: DOMAt(%d): %v", name, at, err)
				continue
			}
			if n := strings.Count(page, "\n"); n > DefaultPageLines+3 {
				t.Errorf("%s: DOMAt(%d) is %d lines, over its budget", name, at, n)
			}
		}
		if _, err := EventLines(r.Records, 0, q.End()); err != nil {
			t.Errorf("%s: events: %v", name, err)
		}
		ns, err := q.Notes(DefaultBefore)
		if err != nil || len(ns) == 0 {
			t.Fatalf("%s: notes: %v (%d)", name, err, len(ns))
		}
		for _, n := range ns {
			if n.Element == "" || n.Landmark == "" || n.Header == "" {
				t.Errorf("%s: note %s lacks its element, landmark or page: %+v", name, n.Note.ID, n)
			}
		}
	}
}

// A note is reported with its state now, not as it was made: the app
// fixture resolves app-n2 after making it.
func TestNotesCarryTheirFinalState(t *testing.T) {
	r, err := recording.ReadDir(recording.Fixture("app"))
	if err != nil {
		t.Fatal(err)
	}
	ns, err := New(r.Records, notes.FakeModel{}).Notes(DefaultBefore)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range ns {
		want := recording.NoteOpen
		if n.Note.ID == "app-n2" {
			want = recording.NoteResolved
		}
		if n.Note.State != want {
			t.Errorf("%s: state %s, want %s", n.Note.ID, n.Note.State, want)
		}
	}
}
