package notes

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestFakeModelMeetsTheModelContract(t *testing.T) {
	RunModelContract(t, func() Model { return FakeModel{} })
}

// Every fixture note resolves, at its own moment, to the element it names.
func TestFakeModelOnFixtures(t *testing.T) {
	for _, name := range recording.Fixtures {
		r, err := recording.ReadDir(recording.Fixture(name))
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range r.Records {
			n, ok := x.(*recording.Note)
			if !ok {
				continue
			}
			st, err := recording.Reduce(r.Records, n.T)
			if err != nil {
				t.Fatal(err)
			}
			if id, ok := (FakeModel{}).Resolve(st, n.At); !ok || id != n.At.Node {
				t.Errorf("%s: note %s doesn't resolve", name, n.ID)
			}
			if n.At.Box != nil {
				if crops, err := (FakeModel{}).Crops(st, []recording.NoteAt{n.At}, nil); err != nil || !crops[0].Contains(*n.At.Box) {
					t.Errorf("%s: note %s crop %v %v", name, n.ID, crops, err)
				}
			}
		}
	}
}
