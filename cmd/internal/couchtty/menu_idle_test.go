package couchtty

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
)

var idleNow = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

// idleMenuState has a primary checkout and its dirty, diverged slot :1, both
// live, with the primary selected so the slot row is the one under test.
func idleMenuState(t *testing.T) (MenuState, couchcore.ActionableThreadSummary) {
	t.Helper()
	primary, one := groupedRow("/workspace/pair", 0, "primary"), groupedRow("/workspace/pair", 1, "one")
	state := NewMenuState([]couchcore.ActionableThreadSummary{primary, one}, primary.Address)
	state.SlotGit = map[string]couchcore.SlotGitStatus{
		one.StartingPath: {Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 1, Behind: 2},
	}
	state.Palette = darkTrueColor
	return state, one
}

func menuLineContaining(t *testing.T, menu, plain string) string {
	t.Helper()
	for _, line := range strings.Split(menu, "\n") {
		if strings.Contains(string(ansi.Strip([]byte(line))), plain) {
			return line
		}
	}
	t.Fatalf("no line shows %q in %q", plain, menu)
	return ""
}

func TestSwitcherFadesAnIdleLiveRowAndItsAmberGlyphs(t *testing.T) {
	state, one := idleMenuState(t)
	for _, tc := range []struct {
		age   time.Duration
		level IdleLevel
	}{{30 * time.Hour, IdleDay}, {4 * 24 * time.Hour, IdleStale}} {
		state.Activity = map[couchcore.ThreadAddress]time.Time{one.Address: idleNow.Add(-tc.age)}
		line := menuLineContaining(t, RenderMenu(state, 100, 16, idleNow, true), "pair:1±*")
		label, amber := FadeStyle(darkTrueColor, tc.level, baseDefault), FadeStyle(darkTrueColor, tc.level, baseAmber)
		if !strings.HasPrefix(line, label) {
			t.Fatalf("%s: row is not wrapped in the level-%d fade: %q", tc.age, tc.level, line)
		}
		if !strings.Contains(line, amber+"±\x1b[0m"+label+amber+"*\x1b[0m"+label) {
			t.Fatalf("%s: glyphs are not the faded amber, restoring the row fade: %q", tc.age, line)
		}
		if strings.Contains(line, attentionSGR) {
			t.Fatalf("%s: full-strength amber survives on an idle row: %q", tc.age, line)
		}
	}
}

// A fresh row, or one no pass has reached yet, draws what it drew before #247.
func TestSwitcherFreshOrUnprobedLiveRowIsUnchanged(t *testing.T) {
	state, one := idleMenuState(t)
	before := state
	before.Palette = Palette{}
	want := menuLineContaining(t, RenderMenu(before, 100, 16, idleNow, true), "pair:1±*")
	for name, activity := range map[string]map[couchcore.ThreadAddress]time.Time{
		"unprobed": nil,
		"fresh":    {one.Address: idleNow.Add(-2 * time.Hour)},
	} {
		state.Activity = activity
		if got := menuLineContaining(t, RenderMenu(state, 100, 16, idleNow, true), "pair:1±*"); got != want {
			t.Fatalf("%s: row = %q, want the pre-#247 %q", name, got, want)
		}
	}
}

// The selected row and a row with attention keep their emphasis however idle.
func TestSwitcherFadeYieldsToSelectionAndAttention(t *testing.T) {
	state, one := idleMenuState(t)
	state.Activity = map[couchcore.ThreadAddress]time.Time{one.Address: idleNow.Add(-5 * 24 * time.Hour)}
	faded := FadeStyle(darkTrueColor, IdleStale, baseDefault)

	selected := state
	selected.Frames = append([]MenuFrame(nil), state.Frames...)
	selected.Frames[0].SelectedAddress = one.Address
	selected.Frames[0].SelectedKey = one.RowKey
	if line := menuLineContaining(t, RenderMenu(selected, 100, 16, idleNow, true), "pair:1±*"); strings.Contains(line, faded) {
		t.Fatalf("the selected row faded: %q", line)
	}

	attended := state
	attended.Attention = map[couchcore.ThreadAddress][]AttentionMessage{one.Address: {{Text: "turn complete"}}}
	if line := menuLineContaining(t, RenderMenu(attended, 100, 16, idleNow, true), "pair:1±*"); strings.Contains(line, faded) {
		t.Fatalf("a row with attention faded: %q", line)
	}
}

// Non-live rows keep their own age ramp; idle fading is for live rows only.
func TestSwitcherNonLiveRowKeepsItsAgeRamp(t *testing.T) {
	state, one := idleMenuState(t)
	parked := one
	parked.State = couchcore.ThreadParked
	parked.LastActiveAt = idleNow.Add(-5 * 24 * time.Hour)
	primary := state.Inventory[0]
	state = NewMenuState([]couchcore.ActionableThreadSummary{primary, parked}, primary.Address)
	state.Palette = darkTrueColor
	state.Activity = map[couchcore.ThreadAddress]time.Time{parked.Address: idleNow.Add(-5 * 24 * time.Hour)}
	line := menuLineContaining(t, RenderMenu(state, 100, 16, idleNow, true), "pair:1")
	if !strings.HasPrefix(line, ageColor(AgeBandFor(idleNow, parked.LastActiveAt))) || strings.Contains(line, "38;2;") {
		t.Fatalf("parked row lost its age ramp or gained a fade: %q", line)
	}
}
