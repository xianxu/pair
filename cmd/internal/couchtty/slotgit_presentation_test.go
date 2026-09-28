package couchtty

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/textwidth"
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
		// Standalone repositories use the primary resting branch too.
		loner.Address: "*",
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

// pair#319, #321: in both views ± and * share the amber attention colour, other
// glyphs keep their row's style, and the plain text and click spans are unchanged.
func TestSlotGlyphColoursInBothViews(t *testing.T) {
	for _, tc := range []struct {
		glyph           string
		diverged, dirty bool
	}{{"±", true, false}, {"±*", true, true}, {"+", false, false}, {"-", false, false}, {"*", false, true}, {"+*", false, true}, {"\ue0a0*", false, true}, {"\ue0a0", false, false}} {
		row := RenderStatusRow(80, StatusModel{Actors: []StatusActor{
			{GroupKey: "g", Label: "pair", Thread: couchcore.ThreadAddress{Tag: "a"}},
			{GroupKey: "g", SlotNumber: 1, Label: "pair:1", Glyph: tc.glyph, Active: true, Thread: couchcore.ThreadAddress{Tag: "b"}},
		}})
		if got := string(ansi.Strip([]byte(row.Body))); got != "pair  [:1"+tc.glyph+"]" {
			t.Fatalf("%s: plain tab bar = %q", tc.glyph, got)
		}
		if strings.Contains(row.Body, attentionSGR+"±\x1b[0m") != tc.diverged || strings.Contains(row.Body, attentionSGR+"*\x1b[0m") != tc.dirty {
			t.Fatalf("%s: tab bar colouring wrong (want amber ±=%v *=%v) in %q", tc.glyph, tc.diverged, tc.dirty, row.Body)
		}
		if strings.Contains(row.Body, "\x1b[38;5;196m") {
			t.Fatalf("%s: red survives in %q", tc.glyph, row.Body)
		}
		for _, other := range []string{"+", "-", "\ue0a0"} {
			if strings.Contains(row.Body, "m"+other+"\x1b[0m") {
				t.Fatalf("%s: %s was coloured in %q", tc.glyph, other, row.Body)
			}
		}
		if n := len(row.Chips); n != 2 || row.Chips[1].End-row.Chips[1].Start != textwidth.Width("[:1"+tc.glyph+"]") {
			t.Fatalf("%s: chip spans = %+v", tc.glyph, row.Chips)
		}
	}

	primary, one := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one")
	state := NewMenuState([]couchcore.ActionableThreadSummary{primary, one}, primary.Address)
	state.SlotGit = map[string]couchcore.SlotGitStatus{
		primary.StartingPath: {Branch: "main", HasUpstream: true, Ahead: 1, Behind: 2},
		one.StartingPath:     {Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 1, Behind: 2},
	}
	menu := RenderMenu(state, 100, 16, time.Unix(1800000000, 0), true)
	var slotLine string
	for _, line := range strings.Split(menu, "\n") {
		if strings.Contains(string(ansi.Strip([]byte(line))), "pair:1±*") {
			slotLine = line
		}
	}
	if !strings.Contains(slotLine, attentionSGR+"±\x1b[0m"+attentionSGR+"*\x1b[0m") {
		t.Fatalf("switcher row lacks the amber ±*: %q", slotLine)
	}
	if plain := RenderMenu(state, 100, 16, time.Unix(1800000000, 0), false); strings.Contains(plain, attentionSGR) {
		t.Fatal("switcher coloured the glyph without 256-colour support")
	}
}

func TestStandaloneGitBadgesInBothViews(t *testing.T) {
	row := groupedRow("/workspace/ducks", 0, "ducks")
	for _, tc := range []struct {
		name   string
		status couchcore.SlotGitStatus
		glyph  string
	}{
		{"ahead", couchcore.SlotGitStatus{Branch: "main", HasUpstream: true, Ahead: 1}, "+"},
		{"behind", couchcore.SlotGitStatus{Branch: "main", HasUpstream: true, Behind: 1}, "-"},
		{"diverged dirty", couchcore.SlotGitStatus{Branch: "main", HasUpstream: true, Ahead: 1, Behind: 1, Dirty: true}, "±*"},
		{"issue dirty", couchcore.SlotGitStatus{Branch: "feature", Dirty: true}, "\ue0a0*"},
		{"clean", couchcore.SlotGitStatus{Branch: "main", HasUpstream: true}, ""},
		{"no upstream", couchcore.SlotGitStatus{Branch: "main", Ahead: 1}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), strings.NewReader(""))
			con.menu = NewMenuState([]couchcore.ActionableThreadSummary{row}, row.Address)
			con.menu.SlotGit = map[string]couchcore.SlotGitStatus{row.StartingPath: tc.status}
			con.order = []string{"ducks"}
			con.panes = map[string]*pane{"ducks": {thread: row.Address, tree: couchcore.Worktree(row.StartingPath), label: "ducks"}}
			model := con.statusModelLocked()
			if len(model.Actors) != 1 || model.Actors[0].Glyph != tc.glyph {
				t.Fatalf("actors=%+v want glyph %q", model.Actors, tc.glyph)
			}
			for _, view := range []string{RenderMenu(con.menu, 100, 16, time.Now(), false), RenderStatusRow(100, model).Body} {
				if !strings.Contains(string(ansi.Strip([]byte(view))), "ducks"+tc.glyph) {
					t.Fatalf("missing badge in %q", view)
				}
			}
		})
	}
}
