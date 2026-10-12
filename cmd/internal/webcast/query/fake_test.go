package query

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/notes"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestFakeSurfaceMeetsTheSurfaceContract(t *testing.T) {
	RunSurfaceContract(t, func(recs []recording.Record) Surface {
		return FakeSurface{Records: recs, Model: notes.FakeModel{}}
	})
}

// Every fixture's notes come back with their element through the fakes: what
// #462 starts from.
func TestFakeSurfaceOnFixtures(t *testing.T) {
	for _, name := range recording.Fixtures() {
		r, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		ns, err := FakeSurface{Records: r.Records, Model: notes.FakeModel{}}.Notes(5000)
		if err != nil || len(ns) == 0 {
			t.Fatalf("%s: %v, %d notes", name, err, len(ns))
		}
		for _, n := range ns {
			if n.Element == "" || n.Context.Node != n.Note.At.Node {
				t.Errorf("%s: note %s has no element or context", name, n.Note.ID)
			}
		}
	}
}
