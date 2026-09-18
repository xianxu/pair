package keyhelp

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func sec(title string, rows ...Binding) Section { return Section{Title: title, Bindings: rows} }

func TestLayerPutsHostFirstAndDropsClaimedChordsEverywhere(t *testing.T) {
	host := []Section{sec("Host", Binding{Key: "Alt+h", Desc: "host help", Chord: workbenchshortcut.ChordAltH, Context: ContextHost})}
	pair := []Section{
		sec("Session", Binding{Key: "Alt+h", Desc: "pair help", Chord: workbenchshortcut.ChordAltH}, Binding{Key: "Alt+l", Desc: "log", Chord: workbenchshortcut.ChordAltL}),
		sec("Only claimed", Binding{Key: "Alt+h", Desc: "pair help again", Chord: workbenchshortcut.ChordAltH}),
		sec("Draft", Binding{Key: "Alt+⏎", Desc: "send"}),
	}
	got := Layer(host, []workbenchshortcut.Chord{workbenchshortcut.ChordAltH}, pair)
	if len(got) != 3 || got[0].Title != "Host" || got[1].Title != "Session" || got[2].Title != "Draft" {
		t.Fatalf("sections = %+v", got)
	}
	for _, s := range got[1:] {
		for _, b := range s.Bindings {
			if b.Chord == workbenchshortcut.ChordAltH {
				t.Errorf("claimed chord survived in %q: %+v", s.Title, b)
			}
		}
	}
	if len(got[1].Bindings) != 1 || got[1].Bindings[0].Key != "Alt+l" {
		t.Errorf("unclaimed row lost: %+v", got[1])
	}
}

// A zero chord means "no identity", so a draft-local row can never be claimed.
func TestLayerNeverDropsChordlessRows(t *testing.T) {
	got := Layer(nil, []workbenchshortcut.Chord{0}, []Section{sec("Draft", Binding{Key: "Alt+⏎", Desc: "send"})})
	if len(got) != 1 || len(got[0].Bindings) != 1 {
		t.Fatalf("chordless row dropped: %+v", got)
	}
}

func TestLayerDoesNotMutateItsInputs(t *testing.T) {
	pair := []Section{sec("Session", Binding{Key: "Alt+h", Chord: workbenchshortcut.ChordAltH}, Binding{Key: "Alt+l", Chord: workbenchshortcut.ChordAltL})}
	Layer(nil, []workbenchshortcut.Chord{workbenchshortcut.ChordAltH}, pair)
	if len(pair[0].Bindings) != 2 || pair[0].Bindings[0].Key != "Alt+h" {
		t.Fatalf("input mutated: %+v", pair)
	}
}
