package couchtty

import (
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func panelAddress(tag string) couchcore.ThreadAddress {
	return couchcore.ThreadAddress{RepoScope: "legacy", Tag: couchcore.ThreadTag(tag)}
}

func summaries() []couchcore.ThreadSummary {
	return []couchcore.ThreadSummary{
		{Address: panelAddress("brain"), WorkingPath: "/w/brain", Name: "brain", PublishedSummary: "the advisor"},
		{Address: panelAddress("pair"), WorkingPath: "/w/pair", Name: "pair", PublishedSummary: "couch tty switching",
			Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive}}},
		{Address: panelAddress("ariadne"), WorkingPath: "/w/ariadne", PublishedSummary: "sdlc gates"},
	}
}

// Filter DELEGATES the match rule; it does not restate it. Decision 12: the
// same resolution serves the CLI, the panel and (in #148) the advisor, so a
// second copy here would drift from the one couchcore owns.
func TestPanelFilterUsesTheInjectedResolver(t *testing.T) {
	m := NewPanelModel(summaries())
	called := ""
	resolve := func(q string) []couchcore.ThreadAddress {
		called = q
		return []couchcore.ThreadAddress{panelAddress("ariadne")}
	}

	rows := m.Filter("anything", resolve)
	if called != "anything" {
		t.Fatalf("the resolver was not consulted (got %q)", called)
	}
	if len(rows) != 1 || rows[0].Tree != "/w/ariadne" {
		t.Fatalf("rows = %+v, want exactly what the resolver named", rows)
	}
}

// An empty query is not a resolution -- it is "show everything", and asking the
// resolver would make the panel's default view depend on a match rule.
func TestPanelFilterWithAnEmptyQueryShowsEverything(t *testing.T) {
	m := NewPanelModel(summaries())
	asked := false
	rows := m.Filter("", func(string) []couchcore.ThreadAddress { asked = true; return nil })
	if asked {
		t.Fatal("an empty query consulted the resolver")
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want all 3", len(rows))
	}
}

// A parked tree -- named, no live actor -- is exactly the thread this project
// exists to stop losing, so it must be listed.
func TestPanelListsParkedTrees(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("", nil)
	for _, r := range rows {
		if r.Tree == "/w/ariadne" {
			if r.Live {
				t.Fatal("a tree with no actors is marked live")
			}
			return
		}
	}
	t.Fatal("the parked tree was omitted")
}

func TestPanelNamesLiveAndParkedRowStates(t *testing.T) {
	got := RenderPanel(NewPanelModel(summaries()).Shown(), 0)
	if !strings.Contains(got, "[live] pair") {
		t.Fatalf("panel does not name the live state: %q", got)
	}
	for _, label := range []string{"brain", "ariadne"} {
		if !strings.Contains(got, "[parked] "+label) {
			t.Fatalf("panel does not name %s as parked: %q", label, got)
		}
	}
}

// Stable selection is only safe if the list does not reorder under the
// operator's cursor.
func TestPanelOrderingIsStable(t *testing.T) {
	m := NewPanelModel(summaries())
	first := m.Filter("", nil)
	for i := 0; i < 5; i++ {
		again := m.Filter("", nil)
		for j := range first {
			if first[j].Tree != again[j].Tree {
				t.Fatalf("row %d moved between refreshes: %q then %q", j, first[j].Tree, again[j].Tree)
			}
		}
	}
}

func TestPanelFilterPreservesSelectedTree(t *testing.T) {
	m := NewPanelModel(summaries())
	m.Filter("", nil)
	m.Move(2) // ariadne
	m.Filter("x", func(string) []couchcore.ThreadAddress {
		return []couchcore.ThreadAddress{panelAddress("brain"), panelAddress("ariadne")}
	})
	got, ok := m.Selected()
	if !ok || got.Tree != "/w/ariadne" {
		t.Fatalf("selected = %+v, %v; want /w/ariadne retained", got, ok)
	}
}

