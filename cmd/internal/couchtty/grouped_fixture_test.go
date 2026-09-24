package couchtty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

func TestGroupedRenderedFixtures(t *testing.T) {
	for _, scenario := range []string{"normal", "absent_primary", "parked", "filtered", "narrow", "glyph_branch", "glyph_dirty", "glyph_ahead", "glyph_clean", "glyph_narrow"} {
		t.Run(scenario, func(t *testing.T) {
			primary, one, two := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")
			rows := []couchcore.ActionableThreadSummary{two, primary, one}
			if scenario == "absent_primary" {
				rows = []couchcore.ActionableThreadSummary{two, one}
			}
			if scenario == "parked" {
				rows[0].State = couchcore.ThreadParked
			}
			state := NewMenuState(rows, one.Address)
			if scenario == "filtered" {
				state.Frames[0].Filter = ":2"
			}
			state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{one.Address: {{Sequence: 1, Text: "review ready"}}}
			width := 100
			if scenario == "narrow" || scenario == "glyph_narrow" {
				width = 40
			}
			if glyphs, ok := groupedGlyphScenarios(primary, one, two)[scenario]; ok {
				state.SlotGit = glyphs
			}
			con := New(hostty.NewFakeHost(ptychild.Size{Rows: 16, Cols: uint16(width)}), strings.NewReader(""))
			con.menu = state
			con.panes = map[string]*pane{}
			for _, r := range rows {
				if r.State != couchcore.ThreadLive {
					continue
				}
				id := string(r.Address.Tag)
				con.order = append(con.order, id)
				con.panes[id] = &pane{tree: couchcore.Worktree(r.StartingPath), thread: r.Address, label: "pair"}
			}
			con.active = "one"
			con.attention.Mark(one.Address, "review ready")
			body := RenderMenu(state, width, 16, time.Unix(1800000000, 0), false) + "\nTAB BAR\n" + RenderStatusRow(width, con.statusModelLocked()).Body + "\n"
			plain := strings.ReplaceAll(string(ansi.Strip([]byte(body))), "\r\n", "\n")
			path := filepath.Join("testdata", "slots_grouped_"+scenario+".txt")
			if os.Getenv("PAIR_UPDATE_GROUPED_FIXTURES") == "1" {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(plain), 0644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if plain != string(want) {
				t.Fatalf("fixture %s differs:\n%s", path, plain)
			}
			if strings.Contains(plain, "ariadne") {
				t.Fatal("incidental dependency clone appeared")
			}
		})
	}
}

func TestGroupedStatusPlaceholderKeepsGroupPosition(t *testing.T) {
	primary, one, two := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one"), groupedRow("/workspace/pair", 2, "two")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100}), strings.NewReader(""))
	con.menu = NewMenuState([]couchcore.ActionableThreadSummary{two, one, primary}, primary.Address)
	con.menu.Reattach = ReattachPass{Phase: ReattachRunning, Loading: one.Address}
	con.panes = map[string]*pane{"primary": {thread: primary.Address, tree: couchcore.Worktree(primary.StartingPath), label: "pair"}, "two": {thread: two.Address, tree: couchcore.Worktree(two.StartingPath), label: "pair"}}
	con.order = []string{"two", "primary"}
	before := con.statusModelLocked()
	if len(before.Actors) != 3 || before.Actors[1].Thread != one.Address || !before.Actors[1].Loading || !before.Actors[1].Placeholder {
		t.Fatalf("pending order: %+v", before.Actors)
	}
	for _, chip := range RenderStatusRow(100, before).Chips {
		if chip.Thread == one.Address {
			t.Fatal("placeholder clickable")
		}
	}
	// Attach becomes observable before either the pass or inventory has refreshed.
	con.panes["one"] = &pane{thread: one.Address, tree: couchcore.Worktree(one.StartingPath), label: "pair"}
	con.order = append(con.order, "one")
	after := con.statusModelLocked()
	if len(after.Actors) != 3 || after.Actors[1].Thread != one.Address || after.Actors[1].Placeholder {
		t.Fatalf("attached order: %+v", after.Actors)
	}
}

// groupedGlyphScenarios pins each slot quick-status state (pair#317) in both the
// switcher and the tab bar. Every scenario observes all three checkouts so a
// glyph that lands on the wrong row cannot hide behind a missing observation.
func groupedGlyphScenarios(primary, one, two couchcore.ActionableThreadSummary) map[string]map[string]couchcore.SlotGitStatus {
	clean := func(branch string) couchcore.SlotGitStatus {
		return couchcore.SlotGitStatus{Branch: branch, HasUpstream: true}
	}
	with := func(slot1 couchcore.SlotGitStatus) map[string]couchcore.SlotGitStatus {
		return map[string]couchcore.SlotGitStatus{primary.StartingPath: clean("main"), one.StartingPath: slot1, two.StartingPath: clean("main-slot2")}
	}
	return map[string]map[string]couchcore.SlotGitStatus{
		"glyph_branch": with(couchcore.SlotGitStatus{Branch: "000317-slot-quick-status-glyph", Dirty: true, HasUpstream: true}),
		"glyph_dirty":  with(couchcore.SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 2}),
		"glyph_ahead":  with(couchcore.SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 2}),
		"glyph_clean":  with(clean("main-slot1")),
		"glyph_narrow": {
			primary.StartingPath: {Branch: "main", HasUpstream: true, Ahead: 1},
			one.StartingPath:     {Detached: true},
			two.StartingPath:     {Branch: "main-slot2", Dirty: true},
		},
	}
}
