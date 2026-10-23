package couchtty

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func stripActors() []StatusActor {
	actors := []StatusActor{hubActor("repo", "g", 0)}
	actors[0].Active = true
	for i, glyph := range []string{"+*", "±", "-", ""} {
		a := hubActor(fmt.Sprintf("repo:%d", i+1), "g", i+1)
		a.Glyph = glyph
		actors = append(actors, a)
	}
	return actors
}

func TestFoldedStateStripRendersOnlyCompleteDetail(t *testing.T) {
	actors := stripActors()
	wide := RenderStatusBar(100, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors})
	if got := rowPlain(wide.Rows[0]); got != "[repo]  +4 *±-·" {
		t.Fatalf("strip missing: %q", got)
	}
	// The same selected actors/counts must survive every scarcity width.
	for width := 1; width <= 100; width++ {
		bar := RenderStatusBar(width, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors})
		plain := rowPlain(bar.Rows[0])
		if textwidth.Width(plain) > width {
			t.Fatalf("overdraw width%d %q", width, plain)
		}
		for _, span := range bar.GroupCounts {
			text := string([]rune(plain)[span.Start:span.End])
			if strings.Contains(text, "*") && !strings.HasSuffix(text, " *±-·") {
				t.Fatalf("partial detail width%d %q", width, text)
			}
		}
	}
	// Exact base width plus separator and four markers; one less omits all detail.
	for _, tc := range []struct {
		width int
		want  string
	}{{14, "[repo]  +4"}, {15, "[repo]  +4 *±-·"}} {
		bar := RenderStatusBar(tc.width, 2, StatusModel{TabStyle: TabStyleCollapse, Actors: actors})
		if got := rowPlain(bar.Rows[0]); got != tc.want {
			t.Errorf("width%d got%q want%q", tc.width, got, tc.want)
		}
	}
}

func TestFoldedStateStripMembershipAndMultiline(t *testing.T) {
	actors := stripActors()
	actors[1].Bell = true
	base := RenderStatusBar(150, 2, StatusModel{TabStyle: TabStyleMultiline, Actors: actors})
	prepared := prepareCollapsedStatusChips(actors, nil)
	count := prepared[len(prepared)-1]
	if count.Folded != 3 || count.FoldedStateStrip != "±-·" {
		t.Fatalf("attention entered hidden strip: %+v", count)
	}
	// Frame-local detail must have no effect on the explicit compatibility mode.
	for i := range actors {
		actors[i].FoldedStateStrip = "*±-·"
	}
	after := RenderStatusBar(150, 2, StatusModel{TabStyle: TabStyleMultiline, Actors: actors})
	if !reflect.DeepEqual(base, after) {
		t.Fatal("detail altered multiline")
	}
}

func TestFoldedStateStripFollowsPresentationOrder(t *testing.T) {
	rows := []couchcore.ActionableThreadSummary{presentationSlot("/src/repo", 10), presentationSlot("/src/repo", 2), presentationSlot("/src/repo", 4)}
	git := map[string]couchcore.SlotGitStatus{}
	for _, row := range rows {
		s := row.Target.Slot
		git[s.WorktreeRoot] = couchcore.SlotGitStatus{Branch: couchcore.RestingBranch(s.Number), HasUpstream: true, Ahead: s.Number % 3, Behind: s.Number % 2, Dirty: s.Number == 4}
	}
	var actors []StatusActor
	for _, p := range PresentThreads(rows, git, "") {
		actors = append(actors, StatusActor{Label: p.Label, GroupKey: p.GroupKey, SlotNumber: p.SlotNumber, Glyph: p.Glyph})
	}
	got := prepareCollapsedStatusChips(actors, nil)
	if len(got) != 1 || got[0].Folded != 3 || got[0].FoldedStateStrip != "+*+" {
		t.Fatalf("numeric order lost: %+v", got)
	}
}