func TestPanelFilterFallsBackToFirstMatch(t *testing.T) {
	m := NewPanelModel(summaries())
	m.Filter("", nil)
	m.Move(1) // pair
	m.Filter("x", func(string) []couchcore.ThreadAddress {
		return []couchcore.ThreadAddress{panelAddress("ariadne")}
	})
	got, ok := m.Selected()
	if !ok || got.Tree != "/w/ariadne" {
		t.Fatalf("selected = %+v, %v; want first visible /w/ariadne", got, ok)
	}
}

func TestPanelZeroMatchesHaveNoSelection(t *testing.T) {
	m := NewPanelModel(summaries())
	m.Filter("x", func(string) []couchcore.ThreadAddress { return nil })
	if got, ok := m.Selected(); ok {
		t.Fatalf("selected = %+v, want no selection", got)
	}
}

func TestPanelSelectThreadAfterRefresh(t *testing.T) {
	m := NewPanelModel(summaries())
	if !m.SelectAddress(panelAddress("ariadne")) {
		t.Fatal("SelectAddress(ariadne) = false")
	}
	got, ok := m.Selected()
	if !ok || got.Tree != "/w/ariadne" {
		t.Fatalf("selected = %+v, %v; want /w/ariadne", got, ok)
	}
}

func TestPanelSelectTreeRefusesAmbiguousWorkingPath(t *testing.T) {
	m := NewPanelModel([]couchcore.ThreadSummary{
		{Address: panelAddress("one"), WorkingPath: "/w/brain"},
		{Address: panelAddress("two"), WorkingPath: "/w/brain"},
	})
	if m.SelectTree("/w/brain") {
		t.Fatal("SelectTree selected an ambiguous working path")
	}
	if got := m.Cursor(); got != 0 {
		t.Fatalf("cursor changed after ambiguous SelectTree: %d", got)
	}
}

// The label is what the operator reads; an unnamed tree must still be
// identifiable rather than showing an empty chip.
func TestPanelRowLabelFallsBackToTheRepo(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("", nil)
	for _, r := range rows {
		if r.Tree == "/w/ariadne" && !strings.Contains(r.Label, "ariadne") {
			t.Fatalf("an unnamed tree rendered as %q", r.Label)
		}
	}
}

// The resolver is free to return matches in any order it likes -- it is a
// lookup, not a view. The panel must impose ITS order, or the numbers under the
// operator's fingers depend on a map iteration somewhere in couchcore.
//
// Found by a deletion check that failed to fire: filtering in the resolver's
// order left every ordering test green, because the fixtures happened to agree.
func TestPanelFilterKeepsTheModelsOrderNotTheResolvers(t *testing.T) {
	m := NewPanelModel(summaries())
	rows := m.Filter("x", func(string) []couchcore.ThreadAddress {
		// Deliberately reversed relative to the model.
		return []couchcore.ThreadAddress{panelAddress("ariadne"), panelAddress("pair"), panelAddress("brain")}
	})
	want := []couchcore.Worktree{"/w/brain", "/w/pair", "/w/ariadne"}
	for i := range want {
		if rows[i].Tree != want[i] {
			t.Fatalf("row %d = %q, want %q — the panel took the resolver's order",
				i, rows[i].Tree, want[i])
		}
	}
	if got, ok := m.Selected(); !ok || got.Tree != "/w/brain" {
		t.Fatalf("selected = %+v, %v; want first displayed /w/brain", got, ok)
	}
}

func TestPanelTargetJoinKeepsParkedRowsAndAddsRoutingSeparately(t *testing.T) {
	m := NewPanelModel(summaries())
	m.BindTargets([]PanelTarget{
		{Address: panelAddress("brain"), Tree: "/w/brain", Target: "child-brain"},
		{Address: panelAddress("pair"), Tree: "/w/pair", Target: "child-pair", Bell: true},
	})

	rows := m.Rows()
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want all three summaries", len(rows))
	}
	for _, row := range rows {
		switch row.Tree {
		case "/w/ariadne":
			if row.Target != "" || row.Live {
				t.Fatalf("parked row gained a live target: %+v", row)
			}
		case "/w/pair":
			if row.Target != "child-pair" || !row.Bell {
				t.Fatalf("live target join = %+v", row)
			}
		}
	}
}

