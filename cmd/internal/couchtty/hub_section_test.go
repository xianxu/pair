package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// pair#491: the hub leads its section and is marked by position only.

func hubRows() []couchcore.ActionableThreadSummary {
	return []couchcore.ActionableThreadSummary{
		presentationSlot("/src/pair", 2), presentationOrdinary("/src/pair", "p"), presentationSlot("/src/pair", 1),
		presentationOrdinary("/src/ops", "o"), presentationSlot("/src/ops", 1),
		presentationSlot("/src/ariadne", 1), presentationOrdinary("/src/ariadne", "a"),
		presentationOrdinary("/src/blog", "b"),
	}
}

func presentationLabels(entries []ThreadPresentation) (labels []string, indents []int, hubs []bool) {
	for _, e := range entries {
		labels, indents, hubs = append(labels, e.Label), append(indents, e.Indent), append(hubs, e.Hub)
	}
	return labels, indents, hubs
}

func TestPresentThreadsLeadsWithTheHubAndIndentsItsSection(t *testing.T) {
	cases := []struct {
		name, hub string
		labels    string
		indents   []int
	}{
		{"no hub", "", "ariadne ariadne:1 blog ops ops:1 pair pair:1 pair:2", []int{0, 2, 0, 0, 2, 0, 2, 2}},
		{"hub is a primary", "/src/ops", "ops ariadne ariadne:1 blog ops:1 pair pair:1 pair:2", []int{0, 2, 4, 2, 4, 2, 4, 4}},
		{"hub is a slot", presentationSlot("/src/ops", 1).Target.Slot.WorktreeRoot, "ops:1 ariadne ariadne:1 blog ops pair pair:1 pair:2", []int{0, 2, 4, 2, 2, 2, 4, 4}},
		{"hub not in the inventory", "/src/elsewhere", "ariadne ariadne:1 blog ops ops:1 pair pair:1 pair:2", []int{0, 2, 0, 0, 2, 0, 2, 2}},
	}
	for _, c := range cases {
		labels, indents, hubs := presentationLabels(PresentThreads(hubRows(), nil, c.hub))
		if got := strings.Join(labels, " "); got != c.labels {
			t.Fatalf("%s: order %q, want %q", c.name, got, c.labels)
		}
		for i := range indents {
			if indents[i] != c.indents[i] {
				t.Fatalf("%s: indents %v, want %v", c.name, indents, c.indents)
			}
			if hubs[i] != (c.hub != "" && i == 0 && strings.HasPrefix(c.name, "hub is")) {
				t.Fatalf("%s: hub flags %v", c.name, hubs)
			}
		}
	}
}

// The switcher's order is the inventory order, installed by the reducer: the
// hub event re-sorts what is already installed, and later inventories keep it.
func TestMenuHubEventOrdersTheSwitcher(t *testing.T) {
	state := NewMenuState(hubRows(), couchcore.ThreadAddress{})
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHub, Hub: "/src/ops"})
	first := func(s MenuState) string { return PresentThreads(menuRows(s)[:1], nil, "")[0].Label }
	if got := first(state); got != "ops" {
		t.Fatalf("switcher leads with %q after the hub event, want ops", got)
	}
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Inventory: hubRows()})
	if got := first(state); got != "ops" || state.Hub != "/src/ops" {
		t.Fatalf("a refreshed inventory leads with %q (hub %q)", got, state.Hub)
	}
}

func TestSwitcherIndentsTheHubSection(t *testing.T) {
	state := NewMenuState(hubRows(), couchcore.ThreadAddress{})
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventHub, Hub: "/src/ops"})
	lines, _ := renderRootMenuFrame(state, state.CurrentFrame(), 120, 40, time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC), false)
	body := strings.Join(lines, "\n")
	// The cursor stays on the row it was on (ariadne); "▸ " and "  " are the
	// marker column, then the indent.
	for _, want := range []string{"\n  ops  /src/ops", "▸   ariadne  ", "\n      ariadne:1", "\n    blog  ", "\n      ops:1", "\n    pair  ", "\n      pair:2"} {
		if !strings.Contains(body, want) {
			t.Fatalf("switcher missing %q:\n%s", want, body)
		}
	}
}

func hubActor(label, group string, n int) StatusActor {
	return StatusActor{Label: label, GroupKey: group, SlotNumber: n, Thread: menuAddress(label)}
}

func hubModel() []StatusActor {
	hub := hubActor("ops", "ops", 0)
	hub.Hub = true
	return []StatusActor{hub,
		hubActor("ariadne", "ariadne", 0), hubActor("ariadne:1", "ariadne", 1), hubActor("ariadne:2", "ariadne", 2),
		hubActor("blog", "blog", 0),
		hubActor("pair", "pair", 0), hubActor("pair:1", "pair", 1), hubActor("pair:2", "pair", 2),
	}
}

