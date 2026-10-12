package notes

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func reduceAll(t *testing.T, b *recording.Builder) *recording.State {
	t.Helper()
	st, err := recording.Reduce(b.Recs, 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// fixtureNote is a fixture's note and the state at its time.
func fixtureNote(t *testing.T, fixture, id string) (*recording.Note, *recording.State) {
	t.Helper()
	r, err := recording.ReadDir(recording.Fixture(fixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range r.Records {
		if n, ok := x.(*recording.Note); ok && n.ID == id {
			st, err := recording.Reduce(r.Records, n.T)
			if err != nil {
				t.Fatal(err)
			}
			return n, st
		}
	}
	t.Fatalf("%s has no note %s", fixture, id)
	return nil, nil
}

// fixtureEnd is a fixture's state after its last record.
func fixtureEnd(t *testing.T, fixture string) *recording.State {
	t.Helper()
	r, err := recording.ReadDir(recording.Fixture(fixture))
	if err != nil {
		t.Fatal(err)
	}
	st, err := recording.Reduce(r.Records, 1<<62)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// walkElements calls f on every element of n's tree, through shadow roots
// frame documents and template contents.
func walkElements(n *recording.DOMNode, f func(*recording.DOMNode)) {
	if n == nil {
		return
	}
	if n.Y == recording.YEl {
		f(n)
	}
	for _, k := range n.Kids {
		walkElements(k, f)
	}
	walkElements(n.Shadow, f)
	walkElements(n.Frame, f)
	walkElements(n.Content, f)
}