func TestPanelKeepsSamePathThreadsDistinctAndBindsExactTarget(t *testing.T) {
	first := couchcore.ThreadSummary{Address: panelAddress("first"), WorkingPath: "/w/brain", Name: "first"}
	second := couchcore.ThreadSummary{Address: panelAddress("second"), WorkingPath: "/w/brain", Name: "second"}
	m := NewPanelModel([]couchcore.ThreadSummary{first, second})
	m.BindTargets([]PanelTarget{{Address: second.Address, Tree: "/w/brain", Target: "child-second"}})
	rows := m.Rows()
	if len(rows) != 2 || rows[0].Target != "" || rows[1].Target != "child-second" {
		t.Fatalf("same-path target join = %+v", rows)
	}
	filtered := m.Filter("second", func(string) []couchcore.ThreadAddress { return []couchcore.ThreadAddress{second.Address} })
	if len(filtered) != 1 || filtered[0].Address != second.Address {
		t.Fatalf("same-path exact filter = %+v", filtered)
	}
}

// The panel may not grow a private verb. Every action it offers must be one
// couch already declares, so the operator's surface and the advisor's cannot
// drift -- the same audit the CLI has.
func TestPanelActionsAreDeclaredOperations(t *testing.T) {
	declared := map[string]bool{}
	for _, n := range couchcore.OperationNames() {
		declared[n] = true
	}
	for _, a := range PanelActions() {
		if !declared[a] {
			t.Errorf("the panel offers %q, which couch does not declare as an operation", a)
		}
	}
}

// And the panel must actually offer the actions the operator needs from it --
// an empty set would pass the audit above vacuously.
func TestPanelOffersLifecycleActions(t *testing.T) {
	got := map[string]bool{}
	for _, a := range PanelActions() {
		got[a] = true
	}
	if len(got) != 3 || !got["start"] || !got["park"] || !got["resume"] {
		t.Fatalf("panel actions = %v, want start/park/resume", got)
	}
}

// Every declared action must be REACHABLE from a keystroke.
//
// A subset check is satisfied by a list that does nothing -- which is exactly
// what shipped: four action names with no dispatch behind them, so the operator
// had no way to start a second child and the audit passed anyway.
func TestEveryPanelActionHasAKey(t *testing.T) {
	keys := PanelActionKeys()
	for _, a := range PanelActions() {
		ks, ok := keys[a]
		if !ok {
			t.Errorf("action %q has no key; it is declared but unreachable", a)
			continue
		}
		if len(ks) == 0 {
			t.Errorf("action %q has no reachable keys", a)
		}
	}
	// And no key may be claimed by two actions.
	seen := map[string]string{}
	for a, ks := range keys {
		for _, k := range ks {
			if prev, dup := seen[k]; dup {
				t.Errorf("key %q is claimed by both %q and %q", k, prev, a)
			}
			seen[k] = a
		}
	}
}

func TestPanelControlsMatchFlatContract(t *testing.T) {
	want := []PanelControl{
		{Keys: "typeahead", Action: "filter"},
		{Keys: "↑↓", Action: "select"},
		{Keys: "Enter", Action: "switch/resume"},
		{Keys: "Ctrl-Space", Action: "start"},
		{Keys: "Alt+x", Action: "park"},
		{Keys: "Escape", Action: "clear/back"},
	}
	got := PanelControls()
	if len(got) != len(want) {
		t.Fatalf("controls = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("control %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPanelRendersWithoutNumberedJumpHints(t *testing.T) {
	got := RenderPanelWithQuery("", NewPanelModel(summaries()).Shown(), 0)
	if strings.Contains(got, "▸ 1") || strings.Contains(got, ":1") || strings.Contains(got, ":s") {
		t.Fatalf("panel still advertises numbered/command jumps: %q", got)
	}
}
