package keyhelp

import "github.com/xianxu/pair/cmd/internal/workbenchshortcut"

// Layer stacks a host's help above Pair's (#282). A chord the host claims never
// reaches Pair, so every Pair row documenting it is dropped -- in every
// context, since a host claims a chord from every pane. Other rows are
// untouched; a Pair section left empty is dropped. Pure and host-agnostic:
// Pair learns which chords it gives up, never who took them. Inputs are not
// modified.
func Layer(host []Section, claimed []workbenchshortcut.Chord, pair []Section) []Section {
	taken := map[workbenchshortcut.Chord]bool{}
	for _, c := range claimed {
		if c != 0 {
			taken[c] = true
		}
	}
	out := append([]Section(nil), host...)
	for _, s := range pair {
		kept := make([]Binding, 0, len(s.Bindings))
		for _, b := range s.Bindings {
			if b.Chord == 0 || !taken[b.Chord] {
				kept = append(kept, b)
			}
		}
		if len(kept) > 0 {
			out = append(out, Section{Title: s.Title, Bindings: kept})
		}
	}
	return out
}