func TestFoldedStateStripAllocationPreservesBasePacking(t *testing.T) {
	actors := []StatusActor{{Folded: 6, FoldedStateStrip: "······"}, {Folded: 3, FoldedStateStrip: "*±·"}}
	for prefix := 0; prefix <= 8; prefix++ {
		for budget := 1; budget <= 40; budget++ {
			cells := []chipCell{{actor: 0, label: "+6"}, {actor: 1, label: "+3"}, {actor: -1, label: "+9 !"}}
			original := append([]chipCell(nil), cells...)
			spare := budget - prefix - chipsWidth(actors, cells, len(cells))
			if prefix > 0 {
				spare -= 2
			}
			decorateFoldedCounts(actors, cells, budget, prefix, nil)
			for i, c := range cells {
				if c.actor != original[i].actor || c.label != original[i].label {
					t.Fatal("decoration displaced base chip")
				}
				want := ""
				if i < 2 {
					cost := 1 + actors[i].Folded
					if cost <= spare {
						want = actors[i].FoldedStateStrip
						spare -= cost
					}
				}
				if c.strip != want {
					t.Fatalf("prefix%d budget%d cell%d strip%q want%q", prefix, budget, i, c.strip, want)
				}
			}
		}
	}
}

func TestFoldedStateStripControlsAndAttentionKeepTheirCells(t *testing.T) {
	for width := 1; width <= 100; width++ {
		model := StatusModel{TabStyle: TabStyleCollapse, Actors: stripActors(), Broadcast: BroadcastLive, Attention: 2}
		bar := RenderStatusBar(width, 2, model)
		// Mask only the source glyphs; identical widths must choose identical actors/counts.
		base := model
		base.Actors = append([]StatusActor(nil), model.Actors...)
		for i := range base.Actors {
			base.Actors[i].Glyph = ""
		}
		other := RenderStatusBar(width, 2, base)
		if !reflect.DeepEqual(bar.Chips, other.Chips) || bar.Rows[0].Control != other.Rows[0].Control || bar.Rows[0].Attention != other.Rows[0].Attention || len(bar.GroupCounts) != len(other.GroupCounts) {
			t.Fatalf("detail changed primary geometry width%d: %q vs%q", width, bar.Body(), other.Body())
		}
	}
}

func TestFoldedStateStripWorkScalesWithMembers(t *testing.T) {
	for _, many := range []bool{false, true} {
		for _, n := range []int{128, 256} {
			var actors []StatusActor
			for i := 0; i < n; i++ {
				group := "group"
				if many {
					group = fmt.Sprintf("group%d", i/4)
				}
				a := hubActor("repo", group, i+1)
				a.Glyph = "+*"
				actors = append(actors, a)
			}
			work := &statusLayoutWork{}
			prepared := prepareCollapsedStatusChips(actors, work)
			cells := make([]chipCell, len(prepared))
			for i, a := range prepared {
				cells[i] = chipCell{actor: i, label: a.Label}
			}
			decorateFoldedCounts(prepared, cells, n*20, 0, work)
			if work.Visits > 7*n || work.Bytes > 20*n {
				t.Fatalf("many%v n%d unbounded work%+v", many, n, work)
			}
			total := 0
			for _, a := range prepared {
				total += len([]rune(a.FoldedStateStrip))
			}
			if total != n {
				t.Fatalf("lost hidden evidence: %d want%d", total, n)
			}
		}
	}
}

func TestFoldedStateStripUsesExistingGlyphOwnerStyles(t *testing.T) {
	for _, palette := range []Palette{darkTrueColor, {FG: black, BG: white, Known: true, TrueColor: true}, {}, {NoColor: true}} {
		for _, other := range []bool{false, true} {
			var actors []StatusActor
			for i, glyph := range []string{"*", "±", ""} {
				a := hubActor("repo", "g", i+1)
				a.Glyph = glyph
				a.OtherHub = other
				actors = append(actors, a)
			}
			_, cells, err := renderChrome(ptychild.Size{Rows: 24, Cols: 100}, StatusModel{TabStyle: TabStyleCollapse, Actors: actors, Palette: palette})
			if err != nil {
				t.Fatal(err)
			}
			reference := StatusActor{Label: "reference", Glyph: "*±·", OtherHub: other}
			_, expected, err := renderChrome(ptychild.Size{Rows: 24, Cols: 100}, StatusModel{TabStyle: TabStyleMultiline, Actors: []StatusActor{reference}, Palette: palette})
			if err != nil {
				t.Fatal(err)
			}
			for _, marker := range []string{"*", "±", "·"} {
				find := func(list []terminal.Cell) terminal.Cell {
					for _, c := range list {
						if c.Content == marker {
							return c
						}
					}
					t.Fatalf("missing marker%q", marker)
					return terminal.Cell{}
				}
				if !reflect.DeepEqual(find(cells).Style, find(expected).Style) {
					t.Fatalf("marker%q owner%v palette%+v lost style", marker, other, palette)
				}
			}
		}
	}
}