func TestRenderStatusRowHubSection(t *testing.T) {
	if got, want := RenderStatusRow(200, StatusModel{Actors: hubModel()}).Body, "ops  ariadne  :1  :2  blog  pair  :1  :2"; got != want {
		t.Fatalf("row %q, want %q", got, want)
	}
	// A numbered hub stands alone: its repository's next slot is not its :N.
	hub := hubActor("ops:1", "ops", 1)
	hub.Hub = true
	actors := []StatusActor{hub, hubActor("ops:2", "ops", 2)}
	if got, want := RenderStatusRow(200, StatusModel{Actors: actors}).Body, "ops:1  ops:2"; got != want {
		t.Fatalf("numbered hub: %q, want %q", got, want)
	}
	// A second hub starts a section after a separator; the active chip keeps [].
	second := hubActor("ops:1", "ops", 1)
	second.Hub = true
	actors = append(hubModel()[:2], second, hubActor("pair:3", "pair", 3), hubActor("pair:4", "pair", 4))
	actors[1].Active = true
	if got, want := RenderStatusRow(200, StatusModel{Actors: actors}).Body, "ops  [ariadne] │ ops:1  pair:3  :4"; got != want {
		t.Fatalf("two sections: %q, want %q", got, want)
	}
}

func rowPlain(r RenderedStatusRow) string { return string(ansi.Strip([]byte(r.Body))) }

func TestRenderStatusRowNarrowCollapsesFromTheRightAndKeepsRed(t *testing.T) {
	full := "ops  ariadne  :1  :2  blog  pair  :1  :2"
	cases := []struct {
		width int
		red   bool
		want  string
	}{
		{len(full), false, full},
		{len(full) - 1, false, "ops  ariadne  :1  :2  blog  pair  +2"},
		{len("ops  ariadne  +2  blog  pair  +2"), false, "ops  ariadne  +2  blog  pair  +2"},
		// pair:2 behind a picker stays drawn, and it stays :2 after its leader.
		{len(full) - 1, true, "ops  ariadne  +2  blog  pair  :2  +1"},
		// Too narrow even collapsed: groups fold until the red chip is in view,
		// and with its leader gone it shows its full label.
		{len("ops  pair:2  +6"), true, "ops  pair:2  +6"},
	}
	for _, c := range cases {
		actors := hubModel()
		actors[7].Blocked = c.red
		row := RenderStatusRow(c.width, StatusModel{Actors: actors})
		if got := rowPlain(row); got != c.want {
			t.Fatalf("width %d red %v: %q, want %q", c.width, c.red, got, c.want)
		}
		// Spans come from the drawing pass: each chip maps to its own thread,
		// and a count maps to nobody.
		plain := rowPlain(row)
		for _, chip := range row.Chips {
			if !strings.HasSuffix(string(chip.Thread.Tag), strings.TrimPrefix(plain[chip.Start:chip.End], ":")) {
				t.Fatalf("width %d: span %q is not %v", c.width, plain[chip.Start:chip.End], chip.Thread)
			}
		}
		if at := strings.LastIndex(plain, "+"); at >= 0 {
			if _, ok := row.ColumnToActor(at); ok {
				t.Fatalf("width %d: the count is clickable", c.width)
			}
		}
	}
}

// A group with no :0 keeps its first member when it collapses, so its count
// reads as its own (`ops:1  +2`), not as the repository before it.
func TestRenderStatusRowCollapsedCountFollowsItsOwnRepository(t *testing.T) {
	hub := hubActor("ops", "ops", 0)
	hub.Hub = true
	actors := []StatusActor{hub, hubActor("blog", "blog", 0), hubActor("ops:1", "ops", 1), hubActor("ops:2", "ops", 2), hubActor("ops:3", "ops", 3)}
	full := "ops  blog  ops:1  :2  :3"
	if got := rowPlain(RenderStatusRow(80, StatusModel{Actors: actors})); got != full {
		t.Fatalf("full row %q, want %q", got, full)
	}
	if got, want := rowPlain(RenderStatusRow(len(full)-1, StatusModel{Actors: actors})), "ops  blog  ops:1  +2"; got != want {
		t.Fatalf("collapsed row %q, want %q", got, want)
	}
}

// A count that doesn't fit whole is left out, never cut to a bare `+`.
func TestRenderStatusRowDropsACountThatDoesNotFit(t *testing.T) {
	collapsed := "ops  ariadne  +2  blog  pair  +2"
	for width := len(collapsed) - 2; width < len(collapsed); width++ {
		got := rowPlain(RenderStatusRow(width, StatusModel{Actors: hubModel()}))
		if strings.HasSuffix(got, "+") || strings.HasSuffix(got, " ") {
			t.Fatalf("width %d: %q ends in a cut count", width, got)
		}
	}
}

// The hub set at startup survives the menu's lazy first build and leads the
// tab bar (pair#491): Couch may hand it over before any thread attaches.
func TestConsoleHubLeadsTheTabBarAcrossTheFirstMenuBuild(t *testing.T) {
	c := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(c.Stop)
	c.SetHub("/src/pair")
	ops, pair := presentationOrdinary("/src/ops", "o"), presentationOrdinary("/src/pair", "p")
	c.attachThreadActor("ops", "ops", ops.Address, "/src/ops", "ops", ptychild.NewFakeChild(nil))
	c.attachThreadActor("pair", "pair", pair.Address, "/src/pair", "pair", ptychild.NewFakeChild(nil))
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureMenuLocked(ops.Address)
	c.menu, _ = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventInventory, Inventory: []couchcore.ActionableThreadSummary{ops, pair}})
	actors := c.statusModelLocked().Actors
	if len(actors) != 2 || actors[0].Label != "pair" || !actors[0].Hub || actors[1].Hub {
		t.Fatalf("tab bar %+v, want the hub pair first", actors)
	}
}
