package couchtty

import (
	"reflect"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestPresentThreadsSlotGlyphScope(t *testing.T) {
	primary, one, two := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")
	second := groupedRow("/workspace/pair", 0, "second")
	loner := groupedRow("/workspace/solo", 0, "solo")
	dirty := func(branch string) couchcore.SlotGitStatus {
		return couchcore.SlotGitStatus{Branch: branch, Dirty: true}
	}
	git := map[string]couchcore.SlotGitStatus{
		primary.StartingPath: dirty("main"),
		one.StartingPath:     dirty("main-slot1"),
		loner.StartingPath:   dirty("main"),
	}
	glyphs := map[couchcore.ThreadAddress]string{}
	for _, p := range PresentThreads([]couchcore.ActionableThreadSummary{two, loner, one, second, primary}, git) {
		glyphs[p.Row.Address] = p.Glyph
	}
	want := map[couchcore.ThreadAddress]string{
		// Every thread in the primary checkout of a slot group reads as :0.
		primary.Address: "*", second.Address: "*",
		one.Address: "*",
		// No observation, no guess.
		two.Address: "",
		// An ordinary repo without slots has no resting branch.
		loner.Address: "",
	}
	if !reflect.DeepEqual(glyphs, want) {
		t.Fatalf("glyphs = %q, want %q", glyphs, want)
	}
	// The resting branch is the slot's own: main-slot1 checked out in :2 is
	// issue work there, not rest.
	git[two.StartingPath] = couchcore.SlotGitStatus{Branch: "main-slot1"}
	for _, p := range PresentThreads([]couchcore.ActionableThreadSummary{two}, git) {
		if p.Glyph != "" {
			t.Fatalf(":2 on main-slot1 glyph = %q", p.Glyph)
		}
	}
}

func TestReduceMenuSlotGitKeepsLastValueOnFailure(t *testing.T) {
	state := NewMenuState(nil, couchcore.ThreadAddress{})
	state.SlotGit = map[string]couchcore.SlotGitStatus{
		"/a":    {Branch: "main-slot1", Dirty: true},
		"/b":    {Branch: "main-slot2"},
		"/gone": {Branch: "main-slot3"},
	}
	before := state.SlotGit
	next, effects := ReduceMenu(state, MenuEvent{
		Kind:          MenuEventSlotGit,
		SlotGit:       map[string]couchcore.SlotGitStatus{"/a": {Branch: "main-slot1"}},
		SlotGitFailed: map[string]bool{"/b": true, "/new": true},
	})
	if len(effects) != 0 {
		t.Fatalf("effects = %+v", effects)
	}
	want := map[string]couchcore.SlotGitStatus{"/a": {Branch: "main-slot1"}, "/b": {Branch: "main-slot2"}}
	if !reflect.DeepEqual(next.SlotGit, want) {
		t.Fatalf("SlotGit = %+v, want %+v", next.SlotGit, want)
	}
	if !before["/a"].Dirty || len(before) != 3 {
		t.Fatal("reducer mutated the prior state's map")
	}
}
